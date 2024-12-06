package providers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadmv1beta1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
)

var kubeadmControlPlaneAPIVersion = fmt.Sprintf("%s/%s", kubeadmv1beta1.GroupVersion.Group, kubeadmv1beta1.GroupVersion.Version)

type kubeadmControlPlane struct {
	Name       string
	Namespace  string
	APIVersion string
}

// GetNamespacedName returns the namespace and name of a cluster
// with a kubeadm-bootstrapped control plane.
func (k kubeadmControlPlane) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-kubeconfig", k.Name),
		Namespace: k.Namespace,
	}
}

// IsKubeconfig determines whether the secret provided is a
// KubeadmControlPlane kubeconfig or not.
func (k kubeadmControlPlane) IsKubeconfig(secret *corev1.Secret) bool {
	switch k.APIVersion {
	case kubeadmControlPlaneAPIVersion:
		if secret.Type != clusterv1beta1.ClusterSecretType {
			logger.V(4).Info("Secret is not a cluster secret",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"secret type", secret.Type,
			)
			return false
		}
		ors := secret.GetOwnerReferences()
		if len(ors) != 1 {
			logger.V(4).Info("Secret has incorrect number of owner references",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"length", len(ors),
			)
			return false
		}
		or := ors[0]
		if or.Kind != string(kubeadmKind) {
			logger.V(4).Info("Secret is not owned by KubeadmControlPlane",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"owner reference", or.Name,
			)
			return false
		}
		return true
	default:
		logger.V(2).Info("APIVersion unsupported for KubeadmControlPlane",
			"APIVersion", k.APIVersion,
		)
		return false
	}
}
