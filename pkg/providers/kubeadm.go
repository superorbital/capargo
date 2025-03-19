package providers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadmv1beta1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var v1beta1KubeadmControlPlane = kubeadmv1beta1.GroupVersion.String()

type kubeadmControlPlane struct {
	client.Client
	ControlPlaneName string
	ClusterName      string
	Namespace        string
	APIVersion       string
}

// GetNamespacedName returns the namespace and name of a cluster
// with a kubeadm-bootstrapped control plane.
func (k kubeadmControlPlane) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-kubeconfig", k.ClusterName),
		Namespace: k.Namespace,
	}
}

// IsKubeconfig determines whether the secret provided is a
// KubeadmControlPlane kubeconfig or not.
func (k kubeadmControlPlane) IsKubeconfig(ctx context.Context, secret *corev1.Secret) bool {
	logger := logf.FromContext(ctx).WithName(loggerName)
	switch k.APIVersion {
	case v1beta1KubeadmControlPlane:
		if secret.Type != capiv1beta1.ClusterSecretType {
			logger.V(4).Info("Secret is not a cluster secret",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"secret type", secret.Type,
			)
			return false
		}
		name := secret.Labels[capiv1beta1.ClusterNameLabel]
		if name != k.ClusterName {
			logger.V(4).Info("Secret cluster name label does not contain cluster name",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"cluster label name", name,
			)
			return false
		}
		kcp := kubeadmv1beta1.KubeadmControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k.ControlPlaneName,
				Namespace: k.Namespace,
			},
		}
		if err := k.Client.Get(ctx, client.ObjectKeyFromObject(&kcp), &kcp, &client.GetOptions{}); err != nil {
			logger.V(4).Info("Could not find KubeadmControlPlane object for secret",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"error", err,
			)
			return false
		}
		if !metav1.IsControlledBy(secret, &kcp) {
			logger.V(4).Info("Secret is not owned by KubeadmControlPlane",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"owner references", secret.GetOwnerReferences(),
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
