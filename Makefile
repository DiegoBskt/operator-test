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
podman-build: ## Build container image for amd64 (OpenShift default).
	podman build --platform linux/amd64 -t ${IMG} .

.PHONY: podman-build-local
podman-build-local: ## Build container image for local architecture.
	podman build -t ${IMG} .

.PHONY: podman-push
podman-push: ## Push container image with the manager.
	podman push ${IMG}

.PHONY: podman-buildx
podman-buildx: ## Build and push multi-arch manifest (amd64 + arm64).
	-podman rmi ${IMG} 2>/dev/null || true
	-podman manifest rm ${IMG} 2>/dev/null || true
	podman manifest create ${IMG}
	podman build --platform linux/amd64 --manifest ${IMG} .
	podman build --platform linux/arm64 --manifest ${IMG} .
	podman manifest push --all ${IMG}

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
	cp config/crd/bases/*.yaml bundle/manifests/

BUNDLE_IMG ?= ghcr.io/diegobskt/cluster-assessment-operator-bundle:v1.0.0

.PHONY: bundle-build
bundle-build: ## Build the bundle image for amd64.
	podman build --platform linux/amd64 -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-build-local
bundle-build-local: ## Build the bundle image for local architecture.
	podman build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	podman push $(BUNDLE_IMG)

.PHONY: bundle-buildx
bundle-buildx: ## Build and push multi-arch bundle (amd64 + arm64).
	-podman rmi $(BUNDLE_IMG) 2>/dev/null || true
	-podman manifest rm $(BUNDLE_IMG) 2>/dev/null || true
	podman manifest create $(BUNDLE_IMG)
	podman build --platform linux/amd64 -f bundle.Dockerfile --manifest $(BUNDLE_IMG) .
	podman build --platform linux/arm64 -f bundle.Dockerfile --manifest $(BUNDLE_IMG) .
	podman manifest push --all $(BUNDLE_IMG)

.PHONY: scorecard
scorecard: ## Run operator-sdk scorecard tests.
	operator-sdk scorecard bundle --selector=suite=basic
	operator-sdk scorecard bundle --selector=suite=olm

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report.
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: preflight
preflight: ## Run Red Hat Preflight certification checks (containerized).
	podman run --rm \
		-v $(HOME)/.docker/config.json:/root/.docker/config.json:ro \
		quay.io/opdev/preflight:stable check container $(IMG)

##@ Dependencies

.PHONY: deps
deps: ## Download dependencies.
	go mod download
	go mod tidy

.PHONY: verify-deps
verify-deps: ## Verify dependencies.
	go mod verify

