#!/bin/bash

# CI test runner script
# This script runs tests in container environment for CI
#
# Usage:
#   ./hack/test-ci.sh
#
# Environment variables:
#   ENVTEST_VERSION - Version of setup-envtest to use
#   ENVTEST_K8S_VERSION - Kubernetes version for test environment
#   GINKGO_PKG - Ginkgo package selector (default: -r)
#   GINKGO_FLAKE_ATTEMPTS - Number of retry attempts for flaky tests
#   GINKGO_VERBOSE - Enable verbose ginkgo output (default: true)
#   COVERAGE_DIR - Directory for coverage output
#   OUTPUT - Output directory for build artifacts
#   GOARCH - Target architecture for container build
#   CONTAINER_FULL_NAME - Full container image name (default: quay.io/jordigilh/korn-build-container:latest)
#
# Note: Ginkgo flags are built dynamically from individual variables at runtime

set -euo pipefail

# Default container image name
CONTAINER_FULL_NAME=${CONTAINER_FULL_NAME:-"quay.io/jordigilh/korn-build-container:latest"}

echo "Running tests for CI..."
# Check if podman is available
if ! command -v podman >/dev/null 2>&1; then
	echo "Error: Podman is not installed or not in PATH"
	exit 1
fi

# Get the current Go architecture if GOARCH not set
GOARCH=${GOARCH:-$(go env GOARCH)}

echo "Pulling container image: ${CONTAINER_FULL_NAME}"
podman pull "${CONTAINER_FULL_NAME}"

echo "Running tests in container..."
podman run --rm \
	--platform=linux/"${GOARCH}" \
	-v "$(pwd)":/src:rw,Z \
	-w /src \
	-e ENVTEST_VERSION="${ENVTEST_VERSION}" \
	-e ENVTEST_K8S_VERSION="${ENVTEST_K8S_VERSION}" \
	-e GINKGO_PKG="${GINKGO_PKG}" \
	-e GINKGO_FLAKE_ATTEMPTS="${GINKGO_FLAKE_ATTEMPTS}" \
	-e GINKGO_VERBOSE="${GINKGO_VERBOSE}" \
	-e COVERAGE_DIR="/src/coverage" \
	-e OUTPUT="/src/output" \
	-e USER_ID="$(id -u)" \
	-e GROUP_ID="$(id -g)" \
	"${CONTAINER_FULL_NAME}" \
	/src/hack/run-test.sh
echo "Tests completed successfully."