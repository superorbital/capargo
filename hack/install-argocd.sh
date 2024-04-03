#!/bin/bash
set -eou pipefail

[ -x kind ] && echo "Please install kind before continuing, see README.md for instructions" >&2 && exit 1
[ -x kubectl ] && echo "Please install kubectl before continuing, see README.md for instructions" >&2 && exit 1

kubeconfig_file="$(mktemp)"
trap 'rm -f ${kubeconfig_file}' 0 2 3 15

cluster_name="${1:-kind}"
kind get kubeconfig --name "${cluster_name}" > "${kubeconfig_file}"

kubectl --kubeconfig "${kubeconfig_file}" create namespace argocd
kubectl --kubeconfig "${kubeconfig_file}" wait --for jsonpath='{.status.phase}=Active' --timeout=5s namespace/argocd

kubectl --kubeconfig "${kubeconfig_file}" apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl --kubeconfig "${kubeconfig_file}" rollout -n argocd status deployment argocd-server
