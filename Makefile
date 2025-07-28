
# Version settings - can be overridden with make VERSION=vX.Y.Z build
VERSION ?= $(shell git describe --tags --always --dirty || echo "dev")
PROJECT_NAME := korn
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)


# Binary filename following GitHub standard: <project-name>_<version>_<os>_<arch>
BINARY_NAME := $(PROJECT_NAME)_$(VERSION)_$(GOOS)_$(GOARCH)

GO_VERSION := 1.23.6
CONTROLLER_TOOLS_VERSION ?= v0.17.2

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.32.0
ENVTEST_VERSION ?= release-0.20
ENVTEST ?= $(LOCALBIN)/setup-envtest
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

GOTOOLCHAIN := go$(GO_VERSION)

COVERAGE_DIR ?= $(shell pwd)/coverage
LOCALBIN ?= $(shell pwd)/bin/$(shell uname -s)_$(shell uname -m)
OUTPUT ?= $(shell pwd)/output

# Default to recursive test if GINKGO_PKG not set
GINKGO_PKG ?= -r
GINKGO_VERBOSE ?= false
GINKGO_FLAKE_ATTEMPTS ?= 3
GINKGO_FLAGS := $(if $(filter 1,$(GINKGO_VERBOSE)),-vv) $(GINKGO_PKG) --mod=mod --randomize-all --randomize-suites --cover --coverprofile=coverage.out --coverpkg=./... --output-dir=$(COVERAGE_DIR) --flake-attempts=$(GINKGO_FLAKE_ATTEMPTS) -p


GOPATH ?= $(HOME)/go
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GINKGO = $(GOBIN)/ginkgo

.PHONY: help clean controller-gen build build-ci linux-amd64 linux-arm64 darwin-arm64 test test-ci fmt vet vet-ci lint ginkgo envtest deps container-env

help: ## Display this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make help                    - Show this help message"
	@echo "  make deps                    - Install build dependencies (Linux: Fedora/RHEL/Debian/Ubuntu)"
	@echo "  make build                   - Build the korn binary locally (native Go build)"
	@echo "  make build-ci                - Build the korn binary using containers"
	@echo "  make linux-amd64             - Build for Linux AMD64 (using containers)"
	@echo "  make linux-arm64             - Build for Linux ARM64 (using containers)"
	@echo "  make darwin-arm64            - Build for Darwin ARM64 (using containers)"
	@echo "  make VERSION=v1.0.0 build    - Build locally with specific version"
	@echo "  make VERSION=v1.0.0 linux-amd64 - Build Linux AMD64 with specific version"
	@echo "  make GOOS=linux GOARCH=amd64 build-ci - Build for specific platform using containers"
	@echo "  make test                    - Run tests locally (no container)"
	@echo "  make test-ci                 - Run tests in container for CI"
	@echo "  make vet-ci                  - Run go vet in container for CI"
	@echo "  make GINKGO_FLAKE_ATTEMPTS=3 test - Run tests with 3 retry attempts for flaky tests"
	@echo "  make container-env           - Build container environment for builds/tests"
	@echo "  make lint                    - Run linting checks"
	@echo "  make clean                   - Remove build artifacts"

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT)
	@echo "Build artifacts removed from $(OUTPUT)"

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

$(OUTPUT):
	mkdir -p $(OUTPUT)

container-env: ## Build container environment for builds and tests
	@echo "Building container environment..."
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "Error: Podman is not installed or not in PATH"; \
		exit 1; \
	fi
	@CONTAINER_PLATFORM=$${CONTAINER_PLATFORM:-linux/$$(go env GOARCH)}; \
	CONTAINER_GOARCH=$${CONTAINER_GOARCH:-$$(go env GOARCH)}; \
	CONTAINER_TAG=$${CONTAINER_TAG:-korn-build-env}; \
	echo "Building container: $$CONTAINER_TAG (platform: $$CONTAINER_PLATFORM, goarch: $$CONTAINER_GOARCH)"; \
	podman build --platform="$$CONTAINER_PLATFORM" \
		--build-arg GOARCH="$$CONTAINER_GOARCH" \
		-f build/Containerfile.build \
		-t "$$CONTAINER_TAG" .

$(GINKGO):
	go install github.com/onsi/ginkgo/v2/ginkgo

ginkgo: $(GINKGO) ## Download ginkgo locally if necessary
	$(GINKGO) version

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.63.4
golangci-lint: ## Install golangci-lint locally if necessary
	@[ -f $(GOLANGCI_LINT) ] || { \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) $(GOLANGCI_LINT_VERSION) ;\
	}


.PHONY: lint
lint: fmt vet golangci-lint ## Run golangci-lint linter & yamllint
	$(GOLANGCI_LINT) run

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code locally
	go vet ./...

.PHONY: vet-ci
vet-ci: ## Run go vet against code in container for CI
	@echo "Running go vet in container..."
	@CONTAINER_TAG="korn-vet-env" make container-env
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "Error: Podman is not installed or not in PATH"; \
		exit 1; \
	fi
	@podman run --rm \
		--platform=linux/$$(go env GOARCH) \
		-v "$$(pwd)":/src:rw,Z \
		-w /src \
		korn-vet-env \
		go vet ./...

.PHONY: test
test: fmt vet deps $(OUTPUT) envtest ginkgo  ## Run tests locally (no container)
	@echo "Running tests locally..."
	@ENVTEST_VERSION="$(ENVTEST_VERSION)" \
		ENVTEST_K8S_VERSION="$(ENVTEST_K8S_VERSION)" \
		GINKGO_FLAGS="$(GINKGO_FLAGS)" \
		GINKGO_FLAKE_ATTEMPTS="$(GINKGO_FLAKE_ATTEMPTS)" \
		COVERAGE_DIR="$(COVERAGE_DIR)" \
		ENVTEST_BIN="$(ENVTEST)" \
		ENVTEST_BIN_DIR="$(LOCALBIN)" \
		LOCALBIN="$(LOCALBIN)" \
		OUTPUT="$(OUTPUT)" \
		USER_ID="$(shell id -u)" \
		GROUP_ID="$(shell id -g)" \
		./hack/run-test.sh

.PHONY: test-ci
test-ci: fmt vet-ci envtest ginkgo ## Run tests in container for CI (container for Linux hosts, native for others)
	@ENVTEST_VERSION="$(ENVTEST_VERSION)" \
		ENVTEST_K8S_VERSION="$(ENVTEST_K8S_VERSION)" \
		GINKGO_FLAGS="$(GINKGO_FLAGS)" \
		GINKGO_FLAKE_ATTEMPTS="$(GINKGO_FLAKE_ATTEMPTS)" \
		COVERAGE_DIR="$(COVERAGE_DIR)" \
		ENVTEST_BIN="$(ENVTEST)" \
		ENVTEST_BIN_DIR="$(LOCALBIN)" \
		LOCALBIN="$(LOCALBIN)" \
		OUTPUT="$(OUTPUT)" \
		GOARCH="$(shell go env GOARCH)" \
		./hack/test-ci.sh



.PHONY: deps
deps: ## Install build dependencies on Linux systems (Fedora/RHEL/Debian/Ubuntu)
	@./hack/dependencies.sh

.PHONY: build-ci
build-ci: $(OUTPUT) ## Build the korn binary using containers (Linux, Darwin via linux/arm64, native for others)
	@chmod +x hack/build-ci.sh
	@GOOS="$(GOOS)" \
		GOARCH="$(GOARCH)" \
		VERSION="$(VERSION)" \
		PROJECT_NAME="$(PROJECT_NAME)" \
		BINARY_NAME="$(BINARY_NAME)" \
		OUTPUT="$(OUTPUT)" \
		./hack/build-ci.sh

.PHONY: linux-amd64
linux-amd64: $(OUTPUT) ## Build for Linux AMD64
	@GOOS=linux GOARCH=amd64 $(MAKE) build-ci

.PHONY: linux-arm64
linux-arm64: $(OUTPUT) ## Build for Linux ARM64
	@GOOS=linux GOARCH=arm64 $(MAKE) build-ci

.PHONY: darwin-arm64
darwin-arm64: $(OUTPUT) ## Build for Darwin ARM64 (Apple Silicon)
	@GOOS=darwin GOARCH=arm64 $(MAKE) build-ci

.PHONY: build
build: fmt vet deps $(OUTPUT) ## Build the korn binary locally (native Go build)
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH) locally..."
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=mod -ldflags="-s -w -X main.version=$(VERSION)" -o $(OUTPUT)/$(BINARY_NAME) main.go
	@echo "Binary built: $(OUTPUT)/$(BINARY_NAME)"
