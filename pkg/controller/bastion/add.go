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
	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the onmetal bastion controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// BastionConfig contains config for the Bastion config.
	BastionConfig controllerconfig.BastionConfig
}

// AddToManagerWithOptions adds a controller with the given AddOptions to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {
	return bastion.Add(mgr, bastion.AddArgs{
		Actuator:          NewActuator(mgr, &opts.BastionConfig),
		ConfigValidator:   NewConfigValidator(mgr.GetClient(), log.Log),
		ControllerOptions: opts.Controller,
		Predicates:        bastion.DefaultPredicates(opts.IgnoreOperationAnnotation),
		Type:              onmetal.Type,
	})
}

// AddToManager adds a controller with the default AddOptions.
func AddToManager(_ context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}
