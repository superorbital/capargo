package providers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/types"
)

type vCluster struct {
	Name       string
	Namespace  string
	APIVersion string
}

// GetNamespacedName returns the namespace and name of a kubeconfig with a
// vCluster control plane.
func (v vCluster) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-kubeconfig", v.Name),
		Namespace: v.Namespace,
	}
}

// IsKubeconfig determines whether the secret provided is a vCluster
// kubeconfig or not.
func (v vCluster) IsKubeconfig(ctx context.Context, secret *corev1.Secret) bool {
	logger := logf.FromContext(ctx).WithName(loggerName)
	switch v.APIVersion {
	case "infrastructure.cluster.x-k8s.io/v1alpha1":
		if secret.Namespace != v.Namespace {
			logger.V(4).Info("Secret is not in the same namespace as VCluster",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
			)
			return false
		}
		if secret.Name != fmt.Sprintf("%s-kubeconfig", v.Name) {
			logger.V(4).Info("Secret does not match '*-kubeconfig' pattern",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
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
