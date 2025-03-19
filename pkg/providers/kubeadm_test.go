package providers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadmv1beta1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
)

var _ = Describe("Kubeadm provider tests", func() {
	When("handling a supported kubeadm cluster", func() {
		var clusterName = "kubeadm-cluster"
		var clusterNamespace = "kubeadm-cluster-namespace"
		cluster := capiv1beta1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
				UID:       types.UID(uuid.New().String()),
			},
			Spec: capiv1beta1.ClusterSpec{
				ControlPlaneRef: &corev1.ObjectReference{
					APIVersion: kubeadmv1beta1.GroupVersion.String(),
					Kind:       "KubeadmControlPlane",
					Name:       clusterName + "-control-plane",
					Namespace:  clusterNamespace,
				},
			},
		}

		// Missing secret type
		var badConfig1 = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-kubeconfig",
				Namespace: clusterNamespace,
				Labels: map[string]string{
					capiv1beta1.ClusterNameLabel: clusterName,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         cluster.Spec.ControlPlaneRef.APIVersion,
						BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
						Controller:         func(v bool) *bool { return &v }(true),
						Kind:               cluster.Spec.ControlPlaneRef.Kind,
						Name:               cluster.Spec.ControlPlaneRef.Name,
						UID:                cluster.GetUID(),
					},
				},
			},
			Data: map[string][]byte{},
		}

		// Missing label
		var badConfig2 = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-kubeconfig",
				Namespace: clusterNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         cluster.Spec.ControlPlaneRef.APIVersion,
						BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
						Controller:         func(v bool) *bool { return &v }(true),
						Kind:               cluster.Spec.ControlPlaneRef.Kind,
						Name:               cluster.Spec.ControlPlaneRef.Name,
						UID:                cluster.GetUID(),
					},
				},
			},
			Type: "cluster.x-k8s.io/secret",
			Data: map[string][]byte{},
		}

		// Multiple owner references
		var badConfig3 = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-kubeconfig",
				Namespace: clusterNamespace,
				Labels: map[string]string{
					capiv1beta1.ClusterNameLabel: clusterName,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         cluster.Spec.ControlPlaneRef.APIVersion,
						BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
						Controller:         func(v bool) *bool { return &v }(true),
						Kind:               cluster.Spec.ControlPlaneRef.Kind,
						Name:               cluster.Spec.ControlPlaneRef.Name,
						UID:                cluster.GetUID(),
					},
					{
						APIVersion: "fake.io/v1alpha1",
						Kind:       "Fake",
						Name:       "FakeObject",
					},
				},
			},
			Type: "cluster.x-k8s.io/secret",
			Data: map[string][]byte{},
		}

		// Invalid owner reference
		var badConfig4 = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-kubeconfig",
				Namespace: clusterNamespace,
				Labels: map[string]string{
					capiv1beta1.ClusterNameLabel: clusterName,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "fake.io/v1alpha1",
						Kind:       "Fake",
						Name:       "FakeObject",
					},
				},
			},
			Type: "cluster.x-k8s.io/secret",
			Data: map[string][]byte{},
		}

		It("should reject all invalid kubeconfigs", func() {
			var p = kubeadmControlPlane{
				Client:           k8sClient,
				APIVersion:       cluster.Spec.ControlPlaneRef.APIVersion,
				ControlPlaneName: cluster.Spec.ControlPlaneRef.Name,
				Namespace:        cluster.Spec.ControlPlaneRef.Namespace,
				ClusterName:      cluster.Name,
			}
			var validated bool
			By("providing kubeconfig secrets with bad configs")
			validated = p.IsKubeconfig(ctx, &badConfig1)
			Expect(validated).To(BeFalse())
			validated = p.IsKubeconfig(ctx, &badConfig2)
			Expect(validated).To(BeFalse())
			validated = p.IsKubeconfig(ctx, &badConfig3)
			Expect(validated).To(BeFalse())
			validated = p.IsKubeconfig(ctx, &badConfig4)
			Expect(validated).To(BeFalse())
		})
	})

	When("handling an unsupported kubeadm cluster", func() {
		var clusterName = "kubeadm-cluster"
		var clusterNamespace = "kubeadm-cluster-namespace"
		// Unsupported API version
		cluster := capiv1beta1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
				UID:       types.UID(uuid.New().String()),
			},
			Spec: capiv1beta1.ClusterSpec{
				ControlPlaneRef: &corev1.ObjectReference{
					APIVersion: "controlplane.cluster.x-k8s.io/v1alpha1",
					Kind:       "KubeadmControlPlane",
					Name:       clusterName,
					Namespace:  clusterNamespace,
				},
			},
		}

		var kubeconfig = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-kubeconfig",
				Namespace: clusterNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         cluster.Spec.ControlPlaneRef.APIVersion,
						BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
						Controller:         func(v bool) *bool { return &v }(true),
						Kind:               "KubeadmControlPlane",
						UID:                cluster.GetUID(),
					},
				},
			},
			Type: "cluster.x-k8s.io/secret",
			Data: map[string][]byte{},
		}

		It("should reject the unsupported cluster", func() {
			var p = kubeadmControlPlane{
				Client:           k8sClient,
				APIVersion:       cluster.Spec.ControlPlaneRef.APIVersion,
				ControlPlaneName: cluster.Spec.ControlPlaneRef.Name,
				Namespace:        cluster.Spec.ControlPlaneRef.Namespace,
				ClusterName:      cluster.Name,
			}
			var validated bool
			By("trying to validate the kubeconfig")
			validated = p.IsKubeconfig(ctx, &kubeconfig)
			Expect(validated).To(BeFalse())
		})
	})
})
