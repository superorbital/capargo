package providers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type vCluster struct {
	Name       string
	Namespace  string
	APIVersion string
}

// IsKubeconfig determines whether the secret provided is a vCluster
// kubeconfig or not.
func (v vCluster) IsKubeconfig(secret *corev1.Secret) bool {
	switch v.APIVersion {
	case "infrastructure.cluster.x-k8s.io/v1alpha1":
		if secret.Namespace != v.Namespace {
			logger.V(4).Info("Secret is not in the same namespace as VCluster",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"vCluster namespace", v.Namespace,
				"vCluster name", v.Name,
			)
			return false
		}
		if secret.Name != fmt.Sprintf("%s-kubeconfig", v.Name) {
			logger.V(4).Info("Secret does not match '*-kubeconfig' pattern",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"vCluster name", v.Name,
			)
			return false
		}
		return true
	default:
		logger.V(2).Info("APIVersion unsupported for VCluster",
			"APIVersion", v.APIVersion,
		)
		return false
	}
}
