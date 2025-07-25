

GO_VERSION := 1.23.6
CONTROLLER_TOOLS_VERSION ?= v0.17.2

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.32.0
ENVTEST_VERSION ?= release-0.20
ENVTEST ?= $(LOCALBIN)/setup-envtest
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

GOTOOLCHAIN := go$(GO_VERSION)

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

.PHONY: help clean controller-gen build test ginkgo envtest

help: ## Display this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make help     - Show this help message"
	@echo "  make build    - Build the korn binary"
	@echo "  make test     - Run all tests with coverage"
	@echo "  make lint     - Run linting checks"
	@echo "  make clean    - Remove build artifacts"

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
test: fmt vet envtest ginkgo  ## Run tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" ENVTEST_K8S_VERSION="$(ENVTEST_K8S_VERSION)" ginkgo $(GINKGO_FLAGS)

.PHONY: build
build: fmt vet $(OUTPUT) ## Build the korn binary
	go build -mod=mod -o $(OUTPUT)/korn main.go