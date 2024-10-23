#!/bin/bash
set -eou pipefail

[ -x kind ] && echo "Please install kind before continuing, see README.md for instructions" >&2 && exit 1
[ -x kubectl ] && echo "Please install kubectl before continuing, see README.md for instructions" >&2 && exit 1
[ -x argocd ] && echo "Please install argocd before continuing, see README.md for instructions" >&2 && exit 1
[ -x vcluster ] && echo "Please install vcluster before continuing, see README.md for instructions" >&2 && exit 1

kubeconfig_file="$(mktemp)"
trap 'rm -f ${kubeconfig_file}' 0 2 3 15

cluster_name="${1:-kind}"
kind get kubeconfig --name "${cluster_name}" > "${kubeconfig_file}"

# Deploy capargo
kubectl --kubeconfig "${kubeconfig_file}" apply -k manifests/kustomize
kubectl --kubeconfig "${kubeconfig_file}" rollout --namespace capargo status deployment capargo

# Create vcluster cluster
VCLUSTER_NAME=vcluster-1
VCLUSTER_NAMESPACE=vcluster
CHART_VERSION=0.19.7

cat <<EOF | kubectl --kubeconfig "${kubeconfig_file}" apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: "${VCLUSTER_NAMESPACE}"
EOF
kubectl --kubeconfig "${kubeconfig_file}" wait --for jsonpath='{.status.phase}=Active' --timeout=5s namespace/vcluster

cat <<EOF | kubectl --kubeconfig "${kubeconfig_file}" apply -f -
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: "${VCLUSTER_NAME}"
  namespace: "${VCLUSTER_NAMESPACE}"
spec:
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: VCluster
    name: "${VCLUSTER_NAME}"
  controlPlaneRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: VCluster
    name: "${VCLUSTER_NAME}"
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: VCluster
metadata:
  name: "${VCLUSTER_NAME}"
  namespace: "${VCLUSTER_NAMESPACE}"
spec:
  helmRelease:
    values: |-
      service:
        type: NodePort
      syncer:
        extraArgs:
        - --tls-san=${VCLUSTER_NAME}.${VCLUSTER_NAMESPACE}.svc
    chart:
      repo: https://charts.loft.sh
      name: vcluster
      version: ${CHART_VERSION}
EOF
kubectl --kubeconfig "${kubeconfig_file}" wait --for jsonpath='{.status.phase}=Provisioned' --timeout=120s --namespace "${VCLUSTER_NAMESPACE}" cluster/"${VCLUSTER_NAME}"

# Log into ArgoCD
KUBECONFIG="${kubeconfig_file}" argocd login \
  --port-forward-namespace argocd \
  --port-forward \
  --username admin \
  --password "$(kubectl --kubeconfig "${kubeconfig_file}" get secret -n argocd argocd-initial-admin-secret -o template --template '{{.data.password}}' | base64 --decode)"

# Create test Application for vcluster
KUBECONFIG="${kubeconfig_file}" argocd app create guestbook \
  --port-forward-namespace argocd \
  --port-forward \
  --repo https://github.com/argoproj/argocd-example-apps.git \
  --path guestbook \
  --dest-namespace default \
  --dest-name "${VCLUSTER_NAME}" \
  --sync-policy=automated

# Wait until vcluster has the Application deployed to it
KUBECONFIG="${kubeconfig_file}" vcluster connect "${VCLUSTER_NAME}" --namespace "${VCLUSTER_NAMESPACE}"
KUBECONFIG="${kubeconfig_file}" kubectl rollout status deployment guestbook-ui
KUBECONFIG="${kubeconfig_file}" vcluster disconnect
