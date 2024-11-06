package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/superorbital/capargo/pkg/providers"
	"github.com/superorbital/capargo/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logger = logf.Log.WithName("capargo-controller")

type ClusterKubeconfigReconciler struct {
	client.Client
	types.Options
}

// Reconcile performs the main logic to create ArgoCD cluster secrets for
// every managed cluster and its kubeconfig.
func (c *ClusterKubeconfigReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	cluster := &clusterv1.Cluster{}
	err := c.Get(ctx, req.NamespacedName, cluster)
	if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	// Remove the ArgoCD cluster secret if the cluster was deleted.
	if err != nil && errors.IsNotFound(err) {
		return reconcile.Result{}, c.deleteArgoCluster(ctx, req.Name)
	}

	// Wait until control plane is ready and our kubeconfig has been generated
	// to create or update the ArgoCD secret.
	if !cluster.Status.ControlPlaneReady {
		return reconcile.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}
	logger.V(4).Info("Cluster received", "cluster", cluster)

	return reconcile.Result{}, c.createOrUpdateArgoCluster(ctx, cluster)
}

// deleteArgoCluster removes the ArgoCD cluster secret from the cluster.
func (c *ClusterKubeconfigReconciler) deleteArgoCluster(ctx context.Context, name string) error {
	err := c.Delete(ctx,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: c.ArgoNamespace,
			},
		}, &client.DeleteOptions{},
	)

	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	logger.V(4).Info("Deleted ArgoCD cluster secret",
		"secret namespace", c.ArgoNamespace, "secret name", name,
	)

	return nil
}

// createOrUpdateArgoCluster uploads the latest version of the cluster
// kubeconfig as an ArgoCD cluster secret to the cluster.
func (c *ClusterKubeconfigReconciler) createOrUpdateArgoCluster(ctx context.Context, cluster *clusterv1.Cluster) error {
	// Find the kubeconfig secret located in the same namespace where the
	// cluster's controlplaneref is.
	cpr := cluster.Spec.ControlPlaneRef
	secrets := &corev1.SecretList{}
	err := c.List(ctx, secrets, client.InNamespace(cpr.Namespace))
	if err != nil {
		return err
	}
	found := false
	index := 0
	for i, secret := range secrets.Items {
		if providers.IsCapiKubeconfig(&secret, cluster) {
			found = true
			index = i
			break
		}
	}

	if !found {
		return fmt.Errorf("kubeconfig secret not found for cluster %s/%s", cluster.Namespace, cluster.Name)
	}

	// Get the kubeconfig from the secret.
	capiSecret := secrets.Items[index]
	configBytes, ok := capiSecret.Data["value"]
	if !ok {
		return fmt.Errorf("secret %s/%s for cluster %s does not contain key \"value\"",
			capiSecret.Namespace, capiSecret.Name, cluster.Name)
	}

	// Create kubeconfig credentials from cluster secret
	config, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if err != nil {
		return fmt.Errorf("failed to build restconfig from the secret %s/%s for cluster %s",
			capiSecret.Namespace, capiSecret.Name, cluster.Name)
	}

	// Build the ArgoCD secret
	clusterConfig := buildClusterConfigFromRestConfig(config)
	ccJson, err := json.Marshal(clusterConfig)
	if err != nil {
		return fmt.Errorf("could not marshal cluster config for cluster %s/%s",
			cluster.Namespace, cluster.Name)
	}

	argoClusterSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: c.ArgoNamespace,
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

	action := "Created ArgoCD cluster secret"
	err = c.Create(ctx, &argoClusterSecret, &client.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	if err != nil && errors.IsAlreadyExists(err) {
		action = "Updated ArgoCD cluster secret"
		err = c.Update(ctx, &argoClusterSecret, &client.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	logger.V(4).Info(action, "cluster", cluster)
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
