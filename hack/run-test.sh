#!/bin/bash

# Container test runner script
# This script runs tests inside a container environment with proper setup
#
# NOTE: This file was generated/enhanced with AI assistance (Cursor)
# All functionality has been reviewed and tested for correctness
#
# Usage:
#   ./hack/run-test.sh                              # Run with default values (verbose output enabled)
#   GINKGO_VERBOSE=false ./hack/run-test.sh         # Run with quiet output
#   GINKGO_VERBOSE=true ./hack/run-test.sh          # Run with verbose output (explicit)
#   GINKGO_PKG="-r --focus=MyTest" ./hack/run-test.sh # Run specific tests
#   GINKGO_FLAKE_ATTEMPTS=3 ./hack/run-test.sh      # Run with 3 retry attempts for flaky tests
#   COVERAGE_DIR=./custom ./hack/run-test.sh        # Use custom coverage directory
#
# Ginkgo flags are built dynamically from: GINKGO_VERBOSE, GINKGO_PKG, GINKGO_FLAKE_ATTEMPTS, COVERAGE_DIR

set -xeuo pipefail

uname -a

env | sort
# Get parameters from environment or use defaults (same as Makefile)
ENVTEST_VERSION=${ENVTEST_VERSION:-"release-0.20"}
ENVTEST_K8S_VERSION=${ENVTEST_K8S_VERSION:-"1.32.0"}
GINKGO_PKG=${GINKGO_PKG:-"-r"}
GINKGO_VERBOSE=${GINKGO_VERBOSE:-"true"}
GINKGO_FLAKE_ATTEMPTS=${GINKGO_FLAKE_ATTEMPTS:-"3"}
USER_ID=${USER_ID:-$(id -u)}
GROUP_ID=${GROUP_ID:-$(id -g)}
COVERAGE_DIR=${COVERAGE_DIR:-$(pwd)/coverage}
LOCALBIN=${LOCALBIN:-$(pwd)/bin/$(uname -s)_$(uname -m)}
OUTPUT=${OUTPUT:-$(pwd)/output}

# Note: Ginkgo flags will be built dynamically at runtime from individual variables

echo "Setting up test environment in container..."
echo "ENVTEST_VERSION: ${ENVTEST_VERSION}"
echo "ENVTEST_K8S_VERSION: ${ENVTEST_K8S_VERSION}"
echo "GINKGO_PKG: ${GINKGO_PKG}"
echo "GINKGO_VERBOSE: ${GINKGO_VERBOSE}"
echo "GINKGO_FLAKE_ATTEMPTS: ${GINKGO_FLAKE_ATTEMPTS}"
echo "COVERAGE_DIR: ${COVERAGE_DIR}"
echo "LOCALBIN: ${LOCALBIN}"

# Validate required commands are available
if ! command -v go >/dev/null 2>&1; then
    echo "Error: go command not found in PATH"
    exit 1
fi


# Add local bin directory to PATH (like Makefile does with GOBIN=$(LOCALBIN))
echo "Creating directories..."
mkdir -p coverage "${LOCALBIN}"
export PATH=$PATH:${LOCALBIN}

# Install test dependencies to local bin directory (like Makefile does with GOBIN=$(LOCALBIN))
echo "Installing setup-envtest..."
GOBIN=${LOCALBIN} go install "sigs.k8s.io/controller-runtime/tools/setup-envtest@${ENVTEST_VERSION}"

echo "Installing ginkgo..."
GOBIN=${LOCALBIN} go install "github.com/onsi/ginkgo/v2/ginkgo"

# Create necessary directories

# Set up test environment
echo "Setting up KUBEBUILDER_ASSETS..."
file $(command -v setup-envtest)

KUBEBUILDER_ASSETS="$(setup-envtest use "${ENVTEST_K8S_VERSION}" --bin-dir ${LOCALBIN} -p path)"
if [[ -z "${KUBEBUILDER_ASSETS}" ]]; then
    echo "Error: Failed to set up KUBEBUILDER_ASSETS"
    exit 1
fi
export KUBEBUILDER_ASSETS
export ENVTEST_K8S_VERSION

echo "KUBEBUILDER_ASSETS: ${KUBEBUILDER_ASSETS}"

# Run tests
echo "Running ginkgo tests..."

# Build ginkgo flags dynamically from individual variables
DYNAMIC_FLAGS=()

# Add verbose flag if enabled
if [[ "${GINKGO_VERBOSE}" == "true" ]]; then
    DYNAMIC_FLAGS+=("-v")
fi

# Add package selector
DYNAMIC_FLAGS+=("${GINKGO_PKG}")

# Add standard flags with current values
DYNAMIC_FLAGS+=("--mod=mod")
DYNAMIC_FLAGS+=("--randomize-all")
DYNAMIC_FLAGS+=("--randomize-suites")
DYNAMIC_FLAGS+=("--cover")
DYNAMIC_FLAGS+=("--coverprofile=coverage.out")
DYNAMIC_FLAGS+=("--coverpkg=./...")
DYNAMIC_FLAGS+=("--output-dir=${COVERAGE_DIR}")
DYNAMIC_FLAGS+=("--flake-attempts=${GINKGO_FLAKE_ATTEMPTS}")
DYNAMIC_FLAGS+=("-p")

echo "Dynamic ginkgo flags: ${DYNAMIC_FLAGS[*]}"

# Execute ginkgo with dynamically built flags
"${LOCALBIN}/ginkgo" "${DYNAMIC_FLAGS[@]}"

# Fix ownership of generated files (only needed in container environments)
if [[ -d "/src" ]] && [[ "${USER_ID}" =~ ^[0-9]+$ ]] && [[ "${GROUP_ID}" =~ ^[0-9]+$ ]]; then
    echo "Container environment detected - fixing file ownership..."
    chown -R "${USER_ID}:${GROUP_ID}" "/src/coverage/}" 2>/dev/null || {
        echo "Warning: Failed to change ownership of generated files"
    }
fi


