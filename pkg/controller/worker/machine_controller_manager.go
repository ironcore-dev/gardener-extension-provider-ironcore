// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
