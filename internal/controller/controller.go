package controller

import (
	"context"

	"github.com/superorbital/capargo/pkg/types"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/kubetypes"
	"k8s.io/client-go/rest"

	corev1 "k8s.io/api/core/v1"
	cav1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// TODO: flags
const (
	vclusterNamespace = "vcluster"
	argoNamespace     = "argocd"
)

func GetArgoConfigFromSecret() types.ClusterConfig {
	return types.ClusterConfig{}
}

func LoadRestConfigFromSecret() rest.Config {
	return rest.Config{}
}

type ArgoClusterSecretController struct {
	client      kube.Client
	vsecretColl krt.Collection[*corev1.Secret]
	argoSecColl krt.Collection[*corev1.Secret]
	clusterColl krt.Collection[*cav1beta1.Cluster]
}

func NewController(client kube.Client) *ArgoClusterSecretController {
	vsecretColl := krt.NewInformerFiltered[*corev1.Secret](client, kubetypes.Filter{
		Namespace: vclusterNamespace,
	})
	clusterColl := krt.NewInformerFiltered[*cav1beta1.Cluster](client, kubetypes.Filter{
		Namespace: vclusterNamespace,
	})
	argoSecColl := krt.NewInformerFiltered[*corev1.Secret](client, kubetypes.Filter{
		Namespace:     argoNamespace,
		LabelSelector: "argocd.argoproj.io/secret-type=cluster",
	})

	return &ArgoClusterSecretController{
		client:      client,
		argoSecColl: argoSecColl,
		vsecretColl: vsecretColl,
		clusterColl: clusterColl,
	}
}

func (a *ArgoClusterSecretController) Start(ctx context.Context) error {
	return nil
}
