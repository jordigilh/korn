#!/bin/bash

# Container test runner script
# This script runs tests inside a container environment with proper setup
#
# Usage:
#   ./hack/run-test.sh                              # Run with default values (verbose output enabled)
#   GINKGO_VERBOSE=false ./hack/run-test.sh         # Run with quiet output
#   GINKGO_VERBOSE=true ./hack/run-test.sh          # Run with verbose output (explicit)
#   GINKGO_FLAKE_ATTEMPTS=3 ./hack/run-test.sh      # Run with 3 retry attempts for flaky tests
#   GINKGO_FLAGS="-r --dry-run" ./hack/run-test.sh  # Run with custom flags
#
# Default GINKGO_FLAGS: "-r --mod=mod --randomize-all --randomize-suites --cover --coverprofile=coverage.out --coverpkg=./... --output-dir=./coverage --flake-attempts=2"

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

# Set default GINKGO_FLAGS using the same logic as the Makefile
# GINKGO_FLAGS := $(if $(filter true,$(GINKGO_VERBOSE)),-vv) $(GINKGO_PKG) --mod=mod --randomize-all --randomize-suites --cover --coverprofile=coverage.out --coverpkg=./... --output-dir=$(COVERAGE_DIR) --flake-attempts=$(GINKGO_FLAKE_ATTEMPTS) -p
if [[ -z "${GINKGO_FLAGS:-}" ]]; then
    # Construct default GINKGO_FLAGS using the same logic as the Makefile
    GINKGO_FLAGS=""
    if [[ "${GINKGO_VERBOSE}" == "true" ]]; then
        GINKGO_FLAGS="${GINKGO_FLAGS} -vv"
    fi
    GINKGO_FLAGS="${GINKGO_FLAGS} ${GINKGO_PKG} --mod=mod --randomize-all --randomize-suites --cover --coverprofile=coverage.out --coverpkg=./... --output-dir=${COVERAGE_DIR} --flake-attempts=${GINKGO_FLAKE_ATTEMPTS} -p"
    # Clean up leading space
    GINKGO_FLAGS="${GINKGO_FLAGS# }"
    echo "Using default GINKGO_FLAGS: ${GINKGO_FLAGS}"
else
    echo "Using provided GINKGO_FLAGS: ${GINKGO_FLAGS}"
fi

echo "Setting up test environment in container..."
echo "ENVTEST_VERSION: ${ENVTEST_VERSION}"
echo "ENVTEST_K8S_VERSION: ${ENVTEST_K8S_VERSION}"
echo "COVERAGE_DIR: ${COVERAGE_DIR}"
echo "GINKGO_FLAKE_ATTEMPTS: ${GINKGO_FLAKE_ATTEMPTS}"
echo "GINKGO_FLAGS: ${GINKGO_FLAGS}"
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
if [[ -n "${GINKGO_FLAGS}" ]]; then
    # Use array to properly handle multiple flags
    read -ra FLAGS <<< "${GINKGO_FLAGS}"
    "${LOCALBIN}/ginkgo" "${FLAGS[@]}"
else
    "${LOCALBIN}/ginkgo"
fi

# Fix ownership of generated files (only needed in container environments)
if [[ -d "/src" ]] && [[ "${USER_ID}" =~ ^[0-9]+$ ]] && [[ "${GROUP_ID}" =~ ^[0-9]+$ ]]; then
    echo "Container environment detected - fixing file ownership..."
    chown -R "${USER_ID}:${GROUP_ID}" "/src/coverage/}" 2>/dev/null || {
        echo "Warning: Failed to change ownership of generated files"
    }
fi


