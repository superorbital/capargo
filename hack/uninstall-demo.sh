#!/bin/bash
set -eou pipefail

[ -x kind ] && echo "Please install kind before continuing, see README.md for instructions" >&2 && exit 1
[ -x kubectl ] && echo "Please install kubectl before continuing, see README.md for instructions" >&2 && exit 1


kubeconfig_file="$(mktemp)"
trap 'rm -f ${kubeconfig_file}' 0 2 3 15

cluster_name="${1:-kind}"
kind get kubeconfig --name "${cluster_name}" > "${kubeconfig_file}"

kubectl --kubeconfig "${kubeconfig_file}" delete application -n argocd guestbook --ignore-not-found
kubectl --kubeconfig "${kubeconfig_file}" delete cluster -n vcluster vcluster-1 --ignore-not-found
kubectl --kubeconfig "${kubeconfig_file}" delete namespace vcluster --ignore-not-found
kubectl --kubeconfig "${kubeconfig_file}" delete namespace capargo --ignore-not-found
