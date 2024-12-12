package providers

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadmv1beta1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
)

var v1beta1KubeadmControlPlane = fmt.Sprintf("%s/%s", kubeadmv1beta1.GroupVersion.Group, kubeadmv1beta1.GroupVersion.Version)

type kubeadmControlPlane struct {
	ControlPlaneName string
	ClusterName      string
	Namespace        string
	APIVersion       string
	UID              types.UID
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
func (k kubeadmControlPlane) IsKubeconfig(secret *corev1.Secret) bool {
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
		ors := secret.GetOwnerReferences()
		if len(ors) != 1 {
			logger.V(4).Info("Secret has incorrect number of owner references",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"length", len(ors),
			)
			return false
		}
		if !reflect.DeepEqual(metav1.OwnerReference{
			APIVersion:         k.APIVersion,
			BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
			Controller:         func(v bool) *bool { return &v }(true),
			Kind:               string(kubeadmKind),
			Name:               k.ControlPlaneName,
			UID:                k.UID,
		}, ors[0]) {
			logger.V(4).Info("Secret is not owned by KubeadmControlPlane",
				"secret namespace", secret.GetNamespace(),
				"secret name", secret.GetName(),
				"owner reference", ors[0],
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
