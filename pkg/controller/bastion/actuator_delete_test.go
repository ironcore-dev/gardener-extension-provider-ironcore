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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
)

var _ = Describe("Bastion Host Delete", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)

	It("should ensure that the bastion host is being deleted", func() {

		By("getting the cluster object")
		cluster, err := extensionscontroller.GetCluster(ctx, k8sClient, ns.Name)
		Expect(err).NotTo(HaveOccurred())

		By("creating bastion host")
		bastionHost := &extensionsv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-bastion-host",
				Namespace: ns.Name,
			},
			Spec: extensionsv1alpha1.BastionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           onmetal.Type,
					ProviderConfig: nil,
				},
				UserData: []byte("my-user"),
				Ingress:  []extensionsv1alpha1.BastionIngressPolicy{},
			},
		}
		Expect(k8sClient.Create(ctx, bastionHost)).Should(Succeed())
		Eventually(Object(bastionHost)).Should(SatisfyAll(
			HaveField("Status.LastOperation.Type", gardencorev1beta1.LastOperationTypeCreate),
		))

		By("generating bastion machine name")
		machineName, err := generateBastionBaseResourceName(cluster.ObjectMeta.Name, bastionHost)
		Expect(err).ShouldNot(HaveOccurred())

		By("creating ignition secret object")
		ignitionSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getIgnitionNameForMachine(machineName),
				Namespace: ns.Name,
			},
		}

		By("creating bastion host machine object")
		bastionMachine := &computev1alpha1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      machineName,
				Namespace: ns.Name,
			},
		}

		By("ensuring bastion host, bastion machine and ignition secret")
		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(bastionHost), bastionHost)
			g.Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(bastionMachine), bastionMachine)
			g.Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ignitionSecret), ignitionSecret)
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())

		By("patching machine with running state")
		machineBase := bastionMachine.DeepCopy()
		bastionMachine.Status.State = computev1alpha1.MachineStateRunning
		bastionMachine.Status.NetworkInterfaces = []computev1alpha1.NetworkInterfaceStatus{{
			Name:      "primary",
			IPs:       []commonv1alpha1.IP{commonv1alpha1.MustParseIP("10.0.0.3")},
			VirtualIP: &commonv1alpha1.IP{Addr: netip.MustParseAddr("10.0.0.4")},
		}}
		Expect(k8sClient.Status().Patch(ctx, bastionMachine, client.MergeFrom(machineBase))).To(Succeed())

		By("ensuring that machine is in Running state")
		Eventually(Object(bastionMachine)).Should(SatisfyAll(
			HaveField("Status.State", computev1alpha1.MachineStateRunning),
		))

		By("deleting bastion host machine")
		Expect(k8sClient.Delete(ctx, bastionHost)).Should(Succeed())
		Eventually(Object(bastionHost)).Should(SatisfyAll(
			HaveField("Status.LastOperation.Type", gardencorev1beta1.LastOperationTypeDelete),
		))

		By("ensure bastion host to be gone")
		Eventually(Get(bastionHost)).Should(BeNil())

		By("ensure bastion host machine to be gone")
		Eventually(Get(bastionMachine)).Should(BeNil())

		By("ensure ignition secret to be gone")
		Eventually(Get(ignitionSecret)).Should(BeNil())
	})
})
