#!/bin/sh

set -e

make run &

(
  export CLUSTER_NAME=kind
  export CLUSTER_NAMESPACE=vcluster
  export KUBERNETES_VERSION=v1.29.2
  export HELM_VALUES="service:\n  type: NodePort"

  clusterctl generate cluster ${CLUSTER_NAME} \
      --infrastructure vcluster \
      --kubernetes-version ${KUBERNETES_VERSION} \
      --target-namespace ${CLUSTER_NAMESPACE} | kubectl apply -f -
)

sleep 30

ARGOCD_PASS=$(kubectl get secret -n argocd argocd-initial-admin-secret -o template --template {{.data.password}} | base64 --decode)

argocd login --username admin --password "${ARGOCD_PASS}" --port-forward-namespace argocd --port-forward

argocd --port-forward-namespace argocd --port-forward app create guestbook --repo https://github.com/argoproj/argocd-example-apps.git --path guestbook --dest-namespace default --dest-name kind --sync-policy=automated

vcluster connect kind -n vcluster
kubectl rollout status deployment guestbook-ui
vcluster disconnect

pkill -f "make run"