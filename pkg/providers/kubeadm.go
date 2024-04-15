package providers

import (
	"fmt"

	"istio.io/istio/pkg/log"
	corev1 "k8s.io/api/core/v1"
)

type kubeadmControlPlane struct {
	Name       string
	Namespace  string
	APIVersion string
}

// IsKubeconfig determines whether the secret provided is a
// KubeadmControlPlane kubeconfig or not.
func (k kubeadmControlPlane) IsKubeconfig(secret *corev1.Secret) bool {
	switch k.APIVersion {
	case "controlplane.cluster.x-k8s.io/v1beta1":
		if secret.Type != "cluster.x-k8s.io/secret" {
			logger.Debugf("Secret %s/%s not a cluster secret",
				secret.GetNamespace(), secret.GetName())
			return false
		}
		ors := secret.GetOwnerReferences()
		if len(ors) != 1 {
			logger.Debugf("Secret %s/%s has incorrect number of owner references: %d",
				secret.GetNamespace(), secret.GetName(), len(ors))
			return false
		}
		or := ors[0]
		if or.Name != "KubeadmControlPlane" {
			logger.Debugf("Secret %s/%s not owned by KubeadmControlPlane: %s",
				secret.GetNamespace(), secret.GetName(), or.Name)
			return false
		}
		if secret.Namespace != k.Namespace {
			logger.Debugf("Secret %s/%s is not in the same namespace as KubeadmControlPlane %s/%s",
				secret.GetNamespace(), secret.GetName(),
				k.Namespace, k.Name)
			return false
		}
		if secret.Name != fmt.Sprintf("%s-kubeconfig", k.Name) {
			logger.Debugf("Secret %s/%s does not match '%s-kubeconfig' pattern",
				secret.GetNamespace(), secret.GetName(),
				k.Name)
			return false
		}
		return true
	default:
		log.Warnf("APIVersion %s unsupported for KubeadmControlPlane", k.APIVersion)
		return false
	}
}
