// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bastion

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// configValidator implements ConfigValidator for onmetal bastion resources.
type configValidator struct {
	common.ClientContext
	client client.Client
	logger logr.Logger
}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator(client client.Client, logger logr.Logger) bastion.ConfigValidator {
	return &configValidator{
		client: client,
		logger: logger.WithName("onmetal-bastion-config-validator"),
	}
}

// Validate validates the provider config of the given bastion resource with the cloud provider.
func (c *configValidator) Validate(ctx context.Context, bastion *extensionsv1alpha1.Bastion, cluster *extensions.Cluster) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}
