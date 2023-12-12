// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
)

var _ = Describe("InfrastructureConfig validation", func() {
	var (
		infra       *apisironcore.InfrastructureConfig
		fldPath     *field.Path
		networkName = "test-network"
	)

	BeforeEach(func() {
		infra = &apisironcore.InfrastructureConfig{
			NetworkRef: &corev1.LocalObjectReference{
				Name: networkName,
			},
		}
	})

	Describe("#ValidateInfrastructureConfig", func() {
		It("should return no errors for a valid configuration", func() {
			Expect(ValidateInfrastructureConfig(infra, nil, nil, nil, fldPath)).To(BeEmpty())
		})

		It("should fail with invalid network reference", func() {
			infra.NetworkRef.Name = "my%network"

			errorList := ValidateInfrastructureConfig(infra, nil, nil, nil, fldPath)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("networkRef.name"),
				})),
			))
		})
	})

	Describe("#ValidateInfrastructureConfigUpdate", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateInfrastructureConfigUpdate(infra, infra, fldPath)).To(BeEmpty())
		})
	})
})
