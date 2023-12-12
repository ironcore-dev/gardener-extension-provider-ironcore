// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/helper"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

// configValidator implements ConfigValidator for ironcore infrastructure resources.
type configValidator struct {
	client client.Client
	logger logr.Logger
}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator(client client.Client, logger logr.Logger) infrastructure.ConfigValidator {
	return &configValidator{
		client: client,
		logger: logger.WithName("ironcore-infrastructure-config-validator"),
	}
}

// Validate validates the provider config of the given infrastructure resource with the cloud provider.
func (c *configValidator) Validate(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) field.ErrorList {
	allErrs := field.ErrorList{}

	if infra == nil || infra.Spec.ProviderConfig == nil {
		return allErrs
	}

	// Get provider config from the infrastructure resource
	config, err := helper.InfrastructureConfigFromInfrastructure(infra)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, err))
		return allErrs
	}

	// check wether a networkRef is set
	if config.NetworkRef == nil {
		return allErrs
	}

	// get ironcore credentials from infrastructure config
	ironcoreClient, namespace, err := ironcore.GetIroncoreClientAndNamespaceFromCloudProviderSecret(ctx, c.client, infra.Namespace)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(nil, fmt.Errorf("could not get ironcore client and namespace: %w", err)))
		return allErrs
	}

	// ensure that the referenced network exists
	network := &networkingv1alpha1.Network{}
	if err := ironcoreClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: config.NetworkRef.Name}, network); err != nil {
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs, field.NotFound(field.NewPath("networkRef"), fmt.Errorf("could not find ironcore network %s: %w", client.ObjectKeyFromObject(network), err)))
			return allErrs
		}
		allErrs = append(allErrs, field.InternalError(field.NewPath("networkRef"), fmt.Errorf("failed to get ironcore network %s: %w", client.ObjectKeyFromObject(network), err)))
	}

	return allErrs
}
