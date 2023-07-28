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
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha1 "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal/helper"
)

// UpdateMachineImagesStatus updates the machine image status
// with the used machine images for the `Worker` resource.
func (w *workerDelegate) UpdateMachineImagesStatus(ctx context.Context) error {
	var machineImages []apiv1alpha1.MachineImage
	for _, pool := range w.worker.Spec.Pools {
		arch := pointer.StringDeref(pool.Architecture, v1beta1constants.ArchitectureAMD64)

		machineImage, err := w.findMachineImage(pool.MachineImage.Name, pool.MachineImage.Version, &arch)
		if err != nil {
			return err
		}

		machineImages = appendMachineImage(machineImages, apiv1alpha1.MachineImage{
			Name:         pool.MachineImage.Name,
			Version:      pool.MachineImage.Version,
			Image:        machineImage,
			Architecture: &arch,
		})
	}

	// Decode the current worker provider status.
	workerStatus, err := w.decodeWorkerProviderStatus()
	if err != nil {
		return fmt.Errorf("unable to decode the worker provider status: %w", err)
	}

	workerStatus.MachineImages = machineImages

	return w.updateWorkerProviderStatus(ctx, workerStatus)
}

func (w *workerDelegate) findMachineImage(name, version string, architecture *string) (string, error) {
	machineImage, err := helper.FindImageFromCloudProfile(w.cloudProfileConfig, name, version, architecture)
	if err == nil {
		return machineImage, nil
	}

	// Try to look up machine image in worker provider status as it was not found in componentconfig.
	if providerStatus := w.worker.Status.ProviderStatus; providerStatus != nil {
		workerStatus := &apiv1alpha1.WorkerStatus{}
		if _, _, err := w.decoder.Decode(providerStatus.Raw, nil, workerStatus); err != nil {
			return "", fmt.Errorf("could not decode worker status of worker '%s': %w", kutil.ObjectName(w.worker), err)
		}

		machineImage, err := helper.FindMachineImage(workerStatus.MachineImages, name, version, architecture)
		if err != nil {
			return "", worker.ErrorMachineImageNotFound(name, version)
		}

		return machineImage.Image, nil
	}

	return "", worker.ErrorMachineImageNotFound(name, version, *architecture)
}

func appendMachineImage(machineImages []apiv1alpha1.MachineImage, machineImage apiv1alpha1.MachineImage) []apiv1alpha1.MachineImage {
	if _, err := helper.FindMachineImage(machineImages, machineImage.Name, machineImage.Version, machineImage.Architecture); err != nil {
		return append(machineImages, machineImage)
	}
	return machineImages
}

func (w *workerDelegate) decodeWorkerProviderStatus() (*apiv1alpha1.WorkerStatus, error) {
	workerStatus := &apiv1alpha1.WorkerStatus{}

	if w.worker.Status.ProviderStatus == nil {
		return workerStatus, nil
	}

	if _, _, err := w.decoder.Decode(w.worker.Status.ProviderStatus.Raw, nil, workerStatus); err != nil {
		return nil, fmt.Errorf("could not decode WorkerStatus '%s': %w", kutil.ObjectName(w.worker), err)
	}

	return workerStatus, nil
}

func (w *workerDelegate) updateWorkerProviderStatus(ctx context.Context, workerStatus *apiv1alpha1.WorkerStatus) error {
	status := &apiv1alpha1.WorkerStatus{}

	if err := w.scheme.Convert(workerStatus, status, nil); err != nil {
		return err
	}

	status.TypeMeta = metav1.TypeMeta{
		APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
		Kind:       "WorkerStatus",
	}

	patch := client.MergeFrom(w.worker.DeepCopy())
	w.worker.Status.ProviderStatus = &runtime.RawExtension{Object: status}
	return w.client.Status().Patch(ctx, w.worker, patch)
}
