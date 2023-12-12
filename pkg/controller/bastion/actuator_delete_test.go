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
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var _ = Describe("Bastion Host Delete", func() {
	ns := SetupTest()

	It("should ensure that the bastion is deleted along with bastion host and ignition secret", func(ctx SpecContext) {
		By("getting the cluster object")
		cluster, err := extensionscontroller.GetCluster(ctx, k8sClient, ns.Name)
		Expect(err).NotTo(HaveOccurred())

		By("creating bastion resource")
		bastion := &extensionsv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-bastion",
				Namespace: ns.Name,
			},
			Spec: extensionsv1alpha1.BastionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           ironcore.Type,
					ProviderConfig: nil,
				},
				UserData: []byte("my-user"),
				Ingress: []extensionsv1alpha1.BastionIngressPolicy{
					{IPBlock: networkingv1.IPBlock{
						CIDR: "10.0.0.0/24",
					}},
				}},
		}
		Expect(k8sClient.Create(ctx, bastion)).Should(Succeed())
		Eventually(Object(bastion)).Should(SatisfyAll(
			HaveField("Status.LastOperation.Type", gardencorev1beta1.LastOperationTypeCreate),
		))

		By("ensuring bastion host is created")
		bastionHostName, err := generateBastionHostResourceName(cluster.ObjectMeta.Name, bastion)
		Expect(err).ShouldNot(HaveOccurred())
		bastionHost := &computev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      bastionHostName,
			},
		}
		Eventually(Get(bastionHost)).Should(Succeed())

		By("ensuring ignition secret is created and owned by bastion host machine")
		ignitionSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getIgnitionNameForMachine(bastionHostName),
				Namespace: ns.Name,
			},
		}
		Eventually(Object(ignitionSecret)).Should(SatisfyAll(
			HaveField("ObjectMeta.OwnerReferences", ContainElement(SatisfyAll(
				HaveField("Name", bastionHost.Name),
				HaveField("Kind", "Machine"),
				HaveField("UID", bastionHost.UID),
			))),
			HaveField("Data", HaveLen(1)),
		))

		By("ensuring network policy is created and owned by bastion host machine")
		networkPolicy := &networkingv1alpha1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bastionHostName,
				Namespace: ns.Name,
			},
		}
		Eventually(Object(networkPolicy)).Should(SatisfyAll(
			HaveField("Spec.NetworkRef.Name", "my-network"),
			HaveField("ObjectMeta.OwnerReferences", ContainElement(SatisfyAll(
				HaveField("Name", bastionHost.Name),
				HaveField("Kind", "Machine"),
				HaveField("UID", bastionHost.UID),
			))),
		))

		By("patching bastion host with Running state")
		bastionHostBase := bastionHost.DeepCopy()
		bastionHost.Status.State = computev1alpha1.MachineStateRunning
		bastionHost.Status.NetworkInterfaces = []computev1alpha1.NetworkInterfaceStatus{{
			Name:      "primary",
			IPs:       []commonv1alpha1.IP{commonv1alpha1.MustParseIP("10.0.0.3")},
			VirtualIP: &commonv1alpha1.IP{Addr: netip.MustParseAddr("10.0.0.4")},
		}}
		Expect(k8sClient.Status().Patch(ctx, bastionHost, client.MergeFrom(bastionHostBase))).To(Succeed())

		Expect(k8sClient.Delete(ctx, bastion)).Should(Succeed())

		By("waiting for the bastion to be gone")
		Eventually(Get(bastion)).Should(Satisfy(apierrors.IsNotFound))

		By("waiting for the bastion host to be gone")
		Eventually(Get(bastionHost)).Should(Satisfy(apierrors.IsNotFound))
	})
})
