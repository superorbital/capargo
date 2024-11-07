package providers

import (
	corev1 "k8s.io/api/core/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("capargo-providers")

type provider interface {
	IsKubeconfig(*corev1.Secret) bool
}

// isCapiKubeconfig determines whether the secret provided is a CAPI kubeconfig
// from a given control plane controller.
func IsCapiKubeconfig(secret *corev1.Secret, cluster *clusterv1beta1.Cluster) bool {
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
		logger.V(2).Info("ControlPlaneRef kind unsupported", "kind", cluster.Spec.ControlPlaneRef.Kind)
		return false
	}
}
