package controller

import (
	"fmt"
	"time"

	argocdcommon "github.com/argoproj/argo-cd/v2/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/superorbital/capargo/pkg/types"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var vclusterKubeconfig443 = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlRENDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUzTXpBNU1UUTROakF3SGhjTk1qUXhNVEEyTVRjME1UQXdXaGNOTXpReE1UQTBNVGMwTVRBdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUzTXpBNU1UUTROakF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFSYUswS1BJRUxhTFNlckd3cmxQZGtQUmFHcnZ0NWRobG5LWTh0eE14U1AKL3FOY1IwYWJxblNBM3NpTkZhZVg2UW5OQjNDYytuVTFYcWYrKzBZTVNuQUVvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVUx4VnVGUlg2aFBpdnhnYy9odUpMCmxOL2c0eTB3Q2dZSUtvWkl6ajBFQXdJRFNRQXdSZ0loQUpLMXI1Y1VzaThqSFRiNFFvSUF6TXBSTmg4MlpxdlgKMm5mV1QveDArdHBBQWlFQW10MUUzYk5UQUVZV1JnOEs5OC9PRExjaWk5bW11MXJxN1ZiRldMNXhyUGc9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    server: https://vcluster-1.vcluster.svc:443
  name: my-vcluster
contexts:
- context:
    cluster: my-vcluster
    user: my-vcluster
  name: my-vcluster
current-context: my-vcluster
kind: Config
preferences: {}
users:
- name: my-vcluster
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrVENDQVRlZ0F3SUJBZ0lJWXpSTWxmQ1dhUVV3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOek13T1RFME9EWXdNQjRYRFRJME1URXdOakUzTkRFd01Gb1hEVEkxTVRFdwpOakUzTkRFd01Gb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJIcEV6WmtmMTdMYVVqMjIKVDE3Sk1EQld2ZmxIWUdBOW51RWxkbmFLQVJ4QWRoRTVYVFVZV3M2ZU9FM0gwdk5DYzBMc3ZibkU1UWVwenUzRgpzZk8yNGl1alNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCUjlLZk1ueDRwMi9OUDkyUkVHOFNpaVgyK0RDakFLQmdncWhrak9QUVFEQWdOSUFEQkYKQWlFQTBCTHNVVUdjNWo2S0NRNHJXTW5lMnRRZzlBbiswS1MxRm5vQS91Q1F2dk1DSUJpMHlTZXlEMGFqZk1oYgpkdWZVVUpSVTdpMDJMMWpqMXpSUXJlWGtvUjZCCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUzTXpBNU1UUTROakF3SGhjTk1qUXhNVEEyTVRjME1UQXdXaGNOTXpReE1UQTBNVGMwTVRBdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUzTXpBNU1UUTROakF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFSbmRWMnVHUUFmZVVwS2IvbjQvcDdjNmtnaG5zZ1VpRTBaa2ZvMzAvdnkKa0lVc1RBazdkMkRGSm53ZXBnMzM0M2oxaTZZbzgwOStDdDFML0psZHJjcDVvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVWZTbnpKOGVLZHZ6VC9ka1JCdkVvCm9sOXZnd293Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUloQU43cE5YQ0pMSFJDZGc5WFlvc1l2ZzdHQXN4Tm55MHUKV2NNRHhRK2Q0WUV0QWlBV01tR1NweHN5aHBDK05Bc1VIek9YZ2NNSmJxWWtyRVhoc2M0UFpVSE00QT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUhwaWlpQUlrSDdETTZuVFZ6WE9wOG9jTlJVOUROL2pqYndVL3UrTUx1dTVvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFZWtUTm1SL1hzdHBTUGJaUFhza3dNRmE5K1VkZ1lEMmU0U1YyZG9vQkhFQjJFVGxkTlJoYQp6cDQ0VGNmUzgwSnpRdXk5dWNUbEI2bk83Y1d4ODdiaUt3PT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=
`

var vclusterKubeconfig6443 = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlRENDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUzTXpBNU1UUTROakF3SGhjTk1qUXhNVEEyTVRjME1UQXdXaGNOTXpReE1UQTBNVGMwTVRBdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUzTXpBNU1UUTROakF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFSYUswS1BJRUxhTFNlckd3cmxQZGtQUmFHcnZ0NWRobG5LWTh0eE14U1AKL3FOY1IwYWJxblNBM3NpTkZhZVg2UW5OQjNDYytuVTFYcWYrKzBZTVNuQUVvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVUx4VnVGUlg2aFBpdnhnYy9odUpMCmxOL2c0eTB3Q2dZSUtvWkl6ajBFQXdJRFNRQXdSZ0loQUpLMXI1Y1VzaThqSFRiNFFvSUF6TXBSTmg4MlpxdlgKMm5mV1QveDArdHBBQWlFQW10MUUzYk5UQUVZV1JnOEs5OC9PRExjaWk5bW11MXJxN1ZiRldMNXhyUGc9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    server: https://vcluster-1.vcluster.svc:6443
  name: my-vcluster
contexts:
- context:
    cluster: my-vcluster
    user: my-vcluster
  name: my-vcluster
current-context: my-vcluster
kind: Config
preferences: {}
users:
- name: my-vcluster
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrVENDQVRlZ0F3SUJBZ0lJWXpSTWxmQ1dhUVV3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOek13T1RFME9EWXdNQjRYRFRJME1URXdOakUzTkRFd01Gb1hEVEkxTVRFdwpOakUzTkRFd01Gb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJIcEV6WmtmMTdMYVVqMjIKVDE3Sk1EQld2ZmxIWUdBOW51RWxkbmFLQVJ4QWRoRTVYVFVZV3M2ZU9FM0gwdk5DYzBMc3ZibkU1UWVwenUzRgpzZk8yNGl1alNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCUjlLZk1ueDRwMi9OUDkyUkVHOFNpaVgyK0RDakFLQmdncWhrak9QUVFEQWdOSUFEQkYKQWlFQTBCTHNVVUdjNWo2S0NRNHJXTW5lMnRRZzlBbiswS1MxRm5vQS91Q1F2dk1DSUJpMHlTZXlEMGFqZk1oYgpkdWZVVUpSVTdpMDJMMWpqMXpSUXJlWGtvUjZCCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUzTXpBNU1UUTROakF3SGhjTk1qUXhNVEEyTVRjME1UQXdXaGNOTXpReE1UQTBNVGMwTVRBdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUzTXpBNU1UUTROakF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFSbmRWMnVHUUFmZVVwS2IvbjQvcDdjNmtnaG5zZ1VpRTBaa2ZvMzAvdnkKa0lVc1RBazdkMkRGSm53ZXBnMzM0M2oxaTZZbzgwOStDdDFML0psZHJjcDVvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVWZTbnpKOGVLZHZ6VC9ka1JCdkVvCm9sOXZnd293Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUloQU43cE5YQ0pMSFJDZGc5WFlvc1l2ZzdHQXN4Tm55MHUKV2NNRHhRK2Q0WUV0QWlBV01tR1NweHN5aHBDK05Bc1VIek9YZ2NNSmJxWWtyRVhoc2M0UFpVSE00QT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUhwaWlpQUlrSDdETTZuVFZ6WE9wOG9jTlJVOUROL2pqYndVL3UrTUx1dTVvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFZWtUTm1SL1hzdHBTUGJaUFhza3dNRmE5K1VkZ1lEMmU0U1YyZG9vQkhFQjJFVGxkTlJoYQp6cDQ0VGNmUzgwSnpRdXk5dWNUbEI2bk83Y1d4ODdiaUt3PT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=
`

var _ = Describe("Cluster API to ArgoCD cluster controller", Ordered, func() {
	Context("When a new cluster is created", func() {
		var (
			argoNamespace string
			testNamespace string
		)
		BeforeAll(func() {
			By("creating the argo namespace")
			argoNamespace = "argocd"
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: argoNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns, &client.CreateOptions{})).To(Succeed())
		})
		BeforeEach(func() {
			By("creating a new namespace for each test")
			testNamespace = fmt.Sprintf("ns-%s", uuid.New().String())
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns, &client.CreateOptions{})).To(Succeed())
		})
		It("should create an ArgoCD cluster secret for a VCluster", func() {
			var (
				err    error
				result reconcile.Result
			)
			vclusterName := "test-vcluster"
			By("creating a cluster object with a VCluster control plane reference")
			vcluster := capiv1beta1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vclusterName,
					Namespace: testNamespace,
				},
				Spec: capiv1beta1.ClusterSpec{
					ControlPlaneEndpoint: capiv1beta1.APIEndpoint{
						Host: vclusterName + ".vcluster.svc",
						Port: 443,
					},
					ControlPlaneRef: &corev1.ObjectReference{
						Kind:       "VCluster",
						Namespace:  testNamespace,
						Name:       vclusterName,
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
					},
					InfrastructureRef: &corev1.ObjectReference{
						Kind:       "VCluster",
						Namespace:  testNamespace,
						Name:       vclusterName,
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &vcluster, &client.CreateOptions{})).To(Succeed())

			By("calling the reconcile function")
			reconciler := &ClusterKubeconfigReconciler{
				Client: k8sClient,
				Options: types.Options{
					ClusterID:        "envTest",
					ClusterNamespace: testNamespace,
					ArgoNamespace:    argoNamespace,
					Timeout:          5 * time.Minute,
				},
			}

			By("ensuring that the reconcile is requeued if the control plane is not ready")
			result, err = reconciler.Reconcile(
				ctx,
				reconcile.Request{
					NamespacedName: apimachinerytypes.NamespacedName{
						Namespace: testNamespace,
						Name:      vclusterName,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(10 * time.Second))

			By("creating a kubeconfig secret")
			kubeconfig := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vclusterName + "-kubeconfig",
					Namespace: testNamespace,
				},
				StringData: map[string]string{
					"value": vclusterKubeconfig443,
				},
			}
			Expect(k8sClient.Create(ctx, &kubeconfig, &client.CreateOptions{})).To(Succeed())

			By("asserting that the control plane is ready")
			vcluster.Status = capiv1beta1.ClusterStatus{
				ControlPlaneReady: true,
			}
			Expect(k8sClient.Status().Update(ctx, &vcluster, &client.SubResourceUpdateOptions{})).To(Succeed())

			By("checking that the ArgoCD cluster secret is created")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: apimachinerytypes.NamespacedName{
				Namespace: testNamespace,
				Name:      vclusterName,
			}})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(0 * time.Second))

			secret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testNamespace + "-" + vclusterName,
					Namespace: argoNamespace,
				},
			}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&secret), &secret, &client.GetOptions{})).NotTo(HaveOccurred())
			Expect(secret.Labels[argocdcommon.LabelKeySecretType]).To(Equal(argocdcommon.LabelValueSecretTypeCluster))
			Expect(secret.Data).NotTo(BeEmpty())
			Expect(secret.Data["server"]).To(Equal([]byte("https://vcluster-1.vcluster.svc:443")))
		})

		It("should update an ArgoCD cluster secret for a VCluster", func() {
			var (
				err    error
				result reconcile.Result
			)
			vclusterName := "test-vcluster"
			By("creating a cluster object with a VCluster control plane reference")
			vcluster := capiv1beta1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vclusterName,
					Namespace: testNamespace,
				},
				Spec: capiv1beta1.ClusterSpec{
					ControlPlaneEndpoint: capiv1beta1.APIEndpoint{
						Host: vclusterName + ".vcluster.svc",
						Port: 443,
					},
					ControlPlaneRef: &corev1.ObjectReference{
						Kind:       "VCluster",
						Namespace:  testNamespace,
						Name:       vclusterName,
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
					},
					InfrastructureRef: &corev1.ObjectReference{
						Kind:       "VCluster",
						Namespace:  testNamespace,
						Name:       vclusterName,
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &vcluster, &client.CreateOptions{})).To(Succeed())

			By("creating a kubeconfig secret")
			kubeconfig := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vclusterName + "-kubeconfig",
					Namespace: testNamespace,
				},
				StringData: map[string]string{
					"value": vclusterKubeconfig443,
				},
			}
			Expect(k8sClient.Create(ctx, &kubeconfig, &client.CreateOptions{})).To(Succeed())

			By("asserting that the control plane is ready")
			vcluster.Status = capiv1beta1.ClusterStatus{
				ControlPlaneReady: true,
			}
			Expect(k8sClient.Status().Update(ctx, &vcluster, &client.SubResourceUpdateOptions{})).To(Succeed())

			By("calling the reconcile function")
			reconciler := &ClusterKubeconfigReconciler{
				Client: k8sClient,
				Options: types.Options{
					ClusterID:        "envTest",
					ClusterNamespace: testNamespace,
					ArgoNamespace:    argoNamespace,
					Timeout:          5 * time.Minute,
				},
			}

			By("checking that the ArgoCD cluster secret is created")
			result, err = reconciler.Reconcile(
				ctx,
				reconcile.Request{
					NamespacedName: apimachinerytypes.NamespacedName{
						Namespace: testNamespace,
						Name:      vclusterName,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(0 * time.Second))

			secret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testNamespace + "-" + vclusterName,
					Namespace: argoNamespace,
				},
			}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&secret), &secret, &client.GetOptions{})).NotTo(HaveOccurred())
			Expect(secret.Labels[argocdcommon.LabelKeySecretType]).To(Equal(argocdcommon.LabelValueSecretTypeCluster))
			Expect(secret.Data).NotTo(BeEmpty())
			Expect(secret.Data["server"]).To(Equal([]byte("https://vcluster-1.vcluster.svc:443")))

			By("updating the kubeconfig secret")
			kubeconfig.StringData = map[string]string{
				"value": vclusterKubeconfig6443,
			}
			Expect(k8sClient.Update(ctx, &kubeconfig, &client.UpdateOptions{})).To(Succeed())

			By("checking that the ArgoCD cluster secret is updated on another reconcile")
			result, err = reconciler.Reconcile(
				ctx,
				reconcile.Request{
					NamespacedName: apimachinerytypes.NamespacedName{
						Namespace: testNamespace,
						Name:      vclusterName,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(0 * time.Second))

			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&secret), &secret, &client.GetOptions{})).NotTo(HaveOccurred())
			Expect(secret.Labels[argocdcommon.LabelKeySecretType]).To(Equal(argocdcommon.LabelValueSecretTypeCluster))
			Expect(secret.Data).NotTo(BeEmpty())
			Expect(secret.Data["server"]).To(Equal([]byte("https://vcluster-1.vcluster.svc:6443")))
		})

		It("should remove an ArgoCD cluster secret when the VCluster is deleted", func() {
			var (
				err    error
				result reconcile.Result
			)
			vclusterName := "test-vcluster"
			By("creating a cluster object with a VCluster control plane reference")
			vcluster := capiv1beta1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vclusterName,
					Namespace: testNamespace,
				},
				Spec: capiv1beta1.ClusterSpec{
					ControlPlaneEndpoint: capiv1beta1.APIEndpoint{
						Host: vclusterName + ".vcluster.svc",
						Port: 443,
					},
					ControlPlaneRef: &corev1.ObjectReference{
						Kind:       "VCluster",
						Namespace:  testNamespace,
						Name:       vclusterName,
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
					},
					InfrastructureRef: &corev1.ObjectReference{
						Kind:       "VCluster",
						Namespace:  testNamespace,
						Name:       vclusterName,
						APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &vcluster, &client.CreateOptions{})).To(Succeed())

			By("creating a kubeconfig secret")
			kubeconfig := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vclusterName + "-kubeconfig",
					Namespace: testNamespace,
				},
				StringData: map[string]string{
					"value": vclusterKubeconfig443,
				},
			}
			Expect(k8sClient.Create(ctx, &kubeconfig, &client.CreateOptions{})).To(Succeed())

			By("asserting that the control plane is ready")
			vcluster.Status = capiv1beta1.ClusterStatus{
				ControlPlaneReady: true,
			}
			Expect(k8sClient.Status().Update(ctx, &vcluster, &client.SubResourceUpdateOptions{})).To(Succeed())

			By("calling the reconcile function")
			reconciler := &ClusterKubeconfigReconciler{
				Client: k8sClient,
				Options: types.Options{
					ClusterID:        "envTest",
					ClusterNamespace: testNamespace,
					ArgoNamespace:    argoNamespace,
					Timeout:          5 * time.Minute,
				},
			}

			By("checking that the ArgoCD cluster secret is created")
			result, err = reconciler.Reconcile(
				ctx,
				reconcile.Request{
					NamespacedName: apimachinerytypes.NamespacedName{
						Namespace: testNamespace,
						Name:      vclusterName,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(0 * time.Second))

			secret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testNamespace + "-" + vclusterName,
					Namespace: argoNamespace,
				},
			}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&secret), &secret, &client.GetOptions{})).NotTo(HaveOccurred())
			Expect(secret.Labels[argocdcommon.LabelKeySecretType]).To(Equal(argocdcommon.LabelValueSecretTypeCluster))
			Expect(secret.Data).NotTo(BeEmpty())
			Expect(secret.Data["server"]).To(Equal([]byte("https://vcluster-1.vcluster.svc:443")))

			By("deleting the kubeconfig secret and cluster")
			Expect(k8sClient.Delete(ctx, &kubeconfig, &client.DeleteOptions{})).To(Succeed())
			Expect(k8sClient.Delete(ctx, &vcluster, &client.DeleteOptions{})).To(Succeed())

			By("checking that the ArgoCD cluster secret is removed on the next reconcile")
			result, err = reconciler.Reconcile(
				ctx,
				reconcile.Request{
					NamespacedName: apimachinerytypes.NamespacedName{
						Namespace: testNamespace,
						Name:      vclusterName,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(0 * time.Second))

			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(&secret), &secret, &client.GetOptions{})
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})
