// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"net/netip"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore/api/core/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	controllerconfig "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/config"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
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
					Type:           ironcore.Type,
					ProviderConfig: nil,
				},
				UserData: []byte("abcd"),
				Ingress: []extensionsv1alpha1.BastionIngressPolicy{
					{IPBlock: networkingv1.IPBlock{
						CIDR: "10.0.0.0/24",
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
				Name:      getIgnitionNameForMachine(bastionHost.Name),
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

		By("ensuring bastion network policy is created with correct spec")
		networkPolicy := &networkingv1alpha1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bastionHost.Name,
				Namespace: ns.Name,
			},
		}
		Eventually(Object(networkPolicy)).Should(SatisfyAll(
			HaveField("Spec.NetworkRef.Name", "my-network"),
			HaveField("Spec.NetworkInterfaceSelector.MatchLabels", HaveKeyWithValue("bastion-host", bastionHost.Name)),
			HaveField("Spec.PolicyTypes", ContainElement(networkingv1alpha1.PolicyTypeIngress)),
			HaveField("Spec.Ingress", ContainElement(SatisfyAll(
				HaveField("Ports", ContainElement(SatisfyAll(
					HaveField("Port", int32(sshPort)),
				))),
				HaveField("From", ContainElement(SatisfyAll(
					HaveField("IPBlock.CIDR", commonv1alpha1.MustParseIPPrefix("10.0.0.0/24")),
				))),
			))),
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
		Expect(err).NotTo(HaveOccurred())
	})
})
