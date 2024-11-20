package providers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

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
	case "controlplane.cluster.x-k8s.io/v1beta1":
		if secret.Type != "cluster.x-k8s.io/secret" {
			logger.V(4).Info("Secret is not a cluster secret",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
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
		if or.Kind != "KubeadmControlPlane" {
			logger.V(4).Info("Secret is not owned by KubeadmControlPlane",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"owner reference", or.Name,
			)
			return false
		}
		if secret.Namespace != k.Namespace {
			logger.V(4).Info("Secret is not in the same namespace as KubeadmControlPlane",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"KubeadmControlPlane namespace", k.Namespace,
				"KubeadmControlPlane name", k.Name,
			)
			return false
		}
		if secret.Name != fmt.Sprintf("%s-kubeconfig", k.Name) {
			logger.V(4).Info("Secret does not match '*-kubeconfig' pattern",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
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
