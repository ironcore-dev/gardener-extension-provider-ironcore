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
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
)

var _ = Describe("Bastion Host Reconcile", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)

	It("should create a igntion secret and machine for a given bastion configuration", func() {

		By("getting the cluster object")
		cluster, err := extensionscontroller.GetCluster(ctx, k8sClient, ns.Name)
		Expect(err).NotTo(HaveOccurred())

		By("creating a bastion configuration")
		bastion := &extensionsv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-bastion",
				Annotations: map[string]string{
					constants.GardenerOperation: constants.GardenerOperationReconcile,
				},
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
		DeferCleanup(k8sClient.Delete, ctx, bastion)

		Eventually(Object(bastion)).Should(SatisfyAll(
			HaveField("Status.LastOperation.Type", gardencorev1beta1.LastOperationTypeCreate),
		))

		By("ensuring that machine is created with correct spec")
		machineName, err := generateBastionBaseResourceName(cluster.ObjectMeta.Name, bastion)
		Expect(err).ShouldNot(HaveOccurred())
		machine := &computev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      machineName,
			},
		}
		Eventually(Object(machine)).Should(SatisfyAll(
			HaveField("Spec.MachineClassRef", corev1.LocalObjectReference{
				Name: "my-machine-class",
			}),
			HaveField("Spec.Image", "my-image"),
			HaveField("Spec.IgnitionRef.Name", getIgnitionNameForMachine(machine.Name)),
			HaveField("Spec.Power", computev1alpha1.PowerOn),
			HaveField("Spec.NetworkInterfaces", ContainElement(SatisfyAll(
				HaveField("Name", "primary"),
				HaveField("NetworkInterfaceSource.Ephemeral.NetworkInterfaceTemplate.Spec.NetworkRef.Name", "my-network"),
			))),
		))

		By("ensuring ignition secret is created")
		ignitionSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getIgnitionNameForMachine(machineName),
				Namespace: ns.Name,
			},
		}
		Eventually(Get(ignitionSecret)).Should(Succeed())

		By("patching machine with running state and network interfaces with private and virtual ip")
		machineBase := machine.DeepCopy()
		machine.Status.State = computev1alpha1.MachineStateRunning
		machine.Status.NetworkInterfaces = []computev1alpha1.NetworkInterfaceStatus{{
			Name:      "primary",
			IPs:       []commonv1alpha1.IP{commonv1alpha1.MustParseIP("10.0.0.1")},
			VirtualIP: &commonv1alpha1.IP{Addr: netip.MustParseAddr("10.0.0.10")},
		}}
		Expect(k8sClient.Status().Patch(ctx, machine, client.MergeFrom(machineBase))).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, machine)

		By("ensuring that machine is created and Running")
		Eventually(Object(machine)).Should(SatisfyAll(
			HaveField("Status.State", computev1alpha1.MachineStateRunning),
		))

		By("ensuring that bastion host is updated with correct virtual/public ip")
		Eventually(Object(bastion)).Should(SatisfyAll(
			HaveField("Status.Ingress.IP", "10.0.0.10"),
		))

		By("error check for nil bastion config")
		err = bastionConfigCheck(nil)
		Expect(err).To(MatchError("bastionConfig must not be empty"))

		By("error check for no Image in bastion config")
		bastionConfig1 := &controllerconfig.BastionConfig{
			MachineClassName: "foo",
		}
		err = bastionConfigCheck(bastionConfig1)
		Expect(err).To(MatchError("bastion not supported as no Image is configured for the bastion host machine"))

		By("error check for no MachineClassName in bastion config")
		bastionConfig2 := &controllerconfig.BastionConfig{
			Image: "bar",
		}
		err = bastionConfigCheck(bastionConfig2)
		Expect(err).To(MatchError("bastion not supported as no flavor is configured for the bastion host machine"))

		By("check for bastion config")
		bastionConfig3 := &controllerconfig.BastionConfig{
			MachineClassName: "foo",
			Image:            "bar",
		}
		err = bastionConfigCheck(bastionConfig3)
		Expect(err).To(BeNil())

	})

})
