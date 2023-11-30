// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

	corev1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

// ValidateCloudProviderSecret checks whether the given secret contains a valid ironcore service account.
func ValidateCloudProviderSecret(secret *corev1.Secret) error {
	if _, ok := secret.Data[ironcore.TokenFieldName]; !ok {
		return fmt.Errorf("missing field: %s in cloud provider secret", ironcore.TokenFieldName)
	}
	namespace, ok := secret.Data[ironcore.NamespaceFieldName]
	if !ok {
		return fmt.Errorf("missing field: %s in cloud provider secret", ironcore.NamespaceFieldName)
	}
	if _, ok := secret.Data[ironcore.UsernameFieldName]; !ok {
		return fmt.Errorf("missing field: %s in cloud provider secret", ironcore.UsernameFieldName)
	}
	errs := apivalidation.ValidateNamespaceName(string(namespace), false)
	if len(errs) > 0 {
		return fmt.Errorf("invalid field: %s in cloud provider secret", ironcore.NamespaceFieldName)
	}

	return nil
}
