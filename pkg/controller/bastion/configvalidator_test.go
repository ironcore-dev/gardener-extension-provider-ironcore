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

package bastion

import (
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	gardenerextensionv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gstruct "github.com/onsi/gomega/gstruct"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("ConfigValidator", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)

	var (
		configValidator bastion.ConfigValidator
		cluster         *extensions.Cluster
	)

	BeforeEach(func() {
		logger := log.Log.WithName("test")
		configValidator = NewConfigValidator(k8sClient, logger)
	})

	It("should pass on an empty bastion config", func() {
		bastion := &gardenerextensionv1alpha1.Bastion{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},
			Spec: gardenerextensionv1alpha1.BastionSpec{},
		}

		errorList := configValidator.Validate(ctx, bastion, cluster)
		Expect(errorList).To(BeEmpty())
	})

	It("should pass on an empty bastion userdata", func() {
		bastion := &gardenerextensionv1alpha1.Bastion{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},
			Spec: gardenerextensionv1alpha1.BastionSpec{
				UserData: []byte{},
			},
		}

		errorList := configValidator.Validate(ctx, bastion, cluster)
		Expect(errorList).To(BeEmpty())
	})

	It("should pass on an empty bastion CIDR", func() {
		bastion := &gardenerextensionv1alpha1.Bastion{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},
			Spec: gardenerextensionv1alpha1.BastionSpec{
				Ingress: []gardenerextensionv1alpha1.BastionIngressPolicy{{
					IPBlock: networkingv1.IPBlock{
						CIDR: "",
					}},
				},
			},
		}

		errorList := configValidator.Validate(ctx, bastion, cluster)
		Expect(errorList).To(BeEmpty())
	})

	It("should pass on an invalid bastion spec CIDR", func() {
		bastion := &gardenerextensionv1alpha1.Bastion{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "bastion",
			},

			Spec: gardenerextensionv1alpha1.BastionSpec{
				DefaultSpec: gardenerextensionv1alpha1.DefaultSpec{
					Type: onmetal.Type,
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
			"Detail": Equal("invalid CIDR address: 213.69.151.260/24"),
		}))

	})

})
