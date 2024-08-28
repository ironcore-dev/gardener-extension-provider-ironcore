// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	machinecontrollerv1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	ironcoreextensionv1alpha1 "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var _ = Describe("Machines", func() {
	ns, _ := SetupTest()

	It("should create the expected machine class for a multi zone cluster", func(ctx SpecContext) {
		By("defining and setting infrastructure status for worker")
		infraStatus := &ironcoreextensionv1alpha1.InfrastructureStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: ironcoreextensionv1alpha1.SchemeGroupVersion.String(),
				Kind:       "InfrastructureStatus",
			},
			NetworkRef: commonv1alpha1.LocalUIDReference{
				Name: "my-network",
				UID:  "1234",
			},
			PrefixRef: commonv1alpha1.LocalUIDReference{
				Name: "my-prefix",
				UID:  "3766",
			},
		}
		w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{Raw: encodeObject(infraStatus)}

		By("deploying the machine class for a given multi zone cluster")
		decoder := serializer.NewCodecFactory(k8sClient.Scheme(), serializer.EnableStrict).UniversalDecoder()
		workerDelegate, err := NewWorkerDelegate(k8sClient, decoder, k8sClient.Scheme(), "", w, testCluster)
		Expect(err).NotTo(HaveOccurred())

		err = workerDelegate.DeployMachineClasses(ctx)
		Expect(err).NotTo(HaveOccurred())

		additionalData := []string{strconv.FormatBool(volumeEncrypted), datVolumeName, volumeSize, volumeType, strconv.FormatBool(volumeEncrypted)}
		workerPoolHash, err := worker.WorkerPoolHash(pool, testCluster, additionalData, nil)
		Expect(err).NotTo(HaveOccurred())

		By("ensuring that the machine class for each pool has been deployed")
		var (
			deploymentName = fmt.Sprintf("%s-%s-z%d", ns.Name, pool.Name, 1)
			className      = fmt.Sprintf("%s-%s", deploymentName, workerPoolHash)
		)

		machineClass := &machinecontrollerv1alpha1.MachineClass{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      className,
			},
		}

		machineClassProviderSpec := map[string]interface{}{
			"image": "registry/my-os",
			"rootDisk": map[string]interface{}{
				"size":            pool.Volume.Size,
				"volumeClassName": pool.Volume.Type,
			},
			"networkName": infraStatus.NetworkRef.Name,
			"prefixName":  infraStatus.PrefixRef.Name,
			"labels": map[string]interface{}{
				ironcore.ClusterNameLabel: testCluster.ObjectMeta.Name,
			},
		}

		Eventually(Object(machineClass)).Should(SatisfyAll(
			HaveField("CredentialsSecretRef", &corev1.SecretReference{
				Namespace: w.Spec.SecretRef.Namespace,
				Name:      w.Spec.SecretRef.Name,
			}),
			HaveField("SecretRef", &corev1.SecretReference{
				Namespace: ns.Name,
				Name:      className,
			}),
			HaveField("Provider", "ironcore"),
			HaveField("NodeTemplate", &machinecontrollerv1alpha1.NodeTemplate{
				Capacity:     pool.NodeTemplate.Capacity,
				InstanceType: pool.MachineType,
				Region:       w.Spec.Region,
				Zone:         "zone1",
			}),
			HaveField("ProviderSpec", runtime.RawExtension{
				Raw: encodeMap(machineClassProviderSpec),
			}),
		))

		By("ensuring that the machine class secret have been applied")
		machineClassSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      className,
			},
		}

		Eventually(Object(machineClassSecret)).Should(SatisfyAll(
			HaveField("ObjectMeta.Labels", HaveKeyWithValue(v1beta1constants.GardenerPurpose, v1beta1constants.GardenPurposeMachineClass)),
			HaveField("Data", HaveKeyWithValue("userData", []byte("some-data"))),
		))
	})

	It("should generate the machine deployments", func(ctx SpecContext) {
		By("creating a worker delegate")
		additionalData := []string{strconv.FormatBool(volumeEncrypted), datVolumeName, volumeSize, volumeType, strconv.FormatBool(volumeEncrypted)}
		workerPoolHash, err := worker.WorkerPoolHash(pool, testCluster, additionalData, nil)
		Expect(err).NotTo(HaveOccurred())
		var (
			deploymentName1 = fmt.Sprintf("%s-%s-z%d", w.Namespace, pool.Name, 1)
			deploymentName2 = fmt.Sprintf("%s-%s-z%d", w.Namespace, pool.Name, 2)
			className1      = fmt.Sprintf("%s-%s", deploymentName1, workerPoolHash)
			className2      = fmt.Sprintf("%s-%s", deploymentName2, workerPoolHash)
		)
		decoder := serializer.NewCodecFactory(k8sClient.Scheme(), serializer.EnableStrict).UniversalDecoder()
		workerDelegate, err := NewWorkerDelegate(k8sClient, decoder, k8sClient.Scheme(), "", w, testCluster)
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
