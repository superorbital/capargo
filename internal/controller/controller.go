package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/superorbital/capargo/pkg/common"
	"github.com/superorbital/capargo/pkg/providers"
	"github.com/superorbital/capargo/pkg/types"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	argocdcommon "github.com/argoproj/argo-cd/v2/common"
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("capargo-controller")

type ClusterKubeconfigReconciler struct {
	client.Client
	types.Options
	providers.ClusterProvider
}

// Reconcile performs the main logic to create ArgoCD cluster secrets for
// every managed cluster and its kubeconfig.
func (c *ClusterKubeconfigReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger = logf.FromContext(ctx)
	cluster := &capiv1beta1.Cluster{}
	err := c.Get(ctx, req.NamespacedName, cluster)
	if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{}, err
	}
	logger.V(4).Info("Cluster received")

	// Remove the ArgoCD cluster secret if the cluster was deleted.
	if errors.IsNotFound(err) {
		return reconcile.Result{}, c.deleteArgoCluster(ctx, req)
	}

	// Wait until control plane is ready and our kubeconfig has been generated
	// to create or update the ArgoCD secret.
	if !cluster.Status.ControlPlaneReady {
		return reconcile.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	return reconcile.Result{}, c.createOrUpdateArgoCluster(ctx, cluster)
}

// deleteArgoCluster removes the ArgoCD cluster secret from the cluster.
func (c *ClusterKubeconfigReconciler) deleteArgoCluster(ctx context.Context, req reconcile.Request) error {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Namespace + "-" + req.Name,
			Namespace: c.ArgoNamespace,
		},
	}
	err := c.Delete(ctx, &secret, &client.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	logger.Info("Deleted ArgoCD cluster secret",
		"secret namespace", secret.GetNamespace(), "secret name", secret.GetName(),
	)

	return nil
}

// createOrUpdateArgoCluster uploads the latest version of the cluster
// kubeconfig as an ArgoCD cluster secret to the cluster.
func (c *ClusterKubeconfigReconciler) createOrUpdateArgoCluster(ctx context.Context, cluster *capiv1beta1.Cluster) error {
	capiSecret := &corev1.Secret{}
	namespacedName, err := c.GetCapiKubeconfigNamespacedName(cluster)
	if err != nil {
		return err
	}
	if err := c.Get(ctx, namespacedName, capiSecret, &client.GetOptions{}); err != nil {
		return err
	}
	valid, err := c.IsCapiKubeconfig(ctx, capiSecret, cluster)
	if err != nil {
		return err
	}

	// Ensure that the secret will contain a kubeconfig, and retrieve it.
	if !valid {
		return fmt.Errorf("secret %s does not contain kubeconfig",
			capiSecret.Name,
		)
	}
	configBytes, ok := capiSecret.Data["value"]
	if !ok {
		return fmt.Errorf("secret %s/%s does not contain key \"value\"",
			capiSecret.Namespace, capiSecret.Name,
		)
	}

	// Create kubeconfig credentials from cluster secret
	config, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if err != nil {
		return fmt.Errorf("failed to build restconfig from the secret %s/%s: %v",
			capiSecret.Namespace, capiSecret.Name, err,
		)
	}

	// Build the ArgoCD secret
	clusterConfig := buildClusterConfigFromRestConfig(config)
	ccJson, err := json.Marshal(clusterConfig)
	if err != nil {
		return fmt.Errorf("could not marshal cluster config: %v", err)
	}

	newArgoClusterSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Namespace + "-" + cluster.Name,
			Namespace: c.ArgoNamespace,
			Annotations: map[string]string{
				common.ClusterNameAnnotation:      cluster.Name,
				common.ClusterNamespaceAnnotation: cluster.Namespace,
			},
			Labels: map[string]string{
				argocdcommon.LabelKeySecretType: argocdcommon.LabelValueSecretTypeCluster,
				common.ControllerNameLabel:      common.ControllerName,
			},
		},
		Data: map[string][]byte{
			"name":   []byte(cluster.Name),
			"server": []byte(config.Host),
			"config": ccJson,
		},
	}

	currentArgoClusterSecret := corev1.Secret{}
	err = c.Get(ctx, client.ObjectKeyFromObject(&newArgoClusterSecret), &currentArgoClusterSecret, &client.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		if err := c.Create(ctx, &newArgoClusterSecret, &client.CreateOptions{}); err != nil {
			return err
		}
		logger.Info("Created ArgoCD cluster", "secret", newArgoClusterSecret.GetName())
	} else {
		if !reflect.DeepEqual(currentArgoClusterSecret.Data, newArgoClusterSecret.Data) ||
			currentArgoClusterSecret.Labels[argocdcommon.LabelValueSecretTypeCluster] != newArgoClusterSecret.Labels[argocdcommon.LabelValueSecretTypeCluster] {
			if err := c.Update(ctx, &newArgoClusterSecret, &client.UpdateOptions{}); err != nil {
				return err
			}
			logger.Info("Updated ArgoCD cluster", "secret", newArgoClusterSecret.GetName())
		}
	}

	return nil
}

func buildClusterConfigFromRestConfig(config *rest.Config) argocdv1alpha1.ClusterConfig {
	var cc argocdv1alpha1.ClusterConfig
	if config.Username != "" {
		cc.Username = config.Username
		cc.Password = config.Password
	}
	if config.BearerToken != "" {
		cc.BearerToken = config.BearerToken
	}
	tlsClientConfig := argocdv1alpha1.TLSClientConfig{
		ServerName: config.TLSClientConfig.ServerName,
		CAData:     config.TLSClientConfig.CAData,
		CertData:   config.TLSClientConfig.CertData,
		KeyData:    config.TLSClientConfig.KeyData,
	}

	cc.TLSClientConfig = tlsClientConfig

	// TODO: AWS Auth Config

	if config.ExecProvider != nil {
		execProviderConfig := &argocdv1alpha1.ExecProviderConfig{
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

func mapEnv(envVar []clientcmdapi.ExecEnvVar) map[string]string {
	outputMap := make(map[string]string, len(envVar))
	for _, env := range envVar {
		outputMap[env.Name] = env.Value
	}
	return outputMap
}
