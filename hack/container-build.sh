#!/bin/bash

# Container build script
# This script builds multiplatform container images for linux/amd64 and linux/arm64
#
# NOTE: This file was generated/enhanced with AI assistance (Cursor)
# All functionality has been reviewed and tested for correctness
#
# Usage:
#   ./hack/container-build.sh
#
# Environment variables:
#   CONTAINER_FULL_NAME - Full container image name (default: quay.io/jordigilh/korn-build-container:latest)

set -euo pipefail

# Default container image name
CONTAINER_FULL_NAME=${CONTAINER_FULL_NAME:-"quay.io/jordigilh/korn-build-container:latest"}

echo "Building multiplatform container images..."

# Check if podman is available
if ! command -v podman >/dev/null 2>&1; then
    echo "Error: Podman is not installed or not in PATH"
    exit 1
fi

echo "Building for linux/amd64..."
podman build --platform=linux/amd64 \
    --build-arg GOARCH=amd64 \
    -f build/Containerfile.build \
    -t "${CONTAINER_FULL_NAME}-amd64" .

echo "Building for linux/arm64..."
podman build --platform=linux/arm64 \
    --build-arg GOARCH=arm64 \
    -f build/Containerfile.build \
    -t "${CONTAINER_FULL_NAME}-arm64" .

echo "Creating multiplatform manifest..."
podman manifest create "${CONTAINER_FULL_NAME}" \
    "${CONTAINER_FULL_NAME}-amd64" \
    "${CONTAINER_FULL_NAME}-arm64"

echo "Multiplatform container images built: ${CONTAINER_FULL_NAME}"