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
	"fmt"
	"net"

	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"

	"github.com/go-logr/logr"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/helper"
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

	if err := validateBastion(bastion); err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	if err := validateCluster(cluster); err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	infrastructureStatus, err := getInfrastructureStatus(ctx, c.client, cluster)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	err = validateInfrastructureStatus(infrastructureStatus)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	return allErrs
}

func validateBastion(bastion *extensionsv1alpha1.Bastion) error {
	if bastion == nil {
		return fmt.Errorf("bastion can't be nil")
	}

	if len(bastion.Spec.UserData) == 0 {
		return fmt.Errorf("bastion spec userdata can't be empty")
	}

	for _, ingress := range bastion.Spec.Ingress {
		if ingress.IPBlock.CIDR == "" {
			return fmt.Errorf("bastion spec Ingress CIDR can't be empty")
		}
		_, _, err := net.ParseCIDR(ingress.IPBlock.CIDR)
		if err != nil {
			return fmt.Errorf("invalid bastion spec Ingress CIDR %w", err)
		}
	}
	return nil
}

func validateCluster(cluster *extensions.Cluster) error {

	if cluster == nil {
		return fmt.Errorf("cluster can't be nil")
	}

	if cluster.Shoot == nil {
		return fmt.Errorf("cluster shoot can't be empty")
	}

	return nil
}

func getInfrastructureStatus(ctx context.Context, c client.Client, cluster *extensions.Cluster) (*api.InfrastructureStatus, error) {
	var infrastructureStatus *api.InfrastructureStatus

	worker := &extensionsv1alpha1.Worker{}
	err := c.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, worker)
	if err != nil {
		return nil, err
	}

	if worker == nil || worker.Spec.InfrastructureProviderStatus == nil {
		return nil, fmt.Errorf("infrastructure provider status must be not empty for worker")
	}

	if infrastructureStatus, err = helper.InfrastructureStatusFromRaw(worker.Spec.InfrastructureProviderStatus); err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	return infrastructureStatus, nil
}

func validateInfrastructureStatus(infrastructureStatus *api.InfrastructureStatus) error {
	emptyref := commonv1alpha1.LocalUIDReference{}

	if infrastructureStatus.NetworkRef == emptyref {
		return fmt.Errorf("network ref must be not empty for infrastructure provider status")
	}

	if infrastructureStatus.PrefixRef == emptyref {
		return fmt.Errorf("prefix ref must be not empty for infrastructure provider status")
	}

	return nil
}
