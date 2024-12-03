// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"context"
	"fmt"
	"net"

	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/go-logr/logr"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllerconfig "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/config"
	api "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/helper"
)

// configValidator implements ConfigValidator for ironcore bastion resources.
type configValidator struct {
	client client.Client
	logger logr.Logger
}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator(client client.Client, logger logr.Logger) bastion.ConfigValidator {
	return &configValidator{
		client: client,
		logger: logger.WithName("ironcore-bastion-config-validator"),
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

	if err = validateInfrastructureStatus(infrastructureStatus); err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	return allErrs
}

func validateBastion(bastion *extensionsv1alpha1.Bastion) error {
	if bastion == nil {
		return fmt.Errorf("bastion can not be nil")
	}

	if len(bastion.Spec.UserData) == 0 {
		return fmt.Errorf("bastion spec userdata can not be empty")
	}

	for _, ingress := range bastion.Spec.Ingress {
		if ingress.IPBlock.CIDR == "" {
			return fmt.Errorf("bastion spec Ingress CIDR can not be empty")
		}
		_, _, err := net.ParseCIDR(ingress.IPBlock.CIDR)
		if err != nil {
			return fmt.Errorf("invalid bastion spec Ingress CIDR: %w", err)
		}
	}
	return nil
}

func validateCluster(cluster *extensions.Cluster) error {

	if cluster == nil {
		return fmt.Errorf("cluster can not be nil")
	}

	if cluster.Shoot == nil {
		return fmt.Errorf("cluster shoot can not be empty")
	}

	return nil
}

func getInfrastructureStatus(ctx context.Context, c client.Client, cluster *extensions.Cluster) (*api.InfrastructureStatus, error) {
	worker := &extensionsv1alpha1.Worker{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, worker); err != nil {
		return nil, err
	}

	if worker == nil || worker.Spec.InfrastructureProviderStatus == nil {
		return nil, fmt.Errorf("infrastructure provider status must be not empty for worker")
	}

	infrastructureStatus, err := helper.InfrastructureStatusFromRaw(worker.Spec.InfrastructureProviderStatus)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	return infrastructureStatus, nil
}

func validateInfrastructureStatus(infrastructureStatus *api.InfrastructureStatus) error {
	emptyref := commonv1alpha1.LocalUIDReference{}

	if infrastructureStatus.NetworkRef == emptyref {
		return fmt.Errorf("network ref must be not empty for infrastructure provider status")
	}

	for _, prefixRef := range infrastructureStatus.PrefixRefs {
		if prefixRef == emptyref {
			return fmt.Errorf("prefix ref must be not empty for infrastructure provider status")
		}
	}

	return nil
}

// validateConfiguration checks whether a bastion configuration is valid.
func validateConfiguration(config *controllerconfig.BastionConfig) error {
	if config == nil {
		return fmt.Errorf("bastionConfig must not be empty")
	}

	if config.MachineClassName == "" {
		return fmt.Errorf("MachineClassName is mandatory")
	}

	if config.VolumeClassName == "" {
		return fmt.Errorf("VolumeClassName is mandatory")
	}

	if config.Image == "" {
		return fmt.Errorf("image is mandatory")
	}
	return nil
}
