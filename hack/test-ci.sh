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
#   GINKGO_FLAGS - Flags to pass to ginkgo
#   GINKGO_FLAKE_ATTEMPTS - Number of retry attempts for flaky tests
#   COVERAGE_DIR - Directory for coverage output
#   OUTPUT - Output directory for build artifacts
#   GOARCH - Target architecture for container build

set -euo pipefail

echo "Running tests for CI..."
# Check if podman is available
if ! command -v podman >/dev/null 2>&1; then
	echo "Error: Podman is not installed or not in PATH"
	exit 1
fi

# Get the current Go architecture if GOARCH not set
GOARCH=${GOARCH:-$(go env GOARCH)}
COVERAGE_DIR=${COVERAGE_DIR:-/src/coverage}

echo "Creating test environment image..."
CONTAINER_PLATFORM="linux/${GOARCH}" \
CONTAINER_GOARCH="${GOARCH}" \
CONTAINER_TAG="korn-test-env" \
make container-env

echo "Running tests in container..."
podman run --rm \
	--platform=linux/"${GOARCH}" \
	-v "$(pwd)":/src:rw,Z \
	-w /src \
	-e ENVTEST_VERSION="${ENVTEST_VERSION}" \
	-e ENVTEST_K8S_VERSION="${ENVTEST_K8S_VERSION}" \
	-e GINKGO_FLAGS="${GINKGO_FLAGS}" \
	-e GINKGO_FLAKE_ATTEMPTS="${GINKGO_FLAKE_ATTEMPTS}" \
	-e COVERAGE_DIR="${COVERAGE_DIR}" \
	-e OUTPUT="${OUTPUT}" \
	-e USER_ID="$(id -u)" \
	-e GROUP_ID="$(id -g)" \
	korn-test-env \
	/src/hack/run-test.sh
echo "Tests completed successfully."