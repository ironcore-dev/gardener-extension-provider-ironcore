// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"context"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/admission/validator"
)

var _ = Describe("SecretBinding validator", func() {

	Describe("#Validate", func() {
		const (
			namespace = "garden-dev"
			name      = "my-provider-account"
		)

		var (
			ctx = context.TODO()

			//nolint:staticcheck // SA1019 ignore deprecated core.SecretBinding usage
			secretBinding = &core.SecretBinding{
				SecretRef: corev1.SecretReference{
					Name:      name,
					Namespace: namespace,
				},
			}

			scheme = func() *runtime.Scheme {
				s := runtime.NewScheme()
				Expect(corev1.AddToScheme(s)).To(Succeed())
				return s
			}()
		)

		newValidator := func(objects ...client.Object) extensionswebhook.Validator {
			apiReader := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			mgr := &test.FakeManager{APIReader: apiReader}
			return validator.NewSecretBindingValidator(mgr)
		}

		It("should return err when obj is not a SecretBinding", func() {
			err := newValidator().Validate(ctx, &corev1.Secret{}, nil)
			Expect(err).To(MatchError("wrong object type *v1.Secret"))
		})

		It("should return err when oldObj is not a SecretBinding", func() {
			//nolint:staticcheck // SA1019 ignore deprecated core.SecretBinding usage
			err := newValidator().Validate(ctx, &core.SecretBinding{}, &corev1.Secret{})
			Expect(err).To(MatchError("wrong object type *v1.Secret for old object"))
		})

		It("should return err if it fails to get the corresponding Secret", func() {
			err := newValidator().Validate(ctx, secretBinding, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should return err when the corresponding Secret is not valid", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data: map[string][]byte{
					"namespace": []byte("foo"),
				},
			}
			err := newValidator(secret).Validate(ctx, secretBinding, nil)
			Expect(err).To(MatchError("missing field: token in cloud provider secret"))
		})

		It("should return nil when the corresponding Secret is valid", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data: map[string][]byte{
					"namespace": []byte("default"),
					"token":     []byte("abcd"),
					"username":  []byte("admin"),
				},
			}
			Expect(newValidator(secret).Validate(ctx, secretBinding, nil)).To(Succeed())
		})
	})
})
