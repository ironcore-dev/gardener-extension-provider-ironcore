// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerextensionv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gstruct "github.com/onsi/gomega/gstruct"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var _ = Describe("ConfigValidator", func() {
	ns := SetupTest()

	var (
		configValidator bastion.ConfigValidator
		cluster         *extensions.Cluster
	)

	BeforeEach(func() {
		logger := log.Log.WithName("test")
		configValidator = NewConfigValidator(k8sClient, logger)
	})

	It("should return error for an empty bastion config", func(ctx SpecContext) {
		errorList := configValidator.Validate(ctx, nil, cluster)
		Expect(errorList).To(ConsistOfFields(gstruct.Fields{
			"Detail": Equal("bastion can not be nil"),
		}))
	})

	It("should return error for an empty bastion userdata", func(ctx SpecContext) {
		bastion := &gardenerextensionv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},
			Spec: gardenerextensionv1alpha1.BastionSpec{
				UserData: []byte{},
			},
		}
		errorList := configValidator.Validate(ctx, bastion, cluster)
		Expect(errorList).To(ConsistOfFields(gstruct.Fields{
			"Detail": Equal("bastion spec userdata can not be empty"),
		}))
	})

	It("should return error for an empty bastion CIDR", func(ctx SpecContext) {
		bastion := &gardenerextensionv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},
			Spec: gardenerextensionv1alpha1.BastionSpec{
				UserData: []byte("foo"),
				Ingress: []gardenerextensionv1alpha1.BastionIngressPolicy{{
					IPBlock: networkingv1.IPBlock{
						CIDR: "",
					}},
				},
			},
		}
		errorList := configValidator.Validate(ctx, bastion, cluster)
		Expect(errorList).To(ConsistOfFields(gstruct.Fields{
			"Detail": Equal("bastion spec Ingress CIDR can not be empty"),
		}))
	})

	It("should return error for an invalid bastion spec CIDR", func(ctx SpecContext) {
		bastion := &gardenerextensionv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},

			Spec: gardenerextensionv1alpha1.BastionSpec{
				DefaultSpec: gardenerextensionv1alpha1.DefaultSpec{
					Type: ironcore.Type,
				},
				UserData: []byte("abcd"),
				Ingress: []gardenerextensionv1alpha1.BastionIngressPolicy{{
					IPBlock: networkingv1.IPBlock{
						CIDR: "213.69.151.260/24",
					}},
				},
			},
		}
		errorList := configValidator.Validate(ctx, bastion, cluster)
		Expect(errorList).To(ConsistOfFields(gstruct.Fields{
			"Detail": Equal("invalid bastion spec Ingress CIDR: invalid CIDR address: 213.69.151.260/24"),
		}))
	})

	It("should return error for an empty cluster", func(ctx SpecContext) {
		bastion := &gardenerextensionv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},

			Spec: gardenerextensionv1alpha1.BastionSpec{
				DefaultSpec: gardenerextensionv1alpha1.DefaultSpec{
					Type: ironcore.Type,
				},
				UserData: []byte("abcd"),
				Ingress: []gardenerextensionv1alpha1.BastionIngressPolicy{{
					IPBlock: networkingv1.IPBlock{
						CIDR: "213.69.151.246/24",
					}},
				},
			},
		}
		errorList := configValidator.Validate(ctx, bastion, nil)
		Expect(errorList).To(ConsistOfFields(gstruct.Fields{
			"Detail": Equal("cluster can not be nil"),
		}))
	})

	It("should return error for an empty cluster shoot", func(ctx SpecContext) {
		bastion := &gardenerextensionv1alpha1.Bastion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},

			Spec: gardenerextensionv1alpha1.BastionSpec{
				DefaultSpec: gardenerextensionv1alpha1.DefaultSpec{
					Type: ironcore.Type,
				},
				UserData: []byte("abcd"),
				Ingress: []gardenerextensionv1alpha1.BastionIngressPolicy{{
					IPBlock: networkingv1.IPBlock{
						CIDR: "213.69.151.246/24",
					}},
				},
			},
		}

		cluster := &extensions.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testns",
			},
			CloudProfile: &gardencorev1beta1.CloudProfile{},
			Shoot:        nil,
		}
		errorList := configValidator.Validate(ctx, bastion, cluster)
		Expect(errorList).To(ConsistOfFields(gstruct.Fields{
			"Detail": Equal("cluster shoot can not be empty"),
		}))
	})
})
