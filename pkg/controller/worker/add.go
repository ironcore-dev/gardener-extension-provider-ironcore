// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	machinescheme "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned/scheme"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{
		RecoverPanic: pointer.Bool(true),
	}
)

// AddOptions are options to apply when adding the ironcore worker controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	RecoverPanic              *bool
	// GardenletManagesMCM specifies whether the machine-controller-manager should be managed.
	GardenletManagesMCM bool
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	schemeBuilder := runtime.NewSchemeBuilder(
		apiextensionsscheme.AddToScheme,
		machinescheme.AddToScheme,
	)
	if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	actuator, err := NewActuator(mgr, opts.GardenletManagesMCM)
	if err != nil {
		return err
	}

	return worker.Add(ctx, mgr, worker.AddArgs{
		Actuator:          actuator,
		ControllerOptions: opts.Controller,
		Predicates:        worker.DefaultPredicates(ctx, mgr, opts.IgnoreOperationAnnotation),
		Type:              ironcore.Type,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}
