package controller

import (
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"k8s.io/client-go/rest"
)

func GetArgoClusterConfig() argoapp.ClusterConfig {
	return argoapp.ClusterConfig{}
}

func LoadRestConfigFromSecret() rest.Config {
	return rest.Config{}
}
