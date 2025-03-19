package providers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadmv1beta1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
)

var _ = Describe("Provider functions", func() {
	Context("When a cluster has a kubeadm controlPlaneRef", func() {
		var clusterName = "kubeadm-cluster"
		var clusterNamespace = "kubeadm-cluster-namespace"
		var controlPlane = kubeadmv1beta1.KubeadmControlPlane{}
		var cluster = capiv1beta1.Cluster{}
		var kubeconfig = corev1.Secret{}

		BeforeEach(func() {
			clusterNamespace = fmt.Sprintf("%s-%d", clusterNamespace, time.Now().UnixMilli())
			controlPlane = kubeadmv1beta1.KubeadmControlPlane{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName + "-control-plane",
					Namespace: clusterNamespace,
				},
			}
			cluster = capiv1beta1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: clusterNamespace,
				},
				Spec: capiv1beta1.ClusterSpec{
					ControlPlaneRef: &corev1.ObjectReference{
						Name:       controlPlane.Name,
						Namespace:  controlPlane.Namespace,
						APIVersion: "controlplane.cluster.x-k8s.io/v1beta1",
						Kind:       "KubeadmControlPlane",
					},
				},
			}
			kubeconfig = corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName + "-kubeconfig",
					Namespace: clusterNamespace,
					Labels: map[string]string{
						capiv1beta1.ClusterNameLabel: clusterName,
					},
				},
				Type: capiv1beta1.ClusterSecretType,
				Data: map[string][]byte{},
			}
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: clusterNamespace}})).To(Succeed())
			Expect(k8sClient.Create(ctx, &controlPlane)).To(Succeed())
			Expect(k8sClient.Create(ctx, &cluster)).To(Succeed())
			Expect(k8sClient.Create(ctx, &kubeconfig)).To(Succeed())
			kubeconfig.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion:         cluster.Spec.ControlPlaneRef.APIVersion,
					BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
					Controller:         func(v bool) *bool { return &v }(true),
					Kind:               cluster.Spec.ControlPlaneRef.Kind,
					Name:               cluster.Spec.ControlPlaneRef.Name,
					UID:                controlPlane.GetUID(),
				},
			}
			Expect(k8sClient.Update(ctx, &kubeconfig)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, &controlPlane)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &cluster)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &kubeconfig)).To(Succeed())
		})

		It("should return the proper name for the kubeconfig", func() {
			clusterProvider := ClusterProvider{
				Client: k8sClient,
			}
			By("providing a namespaced cluster object")
			namespacedName, err := clusterProvider.GetCapiKubeconfigNamespacedName(&cluster)

			By("asserting that the name is correct")
			Expect(err).NotTo(HaveOccurred())
			Expect(namespacedName).To(Equal(types.NamespacedName{Name: kubeconfig.Name, Namespace: kubeconfig.Namespace}))
		})

		It("should validate the kubeconfig", func() {
			clusterProvider := ClusterProvider{
				Client: k8sClient,
			}
			fmt.Printf("namespace: %s, cluster: %s\n", clusterNamespace, cluster.Namespace)

			By("providing a kubeconfig secret object")
			validated, err := clusterProvider.IsCapiKubeconfig(ctx, &kubeconfig, &cluster)

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
			clusterProvider := ClusterProvider{
				Client: k8sClient,
			}

			By("providing a namespaced cluster object")
			namespacedName, err := clusterProvider.GetCapiKubeconfigNamespacedName(&cluster)

			By("asserting that the name is correct")
			Expect(err).NotTo(HaveOccurred())
			Expect(namespacedName).To(Equal(types.NamespacedName{Name: kubeconfig.Name, Namespace: kubeconfig.Namespace}))
		})

		It("should validate the kubeconfig", func() {
			clusterProvider := ClusterProvider{
				Client: k8sClient,
			}

			By("providing a kubeconfig secret object")
			validated, err := clusterProvider.IsCapiKubeconfig(ctx, &kubeconfig, &cluster)

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
			clusterProvider := ClusterProvider{
				Client: k8sClient,
			}

			By("providing a namespaced cluster object")
			namespacedName, err := clusterProvider.GetCapiKubeconfigNamespacedName(&cluster)

			By("asserting that the name is correct")
			Expect(err).NotTo(HaveOccurred())
			Expect(namespacedName).To(Equal(types.NamespacedName{Name: kubeconfig.Name, Namespace: kubeconfig.Namespace}))
		})

		It("should validate the kubeconfig", func() {
			clusterProvider := ClusterProvider{
				Client: k8sClient,
			}

			By("providing a kubeconfig secret object")
			validated, err := clusterProvider.IsCapiKubeconfig(ctx, &kubeconfig, &cluster)

			By("asserting that the secret is an AWS managed kubeconfig")
			Expect(err).NotTo(HaveOccurred())
			Expect(validated).To(BeTrue())
		})
	})
})
