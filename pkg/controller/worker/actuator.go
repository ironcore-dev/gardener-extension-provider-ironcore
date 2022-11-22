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

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	"github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/helper"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/internal/imagevector"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	"k8s.io/client-go/kubernetes"
)

type delegateFactory struct {
	common.RESTConfigContext
}

// NewActuator creates a new Actuator that updates the status of the handled WorkerPoolConfigs.
func NewActuator() worker.Actuator {
	delegateFactory := &delegateFactory{}

	return genericactuator.NewActuator(
		delegateFactory,
		onmetal.MachineControllerManagerName,
		mcmChart,
		mcmShootChart,
		imagevector.ImageVector(),
		extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
	)
}

func (d *delegateFactory) WorkerDelegate(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) (genericactuator.WorkerDelegate, error) {
	clientset, err := kubernetes.NewForConfig(d.RESTConfig())
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	return NewWorkerDelegate(
		d.ClientContext,

		serverVersion.GitVersion,

		worker,
		cluster,
	)
}

type workerDelegate struct {
	common.ClientContext

	serverVersion string

	cloudProfileConfig *api.CloudProfileConfig
	cluster            *extensionscontroller.Cluster
	worker             *extensionsv1alpha1.Worker
}

// NewWorkerDelegate creates a new context for a worker reconciliation.
func NewWorkerDelegate(
	clientContext common.ClientContext,
	serverVersion string,
	worker *extensionsv1alpha1.Worker,
	cluster *extensionscontroller.Cluster,
) (genericactuator.WorkerDelegate, error) {
	config, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}
	return &workerDelegate{
		ClientContext:      clientContext,
		serverVersion:      serverVersion,
		cloudProfileConfig: config,
		cluster:            cluster,
		worker:             worker,
	}, nil
}
