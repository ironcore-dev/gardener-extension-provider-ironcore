// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	controllerconfig "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/config"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the ironcore bastion controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// BastionConfig contains config for the Bastion config.
	BastionConfig controllerconfig.BastionConfig
	// ExtensionClass defines the extension class this extension is responsible for.
	ExtensionClass extensionsv1alpha1.ExtensionClass
}

// AddToManagerWithOptions adds a controller with the given AddOptions to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {
	return bastion.Add(mgr, bastion.AddArgs{
		Actuator:          NewActuator(mgr, &opts.BastionConfig),
		ConfigValidator:   NewConfigValidator(mgr.GetClient(), log.Log),
		ControllerOptions: opts.Controller,
		Predicates:        bastion.DefaultPredicates(opts.IgnoreOperationAnnotation),
		Type:              ironcore.Type,
		ExtensionClass:    opts.ExtensionClass,
	})
}

// AddToManager adds a controller with the default AddOptions.
func AddToManager(_ context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}
