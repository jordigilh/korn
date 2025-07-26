

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
LOCALBIN ?= $(shell pwd)/bin
OUTPUT ?= $(shell pwd)/output

# Default to recursive test if GINKGO_PKG not set
GINKGO_PKG ?= -r
GINKGO_VERBOSE ?= false
GINKGO_FLAGS := $(if $(filter 1,$(GINKGO_VERBOSE)),-v) $(GINKGO_PKG) --mod=mod --randomize-all --randomize-suites --cover --coverprofile=coverage.out --coverpkg=./... --output-dir=$(COVERAGE_DIR)


GOPATH ?= $(HOME)/go
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOIMPORTS = $(GOBIN)/goimports
GINKGO = $(GOBIN)/ginkgo

.PHONY: help clean controller-gen build release linux-amd64 linux-arm64 darwin-arm64 test test-native fmt vet lint ginkgo envtest deps

help: ## Display this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make help                    - Show this help message"
	@echo "  make deps                    - Install build dependencies (Linux: Fedora/RHEL/Debian/Ubuntu)"
	@echo "  make build                   - Build the korn binary locally (native Go build)"
	@echo "  make release                 - Build the korn binary using containers"
	@echo "  make linux-amd64             - Build for Linux AMD64 (using containers)"
	@echo "  make linux-arm64             - Build for Linux ARM64 (using containers)"
	@echo "  make darwin-arm64            - Build for Darwin ARM64 (using containers)"
	@echo "  make VERSION=v1.0.0 build    - Build locally with specific version"
	@echo "  make VERSION=v1.0.0 linux-amd64 - Build Linux AMD64 with specific version"
	@echo "  make GOOS=linux GOARCH=amd64 release - Build for specific platform using containers"
	@echo "  make test                    - Run tests (container for Linux, native for Darwin)"
	@echo "  make test-native             - Run tests natively (no container)"
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
vet: ## Run go vet against code
	go vet ./...

.PHONY: test
test: fmt vet envtest ginkgo  ## Run tests (container for Linux, native for Darwin)
	@echo "Running tests..."
	@if [ "$(shell uname -s)" = "Linux" ]; then \
		echo "Using container test environment for Linux..."; \
		if ! command -v podman >/dev/null 2>&1; then \
			echo "Error: Podman is not installed or not in PATH"; \
			exit 1; \
		fi; \
		echo "Creating test environment image..."; \
		podman build --platform=linux/$(shell go env GOARCH) \
			-f build/Containerfile.build \
			-t korn-test-env .; \
		echo "Running tests in container..."; \
		podman run --rm \
			--platform=linux/$(shell go env GOARCH) \
			-v $(shell pwd):/src:rw,Z \
			-w /src \
			-e ENVTEST_K8S_VERSION="$(ENVTEST_K8S_VERSION)" \
			korn-test-env \
			sh -c "go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION) && go install github.com/onsi/ginkgo/v2/ginkgo && mkdir -p coverage bin && KUBEBUILDER_ASSETS=\"\$$(setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir . -p path)\" ENVTEST_K8S_VERSION=\"$(ENVTEST_K8S_VERSION)\" ginkgo $(GINKGO_FLAGS) && chown -R $(shell id -u):$(shell id -g) /src/coverage/ /src/bin/ || true"; \
	else \
		echo "Using native test environment for $(shell uname -s)..."; \
		$(MAKE) deps; \
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" ENVTEST_K8S_VERSION="$(ENVTEST_K8S_VERSION)" $(GOBIN)/ginkgo $(GINKGO_FLAGS); \
	fi

.PHONY: test-native
test-native: fmt vet envtest ginkgo deps  ## Run tests natively (no container)
	@echo "Running tests natively..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" ENVTEST_K8S_VERSION="$(ENVTEST_K8S_VERSION)" $(GOBIN)/ginkgo $(GINKGO_FLAGS)


.PHONY: deps
deps: ## Install build dependencies on Linux systems (Fedora/RHEL/Debian/Ubuntu)
	@echo "Checking build dependencies..."
	@if [ "$(shell uname -s)" = "Linux" ]; then \
		if command -v dnf >/dev/null 2>&1; then \
			echo "Fedora/RHEL detected - installing build dependencies..."; \
			if [ -f rpm/podman.spec ]; then \
				sudo dnf -y builddep rpm/podman.spec || echo "Warning: Some dependencies may have failed to install"; \
			else \
				echo "Warning: rpm/podman.spec not found"; \
			fi; \
		elif command -v yum >/dev/null 2>&1; then \
			echo "CentOS/RHEL (yum) detected - installing build dependencies..."; \
			if [ -f rpm/podman.spec ]; then \
				sudo yum-builddep -y rpm/podman.spec || echo "Warning: Some dependencies may have failed to install"; \
			else \
				echo "Warning: rpm/podman.spec not found"; \
			fi; \
		elif command -v apt-get >/dev/null 2>&1; then \
			echo "Debian/Ubuntu detected - installing build dependencies..."; \
			if [ -f build/debian-packages.txt ]; then \
				packages=$$(cat build/debian-packages.txt | tr '\n' ' '); \
				sudo apt-get update && sudo apt-get install -y $$packages \
					|| echo "Warning: Some dependencies may have failed to install"; \
			else \
				echo "Warning: build/debian-packages.txt not found"; \
			fi; \
		else \
			echo "Unsupported Linux distribution - skipping dependency installation"; \
		fi; \
	else \
		echo "Non-Linux system detected - skipping dependency installation"; \
	fi

.PHONY: release
release: $(OUTPUT) ## Build the korn binary using containers (Linux, Darwin via linux/arm64, native for others)
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@if [ "$(GOOS)" = "darwin" ] && [ "$(GOARCH)" != "arm64" ]; then \
		echo "Error: Darwin builds are only supported for arm64 architecture"; \
		echo "Use 'make darwin-arm64' or set GOARCH=arm64"; \
		exit 1; \
	fi
	@if [ "$(GOOS)" = "linux" ] || [ "$(GOOS)" = "darwin" ]; then \
		echo "Using container build for $(GOOS)/$(GOARCH) target..."; \
		if ! command -v podman >/dev/null 2>&1; then \
			echo "Error: Podman is not installed or not in PATH"; \
			exit 1; \
		fi; \
		if [ "$(GOOS)" = "linux" ]; then \
			platform="linux/$(GOARCH)"; \
			build_arch="$(GOARCH)"; \
		else \
			platform="linux/arm64"; \
			build_arch="arm64"; \
		fi; \
		echo "Creating build environment image..."; \
		podman build --platform=$$platform \
			--build-arg GOARCH=$$build_arch \
			-f build/Containerfile.build \
			-t korn-build-env .; \
		echo "Building binary in container..."; \
		podman run --rm \
			--platform=$$platform \
			-v $(shell pwd):/src:rw,Z \
			-w /src \
			-e VERSION=$(VERSION) \
			-e GOOS=$(GOOS) \
			-e GOARCH=$(GOARCH) \
			-e PROJECT_NAME=$(PROJECT_NAME) \
			korn-build-env \
			sh -c "mkdir -p output && GOOS=\$$GOOS GOARCH=\$$GOARCH go build -mod=mod -ldflags=\"-s -w -X main.version=\$$VERSION\" -o output/\$${PROJECT_NAME}_\$${VERSION}_\$${GOOS}_\$${GOARCH} main.go"; \
	else \
		echo "Using native build for $(GOOS) target..."; \
		echo "Note: This target requires native Go environment for best compatibility"; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=mod -ldflags="-s -w -X main.version=$(VERSION)" -o $(OUTPUT)/$(BINARY_NAME) main.go; \
	fi
	@echo "Binary built: $(OUTPUT)/$(BINARY_NAME)"

.PHONY: linux-amd64
linux-amd64: $(OUTPUT) ## Build for Linux AMD64
	@GOOS=linux GOARCH=amd64 $(MAKE) release

.PHONY: linux-arm64
linux-arm64: $(OUTPUT) ## Build for Linux ARM64
	@GOOS=linux GOARCH=arm64 $(MAKE) release

.PHONY: darwin-arm64
darwin-arm64: $(OUTPUT) ## Build for Darwin ARM64 (Apple Silicon)
	@GOOS=darwin GOARCH=arm64 $(MAKE) release

.PHONY: build
build: fmt vet deps $(OUTPUT) ## Build the korn binary locally (native Go build)
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH) locally..."
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=mod -ldflags="-s -w -X main.version=$(VERSION)" -o $(OUTPUT)/$(BINARY_NAME) main.go
	@echo "Binary built: $(OUTPUT)/$(BINARY_NAME)"
