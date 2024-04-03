#!/bin/bash
set -eou pipefail

[ -x kind ] && echo "Please install kind before continuing, see README.md for instructions" >&2 && exit 1
[ -x kubectl ] && echo "Please install kubectl before continuing, see README.md for instructions" >&2 && exit 1
[ -x clusterctl ] && echo "Please install clusterctl before continuing, see README.md for instructions" >&2 && exit 1

kubeconfig_file="$(mktemp)"
trap 'rm -f ${kubeconfig_file}' 0 2 3 15

cluster_name="${1:-kind}"
kind get kubeconfig --name "${cluster_name}" > "${kubeconfig_file}"

clusterctl --kubeconfig "${kubeconfig_file}" init --infrastructure vcluster
kubectl --kubeconfig "${kubeconfig_file}" rollout -n capi-system status deployment capi-controller-manager
