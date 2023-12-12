// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
)

var _ = Describe("ControlPlaneConfig validation", func() {
	var (
		controlPlane *apisironcore.ControlPlaneConfig
		fldPath      *field.Path
	)

	BeforeEach(func() {
		controlPlane = &apisironcore.ControlPlaneConfig{}
	})

	Describe("#ValidateControlPlaneConfig", func() {
		It("should return no errors for a valid configuration", func() {
			Expect(ValidateControlPlaneConfig(controlPlane, "", fldPath)).To(BeEmpty())
		})

		It("should fail with invalid CCM feature gates", func() {
			controlPlane.CloudControllerManager = &apisironcore.CloudControllerManagerConfig{
				FeatureGates: map[string]bool{
					"AnyVolumeDataSource":      true,
					"CustomResourceValidation": true,
					"Foo":                      true,
				},
			}

			errorList := ValidateControlPlaneConfig(controlPlane, "1.18.14", fldPath)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("cloudControllerManager.featureGates.CustomResourceValidation"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("cloudControllerManager.featureGates.Foo"),
				})),
			))
		})
	})

	Describe("#ValidateControlPlaneConfigUpdate", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateControlPlaneConfigUpdate(controlPlane, controlPlane, fldPath)).To(BeEmpty())
		})
	})
})
