EXTENSION_PREFIX            := gardener-extension
NAME                        := provider-onmetal
REGISTRY                    := ghcr.io
ADMISSION_NAME              := admission-onmetal
IMAGE_PREFIX                := $(REGISTRY)/extensions
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
LD_FLAGS                    := "-w"
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

TOOLS_DIR := hack/tools
#include vendor/github.com/gardener/gardener/hack/tools.mk

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
INSTALL ?= $(LOCALBIN)/install
CLEAN ?= $(LOCALBIN)/clean 
FORMAT ?= $(LOCALBIN)/format
TEST_COV ?= $(LOCALBIN)/test-cov
TEST_CLEAN ?= $(LOCALBIN)/test-clean
GOIMPORTS ?= $(LOCALBIN)/goimports
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
CHECK ?= $(LOCALBIN)/check
CHECK_CHARTS ?= $(LOCALBIN)/check-charts
VGOPATH ?= $(LOCALBIN)/vgopath
DEEPCOPY_GEN ?= $(LOCALBIN)/deepcopy-gen
CONVERSION_GEN ?= $(LOCALBIN)/conversion-gen
DEFAULTER_GEN ?= $(LOCALBIN)/defaulter-gen
ADDLICENSE ?= $(LOCALBIN)/addlicense

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.2
GOLANGCI_LINT_VERSION ?= v1.50.1
VGOPATH_VERSION ?= v0.0.2
CODE_GENERATOR_VERSION ?= v0.26.1
ADDLICENSE_VERSION ?= v1.1.0

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
		--webhook-config-cert-dir=./example/admission-onmetal-certs

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

INSTALL_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/install.sh"
$(INSTALL): $(LOCALBIN)
	curl -Ss $(INSTALL_SCRIPT_URL) -o $(INSTALL)
	chmod +x $(INSTALL)

.PHONY: install
install: $(INSTALL)
	LD_FLAGS=$(LD_FLAGS) $(INSTALL) -mod=mod ./...

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-images
docker-images:
	@docker build -t $(IMAGE_PREFIX)/$(NAME):latest           -f Dockerfile -m 6g --target $(EXTENSION_PREFIX)-$(NAME) .
	@docker build -t $(IMAGE_PREFIX)/$(ADMISSION_NAME):latest -f Dockerfile -m 6g --target $(EXTENSION_PREFIX)-$(ADMISSION_NAME) .

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

CLEAN_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/clean.sh"
$(CLEAN): $(LOCALBIN)
	curl -Ss $(CLEAN_SCRIPT_URL) -o $(CLEAN)
	chmod +x $(CLEAN)

.PHONY: clean
clean: $(CLEAN)
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	$(CLEAN) ./cmd/... ./pkg/... ./test/...


$(GOIMPORTS): go.mod
	go build -o $(GOIMPORTS) golang.org/x/tools/cmd/goimports

$(GOLANGCI_LINT): $(call tool_version_file,$(GOLANGCI_LINT),$(GOLANGCI_LINT_VERSION))
	@# CGO_ENABLED has to be set to 1 in order for golangci-lint to be able to load plugins
	@# see https://github.com/golangci/golangci-lint/issues/1276
	GOBIN=$(abspath $(LOCALBIN)) CGO_ENABLED=1 go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

CHECK_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/check.sh"
$(CHECK): $(LOCALBIN)
	curl -Ss $(CHECK_SCRIPT_URL) -o $(CHECK)
	chmod +x $(CHECK)

CHECK_CHARTS_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/check-charts.sh"
$(CHECK_CHARTS): $(LOCALBIN)
	curl -Ss $(CHECK_CHARTS_SCRIPT_URL) -o $(CHECK_CHARTS)
	chmod +x $(CHECK_CHARTS)

.PHONY: add-license
add-license: addlicense ## Add license headers to all go files.
	find . -name '*.go' -exec $(ADDLICENSE) -c 'OnMetal authors' {} +

.PHONY: check-license
check-license: addlicense ## Check that every file has a license header present.
	find . -name '*.go' -exec $(ADDLICENSE) -check -c 'OnMetal authors' {} +

.PHONY: check
check: generate add-license lint test # Generate manifests, code, lint, add licenses, test

CHECK_DOCFORGE_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/check-docforge.sh"
$(CHECK_DOCFORGE): $(LOCALBIN)
	curl -Ss $(CHECK_DOCFORGE_SCRIPT_URL) -o $(CHECK_DOCFORGE)
	chmod +x $(CHECK_DOCFORGE)


.PHONY: generate
generate: vgopath deepcopy-gen defaulter-gen conversion-gen
	VGOPATH=$(VGOPATH) \
	DEEPCOPY_GEN=$(DEEPCOPY_GEN) \
	DEFAULTER_GEN=$(DEFAULTER_GEN) \
	CONVERSION_GEN=$(CONVERSION_GEN) \
	./hack/update-codegen.sh
	go generate ./charts/...

FORMAT_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/format.sh"
$(FORMAT): $(LOCALBIN)
	curl -Ss $(FORMAT_SCRIPT_URL) -o $(FORMAT)
	chmod +x $(FORMAT)

.PHONY: format
format: $(FORMAT)
	$(FORMAT) ./cmd ./pkg ./test

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint on the code.
	golangci-lint run ./...

.PHONY: test
test: generate fmt vet envtest checklicense ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

.PHONY: addlicense
addlicense: ## Add license headers to all go files.
	find . -name '*.go' -exec go run github.com/google/addlicense -c 'OnMetal authors' {} +

.PHONY: checklicense
checklicense: ## Check that every file has a license header present.
	find . -name '*.go' -exec go run github.com/google/addlicense  -check -c 'OnMetal authors' {} +

TEST_COV_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/test-cover.sh"
$(TEST_COV): $(LOCALBIN)
	curl -Ss $(TEST_COV_SCRIPT_URL) -o $(TEST_COV)
	chmod +x $(TEST_COV)

.PHONY: test-cov
test-cov: $(TEST_COV)
	$(TEST_COV) -mod=mod ./cmd/... ./pkg/...

TEST_CLEAN_SCRIPT_URL ?= "https://raw.githubusercontent.com/gardener/gardener/master/hack/test-cover-clean.sh"
$(TEST_CLEAN): $(LOCALBIN)
	curl -Ss $(TEST_CLEAN_SCRIPT_URL) -o $(TEST_CLEAN)
	chmod +x $(TEST_CLEAN)

.PHONY: test-clean
test-clean: $(TEST_CLEAN)
	$(TEST_CLEAN)

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check-generate check format test-cov test-clean

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: vgopath
vgopath: $(VGOPATH) ## Download vgopath locally if necessary.
$(VGOPATH): $(LOCALBIN)
	test -s $(LOCALBIN)/vgopath || GOBIN=$(LOCALBIN) go install github.com/onmetal/vgopath@$(VGOPATH_VERSION)

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

.PHONY: addlicense
addlicense: $(ADDLICENSE) ## Download addlicense locally if necessary.
$(ADDLICENSE): $(LOCALBIN)
	test -s $(LOCALBIN)/addlicense || GOBIN=$(LOCALBIN) go install github.com/google/addlicense@$(ADDLICENSE_VERSION)
