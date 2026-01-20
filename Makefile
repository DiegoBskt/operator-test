# VERSION - Single source of truth, read from VERSION file
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "0.0.0")

# Image URLs - all derived from VERSION
REGISTRY ?= ghcr.io/diegobskt
OPERATOR_NAME ?= cluster-assessment-operator
IMG ?= $(REGISTRY)/$(OPERATOR_NAME):v$(VERSION)
CONSOLE_IMG ?= $(REGISTRY)/$(OPERATOR_NAME)-console:v$(VERSION)

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

.PHONY: version
version: ## Show current version from VERSION file.
	@echo "$(VERSION)"

##@ Version Management

.PHONY: update-manifests
update-manifests: ## Update all manifests with current VERSION.
	@echo "Updating manifests to v$(VERSION)..."
	@sed -i '' 's|image: $(REGISTRY)/$(OPERATOR_NAME):v[0-9.]*|image: $(REGISTRY)/$(OPERATOR_NAME):v$(VERSION)|g' config/manager/manager.yaml
	@sed -i '' 's|image: $(REGISTRY)/$(OPERATOR_NAME)-console:v[0-9.]*|image: $(REGISTRY)/$(OPERATOR_NAME)-console:v$(VERSION)|g' config/console-plugin/deployment.yaml
	@sed -i '' 's|containerImage: $(REGISTRY)/$(OPERATOR_NAME):v[0-9.]*|containerImage: $(REGISTRY)/$(OPERATOR_NAME):v$(VERSION)|g' bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
	@echo "Manifests updated to v$(VERSION)"

.PHONY: update-catalogs
update-catalogs: ## Update FBC catalog templates with current VERSION.
	@./scripts/update-catalogs.sh $(VERSION)

.PHONY: release-prep
release-prep: update-manifests update-catalogs catalogs catalog-validate ## Prepare for release: update all manifests and catalogs.
	@echo ""
	@echo "Release preparation complete for v$(VERSION)"
	@echo "Next steps:"
	@echo "  1. Review changes: git diff"
	@echo "  2. Commit: git commit -am 'chore: prepare release v$(VERSION)'"
	@echo "  3. Tag: git tag v$(VERSION)"
	@echo "  4. Push: git push origin main v$(VERSION)"



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

.PHONY: deploy-console-plugin
deploy-console-plugin: ## Deploy the OpenShift Console plugin.
	kubectl apply -f config/console-plugin/

.PHONY: deploy-all
deploy-all: deploy deploy-console-plugin ## Deploy operator and console plugin.

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/manager/ || true
	kubectl delete -f config/rbac/ || true
	kubectl delete -f config/crd/bases/ || true

.PHONY: undeploy-console-plugin
undeploy-console-plugin: ## Undeploy the OpenShift Console plugin.
	kubectl delete -f config/console-plugin/ || true

.PHONY: undeploy-all
undeploy-all: undeploy undeploy-console-plugin ## Undeploy operator and console plugin.

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

BUNDLE_IMG ?= $(REGISTRY)/$(OPERATOR_NAME)-bundle:v$(VERSION)

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

CATALOG_IMG ?= $(REGISTRY)/$(OPERATOR_NAME)-catalog

# OCP versions to build catalogs for (Red Hat OperatorHub requirement)
OCP_VERSIONS ?= v4.12 v4.13 v4.14 v4.15 v4.16 v4.17 v4.18 v4.19 v4.20

##@ File Based Catalog (FBC) - Red Hat OperatorHub Compatible

.PHONY: catalogs
catalogs: ## Generate FBC catalogs from templates for all OCP versions.
	@for version in $(OCP_VERSIONS); do \
		echo "Generating catalog for $$version..."; \
		mkdir -p catalogs/$$version/$(OPERATOR_NAME); \
		opm alpha render-template basic catalog-templates/$$version.yaml -o yaml > catalogs/$$version/$(OPERATOR_NAME)/catalog.yaml; \
	done
	@echo "Catalogs generated for: $(OCP_VERSIONS)"

.PHONY: catalog-validate
catalog-validate: ## Validate all FBC catalogs.
	@for version in $(OCP_VERSIONS); do \
		echo "Validating catalog for $$version..."; \
		opm validate catalogs/$$version; \
	done
	@echo "All catalogs validated successfully!"

.PHONY: catalog-build
catalog-build: ## Build FBC catalog images for all OCP versions (amd64).
	@for version in $(OCP_VERSIONS); do \
		echo "Building catalog image for $$version..."; \
		podman build --platform linux/amd64 \
			--build-arg OCP_VERSION=$$version \
			--build-arg OPERATOR_NAME=$(OPERATOR_NAME) \
			-f catalog.Dockerfile \
			-t $(CATALOG_IMG):$$version .; \
	done
	@echo "Catalog images built: $(OCP_VERSIONS)"

.PHONY: catalog-build-local
catalog-build-local: ## Build FBC catalog images for local architecture.
	@for version in $(OCP_VERSIONS); do \
		echo "Building catalog image for $$version (local arch)..."; \
		podman build \
			--build-arg OCP_VERSION=$$version \
			--build-arg OPERATOR_NAME=$(OPERATOR_NAME) \
			-f catalog.Dockerfile \
			-t $(CATALOG_IMG):$$version .; \
	done

.PHONY: catalog-push
catalog-push: ## Push all catalog images to registry.
	@for version in $(OCP_VERSIONS); do \
		echo "Pushing catalog image for $$version..."; \
		podman push $(CATALOG_IMG):$$version; \
	done
	@echo "All catalog images pushed!"

.PHONY: catalog-build-single
catalog-build-single: ## Build catalog for single OCP version. Usage: make catalog-build-single OCP_VERSION=v4.14
ifndef OCP_VERSION
	$(error OCP_VERSION is required. Usage: make catalog-build-single OCP_VERSION=v4.14)
endif
	opm validate catalogs/$(OCP_VERSION)
	podman build --platform linux/amd64 \
		--build-arg OCP_VERSION=$(OCP_VERSION) \
		--build-arg OPERATOR_NAME=$(OPERATOR_NAME) \
		-f catalog.Dockerfile \
		-t $(CATALOG_IMG):$(OCP_VERSION) .

.PHONY: deploy-olm
deploy-olm: ## Deploy the operator via OLM using operator-sdk.
	operator-sdk run bundle $(BUNDLE_IMG)

.PHONY: cleanup-olm
cleanup-olm: ## Remove the operator installed via OLM.
	operator-sdk cleanup cluster-assessment-operator

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

