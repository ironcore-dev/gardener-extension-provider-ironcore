#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CODEGEN_PKG="${CODEGEN_PKG:-"$( (go mod download > /dev/null 2>&1) && go list -m -f '{{.Dir}}' k8s.io/code-generator)"}"

VGOPATH="$(mktemp -d)"
trap 'rm -rf "$VGOPATH"' EXIT

# Setup virtual GOPATH so the codegen tools work as expected.
(cd "$SCRIPT_DIR/.."; go run github.com/onmetal/vgopath "$VGOPATH")

export GOPATH="$VGOPATH"
export GO111MODULE=off

bash "$CODEGEN_PKG"/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/client \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  "onmetal:v1alpha1" \
  --output-base "$VGOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt"

bash "$CODEGEN_PKG"/generate-internal-groups.sh \
  conversion \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/client \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  "onmetal:v1alpha1" \
  --extra-peer-dirs=github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal,github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --output-base "$VGOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt"

bash "$CODEGEN_PKG"/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/client/componentconfig \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  "config:v1alpha1" \
  --output-base "$VGOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt"

bash "$CODEGEN_PKG"/generate-internal-groups.sh \
  conversion \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/client/componentconfig \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis \
  "config:v1alpha1" \
  --extra-peer-dirs=github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config,github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime,github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1 \
  --output-base "$VGOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt"
