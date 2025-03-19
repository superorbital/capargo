package providers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const loggerName = "capargo-providers"

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

type ClusterProvider struct {
	client.Client
}

type provider interface {
	GetNamespacedName() types.NamespacedName
	IsKubeconfig(context.Context, *corev1.Secret) bool
}

// getProvider returns the provider interface for a given CAPI cluster,
func (c *ClusterProvider) getProvider(cluster *capiv1beta1.Cluster) (provider, error) {
	switch controlPlaneRefKind(cluster.Spec.ControlPlaneRef.Kind) {
	case kubeadmKind:
		var p provider = kubeadmControlPlane{
			Client:           c.Client,
			ClusterName:      cluster.Name,
			APIVersion:       cluster.Spec.ControlPlaneRef.APIVersion,
			ControlPlaneName: cluster.Spec.ControlPlaneRef.Name,
			Namespace:        cluster.Spec.ControlPlaneRef.Namespace,
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
func (c *ClusterProvider) IsCapiKubeconfig(ctx context.Context, secret *corev1.Secret, cluster *capiv1beta1.Cluster) (bool, error) {
	p, err := c.getProvider(cluster)
	if err != nil {
		return false, err
	}
	return p.IsKubeconfig(ctx, secret), nil
}

// GetCapiKubeconfigNamespacedName retrieves the expected namespace and name
// for a CAPI cluster's kubeconfig.
func (c *ClusterProvider) GetCapiKubeconfigNamespacedName(cluster *capiv1beta1.Cluster) (types.NamespacedName, error) {
	p, err := c.getProvider(cluster)
	if err != nil {
		return types.NamespacedName{}, err
	}
	return p.GetNamespacedName(), nil
}
