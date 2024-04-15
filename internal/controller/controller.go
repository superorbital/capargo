package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/superorbital/capargo/pkg/types"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/kclient"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/kubetypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/api/v1beta1"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetArgoConfigFromSecret() types.ClusterConfig {
	return types.ClusterConfig{}
}

func LoadRestConfigFromSecret() rest.Config {
	return rest.Config{}
}

func GetCluster() v1beta1.Cluster {
	return v1beta1.Cluster{}
}

func NewCollection(client kube.Client, options types.Options) krt.Collection[corev1.Secret] {
	recompute := krt.NewRecomputeTrigger()

	capiClusterColl := krt.WrapClient[controllers.Object](kclient.NewDynamic(client, v1beta1.GroupVersion.WithResource(strings.ToLower(fmt.Sprintf("%ss", v1beta1.ClusterKind))), kubetypes.Filter{}))
	capisecretColl := krt.NewInformerFiltered[*corev1.Secret](client, kubetypes.Filter{
		Namespace: options.ClusterNamespace,
		ObjectFilter: kubetypes.NewStaticObjectFilter(func(obj any) bool {
			s, ok := obj.(*corev1.Secret)
			if !ok {
				slog.Warn("Failed to parse secret for object filter")
				return false
			}
			return strings.HasSuffix(s.Name, "-kubeconfig")
		}),
	}, krt.WithName("cluster-api-secrets"))
	coll := krt.NewCollection[*corev1.Secret, corev1.Secret](capisecretColl, func(ctx krt.HandlerContext, c *corev1.Secret) *corev1.Secret {
		cluster := krt.FetchOne[controllers.Object](ctx, capiClusterColl)
		if cluster == nil {
			return nil
		}
		slog.Warn("Got cluster", "name", (*cluster).GetName())
		recompute.MarkDependant(ctx)
		clusterSecret := c
		clusterName := strings.TrimSuffix(c.Name, "-kubeconfig")
		configBytes, ok := clusterSecret.Data["value"]
		if !ok {
			slog.Warn("Cluster secret does not contain key \"value\"", "secretName", clusterSecret.Name, "clusterName", clusterName)
		}

		config, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
		if err != nil {
			slog.Error("Failed to build rest config from kubeconfig for cluster API cluster", "name", c.Name)
		}

		clusterConfig := buildClusterConfigFromRestConfig(config)

		ccJson, err := json.Marshal(clusterConfig)
		if err != nil {
			slog.Error("Failed to prepare cluster config for JSON marshal", "name", c.Name)
		}

		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: options.ArgoNamespace,
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "cluster",
					"controller":                     "capargo",
				},
			},
			StringData: map[string]string{
				"name":   clusterName,
				"server": config.Host,
				"config": string(ccJson),
			},
		}
	}, krt.WithName("argo-secret-collection"))
	coll.Register(func(o krt.Event[corev1.Secret]) {
		tryUpdate := false
		if o.Event == controllers.EventAdd {
			_, err := client.Kube().CoreV1().Secrets(o.New.Namespace).Create(context.TODO(), o.New, metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					tryUpdate = true
				} else {
					slog.Error("Failed to create secret for cluster", "name", o.New.Name, "error", err)
				}
			}
		}
		if o.Event == controllers.EventUpdate || tryUpdate {
			_, err := client.Kube().CoreV1().Secrets(o.New.Namespace).Update(context.TODO(), o.New, metav1.UpdateOptions{})
			if err != nil {
				slog.Error("Failed to update secret for cluster. Waiting 15 seconds then recomputing", "name", o.New.Name, "error", err)
				go func() {
					time.Sleep(15 * time.Second)
					recompute.TriggerRecomputation()
				}()
			}
		} else if o.Event == controllers.EventDelete {
			err := client.Kube().CoreV1().Secrets(o.New.Namespace).Delete(context.TODO(), o.New.Name, metav1.DeleteOptions{})
			if err != nil {
				slog.Error("Failed to delete secret for cluster. Waiting 15 seconds then recomputing", "name", o.New.Name, "error", err)
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
