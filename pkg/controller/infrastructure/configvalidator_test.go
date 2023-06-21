// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package infrastructure

import (
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	gardenerextensionv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
)

var _ = Describe("ConfigValidator", func() {
	ns := SetupTest()

	var (
		configValidator infrastructure.ConfigValidator
	)

	BeforeEach(func() {
		logger := log.Log.WithName("test")
		configValidator = NewConfigValidator(k8sClient, logger)
	})

	It("should pass on an empty infrastructure config", func(ctx SpecContext) {
		infra := &gardenerextensionv1alpha1.Infrastructure{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "infra",
			},
			Spec: gardenerextensionv1alpha1.InfrastructureSpec{},
		}

		errorList := configValidator.Validate(ctx, infra)
		Expect(errorList).To(BeEmpty())
	})

	It("should pass if the referenced network exists", func(ctx SpecContext) {
		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "network",
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		infra := &gardenerextensionv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "infra",
			},
			Spec: gardenerextensionv1alpha1.InfrastructureSpec{
				DefaultSpec: gardenerextensionv1alpha1.DefaultSpec{
					Type: onmetal.Type,
					ProviderConfig: &runtime.RawExtension{Object: &v1alpha1.InfrastructureConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha1.SchemeGroupVersion.String(),
							Kind:       "InfrastructureConfig",
						},
						NetworkRef: &corev1.LocalObjectReference{
							Name: "network",
						},
					}}},
			},
		}
		Expect(k8sClient.Create(ctx, infra)).Should(Succeed())

		errorList := configValidator.Validate(ctx, infra)
		Expect(errorList).NotTo(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeNotFound),
			"Field": Equal("networkRef"),
		}))
	})

	It("should not pass if the referenced network does not exists", func(ctx SpecContext) {
		infra := &gardenerextensionv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "infra",
			},
			Spec: gardenerextensionv1alpha1.InfrastructureSpec{
				DefaultSpec: gardenerextensionv1alpha1.DefaultSpec{
					Type: onmetal.Type,
					ProviderConfig: &runtime.RawExtension{Object: &v1alpha1.InfrastructureConfig{
						TypeMeta: metav1.TypeMeta{
							APIVersion: v1alpha1.SchemeGroupVersion.String(),
							Kind:       "InfrastructureConfig",
						},
						NetworkRef: &corev1.LocalObjectReference{
							Name: "non-existing-network",
						},
					}}},
			},
		}
		Expect(k8sClient.Create(ctx, infra)).Should(Succeed())

		errorList := configValidator.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeNotFound),
			"Field": Equal("networkRef"),
		}))
	})
})
