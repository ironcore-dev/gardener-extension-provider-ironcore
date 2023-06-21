// Copyright 2023 OnMetal authors
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

package bastion

import (
	"net/netip"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
	corev1alpha1 "github.com/onmetal/onmetal-api/api/core/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
)

var _ = Describe("Bastion Host Reconcile", func() {
	ns := SetupTest()

	It("should create igntion secret and machine for a given bastion configuration", func(ctx SpecContext) {
		By("getting the cluster object")
		cluster, err := extensionscontroller.GetCluster(ctx, k8sClient, ns.Name)
		Expect(err).NotTo(HaveOccurred())

		By("creating bastion resource")
		bastion := &extensionsv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-bastion",
			},
			Spec: extensionsv1alpha1.BastionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           onmetal.Type,
					ProviderConfig: nil,
				},
				UserData: []byte("abcd"),
				Ingress: []extensionsv1alpha1.BastionIngressPolicy{
					{IPBlock: networkingv1.IPBlock{
						CIDR: "213.69.151.0/24",
					}},
				},
			},
		}
		Expect(k8sClient.Create(ctx, bastion)).Should(Succeed())
		DeferCleanup(k8sClient.Delete, bastion)

		Eventually(Object(bastion)).Should(SatisfyAll(
			HaveField("Status.LastOperation.Type", gardencorev1beta1.LastOperationTypeCreate),
		))

		By("ensuring bastion host is created with correct spec")
		bastionHostName, err := generateBastionHostResourceName(cluster.ObjectMeta.Name, bastion)
		Expect(err).ShouldNot(HaveOccurred())
		bastionHost := &computev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      bastionHostName,
			},
		}
		Eventually(Object(bastionHost)).Should(SatisfyAll(
			HaveField("Spec.MachineClassRef", corev1.LocalObjectReference{
				Name: "my-machine-class",
			}),
			HaveField("Spec.IgnitionRef.Name", getIgnitionNameForMachine(bastionHost.Name)),
			HaveField("Spec.Power", computev1alpha1.PowerOn),
			HaveField("Spec.Volumes", ContainElement(SatisfyAll(
				HaveField("Name", "root"),
				HaveField("VolumeSource.Ephemeral.VolumeTemplate.Spec.VolumeClassRef.Name", "my-volume-class"),
				HaveField("VolumeSource.Ephemeral.VolumeTemplate.Spec.Image", "my-image"),
				HaveField("VolumeSource.Ephemeral.VolumeTemplate.Spec.Resources", Equal(corev1alpha1.ResourceList{
					corev1alpha1.ResourceStorage: resource.MustParse("10Gi"),
				})),
			))),
			HaveField("Spec.NetworkInterfaces", ContainElement(SatisfyAll(
				HaveField("Name", "primary"),
				HaveField("NetworkInterfaceSource.Ephemeral.NetworkInterfaceTemplate.Spec.NetworkRef.Name", "my-network"),
				HaveField("NetworkInterfaceSource.Ephemeral.NetworkInterfaceTemplate.Spec.IPFamilies", ConsistOf(corev1.IPv4Protocol)),
				HaveField("NetworkInterfaceSource.Ephemeral.NetworkInterfaceTemplate.Spec.VirtualIP.Ephemeral.VirtualIPTemplate.Spec.Type", networkingv1alpha1.VirtualIPTypePublic),
				HaveField("NetworkInterfaceSource.Ephemeral.NetworkInterfaceTemplate.Spec.VirtualIP.Ephemeral.VirtualIPTemplate.Spec.IPFamily", corev1.IPv4Protocol),
			))),
		))

		By("ensuring ignition secret is created and owned by bastion host machine")
		ignitionSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getIgnitionNameForMachine(bastionHostName),
				Namespace: ns.Name,
			},
		}
		Eventually(Object(ignitionSecret)).Should(SatisfyAll(
			HaveField("ObjectMeta.OwnerReferences", ContainElement(SatisfyAll(
				HaveField("Kind", "Machine"),
				HaveField("UID", bastionHost.UID),
				HaveField("Name", bastionHost.Name),
			))),
			HaveField("Data", HaveLen(1)),
		))

		By("patching bastion host with running state and network interfaces with private and virtual ip")
		machineBase := bastionHost.DeepCopy()
		bastionHost.Status.State = computev1alpha1.MachineStateRunning
		bastionHost.Status.NetworkInterfaces = []computev1alpha1.NetworkInterfaceStatus{{
			Name:      "primary",
			IPs:       []commonv1alpha1.IP{commonv1alpha1.MustParseIP("10.0.0.1")},
			VirtualIP: &commonv1alpha1.IP{Addr: netip.MustParseAddr("10.0.0.10")},
		}}
		Expect(k8sClient.Status().Patch(ctx, bastionHost, client.MergeFrom(machineBase))).To(Succeed())
		DeferCleanup(k8sClient.Delete, bastionHost)

		By("ensuring that bastion host is created and Running")
		Eventually(Object(bastionHost)).Should(SatisfyAll(
			HaveField("Status.State", computev1alpha1.MachineStateRunning),
		))

		By("ensuring that bastion host is updated with correct virtual/public ip")
		Eventually(Object(bastion)).Should(SatisfyAll(
			HaveField("Status.Ingress.IP", "10.0.0.10"),
		))
	})

	It("should validate and return an appropriate error when attempting to create a machine with an invalid bastion configuration", func() {
		By("checking for nil bastion config")
		err := validateConfiguration(nil)
		Expect(err).To(MatchError("bastionConfig must not be empty"))

		By("checking for missing Image in bastion config")
		bastionConfig := &controllerconfig.BastionConfig{
			MachineClassName: "foo",
			VolumeClassName:  "bar",
		}
		err = validateConfiguration(bastionConfig)
		Expect(err).To(MatchError("image is mandatory"))

		By("checking for missing MachineClassName in bastion config")
		bastionConfig = &controllerconfig.BastionConfig{
			Image:           "bar",
			VolumeClassName: "foo",
		}
		err = validateConfiguration(bastionConfig)
		Expect(err).To(MatchError("MachineClassName is mandatory"))

		By("checking for missing VolumeClassName in bastion config")
		bastionConfig = &controllerconfig.BastionConfig{
			Image:            "bar",
			MachineClassName: "foo",
		}
		err = validateConfiguration(bastionConfig)
		Expect(err).To(MatchError("VolumeClassName is mandatory"))

		By("checking for valid bastion config")
		bastionConfig = &controllerconfig.BastionConfig{
			MachineClassName: "foo",
			VolumeClassName:  "foo",
			Image:            "bar",
		}
		err = validateConfiguration(bastionConfig)
		Expect(err).To(BeNil())
	})
})
