// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ironcorevalidation "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/validation"
)

type secret struct{}

// NewSecretValidator returns a new instance of a secret validator.
func NewSecretValidator() extensionswebhook.Validator {
	return &secret{}
}

// Validate checks whether the given new secret contains a valid ironcore service account.
func (s *secret) Validate(_ context.Context, newObj, oldObj client.Object) error {
	secret, ok := newObj.(*corev1.Secret)
	if !ok {
		return fmt.Errorf("wrong object type %T", newObj)
	}

	if oldObj != nil {
		oldSecret, ok := oldObj.(*corev1.Secret)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", oldObj)
		}

		if equality.Semantic.DeepEqual(secret.Data, oldSecret.Data) {
			return nil
		}
	}

	return ironcorevalidation.ValidateCloudProviderSecret(secret)
}
