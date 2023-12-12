// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Infrastructure Reconcile", func() {
	ns := SetupTest()

	It("should ensure that the network, natgateway and prefix is being deleted", func(ctx SpecContext) {
		By("getting the cluster object")
		cluster, err := extensionscontroller.GetCluster(ctx, k8sClient, ns.Name)
		Expect(err).NotTo(HaveOccurred())

		providerConfig := &v1alpha1.InfrastructureConfig{}

		By("creating an infrastructure configuration")
		infra := &extensionsv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-infra",
				Annotations: map[string]string{
					constants.GardenerOperation: constants.GardenerOperationReconcile,
				},
			},
			Spec: extensionsv1alpha1.InfrastructureSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           ironcore.Type,
					ProviderConfig: &runtime.RawExtension{Object: providerConfig},
				},
				Region: "foo",
			},
		}
		Expect(k8sClient.Create(ctx, infra)).Should(Succeed())

		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(infra), infra)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(infra.Status.LastOperation).NotTo(BeNil())
		}).Should(Succeed())

		By("expecting a network being created")
		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      generateResourceNameFromCluster(cluster),
			},
		}

		By("expecting a nat gateway being created")
		natGateway := &networkingv1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      generateResourceNameFromCluster(cluster),
			},
		}

		By("expecting a prefix being created")
		prefix := &ipamv1alpha1.Prefix{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      generateResourceNameFromCluster(cluster),
			},
		}

		By("deleting the infrastructure resource")
		Expect(k8sClient.Delete(ctx, infra)).Should(Succeed())

		By("waiting for the network to be gone")
		Eventually(Get(network)).Should(Satisfy(apierrors.IsNotFound))

		By("waiting for the natgateway to be gone")
		Eventually(Get(natGateway)).Should(Satisfy(apierrors.IsNotFound))

		By("waiting for the prefix to be gone")
		Eventually(Get(prefix)).Should(Satisfy(apierrors.IsNotFound))
	})
})
