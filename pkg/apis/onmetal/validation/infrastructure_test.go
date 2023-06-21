// Copyright 2022 OnMetal authors
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

package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
)

var _ = Describe("InfrastructureConfig validation", func() {
	var (
		infra       *apisonmetal.InfrastructureConfig
		fldPath     *field.Path
		networkName = "test-network"
	)

	BeforeEach(func() {
		infra = &apisonmetal.InfrastructureConfig{
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
