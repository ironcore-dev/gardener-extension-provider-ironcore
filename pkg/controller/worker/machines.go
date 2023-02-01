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
	"encoding/json"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	machinecontrollerv1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MachineClassKind yields the name of the machine class kind used by onmetal provider.
func (w *workerDelegate) MachineClassKind() string {
	return "MachineClass"
}

// MachineClass yields a newly initialized machine class object.
func (w *workerDelegate) MachineClass() client.Object {
	return &machinecontrollerv1alpha1.MachineClass{}
}

// MachineClassList yields a newly initialized MachineClassList object.
func (w *workerDelegate) MachineClassList() client.ObjectList {
	return &machinecontrollerv1alpha1.MachineClassList{}
}

// DeployMachineClasses generates and creates the onmetal specific machine classes.
func (w *workerDelegate) DeployMachineClasses(ctx context.Context) error {
	machineClasses, machineClassSecrets, err := w.generateMachineClassAndSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate machine classes and machine class secrets: %w", err)
	}

	// apply machine classes and machine secrets
	for _, class := range machineClasses {
		if _, err := controllerutil.CreateOrPatch(ctx, w.Client(), class, nil); err != nil {
			return fmt.Errorf("failed to create/patch machineclass %s: %w", client.ObjectKeyFromObject(class), err)
		}
	}
	for _, secret := range machineClassSecrets {
		if _, err := controllerutil.CreateOrPatch(ctx, w.Client(), secret, nil); err != nil {
			return fmt.Errorf("failed to create/patch machineclass secret %s: %w", client.ObjectKeyFromObject(secret), err)
		}
	}

	return nil
}

// GenerateMachineDeployments generates the configuration for the desired machine deployments.
func (w *workerDelegate) GenerateMachineDeployments(ctx context.Context) (worker.MachineDeployments, error) {
	var (
		machineDeployments = worker.MachineDeployments{}
	)

	for _, pool := range w.worker.Spec.Pools {
		zoneLen := int32(len(pool.Zones))
		for zoneIndex := range pool.Zones {
			workerPoolHash, err := w.generateHashForWorkerPool(pool)
			if err != nil {
				return nil, err
			}
			var (
				deploymentName = fmt.Sprintf("%s-%s-z%d", w.worker.Namespace, pool.Name, zoneIndex+1)
				className      = fmt.Sprintf("%s-%s", deploymentName, workerPoolHash)
			)
			zoneIdx := int32(zoneIndex)

			machineDeployments = append(machineDeployments, worker.MachineDeployment{
				Name:                 deploymentName,
				ClassName:            className,
				SecretName:           className,
				Minimum:              worker.DistributeOverZones(zoneIdx, pool.Minimum, zoneLen),
				Maximum:              worker.DistributeOverZones(zoneIdx, pool.Maximum, zoneLen),
				MaxSurge:             worker.DistributePositiveIntOrPercent(zoneIdx, pool.MaxSurge, zoneLen, pool.Maximum),
				MaxUnavailable:       worker.DistributePositiveIntOrPercent(zoneIdx, pool.MaxUnavailable, zoneLen, pool.Minimum),
				Labels:               pool.Labels,
				Annotations:          pool.Annotations,
				Taints:               pool.Taints,
				MachineConfiguration: genericworkeractuator.ReadMachineConfiguration(pool),
			})
		}
	}

	return machineDeployments, nil
}

func (w *workerDelegate) generateMachineClassAndSecrets(ctx context.Context) ([]*machinecontrollerv1alpha1.MachineClass, []*corev1.Secret, error) {
	var (
		machineClasses      []*machinecontrollerv1alpha1.MachineClass
		machineClassSecrets []*corev1.Secret
	)

	infrastructureStatus := &apisonmetal.InfrastructureStatus{}
	if _, _, err := w.Decoder().Decode(w.worker.Spec.InfrastructureProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return nil, nil, err
	}

	for _, pool := range w.worker.Spec.Pools {
		workerPoolHash, err := w.generateHashForWorkerPool(pool)
		if err != nil {
			return nil, nil, err
		}
		//arch := pointer.StringDeref(pool.Architecture, v1beta1constants.ArchitectureAMD64)

		//machineImage, err := w.findMachineImage(pool.MachineImage.Name, pool.MachineImage.Version, &arch)
		//if err != nil {
		//	return nil, nil, err
		//}

		//disks := make([]map[string]interface{}, 0)
		//// root volume
		//if pool.Volume != nil {
		//	disk, err := createDiskSpecForVolume(*pool.Volume, w.worker.Name, machineImage, true)
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//
		//	disks = append(disks, disk)
		//}

		for zoneIndex, zone := range pool.Zones {
			var (
				deploymentName = fmt.Sprintf("%s-%s-z%d", w.worker.Namespace, pool.Name, zoneIndex+1)
				className      = fmt.Sprintf("%s-%s", deploymentName, workerPoolHash)
			)

			// Here we are going to create the necessary objects:
			// 1. construct a MachineClass per zone containing the ProviderSpec needed by the MCM
			// 2. construct a Secret for each MachineClass containing the user data to create a Kubernetes Node
			// 3. construct a CredentialSecret where we inject the user token and construct an effective kubeconfig
			//    for a given region

			nodeTemplate := &machinecontrollerv1alpha1.NodeTemplate{}
			if pool.NodeTemplate != nil {
				nodeTemplate = &machinecontrollerv1alpha1.NodeTemplate{
					Capacity:     pool.NodeTemplate.Capacity,
					InstanceType: pool.MachineType,
					Region:       w.worker.Spec.Region,
					Zone:         zone,
				}
			}

			// TODO: adjust according the MachineClassSpec of the Onmetal MCM
			machineClassProviderSpec := map[string]interface{}{
				"machineClassRefName": pool.MachineType,
				"machinePoolRefName":  zone,
				"image":               pool.MachineImage,
			}
			machineClassProviderSpecJSON, err := json.Marshal(machineClassProviderSpec)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal machine class for machine pool %s: %w", pool.Name, err)
			}

			machineClass := &machinecontrollerv1alpha1.MachineClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      className,
					Namespace: w.worker.Namespace,
					Labels: map[string]string{
						v1beta1constants.GardenerPurpose: genericworkeractuator.GardenPurposeMachineClass,
					},
				},
				NodeTemplate: nodeTemplate,
				CredentialsSecretRef: &corev1.SecretReference{
					Name:      w.worker.Spec.SecretRef.Name,
					Namespace: w.worker.Spec.SecretRef.Namespace,
				},
				ProviderSpec: runtime.RawExtension{
					Raw: machineClassProviderSpecJSON,
				},
				Provider: onmetal.ProviderName,
				SecretRef: &corev1.SecretReference{
					Name:      className,
					Namespace: w.worker.Namespace,
				},
			}

			machineClassSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      className,
					Namespace: w.worker.Namespace,
					Labels: map[string]string{
						v1beta1constants.GardenerPurpose: genericworkeractuator.GardenPurposeMachineClass,
					},
				},
				Data: map[string][]byte{
					"userData": pool.UserData,
				},
			}

			machineClasses = append(machineClasses, machineClass)
			machineClassSecrets = append(machineClassSecrets, machineClassSecret)
		}
	}

	return machineClasses, machineClassSecrets, nil
}

func (w *workerDelegate) generateHashForWorkerPool(pool v1alpha1.WorkerPool) (string, error) {
	workerPoolHash, err := worker.WorkerPoolHash(pool, w.cluster)
	if err != nil {
		return "", fmt.Errorf("failed to generate hash for worke pool %s: %w", pool.Name, err)
	}
	return workerPoolHash, err
}

//func createDiskSpecForVolume(volume v1alpha1.Volume, workerName string, machineImage string, boot bool) (map[string]interface{}, error) {
//	return createDiskSpec(volume.Size, workerName, boot, &machineImage, volume.Type)
//}

//func createDiskSpec(size, workerName string, boot bool, machineImage, volumeType *string) (map[string]interface{}, error) {
//	volumeSize, err := worker.DiskSize(size)
//	if err != nil {
//		return nil, err
//	}
//
//	disk := map[string]interface{}{
//		"autoDelete": true,
//		"boot":       boot,
//		"sizeGb":     volumeSize,
//		"labels": map[string]interface{}{
//			"name": workerName,
//		},
//	}
//
//	if machineImage != nil {
//		disk["image"] = *machineImage
//	}
//
//	if volumeType != nil {
//		disk["type"] = *volumeType
//	}
//
//	return disk, nil
//}
