// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
)

// ValidateInfrastructureConfig validates a InfrastructureConfig object.
func ValidateInfrastructureConfig(infra *apisironcore.InfrastructureConfig, nodesCIDR, podsCIDR, servicesCIDR *string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if infra.NetworkRef != nil {
		for _, msg := range apivalidation.NameIsDNSLabel(infra.NetworkRef.Name, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("networkRef").Child("name"), infra.NetworkRef.Name, msg))
		}
	}

	return allErrs
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *apisironcore.InfrastructureConfig, fldPath *field.Path) field.ErrorList {
	var (
		allErrs = field.ErrorList{}
	)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.NetworkRef, oldConfig.NetworkRef, fldPath.Child("networkRef"))...)

	return allErrs
}
