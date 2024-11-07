#!/bin/bash
set -eou pipefail

LOCAL_BIN="${1}"
BINARY="${2}"
PACKAGE_NAME="${3}"
PACKAGE_VERSION="${4}"

TOOL_SYMLINK="${LOCAL_BIN}/${BINARY}"
TOOL_FULLPATH="${TOOL_SYMLINK}-${PACKAGE_VERSION}"

if [ ! -f "${TOOL_FULLPATH}" ]; then
    echo "Downloading ${BINARY}@${PACKAGE_VERSION}"
    rm -f "${TOOL_SYMLINK}" || true
    GOBIN="${LOCAL_BIN}" go install "${PACKAGE_NAME}@${PACKAGE_VERSION}"
    mv "${TOOL_SYMLINK}" "${TOOL_FULLPATH}"
fi

ln -sf "${TOOL_FULLPATH}" "${TOOL_SYMLINK}"
