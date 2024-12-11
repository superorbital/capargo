#!/bin/bash
set -eou pipefail

OUTPUT_DIR="${OUTPUT_DIR:-_output}"
IMAGE_VERSION=${1}

mkdir -p "${OUTPUT_DIR}"


SED="sed"
# Requires gsed to be installed via brew if on MacOS
if [[ $(uname -s | tr '[:upper:]' '[:lower:]') == "darwin" ]]; then
  SED="gsed"
fi

# Create install manifest
echo "# This is an auto-generated file. DO NOT EDIT." > "${OUTPUT_DIR}"/install.yaml
kubectl kustomize manifests/overlays/default >> "${OUTPUT_DIR}"/install.yaml
"${SED}" -i -e "s|image: superorbital/capargo:latest|image: superorbital/capargo:${IMAGE_VERSION}|g" "${OUTPUT_DIR}"/install.yaml
