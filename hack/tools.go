//go:build tools
// +build tools

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "github.com/gardener/gardener/hack/api-reference/template"

	_ "github.com/ahmetb/gen-crd-api-reference-docs"
	_ "github.com/ironcore-dev/vgopath"
	_ "go.uber.org/mock/mockgen"
	_ "golang.org/x/tools/cmd/goimports"
	_ "k8s.io/code-generator"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
