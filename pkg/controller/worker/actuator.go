// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	"github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	api "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/helper"
)

type delegateFactory struct {
	gardenReader client.Reader
	seedClient   client.Client
	decoder      runtime.Decoder
	restConfig   *rest.Config
	scheme       *runtime.Scheme
}

type actuator struct {
	worker.Actuator
	workerDelegate *delegateFactory
}

// NewActuator creates a new Actuator that updates the status of the handled WorkerPoolConfigs.
func NewActuator(mgr manager.Manager, gardenCluster cluster.Cluster) worker.Actuator {
	workerDelegate := &delegateFactory{
		gardenReader: gardenCluster.GetAPIReader(),
		seedClient:   mgr.GetClient(),
		decoder:      serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		restConfig:   mgr.GetConfig(),
		scheme:       mgr.GetScheme(),
	}

	return &actuator{
		Actuator:       genericactuator.NewActuator(mgr, gardenCluster, workerDelegate, nil),
		workerDelegate: workerDelegate,
	}
}

func (d *delegateFactory) WorkerDelegate(_ context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) (genericactuator.WorkerDelegate, error) {
	clientset, err := kubernetes.NewForConfig(d.restConfig)
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	return NewWorkerDelegate(
		d.seedClient,
		d.decoder,
		d.scheme,
		serverVersion.GitVersion,
		worker,
		cluster,
	)
}

type workerDelegate struct {
	client  client.Client
	decoder runtime.Decoder
	scheme  *runtime.Scheme

	serverVersion      string
	cloudProfileConfig *api.CloudProfileConfig
	cluster            *extensionscontroller.Cluster
	worker             *extensionsv1alpha1.Worker
}

// NewWorkerDelegate creates a new context for a worker reconciliation.
func NewWorkerDelegate(
	client client.Client,
	decoder runtime.Decoder,
	scheme *runtime.Scheme,
	serverVersion string,
	worker *extensionsv1alpha1.Worker,
	cluster *extensionscontroller.Cluster,
) (
	genericactuator.WorkerDelegate,
	error,
) {
	config, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	return &workerDelegate{
		scheme:             scheme,
		client:             client,
		decoder:            decoder,
		serverVersion:      serverVersion,
		cloudProfileConfig: config,
		cluster:            cluster,
		worker:             worker,
	}, nil
}
