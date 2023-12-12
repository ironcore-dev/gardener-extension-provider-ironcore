// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"encoding/json"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var _ = Describe("Infrastructure Reconcile", func() {
	ns := SetupTest()

	It("should create a network, natgateway and prefix for a given infrastructure configuration", func(ctx SpecContext) {
		By("getting the cluster object")
		cluster, err := extensionscontroller.GetCluster(ctx, k8sClient, ns.Name)
		Expect(err).NotTo(HaveOccurred())

		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-network",
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("creating an infrastructure configuration")
		infra := &extensionsv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-infra-with-network",
				Annotations: map[string]string{
					constants.GardenerOperation: constants.GardenerOperationReconcile,
				},
			},
			Spec: extensionsv1alpha1.InfrastructureSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type: ironcore.Type,
					ProviderConfig: &runtime.RawExtension{Object: &v1alpha1.InfrastructureConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha1.SchemeGroupVersion.String(),
							Kind:       "InfrastructureConfig",
						},
						NetworkRef: &corev1.LocalObjectReference{
							Name: "my-network",
						},
					}},
				},
				Region: "foo",
				SecretRef: corev1.SecretReference{
					Namespace: ns.Name,
					Name:      "my-infra-creds",
				},
			},
		}
		Expect(k8sClient.Create(ctx, infra)).Should(Succeed())

		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(infra), infra)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(infra.Status.LastOperation).NotTo(BeNil())
		}).Should(Succeed())

		Eventually(Object(network)).Should(SatisfyAll(
			HaveField("ObjectMeta.Namespace", ns.Name),
			HaveField("ObjectMeta.Name", "my-network"),
		))

		By("expecting a nat gateway being created")
		natGateway := &networkingv1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      generateResourceNameFromCluster(cluster),
			},
		}

		Eventually(Object(natGateway)).Should(SatisfyAll(
			HaveField("Spec.Type", networkingv1alpha1.NATGatewayTypePublic),
			HaveField("Spec.IPFamily", corev1.IPv4Protocol),
			HaveField("Spec.NetworkRef", corev1.LocalObjectReference{
				Name: network.Name,
			}),
		))

		By("expecting a prefix being created")
		prefix := &ipamv1alpha1.Prefix{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      generateResourceNameFromCluster(cluster),
			},
		}

		Eventually(Object(prefix)).Should(SatisfyAll(
			HaveField("Spec.IPFamily", corev1.IPv4Protocol),
			HaveField("Spec.Prefix", commonv1alpha1.MustParseNewIPPrefix("10.0.0.0/24")),
		))

		By("ensuring that the infrastructure state contains the correct refs")
		providerStatus := map[string]interface{}{
			"apiVersion": "ironcore.provider.extensions.gardener.cloud/v1alpha1",
			"kind":       "InfrastructureStatus",
			"networkRef": map[string]interface{}{
				"name": network.Name,
				"uid":  network.UID,
			},
			"natGatewayRef": map[string]interface{}{
				"name": natGateway.Name,
				"uid":  natGateway.UID,
			},
			"prefixRef": map[string]interface{}{
				"name": prefix.Name,
				"uid":  prefix.UID,
			},
		}
		providerStatusJSON, err := json.Marshal(providerStatus)
		Expect(err).NotTo(HaveOccurred())
		Eventually(Object(infra)).Should(SatisfyAll(
			HaveField("Status.ProviderStatus", &runtime.RawExtension{Raw: providerStatusJSON}),
		))
	})

	It("should create a network, natgateway and prefix for a given infrastructure configuration", func(ctx SpecContext) {
		By("getting the cluster object")
		cluster, err := extensionscontroller.GetCluster(ctx, k8sClient, ns.Name)
		Expect(err).NotTo(HaveOccurred())

		By("creating an infrastructure configuration")
		infra := &extensionsv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-infra-without-network",
				Annotations: map[string]string{
					constants.GardenerOperation: constants.GardenerOperationReconcile,
				},
			},
			Spec: extensionsv1alpha1.InfrastructureSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           ironcore.Type,
					ProviderConfig: &runtime.RawExtension{Object: &v1alpha1.InfrastructureConfig{}},
				},
				Region: "foo",
				SecretRef: corev1.SecretReference{
					Namespace: ns.Name,
					Name:      "my-infra-creds",
				},
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

		Eventually(Object(network)).Should(SatisfyAll(
			HaveField("ObjectMeta.Namespace", ns.Name),
			HaveField("ObjectMeta.Name", generateResourceNameFromCluster(cluster)),
		))

		By("expecting a nat gateway being created")
		natGateway := &networkingv1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      generateResourceNameFromCluster(cluster),
			},
		}

		Eventually(Object(natGateway)).Should(SatisfyAll(
			HaveField("Spec.Type", networkingv1alpha1.NATGatewayTypePublic),
			HaveField("Spec.IPFamily", corev1.IPv4Protocol),
			HaveField("Spec.NetworkRef", corev1.LocalObjectReference{
				Name: network.Name,
			}),
		))

		By("expecting a prefix being created")
		prefix := &ipamv1alpha1.Prefix{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      generateResourceNameFromCluster(cluster),
			},
		}

		Eventually(Object(prefix)).Should(SatisfyAll(
			HaveField("Spec.IPFamily", corev1.IPv4Protocol),
			HaveField("Spec.Prefix", commonv1alpha1.MustParseNewIPPrefix("10.0.0.0/24")),
		))

		By("ensuring that the infrastructure state contains the correct refs")
		providerStatus := map[string]interface{}{
			"apiVersion": "ironcore.provider.extensions.gardener.cloud/v1alpha1",
			"kind":       "InfrastructureStatus",
			"networkRef": map[string]interface{}{
				"name": network.Name,
				"uid":  network.UID,
			},
			"natGatewayRef": map[string]interface{}{
				"name": natGateway.Name,
				"uid":  natGateway.UID,
			},
			"prefixRef": map[string]interface{}{
				"name": prefix.Name,
				"uid":  prefix.UID,
			},
		}
		providerStatusJSON, err := json.Marshal(providerStatus)
		Expect(err).NotTo(HaveOccurred())
		Eventually(Object(infra)).Should(SatisfyAll(
			HaveField("Status.ProviderStatus", &runtime.RawExtension{Raw: providerStatusJSON}),
		))
	})
})
