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
	"fmt"

	gardenercore "github.com/gardener/gardener/pkg/apis/core"
	gardenercorehelper "github.com/gardener/gardener/pkg/apis/core/helper"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"

	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
)

// ValidateCloudProfileConfig validates a CloudProfileConfig object.
func ValidateCloudProfileConfig(cpConfig *apisonmetal.CloudProfileConfig, machineImages []gardenercore.MachineImage, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	machineImagesPath := fldPath.Child("machineImages")

	for _, image := range machineImages {
		var processed bool
		for i, imageConfig := range cpConfig.MachineImages {
			if image.Name == imageConfig.Name {
				allErrs = append(allErrs, validateVersions(imageConfig.Versions, gardenercorehelper.ToExpirableVersions(image.Versions), machineImagesPath.Index(i).Child("versions"))...)
				processed = true
				break
			}
		}
		if !processed && len(image.Versions) > 0 {
			allErrs = append(allErrs, field.Required(machineImagesPath, fmt.Sprintf("must provide an image mapping for image %q", image.Name)))
		}
	}

	if cpConfig.StorageClasses.Default != nil {
		for _, msg := range apivalidation.NameIsDNSLabel(cpConfig.StorageClasses.Default.Name, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("storageClasses").Child("defaultStorageClasses").Child("name"), cpConfig.StorageClasses.Default.Name, msg))
		}
	}

	for i, sc := range cpConfig.StorageClasses.Additional {
		for _, msg := range apivalidation.NameIsDNSLabel(sc.Name, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("storageClasses").Child("additionalStorageClasses").Index(i).Child("name"), sc.Name, msg))
		}
	}

	return allErrs
}

func validateVersions(versionsConfig []apisonmetal.MachineImageVersion, versions []gardenercore.ExpirableVersion, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, version := range versions {
		var processed bool
		for j, versionConfig := range versionsConfig {
			jdxPath := fldPath.Index(j)
			if version.Version == versionConfig.Version {
				if len(versionConfig.Image) == 0 {
					allErrs = append(allErrs, field.Required(jdxPath.Child("image"), "must provide an image"))
				}
				if !slices.Contains(v1beta1constants.ValidArchitectures, *versionConfig.Architecture) {
					allErrs = append(allErrs, field.NotSupported(jdxPath.Child("architecture"), *versionConfig.Architecture, v1beta1constants.ValidArchitectures))
				}
				processed = true
				break
			}
		}
		if !processed {
			allErrs = append(allErrs, field.Required(fldPath, fmt.Sprintf("must provide an image mapping for version %q", version.Version)))
		}
	}

	return allErrs
}
