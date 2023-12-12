ENSURE_GARDENER_MOD         := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR           := $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack
EXTENSION_PREFIX            := gardener-extension
NAME                        := provider-ironcore
REGISTRY                    := ghcr.io
ADMISSION_NAME              := admission-ironcore
IMAGE_PREFIX                := $(REGISTRY)/extensions
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
LD_FLAGS                    := "-w $(shell bash $(GARDENER_HACK_DIR)/get-build-ld-flags.sh k8s.io/component-base $(REPO_ROOT)/VERSION $(EXTENSION_PREFIX))"
LEADER_ELECTION             := false
IGNORE_OPERATION_ANNOTATION := true

WEBHOOK_CONFIG_PORT	:= 8443
WEBHOOK_CONFIG_MODE	:= url
WEBHOOK_CONFIG_URL	:= host.docker.internal:$(WEBHOOK_CONFIG_PORT)
EXTENSION_NAMESPACE	:=

WEBHOOK_PARAM := --webhook-config-url=$(WEBHOOK_CONFIG_URL)
ifeq ($(WEBHOOK_CONFIG_MODE), service)
	WEBHOOK_PARAM := --webhook-config-namespace=$(EXTENSION_NAMESPACE)
endif

#########################################
# Tools                                 #
#########################################

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.28.0

TOOLS_DIR := $(CURDIR)/hack/tools
include $(GARDENER_HACK_DIR)/tools.mk

#########################################
# Rules for local development scenarios #
#########################################

.PHONY: start
start:
	@LEADER_ELECTION_NAMESPACE=garden GO111MODULE=on go run \
		-ldflags $(LD_FLAGS) \
		./cmd/$(EXTENSION_PREFIX)-$(NAME) \
		--config-file=./example/00-componentconfig.yaml \
		--ignore-operation-annotation=$(IGNORE_OPERATION_ANNOTATION) \
		--leader-election=$(LEADER_ELECTION) \
		--webhook-config-server-host=0.0.0.0 \
		--webhook-config-server-port=$(WEBHOOK_CONFIG_PORT) \
		--webhook-config-mode=$(WEBHOOK_CONFIG_MODE) \
		--gardener-version="v1.39.0" \
		$(WEBHOOK_PARAM)

.PHONY: start-admission
start-admission:
	@LEADER_ELECTION_NAMESPACE=garden GO111MODULE=on go run \
		-ldflags $(LD_FLAGS) \
		./cmd/$(EXTENSION_PREFIX)-$(ADMISSION_NAME) \
		--webhook-config-server-host=0.0.0.0 \
		--webhook-config-server-port=9443 \
		--webhook-config-cert-dir=./example/admission-ironcore-certs

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: install
install:
	@LD_FLAGS=$(LD_FLAGS) EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) \
	@bash $(GARDENER_HACK_DIR)/install.sh ./...

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-images
docker-images:
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(IMAGE_PREFIX)/$(NAME):$(VERSION)           -t $(IMAGE_PREFIX)/$(NAME):latest           -f Dockerfile -m 6g --target $(EXTENSION_PREFIX)-$(NAME)           .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(IMAGE_PREFIX)/$(ADMISSION_NAME):$(VERSION) -t $(IMAGE_PREFIX)/$(ADMISSION_NAME):latest -f Dockerfile -m 6g --target $(EXTENSION_PREFIX)-$(ADMISSION_NAME) .

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: tidy
tidy:
	@GO111MODULE=on go mod tidy
	@cp $(GARDENER_HACK_DIR)/cherry-pick-pull.sh $(HACK_DIR)/cherry-pick-pull.sh && chmod +xw $(HACK_DIR)/cherry-pick-pull.sh

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@bash $(GARDENER_HACK_DIR)/clean.sh ./cmd/... ./pkg/... ./test/...

.PHONY: check-generate
check-generate:
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check-generate.sh $(REPO_ROOT)

.PHONY: generate-mocks
generate-mocks: mockgen ## Generate code (mocks etc.).
	MOCKGEN=$(MOCKGEN) go generate ./pkg/controller/...

.PHONY: add-license
add-license: addlicense ## Add license headers to all go files.
	find . -name '*.go' -exec $(ADDLICENSE) -f hack/license-header.txt {} +

.PHONY: check-license
check-license: addlicense ## Check that every file has a license header present.
	find . -name '*.go' -exec $(ADDLICENSE) -check -c 'IronCore authors' {} +

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT)
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check-charts.sh ./charts

.PHONY: generate
generate: deepcopy-gen defaulter-gen conversion-gen $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM) $(MOCKGEN) $(YQ) $(VGOPATH)
	@VGOPATH=$(VGOPATH) \
		MOCKGEN=$(MOCKGEN) \
		DEEPCOPY_GEN=$(DEEPCOPY_GEN) \
		DEFAULTER_GEN=$(DEFAULTER_GEN) \
		CONVERSION_GEN=$(CONVERSION_GEN) \
		REPO_ROOT=$(REPO_ROOT) \
		GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) \
		bash $(GARDENER_HACK_DIR)/generate-sequential.sh ./charts/... ./cmd/... ./example/... ./pkg/...
	$(MAKE) format

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./pkg ./test

.PHONY: test
test: generate-mocks fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-cov
test-cov:
	@bash $(GARDENER_HACK_DIR)/test-cover.sh ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@bash $(GARDENER_HACK_DIR)/hack/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check-generate check format test-cov test-clean

.PHONY: docs
docs: gen-crd-api-reference-docs ## Run go generate to generate API reference documentation.
	$(GEN_CRD_API_REFERENCE_DOCS) -api-dir ./pkg/apis/ironcore/v1alpha1 -config ./hack/api-reference/api.json -template-dir ./hack/api-reference/template -out-file ./hack/api-reference/api.md
	$(GEN_CRD_API_REFERENCE_DOCS) -api-dir ./pkg/apis/config/v1alpha1 -config ./hack/api-reference/config.json -template-dir ./hack/api-reference/template -out-file ./hack/api-reference/config.md

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check-generate check format test-cov test-clean

##@ Tools

## Location to install dependencies to
LOCALBIN ?= $(TOOLS_DIR)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
#KUSTOMIZE ?= $(LOCALBIN)/kustomize
#CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
#ENVTEST ?= $(LOCALBIN)/setup-envtest
#GOIMPORTS ?= $(LOCALBIN)/goimports
#GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
#VGOPATH ?= $(LOCALBIN)/vgopath
DEEPCOPY_GEN ?= $(LOCALBIN)/deepcopy-gen
CONVERSION_GEN ?= $(LOCALBIN)/conversion-gen
DEFAULTER_GEN ?= $(LOCALBIN)/defaulter-gen
#ADDLICENSE ?= $(LOCALBIN)/addlicense
#GEN_CRD_API_REFERENCE_DOCS ?= $(LOCALBIN)/gen-crd-api-reference-docs
#MOCKGEN ?= $(LOCALBIN)/mockgen

## Tool Versions
#KUSTOMIZE_VERSION ?= v3.8.7
#CONTROLLER_TOOLS_VERSION ?= v0.13.0
#VGOPATH_VERSION ?= v0.1.3
CODE_GENERATOR_VERSION ?= v0.28.4
#ADDLICENSE_VERSION ?= v1.1.1
#GOIMPORTS_VERSION ?= v0.14.0
#GEN_CRD_API_REFERENCE_DOCS_VERSION ?= v0.3.0
#MOCKGEN_VERSION ?= v0.3.0
#GOLANGCI_LINT_VERSION ?= v1.55.2

#.PHONY: envtest
#envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
#$(ENVTEST): $(LOCALBIN)
#	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
#
#.PHONY: controller-gen
#controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
#$(CONTROLLER_GEN): $(LOCALBIN)
#	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
#	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
#
.PHONY: deepcopy-gen
deepcopy-gen: $(DEEPCOPY_GEN) ## Download deepcopy-gen locally if necessary.
$(DEEPCOPY_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/deepcopy-gen || GOBIN=$(LOCALBIN) go install k8s.io/code-generator/cmd/deepcopy-gen@$(CODE_GENERATOR_VERSION)

.PHONY: defaulter-gen
defaulter-gen: $(DEFAULTER_GEN) ## Download defaulter-gen locally if necessary.
$(DEFAULTER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/defaulter-gen || GOBIN=$(LOCALBIN) go install k8s.io/code-generator/cmd/defaulter-gen@$(CODE_GENERATOR_VERSION)

.PHONY: conversion-gen
conversion-gen: $(CONVERSION_GEN) ## Download conversion-gen locally if necessary.
$(CONVERSION_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/conversion-gen || GOBIN=$(LOCALBIN) go install k8s.io/code-generator/cmd/conversion-gen@$(CODE_GENERATOR_VERSION)


#.PHONY: vgopath
#vgopath: $(VGOPATH) ## Download vgopath locally if necessary.
#$(VGOPATH): $(LOCALBIN)
#	test -s $(LOCALBIN)/vgopath || GOBIN=$(LOCALBIN) go install github.com/ironcore-dev/vgopath@$(VGOPATH_VERSION)
#
#.PHONY: addlicense
#addlicense: $(ADDLICENSE) ## Download addlicense locally if necessary.
#$(ADDLICENSE): $(LOCALBIN)
#	test -s $(LOCALBIN)/addlicense || GOBIN=$(LOCALBIN) go install github.com/google/addlicense@$(ADDLICENSE_VERSION)
#
#.PHONY: goimports
#goimports: $(GOIMPORTS) ## Download goimports locally if necessary.
#$(GOIMPORTS): $(LOCALBIN)
#	test -s $(LOCALBIN)/goimports || GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
#
#.PHONY: gen-crd-api-reference-docs
#gen-crd-api-reference-docs: $(GEN_CRD_API_REFERENCE_DOCS) ## Download gen-crd-api-reference-docs locally if necessary.
#$(GEN_CRD_API_REFERENCE_DOCS): $(LOCALBIN)
#	test -s $(LOCALBIN)/gen-crd-api-reference-docs || GOBIN=$(LOCALBIN) go install github.com/ahmetb/gen-crd-api-reference-docs@$(GEN_CRD_API_REFERENCE_DOCS_VERSION)
#
#.PHONY: mockgen
#mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
#$(MOCKGEN): $(LOCALBIN)
#	test -s $(LOCALBIN)/mockgen || GOBIN=$(LOCALBIN) go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
#
#.PHONY: golangci-lint
#golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
#$(GOLANGCI_LINT): $(LOCALBIN)
#	test -s $(LOCALBIN)/golangci-lint || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
