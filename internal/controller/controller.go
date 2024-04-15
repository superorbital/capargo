package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superorbital/capargo/pkg/providers"
	"github.com/superorbital/capargo/pkg/types"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/kubeclient"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/kubetypes"
	istiolog "istio.io/istio/pkg/log"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/api/v1beta1"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Logger options
var logger = istiolog.RegisterScope("capargo-controller", "")

func GetArgoConfigFromSecret() types.ClusterConfig {
	return types.ClusterConfig{}
}

func LoadRestConfigFromSecret() rest.Config {
	return rest.Config{}
}

func GetCluster() v1beta1.Cluster {
	return v1beta1.Cluster{}
}

func NewCollection(ctx context.Context, client kube.Client, options types.Options) krt.Collection[corev1.Secret] {
	// Create CAPI cluster informer
	r := strings.ToLower(fmt.Sprintf("%ss", v1beta1.ClusterKind))
	gvr := v1beta1.GroupVersion.WithResource(r)
	kubeclient.Register[*unstructured.Unstructured](kubeclient.NewTypeRegistration[*unstructured.Unstructured](
		v1.SchemeGroupVersion.WithResource(r),
		config.GroupVersionKind{
			Group:   gvr.Group,
			Version: gvr.Version,
			Kind:    v1beta1.ClusterKind,
		},
		&unstructured.Unstructured{},
		func(c kubeclient.ClientGetter, o kubetypes.InformerOptions) cache.ListerWatcher {
			cs := c.Dynamic().Resource(gvr).Namespace(o.Namespace)
			return &cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					options.FieldSelector = o.FieldSelector
					options.LabelSelector = o.LabelSelector
					return cs.List(ctx, options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					options.FieldSelector = o.FieldSelector
					options.LabelSelector = o.LabelSelector
					return cs.Watch(ctx, options)
				},
				DisableChunking: true,
			}
		},
	))
	capiClusters := krt.NewInformer[*unstructured.Unstructured](client, krt.WithName("cluster-api-clusters"))

	// Create the CAPI cluster secrets informer
	capiSecrets := krt.NewInformerFiltered[*corev1.Secret](
		client,
		kubetypes.Filter{
			Namespace: options.ClusterNamespace,
			ObjectFilter: kubetypes.NewStaticObjectFilter(func(obj any) bool {
				_, ok := obj.(*corev1.Secret)
				if !ok {
					logger.Error("Failed to parse secret for object filter")
					return false
				}
				return true
			}),
		}, krt.WithName("cluster-api-secrets"))

	// Create transformation function -- CAPI cluster + secret to ArgoCD secret.
	recompute := krt.NewRecomputeTrigger()
	coll := krt.NewCollection[*corev1.Secret, corev1.Secret](capiSecrets, func(ctx krt.HandlerContext, s *corev1.Secret) *corev1.Secret {
		obj := krt.FetchOne[*unstructured.Unstructured](ctx, capiClusters)
		if obj == nil {
			return nil
		}
		var cluster v1beta1.Cluster
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured((*obj).Object, &cluster); err != nil {
			logger.Errorf("Could not convert cluster %s/%s, %v", (*obj).GetNamespace(), (*obj).GetName(), err)
			return nil
		}

		if !providers.IsCapiKubeconfig(s, &cluster) {
			return nil
		}

		configBytes, ok := s.Data["value"]
		if !ok {
			logger.Errorf("Secret %s/%s does not contain key \"value\" for cluster %s",
				s.Namespace, s.Name, cluster.Name)
			return nil
		}
		recompute.MarkDependant(ctx)

		// Create kubeconfig credentials from cluster secret
		config, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
		if err != nil {
			logger.Errorf("Failed to build restconfig from the value in secret %s/%s for cluster %s",
				s.Namespace, s.Name,
				cluster.Name,
			)
			return nil
		}

		// Build the ArgoCD secret
		clusterConfig := buildClusterConfigFromRestConfig(config)
		ccJson, err := json.Marshal(clusterConfig)
		if err != nil {
			logger.Errorf("Could not marshal cluster config for cluster %s/%s",
				cluster.Namespace, cluster.Name)
			return nil
		}

		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name,
				Namespace: options.ArgoNamespace,
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "cluster",
					"controller":                     "capargo",
				},
			},
			StringData: map[string]string{
				"name":   cluster.Name,
				"server": config.Host,
				"config": string(ccJson),
			},
		}
	}, krt.WithName("argo-secret-collection"))

	coll.Register(func(o krt.Event[corev1.Secret]) {
		tryUpdate := false
		if o.Event == controllers.EventAdd || o.Event == controllers.EventUpdate {
			cctx, cancel := context.WithTimeout(ctx, options.Timeout)
			defer cancel()
			_, err := client.Kube().CoreV1().Secrets(o.New.Namespace).Create(cctx, o.New, metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					tryUpdate = true
				} else {
					logger.Errorf("Failed to create ArgoCD secret %s/%s: %v",
						o.New.Namespace, o.New.Name, err,
					)
				}
			}
		}
		if tryUpdate {
			cctx, cancel := context.WithTimeout(ctx, options.Timeout)
			defer cancel()
			_, err := client.Kube().CoreV1().Secrets(o.New.Namespace).Update(cctx, o.New, metav1.UpdateOptions{})
			if err != nil {
				logger.Errorf("Failed to update ArgoCD secret %s/%s: %v. Waiting 15 seconds then recomputing",
					o.New.Namespace, o.New.Name, err,
				)
				go func() {
					time.Sleep(15 * time.Second)
					recompute.TriggerRecomputation()
				}()
			}
		} else if o.Event == controllers.EventDelete {
			cctx, cancel := context.WithTimeout(ctx, options.Timeout)
			defer cancel()
			err := client.Kube().CoreV1().Secrets(o.Old.Namespace).Delete(cctx, o.Old.Name, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				logger.Errorf("Failed to delete ArgoCD secret %s/%s: %v. Waiting 15 seconds then recomputing",
					o.Old.Namespace, o.Old.Name, err,
				)
				go func() {
					time.Sleep(15 * time.Second)
					recompute.TriggerRecomputation()
				}()
			}
		}
	})

	return coll
}

func buildClusterConfigFromRestConfig(config *rest.Config) types.ClusterConfig {
	var cc types.ClusterConfig
	if config.Username != "" {
		cc.Username = config.Username
		cc.Password = config.Password
	}
	if config.BearerToken != "" {
		cc.BearerToken = config.BearerToken
	}
	// TODO: I have hardcoded insecure and removing cadata due to the oddities of the capi vcluster provider
	tlsClientConfig := types.TLSClientConfig{
		Insecure:   true,
		ServerName: config.TLSClientConfig.ServerName,
		//CAData:     config.TLSClientConfig.CAData,
		CertData: config.TLSClientConfig.CertData,
		KeyData:  config.TLSClientConfig.KeyData,
	}

	cc.TLSClientConfig = tlsClientConfig

	// TODO: AWS Auth Config

	if config.ExecProvider != nil {
		execProviderConfig := &types.ExecProviderConfig{
			Command:     config.ExecProvider.Command,
			Args:        config.ExecProvider.Args,
			Env:         mapEnv(config.ExecProvider.Env),
			APIVersion:  config.ExecProvider.APIVersion,
			InstallHint: config.ExecProvider.InstallHint,
		}
		cc.ExecProviderConfig = execProviderConfig
	}
	return cc
}

func mapEnv(envVar []api.ExecEnvVar) map[string]string {
	outputMap := make(map[string]string, len(envVar))
	for _, env := range envVar {
		outputMap[env.Name] = env.Value
	}
	return outputMap
}
