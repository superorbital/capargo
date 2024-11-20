package providers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type awsManagedControlPlane struct {
	Name       string
	Namespace  string
	APIVersion string
}

// GetNamespacedName returns the namespace and name of a cluster
// with an AWS managed control plane.
func (a awsManagedControlPlane) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-user-kubeconfig", a.Name),
		Namespace: a.Namespace,
	}
}

// IsKubeconfig determines whether the secret provided is an
// AWSManagedControlPlane kubeconfig or not.
func (a awsManagedControlPlane) IsKubeconfig(secret *corev1.Secret) bool {
	switch a.APIVersion {
	case "controlplane.cluster.x-k8s.io/v1beta2":
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
		if or.Kind != "AWSManagedControlPlane" {
			logger.V(4).Info("Secret is not owned by AWSManagedControlPlane",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"owner reference", or.Name,
			)
			return false
		}
		if secret.Namespace != a.Namespace {
			logger.V(4).Info("Secret is not in the same namespace as AWSManagedControlPlane",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"AWSManagedControlPlane namespace", a.Namespace,
				"AWSManagedControlPlane name", a.Name,
			)
			return false
		}
		if secret.Name != fmt.Sprintf("%s-user-kubeconfig", a.Name) {
			logger.V(4).Info("Secret does not match '*-user-kubeconfig' pattern",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
			)
			return false
		}
		return true
	default:
		logger.V(2).Info("APIVersion unsupported for AWSManagedControlPlane",
			"APIVersion", a.APIVersion,
		)
		return false
	}
}
