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
			logger.Debugf("Secret %s/%s is not in the same namespace as VCluster %s/%s",
				secret.GetNamespace(), secret.GetName(),
				v.Namespace, v.Name)
			return false
		}
		if secret.Name != fmt.Sprintf("%s-kubeconfig", v.Name) {
			logger.Debugf("Secret %s/%s does not match '%s-kubeconfig' pattern",
				secret.GetNamespace(), secret.GetName(),
				v.Name)
			return false
		}
		return true
	default:
		logger.Warnf("APIVersion %s unsupported for VCluster", v.APIVersion)
		return false
	}
}
