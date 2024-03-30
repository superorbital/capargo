#!/bin/sh

kubectl create ns argocd

sleep 2

kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

kubectl rollout -n argocd status deployment argocd-server

clusterctl init --infrastructure vcluster

(
  export CLUSTER_NAME=kind
  export CLUSTER_NAMESPACE=vcluster
  export KUBERNETES_VERSION=v1.29.2
  export HELM_VALUES="service:\n  type: NodePort"

  kubectl create namespace ${CLUSTER_NAMESPACE}
  clusterctl generate cluster ${CLUSTER_NAME} \
      --infrastructure vcluster \
      --kubernetes-version ${KUBERNETES_VERSION} \
      --target-namespace ${CLUSTER_NAMESPACE} | kubectl apply -f -
)
