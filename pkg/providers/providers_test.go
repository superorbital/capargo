package providers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("Provider functions", func() {
	Context("When a cluster has a kubeadm controlPlaneRef", func() {
		clusterName := "kubeadm-cluster"
		clusterNamespace := "kubeadm-cluster-namespace"
		cluster := capiv1beta1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
				UID:       types.UID(uuid.New().String()),
			},
			Spec: capiv1beta1.ClusterSpec{
				ControlPlaneRef: &corev1.ObjectReference{
					APIVersion: "controlplane.cluster.x-k8s.io/v1beta1",
					Kind:       "KubeadmControlPlane",
					Name:       clusterName + "-control-plane",
					Namespace:  clusterNamespace,
				},
			},
		}

		var kubeconfig = corev1.Secret{
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
			Type: capiv1beta1.ClusterSecretType,
			Data: map[string][]byte{},
		}
		It("should return the proper name for the kubeconfig", func() {
			By("providing a namespaced cluster object")
			namespacedName, err := GetCapiKubeconfigNamespacedName(&cluster)

			By("asserting that the name is correct")
			Expect(err).NotTo(HaveOccurred())
			Expect(namespacedName).To(Equal(types.NamespacedName{Name: kubeconfig.Name, Namespace: kubeconfig.Namespace}))
		})

		It("should validate the kubeconfig", func() {
			By("providing a kubeconfig secret object")
			validated, err := IsCapiKubeconfig(&kubeconfig, &cluster)

			By("asserting that the secret is a kubeadm kubeconfig")
			Expect(err).NotTo(HaveOccurred())
			Expect(validated).To(BeTrue())
		})
	})

	Context("When a cluster has a vcluster controlPlaneRef", func() {
		clusterName := "vcluster-cluster"
		clusterNamespace := "vcluster-cluster-namespace"
		cluster := capiv1beta1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
				UID:       types.UID(uuid.New().String()),
			},
			Spec: capiv1beta1.ClusterSpec{
				ControlPlaneRef: &corev1.ObjectReference{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
					Kind:       "VCluster",
					Name:       clusterName,
					Namespace:  clusterNamespace,
				},
			},
		}

		var kubeconfig = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-kubeconfig",
				Namespace: clusterNamespace,
			},
			Data: map[string][]byte{},
		}
		It("should return the proper name for the kubeconfig", func() {
			By("providing a namespaced cluster object")
			namespacedName, err := GetCapiKubeconfigNamespacedName(&cluster)

			By("asserting that the name is correct")
			Expect(err).NotTo(HaveOccurred())
			Expect(namespacedName).To(Equal(types.NamespacedName{Name: kubeconfig.Name, Namespace: kubeconfig.Namespace}))
		})

		It("should validate the kubeconfig", func() {
			By("providing a kubeconfig secret object")
			validated, err := IsCapiKubeconfig(&kubeconfig, &cluster)

			By("asserting that the secret is a vcluster kubeconfig")
			Expect(err).NotTo(HaveOccurred())
			Expect(validated).To(BeTrue())
		})
	})

	Context("When a cluster has an AWS managed controlPlaneRef", func() {
		clusterName := "eks-cluster"
		clusterNamespace := "eks-cluster-namespace"
		cluster := capiv1beta1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
				UID:       types.UID(uuid.New().String()),
			},
			Spec: capiv1beta1.ClusterSpec{
				ControlPlaneRef: &corev1.ObjectReference{
					APIVersion: "controlplane.cluster.x-k8s.io/v1beta2",
					Kind:       "AWSManagedControlPlane",
					Name:       clusterName,
					Namespace:  clusterNamespace,
				},
			},
		}

		var kubeconfig = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-user-kubeconfig",
				Namespace: clusterNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "controlplane.cluster.x-k8s.io/v1beta2",
						BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
						Controller:         func(v bool) *bool { return &v }(true),
						Kind:               "AWSManagedControlPlane",
						UID:                cluster.GetUID(),
					},
				},
			},
			Type: "cluster.x-k8s.io/secret",
			Data: map[string][]byte{},
		}
		It("should return the proper name for the kubeconfig", func() {
			By("providing a namespaced cluster object")
			namespacedName, err := GetCapiKubeconfigNamespacedName(&cluster)

			By("asserting that the name is correct")
			Expect(err).NotTo(HaveOccurred())
			Expect(namespacedName).To(Equal(types.NamespacedName{Name: kubeconfig.Name, Namespace: kubeconfig.Namespace}))
		})

		It("should validate the kubeconfig", func() {
			By("providing a kubeconfig secret object")
			validated, err := IsCapiKubeconfig(&kubeconfig, &cluster)

			By("asserting that the secret is an AWS managed kubeconfig")
			Expect(err).NotTo(HaveOccurred())
			Expect(validated).To(BeTrue())
		})
	})
})
