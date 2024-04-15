package providers

import (
	istiolog "istio.io/istio/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/api/v1beta1"
)

var logger = istiolog.RegisterScope("capargo-providers", "")

type provider interface {
	IsKubeconfig(*corev1.Secret) bool
}

// isCapiKubeconfig determines whether the secret provided is a CAPI kubeconfig
// from a given control plane controller.
func IsCapiKubeconfig(secret *corev1.Secret, cluster *v1beta1.Cluster) bool {
	switch cluster.Spec.ControlPlaneRef.Kind {
	case "KubeadmControlPlane":
		var p provider = kubeadmControlPlane{
			APIVersion: cluster.Spec.ControlPlaneRef.APIVersion,
			Name:       cluster.Spec.ControlPlaneRef.Name,
			Namespace:  cluster.Spec.ControlPlaneRef.Namespace,
		}
		return p.IsKubeconfig(secret)
	case "AWSManagedControlPlane":
		var p provider = awsManagedControlPlane{
			APIVersion: cluster.Spec.ControlPlaneRef.APIVersion,
			Name:       cluster.Spec.ControlPlaneRef.Name,
			Namespace:  cluster.Spec.ControlPlaneRef.Namespace,
		}
		return p.IsKubeconfig(secret)
	case "VCluster":
		var p provider = vCluster{
			APIVersion: cluster.Spec.ControlPlaneRef.APIVersion,
			Name:       cluster.Spec.ControlPlaneRef.Name,
			Namespace:  cluster.Spec.ControlPlaneRef.Namespace,
		}
		return p.IsKubeconfig(secret)
	default:
		logger.Warnf("ControlPlaneRef kind %s unsupported", cluster.Spec.ControlPlaneRef.Kind)
		return false
	}
}
