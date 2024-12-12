package providers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("capargo-providers")

// controlPlaneRefKind
type controlPlaneRefKind string

const (
	// kubeadmKind
	kubeadmKind controlPlaneRefKind = "KubeadmControlPlane"

	// awsManagedKind
	awsManagedKind controlPlaneRefKind = "AWSManagedControlPlane"

	// vclusterKind
	vclusterKind controlPlaneRefKind = "VCluster"
)

type provider interface {
	GetNamespacedName() types.NamespacedName
	IsKubeconfig(*corev1.Secret) bool
}

// getProvider returns the provider interface for a given CAPI cluster,
func getProvider(cluster *clusterv1beta1.Cluster) (provider, error) {
	switch controlPlaneRefKind(cluster.Spec.ControlPlaneRef.Kind) {
	case kubeadmKind:
		var p provider = kubeadmControlPlane{
			APIVersion:       cluster.Spec.ControlPlaneRef.APIVersion,
			ControlPlaneName: cluster.Spec.ControlPlaneRef.Name,
			Namespace:        cluster.Spec.ControlPlaneRef.Namespace,
			ClusterName:      cluster.Name,
			UID:              cluster.UID,
		}
		return p, nil
	case awsManagedKind:
		var p provider = awsManagedControlPlane{
			APIVersion: cluster.Spec.ControlPlaneRef.APIVersion,
			Name:       cluster.Spec.ControlPlaneRef.Name,
			Namespace:  cluster.Spec.ControlPlaneRef.Namespace,
		}
		return p, nil
	case vclusterKind:
		var p provider = vCluster{
			APIVersion: cluster.Spec.ControlPlaneRef.APIVersion,
			Name:       cluster.Spec.ControlPlaneRef.Name,
			Namespace:  cluster.Spec.ControlPlaneRef.Namespace,
		}
		return p, nil
	default:
		return nil, fmt.Errorf("controlPlaneRef kind %s unsupported", cluster.Spec.ControlPlaneRef.Kind)
	}
}

// IsCapiKubeconfig determines whether the secret provided is a CAPI kubeconfig
// from a given control plane controller.
func IsCapiKubeconfig(secret *corev1.Secret, cluster *clusterv1beta1.Cluster) (bool, error) {
	p, err := getProvider(cluster)
	if err != nil {
		return false, err
	}
	return p.IsKubeconfig(secret), nil
}

// GetCapiKubeconfigNamespacedName retrieves the expected namespace and name
// for a CAPI cluster's kubeconfig.
func GetCapiKubeconfigNamespacedName(cluster *clusterv1beta1.Cluster) (types.NamespacedName, error) {
	p, err := getProvider(cluster)
	if err != nil {
		return types.NamespacedName{}, err
	}
	return p.GetNamespacedName(), nil
}
