# VERSION - Single source of truth, read from VERSION file
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "0.0.0")

# Image URLs - all derived from VERSION
REGISTRY ?= ghcr.io/diegobskt
OPERATOR_NAME ?= cluster-assessment-operator
IMG ?= $(REGISTRY)/$(OPERATOR_NAME):v$(VERSION)
CONSOLE_IMG ?= $(REGISTRY)/$(OPERATOR_NAME)-console:v$(VERSION)
BUNDLE_IMG ?= $(REGISTRY)/$(OPERATOR_NAME)-bundle:v$(VERSION)
CATALOG_IMG ?= $(REGISTRY)/$(OPERATOR_NAME)-catalog

# OCP versions to build catalogs for
OCP_VERSIONS ?= v4.12 v4.13 v4.14 v4.15 v4.16 v4.17 v4.18 v4.19 v4.20

# Go configuration
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: version
version: ## Show current version.
	@echo "$(VERSION)"

##@ Development

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -ldflags "-X github.com/openshift-assessment/cluster-assessment-operator/pkg/version.Version=$(VERSION)" -o bin/manager main.go

.PHONY: run
run: fmt vet ## Run controller locally (for development).
	go run ./main.go

.PHONY: fmt
fmt: ## Run go fmt.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests with coverage.
	go test ./... -coverprofile cover.out

.PHONY: lint
lint: ## Run golangci-lint.
	golangci-lint run

.PHONY: deps
deps: ## Download and tidy dependencies.
	go mod download
	go mod tidy

##@ Deployment (Direct - for Development)

.PHONY: deploy
deploy: ## Deploy operator + console plugin directly (dev mode, no OLM).
	@echo "Deploying operator and console plugin..."
	kubectl apply -f config/crd/bases/
	kubectl apply -f config/rbac/
	kubectl apply -f config/manager/
	kubectl apply -f config/console-plugin/
	@echo ""
	@echo "Enabling console plugin..."
	oc patch consoles.operator.openshift.io cluster --type=merge --patch='{"spec":{"plugins":["cluster-assessment-plugin"]}}' 2>/dev/null || true
	@echo "Deployment complete! Run: oc get pods -n cluster-assessment-operator"

.PHONY: undeploy
undeploy: ## Undeploy operator + console plugin (dev mode).
	@echo "Undeploying operator and console plugin..."
	-oc patch consoles.operator.openshift.io cluster --type=json --patch='[{"op": "remove", "path": "/spec/plugins"}]' 2>/dev/null
	-kubectl delete -f config/console-plugin/
	-kubectl delete -f config/manager/
	-kubectl delete -f config/rbac/
	-kubectl delete -f config/crd/bases/
	@echo "Undeploy complete!"

##@ Deployment (OLM - for Production)

.PHONY: deploy-olm
deploy-olm: ## Deploy operator via OLM CatalogSource (production).
	@echo "Deploying via OLM..."
	@OCP_VERSION=$$(oc version -o json 2>/dev/null | jq -r '.openshiftVersion' | cut -d. -f1,2 | sed 's/^/v/' || echo "v4.20"); \
	echo "Detected OpenShift version: $$OCP_VERSION"; \
	oc apply -f - <<EOF
	---
	apiVersion: operators.coreos.com/v1alpha1
	kind: CatalogSource
	metadata:
	  name: cluster-assessment-catalog
	  namespace: openshift-marketplace
	spec:
	  sourceType: grpc
	  image: $(CATALOG_IMG):$$OCP_VERSION
	  displayName: Cluster Assessment Operator
	  publisher: Community
	  updateStrategy:
	    registryPoll:
	      interval: 10m
	---
	apiVersion: v1
	kind: Namespace
	metadata:
	  name: cluster-assessment-operator
	  labels:
	    openshift.io/cluster-monitoring: "true"
	---
	apiVersion: operators.coreos.com/v1
	kind: OperatorGroup
	metadata:
	  name: cluster-assessment-operator
	  namespace: cluster-assessment-operator
	spec: {}
	---
	apiVersion: operators.coreos.com/v1alpha1
	kind: Subscription
	metadata:
	  name: cluster-assessment-operator
	  namespace: cluster-assessment-operator
	spec:
	  channel: stable-v1
	  name: cluster-assessment-operator
	  source: cluster-assessment-catalog
	  sourceNamespace: openshift-marketplace
	  installPlanApproval: Automatic
	EOF
	@echo ""
	@echo "Enabling console plugin..."
	@sleep 5
	oc patch consoles.operator.openshift.io cluster --type=merge --patch='{"spec":{"plugins":["cluster-assessment-plugin"]}}'
	@echo ""
	@echo "OLM deployment initiated! Monitor with: oc get csv -n cluster-assessment-operator -w"

.PHONY: undeploy-olm
undeploy-olm: ## Undeploy operator via OLM (full cleanup).
	@echo "Cleaning up OLM deployment..."
	-oc delete clusterassessment --all --ignore-not-found
	-oc delete subscription.operators.coreos.com cluster-assessment-operator -n cluster-assessment-operator --ignore-not-found
	-oc delete csv -n cluster-assessment-operator -l operators.coreos.com/cluster-assessment-operator.cluster-assessment-operator --ignore-not-found
	-oc delete operatorgroup cluster-assessment-operator -n cluster-assessment-operator --ignore-not-found
	-oc patch consoles.operator.openshift.io cluster --type=json --patch='[{"op": "remove", "path": "/spec/plugins"}]' 2>/dev/null || true
	-oc delete catalogsource cluster-assessment-catalog -n openshift-marketplace --ignore-not-found
	-oc delete namespace cluster-assessment-operator --ignore-not-found --wait=false
	-oc delete crd clusterassessments.assessment.openshift.io --ignore-not-found
	@echo "OLM cleanup complete!"

##@ Release

.PHONY: release-prep
release-prep: update-manifests update-catalogs ## Prepare release: update VERSION → manifests → catalogs.
	@echo ""
	@echo "============================================================"
	@echo "Release preparation complete for v$(VERSION)"
	@echo "============================================================"
	@echo ""
	@echo "Verify bundle CSV version:"
	@grep "name: cluster-assessment-operator.v" bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml | head -1
	@echo ""
	@echo "Next steps:"
	@echo "  1. git diff"
	@echo "  2. git commit -am 'chore: release v$(VERSION)'"
	@echo "  3. git tag v$(VERSION)"
	@echo "  4. git push origin main v$(VERSION)"

.PHONY: update-manifests
update-manifests: ## Update all manifests with current VERSION.
	@echo "Updating manifests to v$(VERSION)..."
	@sed -i '' 's|image: $(REGISTRY)/$(OPERATOR_NAME):v[0-9.]*|image: $(REGISTRY)/$(OPERATOR_NAME):v$(VERSION)|g' config/manager/manager.yaml
	@sed -i '' 's|image: $(REGISTRY)/$(OPERATOR_NAME)-console:v[0-9.]*|image: $(REGISTRY)/$(OPERATOR_NAME)-console:v$(VERSION)|g' config/console-plugin/deployment.yaml
	@sed -i '' 's|containerImage: $(REGISTRY)/$(OPERATOR_NAME):v[0-9.]*|containerImage: $(REGISTRY)/$(OPERATOR_NAME):v$(VERSION)|g' bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
	@sed -i '' 's|name: cluster-assessment-operator.v[0-9.]*|name: cluster-assessment-operator.v$(VERSION)|g' bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
	@sed -i '' 's|^  version: [0-9.]*|  version: $(VERSION)|g' bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
	@sed -i '' 's|olm.skipRange: ">=1.0.0 <[0-9.]*"|olm.skipRange: ">=1.0.0 <$(VERSION)"|g' bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
	@sed -i '' 's|image: $(REGISTRY)/$(OPERATOR_NAME):v[0-9.]*|image: $(REGISTRY)/$(OPERATOR_NAME):v$(VERSION)|g' bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
	@sed -i '' 's|image: $(REGISTRY)/$(OPERATOR_NAME)-console:v[0-9.]*|image: $(REGISTRY)/$(OPERATOR_NAME)-console:v$(VERSION)|g' bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
	@echo "Manifests updated to v$(VERSION)"

.PHONY: update-catalogs
update-catalogs: ## Update catalog templates with current VERSION.
	@./scripts/update-catalogs.sh $(VERSION)

##@ Build Images

.PHONY: image-build
image-build: ## Build operator image (amd64).
	podman build --platform linux/amd64 --build-arg VERSION=$(VERSION) -t $(IMG) .

.PHONY: image-push
image-push: ## Push operator image.
	podman push $(IMG)

.PHONY: image-buildx
image-buildx: ## Build and push multi-arch operator image (amd64 + arm64).
	-podman manifest rm $(IMG) 2>/dev/null || true
	podman manifest create $(IMG)
	podman build --platform linux/amd64 --build-arg VERSION=$(VERSION) --manifest $(IMG) .
	podman build --platform linux/arm64 --build-arg VERSION=$(VERSION) --manifest $(IMG) .
	podman manifest push --all $(IMG)

.PHONY: bundle-build
bundle-build: ## Build bundle image (amd64).
	podman build --platform linux/amd64 -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push bundle image.
	podman push $(BUNDLE_IMG)

.PHONY: bundle-buildx
bundle-buildx: ## Build and push multi-arch bundle image.
	-podman manifest rm $(BUNDLE_IMG) 2>/dev/null || true
	podman manifest create $(BUNDLE_IMG)
	podman build --platform linux/amd64 -f bundle.Dockerfile --manifest $(BUNDLE_IMG) .
	podman build --platform linux/arm64 -f bundle.Dockerfile --manifest $(BUNDLE_IMG) .
	podman manifest push --all $(BUNDLE_IMG)

##@ Catalog Images

.PHONY: catalogs
catalogs: ## Generate FBC catalogs from templates.
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
	@echo "All catalogs valid!"

.PHONY: catalog-build
catalog-build: ## Build catalog images for all OCP versions.
	@for version in $(OCP_VERSIONS); do \
		echo "Building catalog for $$version..."; \
		podman build --platform linux/amd64 \
			--build-arg OCP_VERSION=$$version \
			--build-arg OPERATOR_NAME=$(OPERATOR_NAME) \
			-f catalog.Dockerfile \
			-t $(CATALOG_IMG):$$version .; \
	done

.PHONY: catalog-push
catalog-push: ## Push all catalog images.
	@for version in $(OCP_VERSIONS); do \
		echo "Pushing catalog for $$version..."; \
		podman push $(CATALOG_IMG):$$version; \
	done

##@ Testing

.PHONY: scorecard
scorecard: ## Run OLM scorecard tests.
	operator-sdk scorecard bundle --selector=suite=basic
	operator-sdk scorecard bundle --selector=suite=olm

.PHONY: preflight
preflight: ## Run Red Hat Preflight certification checks.
	podman run --rm \
		-v $(HOME)/.docker/config.json:/root/.docker/config.json:ro \
		quay.io/opdev/preflight:stable check container $(IMG)
