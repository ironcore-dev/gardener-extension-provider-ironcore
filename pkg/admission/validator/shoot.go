// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"
	"reflect"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/admission"
	apisironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	ironcorevalidation "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/validation"
)

type shoot struct {
	client         client.Client
	decoder        runtime.Decoder
	lenientDecoder runtime.Decoder
}

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator(mgr manager.Manager) extensionswebhook.Validator {
	return &shoot{
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

// Validate validates the given shoot objects.
func (s *shoot) Validate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if old != nil {
		oldShoot, ok := old.(*core.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", old)
		}
		return s.validateUpdate(ctx, oldShoot, shoot)
	}

	return s.validateCreate(ctx, shoot)
}

var (
	specPath = field.NewPath("spec")

	networkPath  = specPath.Child("networking")
	providerPath = specPath.Child("provider")

	infrastructureConfigPath = providerPath.Child("infrastructureConfig")
	controlPlaneConfigPath   = providerPath.Child("controlPlaneConfig")
	workersPath              = providerPath.Child("workers")
)

type validationContext struct {
	shoot                *core.Shoot
	infrastructureConfig *apisironcore.InfrastructureConfig
	controlPlaneConfig   *apisironcore.ControlPlaneConfig
	cloudProfile         *gardencorev1beta1.CloudProfile
}

func (s *shoot) validateContext(valContext *validationContext) field.ErrorList {
	var (
		allErrors = field.ErrorList{}
	)

	allErrors = append(allErrors, ironcorevalidation.ValidateNetworking(valContext.shoot.Spec.Networking, networkPath)...)
	allErrors = append(allErrors, ironcorevalidation.ValidateInfrastructureConfig(valContext.infrastructureConfig, valContext.shoot.Spec.Networking.Nodes, valContext.shoot.Spec.Networking.Pods, valContext.shoot.Spec.Networking.Services, infrastructureConfigPath)...)
	allErrors = append(allErrors, ironcorevalidation.ValidateWorkers(valContext.shoot.Spec.Provider.Workers, workersPath)...)
	allErrors = append(allErrors, ironcorevalidation.ValidateControlPlaneConfig(valContext.controlPlaneConfig, valContext.shoot.Spec.Kubernetes.Version, controlPlaneConfigPath)...)

	return allErrors
}

func (s *shoot) validateCreate(ctx context.Context, shoot *core.Shoot) error {
	validationContext, err := newValidationContext(ctx, s.decoder, s.client, shoot)
	if err != nil {
		return err
	}

	return s.validateContext(validationContext).ToAggregate()
}

func (s *shoot) validateUpdate(ctx context.Context, oldShoot, currentShoot *core.Shoot) error {
	oldValContext, err := newValidationContext(ctx, s.lenientDecoder, s.client, oldShoot)
	if err != nil {
		return err
	}

	currentValContext, err := newValidationContext(ctx, s.decoder, s.client, currentShoot)
	if err != nil {
		return err
	}

	var (
		oldInfrastructureConfig, currentInfrastructureConfig = oldValContext.infrastructureConfig, currentValContext.infrastructureConfig
		oldControlPlaneConfig, currentControlPlaneConfig     = oldValContext.controlPlaneConfig, currentValContext.controlPlaneConfig
		allErrors                                            = field.ErrorList{}
	)

	if !reflect.DeepEqual(oldInfrastructureConfig, currentInfrastructureConfig) {
		allErrors = append(allErrors, ironcorevalidation.ValidateInfrastructureConfigUpdate(oldInfrastructureConfig, currentInfrastructureConfig, infrastructureConfigPath)...)
	}

	if !reflect.DeepEqual(oldControlPlaneConfig, currentControlPlaneConfig) {
		allErrors = append(allErrors, ironcorevalidation.ValidateControlPlaneConfigUpdate(oldControlPlaneConfig, currentControlPlaneConfig, controlPlaneConfigPath)...)
	}

	allErrors = append(allErrors, ironcorevalidation.ValidateWorkersUpdate(oldValContext.shoot.Spec.Provider.Workers, currentValContext.shoot.Spec.Provider.Workers, workersPath)...)
	allErrors = append(allErrors, s.validateContext(currentValContext)...)

	return allErrors.ToAggregate()

}

func newValidationContext(ctx context.Context, decoder runtime.Decoder, c client.Client, shoot *core.Shoot) (*validationContext, error) {
	if shoot.Spec.Provider.InfrastructureConfig == nil {
		return nil, field.Required(infrastructureConfigPath, "infrastructureConfig must be set for ironcore shoots")
	}
	infrastructureConfig, err := admission.DecodeInfrastructureConfig(decoder, shoot.Spec.Provider.InfrastructureConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding infrastructureConfig: %v", err)
	}

	if shoot.Spec.Provider.ControlPlaneConfig == nil {
		return nil, field.Required(controlPlaneConfigPath, "controlPlaneConfig must be set for ironcore shoots")
	}
	controlPlaneConfig, err := admission.DecodeControlPlaneConfig(decoder, shoot.Spec.Provider.ControlPlaneConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding controlPlaneConfig: %v", err)
	}

	cloudProfile := &gardencorev1beta1.CloudProfile{}
	if err := c.Get(ctx, client.ObjectKey{Name: shoot.Spec.CloudProfileName}, cloudProfile); err != nil {
		return nil, err
	}

	if cloudProfile.Spec.ProviderConfig == nil {
		return nil, fmt.Errorf("providerConfig is not given for cloud profile %q", cloudProfile.Name)
	}

	return &validationContext{
		shoot:                shoot,
		infrastructureConfig: infrastructureConfig,
		controlPlaneConfig:   controlPlaneConfig,
		cloudProfile:         cloudProfile,
	}, nil
}
