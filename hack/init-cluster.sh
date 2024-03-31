#!/bin/sh

kubectl create ns argocd

sleep 2

kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

kubectl rollout -n argocd status deployment argocd-server

clusterctl init --infrastructure vcluster

kubectl rollout -n capi-system status deployment argocd-server

sleep 5
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

if [ "${FULL_TEST_RUN}" = "true" ]; then

  ARGOCD_PASS=$(kubectl get secret -n argocd argocd-initial-admin-secret -o template --template {{.data.password}} | base64 --decode)

  argocd login --username admin --password "${ARGOCD_PASS}" --port-forward-namespace argocd --port-forward

  argocd app create guestbook --repo https://github.com/argoproj/argocd-example-apps.git --path guestbook --dest-namespace default --dest-name vcluster-kind --sync-policy=automated

  vcluster connect kind -n vcluster
  kubectl rollout status deployment guestbook-ui
  vcluster disconnect
fi