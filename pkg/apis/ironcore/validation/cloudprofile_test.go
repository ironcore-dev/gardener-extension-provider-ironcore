// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	apisironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
)

func InvalidField(fld string) types.GomegaMatcher {
	return SimpleMatchField(field.ErrorTypeInvalid, fld)
}

func SimpleMatchField(errorType field.ErrorType, fld string) types.GomegaMatcher {
	return HaveValue(MatchFields(IgnoreExtras, Fields{
		"Type":  Equal(errorType),
		"Field": Equal(fld),
	}))
}

var _ = Describe("CloudProfileConfig validation", func() {
	Describe("#ValidateCloudProfileConfig", func() {
		var (
			cloudProfileConfig  *apisironcore.CloudProfileConfig
			machineImages       []core.MachineImage
			nilPath             *field.Path
			machineImageName    string
			machineImageVersion string
		)

		BeforeEach(func() {
			machineImageName = "ubuntu"
			machineImageVersion = "1.2.3"
			cloudProfileConfig = &apisironcore.CloudProfileConfig{
				MachineImages: []apisironcore.MachineImages{
					{
						Name: machineImageName,
						Versions: []apisironcore.MachineImageVersion{
							{
								Version:      machineImageVersion,
								Image:        "registry/image:sha1234",
								Architecture: pointer.String("amd64"),
							},
						},
					},
				},
				StorageClasses: apisironcore.StorageClasses{
					Default: &apisironcore.StorageClass{
						Name: "default",
						Type: "defaultType",
					},
					Additional: []apisironcore.StorageClass{
						{
							Name: "foo",
							Type: "fooType",
						},
						{
							Name: "bar",
							Type: "barType",
						},
					},
				},
			}
			machineImages = []core.MachineImage{
				{
					Name: machineImageName,
					Versions: []core.MachineImageVersion{
						{
							ExpirableVersion: core.ExpirableVersion{Version: machineImageVersion},
						},
					},
				},
			}
		})

		Describe("machine image validation", func() {
			It("should pass validation", func() {
				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, nilPath)
				Expect(errorList).To(BeEmpty())
			})

			It("should not require a machine image mapping because no versions are configured", func() {
				machineImages = append(machineImages, core.MachineImage{
					Name:     "suse",
					Versions: nil,
				})
				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, nilPath)
				Expect(errorList).To(BeEmpty())
			})

			It("should require a machine image mapping to be configured", func() {
				machineImages = append(machineImages, core.MachineImage{
					Name: "suse",
					Versions: []core.MachineImageVersion{
						{
							ExpirableVersion: core.ExpirableVersion{
								Version: machineImageVersion,
							},
						},
					},
				})
				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, nilPath)
				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("machineImages"),
					})),
				))
			})

			It("should forbid unsupported machine image version configuration", func() {
				cloudProfileConfig.MachineImages[0].Versions[0].Image = ""
				cloudProfileConfig.MachineImages[0].Versions[0].Architecture = pointer.String("foo")
				machineImages[0].Versions = append(machineImages[0].Versions, core.MachineImageVersion{ExpirableVersion: core.ExpirableVersion{Version: "2.0.0"}})
				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, nilPath)

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("machineImages[0].versions"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("machineImages[0].versions[0].image"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal("machineImages[0].versions[0].architecture"),
					})),
				))
			})
		})

		DescribeTable("ValidateCloudProfileConfig StorageClass name",
			func(cpConfig *apisironcore.CloudProfileConfig, machineImages []core.MachineImage, fldPath *field.Path, match types.GomegaMatcher) {
				errList := ValidateCloudProfileConfig(cpConfig, machineImages, fldPath)
				Expect(errList).To(match)
			},
			Entry("invalid storageClass name in default StorageClass",
				&apisironcore.CloudProfileConfig{
					StorageClasses: apisironcore.StorageClasses{
						Default: &apisironcore.StorageClass{
							Name: "foo*",
							Type: "defaultType",
						},
					},
				},
				machineImages,
				nilPath,
				ContainElement(InvalidField("storageClasses.defaultStorageClasses.name")),
			),
			Entry("invalid storageClass name in additional storageClasses",
				&apisironcore.CloudProfileConfig{
					StorageClasses: apisironcore.StorageClasses{
						Additional: []apisironcore.StorageClass{
							{
								Name: "foo*",
								Type: "defaultType",
							},
						},
					},
				},
				machineImages,
				nilPath,
				ContainElement(InvalidField("storageClasses.additionalStorageClasses[0].name")),
			),
		)

	})
})
