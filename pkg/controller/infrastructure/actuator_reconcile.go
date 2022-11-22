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

package infrastructure

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/helper"
)

// Reconcile implements infrastructure.Actuator.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	return a.reconcile(ctx, log, infra, cluster)
}

func (a *actuator) reconcile(ctx context.Context, logger logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	config, err := helper.InfrastructureConfigFromInfrastructure(infra)
	if err != nil {
		return err
	}

	return a.updateProviderStatus(ctx, infra, config)
}
