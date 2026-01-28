# Main Makefile for nvsentinel project
# Coordinates between multiple sub-Makefiles organized by functionality

# Main Makefile for NVIDIA Device API

# ==============================================================================
# Configuration
# ==============================================================================

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Modules to manage
MODULES := api client-go code-generator

# Go build settings
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TREE_STATE ?= $(shell if git diff --quiet 2>/dev/null; then echo "clean"; else echo "dirty"; fi)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Version package path for ldflags
VERSION_PKG = github.com/nvidia/nvsentinel/pkg/version

# Container settings
CONTAINER_RUNTIME ?= docker
IMAGE_REGISTRY ?= ghcr.io/nvidia/nvsentinel
DOCKERFILE := deployments/container/Dockerfile

# Linker flags
LDFLAGS = -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
	-X $(VERSION_PKG).GitTreeState=$(GIT_TREE_STATE) \
	-X $(VERSION_PKG).BuildDate=$(BUILD_DATE)

# ==============================================================================
# Targets
# ==============================================================================

# Variables
GOPATH ?= $(shell go env GOPATH)
GO_CACHE_DIR ?= $(shell go env GOCACHE)

# Load tool versions from .versions.yaml (single source of truth)
# Requires yq to be installed: brew install yq (macOS) or see https://github.com/mikefarah/yq
YQ := $(shell command -v yq 2> /dev/null)

ifndef YQ
$(error yq is required but not found. Install it with: brew install yq (macOS) or see https://github.com/mikefarah/yq)
endif

# Load versions from .versions.yaml
ADDLICENSE_VERSION := $(shell $(YQ) '.linting.addlicense' .versions.yaml)
BLACK_VERSION := $(shell $(YQ) '.linting.black' .versions.yaml)
DOCKER_BUILDX_VERSION := $(shell $(YQ) '.container_tools.docker_buildx' .versions.yaml)
GO_VERSION := $(shell $(YQ) '.languages.go' .versions.yaml)
GOCOVER_COBERTURA_VERSION := $(shell $(YQ) '.go_tools.gocover_cobertura' .versions.yaml)
GOLANGCI_LINT_VERSION := $(shell $(YQ) '.go_tools.golangci_lint' .versions.yaml)
GOTESTSUM_VERSION := $(shell $(YQ) '.go_tools.gotestsum' .versions.yaml)
GRPCIO_TOOLS_VERSION := $(shell $(YQ) '.protobuf.grpcio_tools' .versions.yaml)
KO_VERSION := $(shell $(YQ) '.container_tools.ko' .versions.yaml)
POETRY_VERSION := $(shell $(YQ) '.build_tools.poetry' .versions.yaml)
PROTOBUF_VERSION := $(shell $(YQ) '.protobuf.protobuf' .versions.yaml)
PROTOC_GEN_GO_GRPC_VERSION := $(shell $(YQ) '.protobuf.protoc_gen_go_grpc' .versions.yaml)
PROTOC_GEN_GO_VERSION := $(shell $(YQ) '.protobuf.protoc_gen_go' .versions.yaml)
PYTHON_VERSION := $(shell $(YQ) '.languages.python' .versions.yaml)
SHELLCHECK_VERSION := $(shell $(YQ) '.linting.shellcheck' .versions.yaml)

# Go modules with specific patterns from CI
GO_MODULES := \
	api \
	health-monitors/syslog-health-monitor \
	health-monitors/csp-health-monitor \
	health-monitors/kubernetes-object-monitor \
	platform-connectors \
	health-events-analyzer \
	fault-quarantine \
	labeler \
	node-drainer \
	fault-remediation \
	janitor \
	metadata-collector \
	event-exporter \
	store-client \
	commons


# Python modules
PYTHON_MODULES := \
	health-monitors/gpu-health-monitor

# Container-only modules
CONTAINER_MODULES := \
	log-collector \
    gpu-reset

# Special modules requiring private repo access
PRIVATE_MODULES := \
	health-monitors/csp-health-monitor \
	health-events-analyzer \
	fault-quarantine \
	labeler \
	node-drainer \
	fault-remediation \
	janitor

# Modules requiring kubebuilder for tests
KUBEBUILDER_MODULES := \
	node-drainer \
	fault-remediation

# Default target
.PHONY: all
all: code-gen test build ## Run code generation, test, and build for all modules.

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: code-gen
code-gen: ## Run code generation in all modules.
	$(MAKE) -C api code-gen
	@if [ -d client-go ] && [ -f client-go/Makefile ]; then $(MAKE) -C client-go code-gen; fi

.PHONY: verify-codegen
verify-codegen: code-gen ## Verify generated code is up-to-date.
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: Generated code is out of date. Run 'make code-gen'."; \
		git status --porcelain; \
		git --no-pager diff; \
		exit 1; \
	fi

##@ Build

.PHONY: build
build: build-modules build-server ## Build all modules and server.

.PHONY: build-modules
build-modules: ## Build all modules.
	@for mod in $(MODULES); do \
		if [ -f $$mod/Makefile ]; then \
			$(MAKE) -C $$mod build; \
		fi \
	done

.PHONY: build-server
build-server: ## Build the Device API Server
	@echo "Building device-api-server..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o bin/device-api-server \
		./cmd/device-api-server
	@echo "Built bin/device-api-server"

.PHONY: build-nvml-provider
build-nvml-provider: ## Build the NVML Provider sidecar (requires CGO)
	@echo "Building nvml-provider..."
	@mkdir -p bin
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-tags=nvml \
		-ldflags "$(LDFLAGS)" \
		-o bin/nvml-provider \
		./cmd/nvml-provider
	@echo "Built bin/nvml-provider"

##@ Testing

.PHONY: test
test: test-modules test-server ## Run tests in all modules.

.PHONY: test-modules
test-modules: ## Run tests in all modules.
	@for mod in $(MODULES); do \
		if [ -f $$mod/Makefile ]; then \
			$(MAKE) -C $$mod test; \
		fi \
	done

.PHONY: test-server
test-server: ## Run server tests only
	go test -race -v ./pkg/...

##@ Linting

.PHONY: lint
lint: ## Run linting on all modules.
	@for mod in $(MODULES); do \
		if [ -f $$mod/Makefile ]; then \
			$(MAKE) -C $$mod lint; \
		fi \
	done
	go vet ./...

##@ Container Images

.PHONY: docker-build
docker-build: docker-build-server docker-build-nvml-provider ## Build all container images

.PHONY: docker-build-server
docker-build-server: ## Build device-api-server container image
	$(CONTAINER_RUNTIME) build \
		--target device-api-server \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_TREE_STATE=$(GIT_TREE_STATE) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_REGISTRY)/device-api-server:$(VERSION) \
		-f $(DOCKERFILE) .

.PHONY: docker-build-nvml-provider
docker-build-nvml-provider: ## Build nvml-provider container image
	$(CONTAINER_RUNTIME) build \
		--target nvml-provider \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_TREE_STATE=$(GIT_TREE_STATE) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_REGISTRY)/nvml-provider:$(VERSION) \
		-f $(DOCKERFILE) .

.PHONY: docker-push
docker-push: ## Push all container images
	$(CONTAINER_RUNTIME) push $(IMAGE_REGISTRY)/device-api-server:$(VERSION)
	$(CONTAINER_RUNTIME) push $(IMAGE_REGISTRY)/nvml-provider:$(VERSION)

##@ Helm

.PHONY: helm-lint
helm-lint: ## Lint Helm chart
	helm lint deployments/helm/nvsentinel

.PHONY: helm-template
helm-template: ## Render Helm chart templates
	helm template nvsentinel deployments/helm/nvsentinel

.PHONY: helm-package
helm-package: ## Package Helm chart
	@mkdir -p dist/
	helm package deployments/helm/nvsentinel -d dist/

##@ Cleanup

.PHONY: clean
clean: ## Clean generated artifacts in all modules.
	@for mod in $(MODULES); do \
		if [ -f $$mod/Makefile ]; then \
			$(MAKE) -C $$mod clean; \
		fi \
	done
	rm -rf bin/

.PHONY: tidy
tidy: ## Run go mod tidy on all modules.
	@for mod in $(MODULES); do \
		echo "Tidying $$mod..."; \
		(cd $$mod && go mod tidy); \
	done
	go mod tidy
