// Copyright 2022 IronCore authors
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
	"fmt"
	"path/filepath"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils/chart"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var (
	mcmChart = &chart.Chart{
		Name:   ironcore.MachineControllerManagerName,
		Path:   filepath.Join(ironcore.InternalChartsPath, ironcore.MachineControllerManagerName, "seed"),
		Images: []string{ironcore.MachineControllerManagerImageName, ironcore.MachineControllerManagerProviderIroncoreImageName},
		Objects: []*chart.Object{
			{Type: &appsv1.Deployment{}, Name: ironcore.MachineControllerManagerName},
			{Type: &corev1.Service{}, Name: ironcore.MachineControllerManagerName},
			{Type: &corev1.ServiceAccount{}, Name: ironcore.MachineControllerManagerName},
			{Type: &corev1.Secret{}, Name: ironcore.MachineControllerManagerName},
			{Type: extensionscontroller.GetVerticalPodAutoscalerObject(), Name: ironcore.MachineControllerManagerVpaName},
			{Type: &corev1.ConfigMap{}, Name: ironcore.MachineControllerManagerMonitoringConfigName},
		},
	}

	mcmShootChart = &chart.Chart{
		Name: ironcore.MachineControllerManagerName,
		Path: filepath.Join(ironcore.InternalChartsPath, ironcore.MachineControllerManagerName, "shoot"),
		Objects: []*chart.Object{
			{Type: &rbacv1.ClusterRole{}, Name: fmt.Sprintf("extensions.gardener.cloud:%s:%s", ironcore.ProviderName, ironcore.MachineControllerManagerName)},
			{Type: &rbacv1.ClusterRoleBinding{}, Name: fmt.Sprintf("extensions.gardener.cloud:%s:%s", ironcore.ProviderName, ironcore.MachineControllerManagerName)},
		},
	}
)

func (w *workerDelegate) GetMachineControllerManagerChartValues(ctx context.Context) (map[string]interface{}, error) {
	namespace := &corev1.Namespace{}
	if err := w.client.Get(ctx, kutil.Key(w.worker.Namespace), namespace); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"providerName": ironcore.ProviderName,
		"namespace": map[string]interface{}{
			"uid": namespace.UID,
		},
		"podLabels": map[string]interface{}{
			v1beta1constants.LabelPodMaintenanceRestart: "true",
		},
	}, nil
}

func (w *workerDelegate) GetMachineControllerManagerShootChartValues(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"providerName": ironcore.ProviderName,
	}, nil
}
