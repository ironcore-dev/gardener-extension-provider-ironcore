// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/internal/imagevector"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the ironcore controlplane controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// WebhookServerNamespace is the namespace in which the webhook server runs.
	WebhookServerNamespace string
	// WebhookServerPort is the port on which the webhook server listens.
	WebhookServerPort int
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	webhookServer := mgr.GetWebhookServer()
	defaultServer, ok := webhookServer.(*webhook.DefaultServer)
	if !ok {
		return fmt.Errorf("expected *webhook.DefaultServer, got %T", webhookServer)
	}

	genericActuator, err := genericactuator.NewActuator(mgr, ironcore.ProviderName,
		secretConfigsFunc, shootAccessSecretsFunc, nil, nil,
		configChart, controlPlaneChart, controlPlaneShootChart, nil, storageClassChart, nil,
		NewValuesProvider(mgr), extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
		imagevector.ImageVector(), ironcore.CloudProviderConfigName, nil, opts.WebhookServerNamespace, defaultServer.Options.Port)

	if err != nil {
		return err
	}

	return controlplane.Add(ctx, mgr, controlplane.AddArgs{
		Actuator:          genericActuator,
		ControllerOptions: opts.Controller,
		Predicates:        controlplane.DefaultPredicates(ctx, mgr, opts.IgnoreOperationAnnotation),
		Type:              ironcore.Type,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}
