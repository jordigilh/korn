#!/bin/bash

# CI build runner script
# This script builds the korn binary using containers for consistent builds
#
# Usage:
#   ./hack/build-ci.sh
#
# Environment variables:
#   GOOS - Target operating system
#   GOARCH - Target architecture
#   VERSION - Version to embed in binary
#   PROJECT_NAME - Name of the project/binary
#   BINARY_NAME - Name of the output binary
#   OUTPUT - Output directory for build artifacts
#   CONTAINER_FULL_NAME - Full container image name (default: quay.io/jordigilh/korn-build-container:latest)

set -euo pipefail

# Default container image name
CONTAINER_FULL_NAME=${CONTAINER_FULL_NAME:-"quay.io/jordigilh/korn-build-container:latest"}

echo "Building ${BINARY_NAME} for ${GOOS}/${GOARCH}..."

# Validate Darwin architecture restriction
if [ "${GOOS}" = "darwin" ] && [ "${GOARCH}" != "arm64" ]; then
    echo "Error: Darwin builds are only supported for arm64 architecture"
    echo "Use 'make darwin-arm64' or set GOARCH=arm64"
    exit 1
fi

# Container builds for Linux and Darwin
if [ "${GOOS}" = "linux" ] || [ "${GOOS}" = "darwin" ]; then
    echo "Using container build for ${GOOS}/${GOARCH} target..."

    # Check if podman is available
    if ! command -v podman >/dev/null 2>&1; then
        echo "Error: Podman is not installed or not in PATH"
        exit 1
    fi

    # Set platform and build architecture
    if [ "${GOOS}" = "linux" ]; then
        platform="linux/${GOARCH}"
        build_arch="${GOARCH}"
    else
        # Darwin builds use linux/arm64 platform for cross-compilation
        platform="linux/arm64"
        build_arch="arm64"
    fi

    echo "Pulling container image: ${CONTAINER_FULL_NAME}"
    podman pull "${CONTAINER_FULL_NAME}"

    echo "Building binary in container..."
    podman run --rm \
        --platform="${platform}" \
        -v "$(pwd)":/src:rw,Z \
        -w /src \
        -e VERSION="${VERSION}" \
        -e GOOS="${GOOS}" \
        -e GOARCH="${GOARCH}" \
        -e PROJECT_NAME="${PROJECT_NAME}" \
        "${CONTAINER_FULL_NAME}" \
        sh -c "mkdir -p output && GOOS=\${GOOS} GOARCH=\${GOARCH} go build -mod=mod -ldflags=\"-s -w -X main.version=\${VERSION}\" -o output/\${PROJECT_NAME}_\${VERSION}_\${GOOS}_\${GOARCH} main.go"
else
    # Native builds for other platforms
    echo "Using native build for ${GOOS} target..."
    echo "Note: This target requires native Go environment for best compatibility"
    GOOS="${GOOS}" GOARCH="${GOARCH}" go build -mod=mod -ldflags="-s -w -X main.version=${VERSION}" -o "${OUTPUT}/${BINARY_NAME}" main.go
fi

echo "Binary built: ${OUTPUT}/${BINARY_NAME}"