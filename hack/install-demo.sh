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
kubectl --kubeconfig "${kubeconfig_file}" create namespace vcluster
kubectl --kubeconfig "${kubeconfig_file}" wait --for jsonpath='{.status.phase}=Active' --timeout=5s namespace/vcluster
HELM_VALUES="service:\n  type: NodePort" clusterctl generate cluster vcluster-1 \
  --kubeconfig "${kubeconfig_file}" \
  --infrastructure vcluster \
  --kubernetes-version v1.29.2 \
  --target-namespace vcluster | kubectl  --kubeconfig "${kubeconfig_file}" apply -f -
kubectl --kubeconfig "${kubeconfig_file}" wait --for jsonpath='{.status.phase}=Provisioned' --timeout=120s --namespace vcluster cluster/vcluster-1

# Log into ArgoCD
KUBECONFIG="${kubeconfig_file}" argocd login \
  --port-forward-namespace argocd \
  --port-forward \
  --username admin \
  --password "$(kubectl --kubeconfig "${kubeconfig_file}" get secret -n argocd argocd-initial-admin-secret -o template --template '{{.data.password}}' | base64 --decode)"

# Create test Application for vcluster-1
KUBECONFIG="${kubeconfig_file}" argocd app create guestbook \
  --port-forward-namespace argocd \
  --port-forward \
  --repo https://github.com/argoproj/argocd-example-apps.git \
  --path guestbook \
  --dest-namespace default \
  --dest-name vcluster-1 \
  --sync-policy=automated

# Wait until vcluster-1 has the Application deployed to it
KUBECONFIG="${kubeconfig_file}" vcluster connect vcluster-1 --namespace vcluster
KUBECONFIG="${kubeconfig_file}" kubectl rollout status deployment guestbook-ui
KUBECONFIG="${kubeconfig_file}" vcluster disconnect
