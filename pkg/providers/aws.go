package providers

import (
	"fmt"

	"istio.io/istio/pkg/log"
	corev1 "k8s.io/api/core/v1"
)

type awsManagedControlPlane struct {
	Name       string
	Namespace  string
	APIVersion string
}

// IsKubeconfig determines whether the secret provided is an
// AWSManagedControlPlane kubeconfig or not.
func (a awsManagedControlPlane) IsKubeconfig(secret *corev1.Secret) bool {
	switch a.APIVersion {
	case "controlplane.cluster.x-k8s.io/v1beta2":
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
		if or.Name != "AWSManagedControlPlane" {
			logger.Debugf("Secret %s/%s not owned by AWSManagedControlPlane: %s",
				secret.GetNamespace(), secret.GetName(), or.Name)
			return false
		}
		if secret.Namespace != a.Namespace {
			logger.Debugf("Secret %s/%s is not in the same namespace as AWSManagedControlPlane %s/%s",
				secret.GetNamespace(), secret.GetName(),
				a.Namespace, a.Name)
			return false
		}
		if secret.Name != fmt.Sprintf("%s-user-kubeconfig", a.Name) {
			logger.Debugf("Secret %s/%s does not match '%s-user-kubeconfig' pattern",
				secret.GetNamespace(), secret.GetName(),
				a.Name)
			return false
		}
		return true
	default:
		log.Warnf("APIVersion %s unsupported for AWSManagedControlPlane", a.APIVersion)
		return false
	}
}
