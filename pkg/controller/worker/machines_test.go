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
	"encoding/json"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	machinecontrollerv1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Machines", func() {
	ctx := testutils.SetupContext()
	ns, _ := SetupTest(ctx)

	It("should create the correct kind of the machine class", func() {
		workerDelegate, err := NewWorkerDelegate(common.NewClientContext(nil, nil, nil), "", nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(workerDelegate.MachineClassKind()).To(Equal("MachineClass"))
	})

	It("should create the correct type for the machine class", func() {
		workerDelegate, err := NewWorkerDelegate(common.NewClientContext(nil, nil, nil), "", nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(workerDelegate.MachineClass()).To(Equal(&machinev1alpha1.MachineClass{}))
	})

	It("should create the correct type for the machine class list", func() {
		workerDelegate, err := NewWorkerDelegate(common.NewClientContext(nil, nil, nil), "", nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(workerDelegate.MachineClassList()).To(Equal(&machinev1alpha1.MachineClassList{}))
	})

	It("should create the expected machine classes for a multi zone cluster", func() {
		By("defining and setting infrastructure status for worker")
		infraStatus := &api.InfrastructureStatus{
			TypeMeta: metav1.TypeMeta{
				Kind:       "InfrastructureStatus",
				APIVersion: "onmetal.provider.extensions.gardener.cloud/v1alpha1",
			},
			NetworkRef: commonv1alpha1.LocalUIDReference{
				Name: "my-network",
				UID:  "1234",
			},
		}
		w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{Raw: encodeObject(infraStatus)}

		By("deploying the machine classes for a given multi zone cluster")
		decoder := serializer.NewCodecFactory(k8sClient.Scheme(), serializer.EnableStrict).UniversalDecoder()
		workerDelegate, err := NewWorkerDelegate(common.NewClientContext(k8sClient, k8sClient.Scheme(), decoder), "", w, cluster)
		Expect(err).NotTo(HaveOccurred())

		err = workerDelegate.DeployMachineClasses(ctx)
		Expect(err).NotTo(HaveOccurred())

		workerPoolHash, err := worker.WorkerPoolHash(pool, cluster)
		Expect(err).NotTo(HaveOccurred())

		By("ensuring that the machine classes for each pool has been deployed")
		var (
			deploymentName1 = fmt.Sprintf("%s-%s-z%d", ns.Name, "pool", 1)
			deploymentName2 = fmt.Sprintf("%s-%s-z%d", ns.Name, "pool", 2)
			className1      = fmt.Sprintf("%s-%s", deploymentName1, workerPoolHash)
			className2      = fmt.Sprintf("%s-%s", deploymentName2, workerPoolHash)
		)
		Eventually(func(g Gomega) {
			machineClass1 := &machinecontrollerv1alpha1.MachineClass{}
			machineClassKey1 := types.NamespacedName{Namespace: ns.Name, Name: className1}
			err := k8sClient.Get(ctx, machineClassKey1, machineClass1)
			Expect(client.IgnoreNotFound(err)).To(Succeed())
			g.Expect(err).NotTo(HaveOccurred())

			machineClass2 := &machinecontrollerv1alpha1.MachineClass{}
			machineClassKey2 := types.NamespacedName{Namespace: ns.Name, Name: className2}
			err = k8sClient.Get(ctx, machineClassKey2, machineClass2)
			Expect(client.IgnoreNotFound(err)).To(Succeed())
			g.Expect(err).NotTo(HaveOccurred())

			machineClassProviderSpec1 := map[string]interface{}{
				"machineClassRefName": pool.MachineType,
				"machinePoolRefName":  "zone1",
				"image":               pool.MachineImage,
			}

			machineClassProviderSpec2 := map[string]interface{}{
				"machineClassRefName": pool.MachineType,
				"machinePoolRefName":  "zone2",
				"image":               pool.MachineImage,
			}

			g.Expect(machineClass1).Should(SatisfyAll(
				HaveField("ObjectMeta.Labels", HaveKeyWithValue(v1beta1constants.GardenerPurpose, genericworkeractuator.GardenPurposeMachineClass)),
				HaveField("CredentialsSecretRef", &corev1.SecretReference{
					Namespace: w.Spec.SecretRef.Namespace,
					Name:      w.Spec.SecretRef.Name,
				}),
				HaveField("SecretRef", &corev1.SecretReference{
					Namespace: ns.Name,
					Name:      className1,
				}),
				HaveField("Provider", onmetal.ProviderName),
				HaveField("NodeTemplate", &machinecontrollerv1alpha1.NodeTemplate{
					Capacity:     pool.NodeTemplate.Capacity,
					InstanceType: pool.MachineType,
					Region:       w.Spec.Region,
					Zone:         "zone1",
				}),
				HaveField("ProviderSpec", runtime.RawExtension{
					Raw: encodeMap(machineClassProviderSpec1),
				}),
			))

			g.Expect(machineClass2).Should(SatisfyAll(
				HaveField("ObjectMeta.Labels", HaveKeyWithValue(v1beta1constants.GardenerPurpose, genericworkeractuator.GardenPurposeMachineClass)),
				HaveField("CredentialsSecretRef", &corev1.SecretReference{
					Namespace: w.Spec.SecretRef.Namespace,
					Name:      w.Spec.SecretRef.Name,
				}),
				HaveField("SecretRef", &corev1.SecretReference{
					Namespace: ns.Name,
					Name:      className2,
				}),
				HaveField("Provider", onmetal.ProviderName),
				HaveField("NodeTemplate", &machinecontrollerv1alpha1.NodeTemplate{
					Capacity:     pool.NodeTemplate.Capacity,
					InstanceType: pool.MachineType,
					Region:       w.Spec.Region,
					Zone:         "zone2",
				}),
				HaveField("ProviderSpec", runtime.RawExtension{
					Raw: encodeMap(machineClassProviderSpec2),
				}),
			))
		}).Should(Succeed())

		By("ensuring that the machine class secrets have been applied")
		Eventually(func(g Gomega) {
			machineClassSecret1 := &corev1.Secret{}
			machineClassSecretKey1 := types.NamespacedName{Namespace: ns.Name, Name: className1}
			err := k8sClient.Get(ctx, machineClassSecretKey1, machineClassSecret1)
			Expect(client.IgnoreNotFound(err)).To(Succeed())
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(machineClassSecret1).Should(SatisfyAll(
				HaveField("ObjectMeta.Labels", HaveKeyWithValue(v1beta1constants.GardenerPurpose, genericworkeractuator.GardenPurposeMachineClass)),
				HaveField("Data", HaveKeyWithValue("userData", []byte("some-data"))),
			))

			machineClassSecret2 := &corev1.Secret{}
			machineClassSecretKey2 := types.NamespacedName{Namespace: ns.Name, Name: className2}
			err = k8sClient.Get(ctx, machineClassSecretKey2, machineClassSecret2)
			Expect(client.IgnoreNotFound(err)).To(Succeed())
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(machineClassSecret2).Should(SatisfyAll(
				HaveField("ObjectMeta.Labels", HaveKeyWithValue(v1beta1constants.GardenerPurpose, genericworkeractuator.GardenPurposeMachineClass)),
				HaveField("Data", HaveKeyWithValue("userData", []byte("some-data"))),
			))
		}).Should(Succeed())
	})

	It("should generate the machine deployments", func() {
		By("creating a worker delegate")
		workerPoolHash, err := worker.WorkerPoolHash(pool, cluster)
		Expect(err).NotTo(HaveOccurred())
		var (
			deploymentName1 = fmt.Sprintf("%s-%s-z%d", w.Namespace, pool.Name, 1)
			deploymentName2 = fmt.Sprintf("%s-%s-z%d", w.Namespace, pool.Name, 2)
			className1      = fmt.Sprintf("%s-%s", deploymentName1, workerPoolHash)
			className2      = fmt.Sprintf("%s-%s", deploymentName2, workerPoolHash)
		)
		decoder := serializer.NewCodecFactory(k8sClient.Scheme(), serializer.EnableStrict).UniversalDecoder()
		workerDelegate, err := NewWorkerDelegate(common.NewClientContext(k8sClient, k8sClient.Scheme(), decoder), "", w, cluster)
		Expect(err).NotTo(HaveOccurred())

		By("generating the machine deployments")
		machineDeployments, err := workerDelegate.GenerateMachineDeployments(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(machineDeployments).To(Equal(worker.MachineDeployments{
			worker.MachineDeployment{
				Name:                 deploymentName1,
				ClassName:            className1,
				SecretName:           className1,
				Minimum:              worker.DistributeOverZones(0, pool.Minimum, 2),
				Maximum:              worker.DistributeOverZones(0, pool.Maximum, 2),
				MaxSurge:             worker.DistributePositiveIntOrPercent(0, pool.MaxSurge, 2, pool.Maximum),
				MaxUnavailable:       worker.DistributePositiveIntOrPercent(0, pool.MaxUnavailable, 2, pool.Minimum),
				Labels:               pool.Labels,
				Annotations:          pool.Annotations,
				Taints:               pool.Taints,
				MachineConfiguration: genericworkeractuator.ReadMachineConfiguration(pool),
			},
			worker.MachineDeployment{
				Name:                 deploymentName2,
				ClassName:            className2,
				SecretName:           className2,
				Minimum:              worker.DistributeOverZones(1, pool.Minimum, 2),
				Maximum:              worker.DistributeOverZones(1, pool.Maximum, 2),
				MaxSurge:             worker.DistributePositiveIntOrPercent(1, pool.MaxSurge, 2, pool.Maximum),
				MaxUnavailable:       worker.DistributePositiveIntOrPercent(1, pool.MaxUnavailable, 2, pool.Minimum),
				Labels:               pool.Labels,
				Annotations:          pool.Annotations,
				Taints:               pool.Taints,
				MachineConfiguration: genericworkeractuator.ReadMachineConfiguration(pool),
			},
		}))
	})
})

func encodeObject(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func encodeMap(m map[string]interface{}) []byte {
	data, _ := json.Marshal(m)
	return data
}
