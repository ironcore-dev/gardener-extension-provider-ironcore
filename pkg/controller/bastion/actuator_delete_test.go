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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
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
					Type:           onmetal.Type,
					ProviderConfig: nil,
				},
				UserData: []byte("my-user"),
				Ingress:  []extensionsv1alpha1.BastionIngressPolicy{},
			},
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

		By("patching bastion host with Running state")
		bastionHostBase := bastionHost.DeepCopy()
		bastionHost.Status.State = computev1alpha1.MachineStateRunning
		bastionHost.Status.NetworkInterfaces = []computev1alpha1.NetworkInterfaceStatus{{
			Name:      "primary",
			IPs:       []commonv1alpha1.IP{commonv1alpha1.MustParseIP("10.0.0.3")},
			VirtualIP: &commonv1alpha1.IP{Addr: netip.MustParseAddr("10.0.0.4")},
		}}
		Expect(k8sClient.Status().Patch(ctx, bastionHost, client.MergeFrom(bastionHostBase))).To(Succeed())

		Expect(k8sClient.Delete(ctx, bastion)).To(Succeed())

		By("waiting for the bastion to be gone")
		Eventually(Get(bastion)).Should(Satisfy(apierrors.IsNotFound))

		By("waiting for the bastion host to be gone")
		Eventually(Get(bastionHost)).Should(Satisfy(apierrors.IsNotFound))
	})
})
