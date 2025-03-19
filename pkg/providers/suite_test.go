package providers

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadmv1beta1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
)

var (
	cfg       *rest.Config
	ctx       context.Context
	cancel    context.CancelFunc
	envTest   *envtest.Environment
	crds      []crdInfo
	k8sClient client.Client
)

func TestProviders(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(time.Second * 10)
	SetDefaultEventuallyPollingInterval(time.Millisecond * 100)
	RunSpecs(t, "Providers Suite")
}

type crdInfo struct {
	Names        apiextensionsv1.CustomResourceDefinitionNames
	Scope        apiextensionsv1.ResourceScope
	GroupVersion schema.GroupVersion
}

func createCRDs(info []crdInfo) []*apiextensionsv1.CustomResourceDefinition {
	crds := []*apiextensionsv1.CustomResourceDefinition{}
	for i := range info {
		crd := apiextensionsv1.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				Kind:       info[i].Names.Kind,
				APIVersion: fmt.Sprintf("%s/%s", info[i].GroupVersion.Group, info[i].GroupVersion.Version),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", info[i].Names.Plural, info[i].GroupVersion.Group),
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: info[i].GroupVersion.Group,
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    info[i].GroupVersion.Version,
						Served:  true,
						Storage: true,
						Schema: &apiextensionsv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
								Type:                   "object",
								XPreserveUnknownFields: func(v bool) *bool { return &v }(true),
							},
						},
						AdditionalPrinterColumns: []apiextensionsv1.CustomResourceColumnDefinition{},
						Subresources: &apiextensionsv1.CustomResourceSubresources{
							Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
						},
					},
				},
				Scope: info[i].Scope,
				Names: info[i].Names,
			},
		}
		crds = append(crds, &crd)
	}
	return crds
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())
	crds = []crdInfo{
		{
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Singular: "cluster",
				Plural:   "clusters",
				Kind:     "Cluster",
			},
			Scope:        apiextensionsv1.NamespaceScoped,
			GroupVersion: capiv1beta1.GroupVersion,
		},
		{
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Singular: "kubeadmcontrolplane",
				Plural:   "kubeadmcontrolplanes",
				Kind:     "KubeadmControlPlane",
			},
			Scope:        apiextensionsv1.NamespaceScoped,
			GroupVersion: kubeadmv1beta1.GroupVersion,
		},
	}
	testCRDs := createCRDs(crds)
	By("bootstrapping the envtest test environment")
	envTest = &envtest.Environment{
		CRDs: testCRDs,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}
	var err error
	cfg, err = envTest.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Add CRDs to Scheme
	err = capiv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = kubeadmv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Create client for envTest
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := envTest.Stop()
	Expect(err).NotTo(HaveOccurred())
})
