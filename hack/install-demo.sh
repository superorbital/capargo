#!/bin/bash
set -eou pipefail

[ -x kind ] && echo "Please install kind before continuing, see README.md for instructions" >&2 && exit 1
[ -x kubectl ] && echo "Please install kubectl before continuing, see README.md for instructions" >&2 && exit 1
[ -x argocd ] && echo "Please install argocd before continuing, see README.md for instructions" >&2 && exit 1

# Installs a released binary from a given Github repository
# Inputs:
# 1. Binary name
# 2. Version of the binary
# 3. Installation directory for the binary
# 4. Org name and repo name as <ORG>/<REPO>, e.g.: "argoproj/argocd"
install_github_binary() {
  local binary="${1}"
  local binary_version="${2}"
  local binary_install_dir="${3}"
  local github_path="${4}"
  
  local fullpath="${binary_install_dir}/${binary}-${binary_version}"
  local arch
  if [ ! -f "${fullpath}" ]; then
    arch="$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)"
    echo "Downloading ${binary} ${binary_version} for ${arch}"
    curl --location \
        --silent \
        --fail \
        --output "${fullpath}" \
        "https://github.com/${github_path}/releases/download/${binary_version}/${binary}-${arch}"
    chmod 755 "${fullpath}"
  fi
  ln -sf "${fullpath}" "${binary_install_dir}/${binary}"
}

kubeconfig_file="$(mktemp)"
trap 'rm -f ${kubeconfig_file}' 0 2 3 15

cluster_name="${1:-kind}"
kind get kubeconfig --name "${cluster_name}" > "${kubeconfig_file}"

# Deploy capargo
kubectl --kubeconfig "${kubeconfig_file}" apply -k manifests/overlays/local
kubectl --kubeconfig "${kubeconfig_file}" rollout --namespace capargo status deployment capargo

# Install vcluster binary
VCLUSTER=bin/vcluster
VCLUSTER_BIN_VERSION=v0.19.7
install_github_binary "vcluster" "${VCLUSTER_BIN_VERSION}" "$(pwd)/bin" "loft-sh/vcluster"

# Install argocd binary
ARGOCD=bin/argocd
ARGOCD_BIN_VERSION=v2.13.1
install_github_binary "argocd" "${ARGOCD_BIN_VERSION}" "$(pwd)/bin" "argoproj/argo-cd"

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
KUBECONFIG="${kubeconfig_file}" "${ARGOCD}" login \
  --port-forward-namespace argocd \
  --port-forward \
  --username admin \
  --password "$(kubectl --kubeconfig "${kubeconfig_file}" get secret -n argocd argocd-initial-admin-secret -o template --template '{{.data.password}}' | base64 --decode)"

# Create test Application for vcluster
KUBECONFIG="${kubeconfig_file}" "${ARGOCD}" app create guestbook \
  --port-forward-namespace argocd \
  --port-forward \
  --repo https://github.com/argoproj/argocd-example-apps.git \
  --path guestbook \
  --dest-namespace default \
  --dest-name "${VCLUSTER_NAME}" \
  --sync-policy=automated

# Wait until vcluster has the Application deployed to it
KUBECONFIG="${kubeconfig_file}" "${VCLUSTER}" connect "${VCLUSTER_NAME}" --namespace "${VCLUSTER_NAMESPACE}"
KUBECONFIG="${kubeconfig_file}" kubectl rollout status deployment guestbook-ui
KUBECONFIG="${kubeconfig_file}" "${VCLUSTER}" disconnect
