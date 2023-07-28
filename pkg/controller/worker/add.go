// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	machinescheme "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned/scheme"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{
		RecoverPanic: pointer.Bool(true),
	}
)

// AddOptions are options to apply when adding the onmetal worker controller to the manager.
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
		Type:              onmetal.Type,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}
