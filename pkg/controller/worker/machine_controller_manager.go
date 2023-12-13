// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
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
