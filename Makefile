# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/diegobskt/cluster-assessment-operator:v1.0.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

.PHONY: lint
lint: ## Run golangci-lint against code.
	golangci-lint run

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: podman-build
podman-build: ## Build container image with the manager.
	podman build -t ${IMG} .

.PHONY: podman-push
podman-push: ## Push container image with the manager.
	podman push ${IMG}

.PHONY: podman-buildx
podman-buildx: ## Build and push container image for cross-platform support.
	podman build --platform linux/amd64,linux/arm64 --manifest ${IMG} .
	podman manifest push ${IMG}

##@ Deployment

.PHONY: install
install: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/crd/bases/

.PHONY: deploy
deploy: ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/crd/bases/
	kubectl apply -f config/rbac/
	kubectl apply -f config/manager/

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/manager/ || true
	kubectl delete -f config/rbac/ || true
	kubectl delete -f config/crd/bases/ || true

##@ Release

.PHONY: release-manifests
release-manifests: ## Generate release manifests.
	mkdir -p dist
	cat config/crd/bases/*.yaml > dist/install.yaml
	echo "---" >> dist/install.yaml
	cat config/rbac/*.yaml >> dist/install.yaml
	echo "---" >> dist/install.yaml
	cat config/manager/*.yaml >> dist/install.yaml

.PHONY: bundle
bundle: release-manifests ## Generate bundle for OLM.
	@echo "Bundle generation not yet implemented"

##@ Dependencies

.PHONY: deps
deps: ## Download dependencies.
	go mod download
	go mod tidy

.PHONY: verify-deps
verify-deps: ## Verify dependencies.
	go mod verify
