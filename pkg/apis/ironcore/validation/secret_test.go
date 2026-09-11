// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Secret validation", func() {
	DescribeTable("#ValidateCloudProviderSecret",
		func(data map[string][]byte, matcher gomegatypes.GomegaMatcher) {
			secret := &corev1.Secret{
				Data: data,
			}
			err := ValidateCloudProviderSecret(secret)
			Expect(err).To(matcher)
		},
		Entry("should return error when the token field is missing",
			map[string][]byte{
				"namespace": []byte("foo"),
				"username":  []byte("admin"),
			}, HaveOccurred()),
		Entry("should return error when the namespace field is missing",
			map[string][]byte{
				"token":    []byte("foo"),
				"username": []byte("admin"),
			}, HaveOccurred()),
		Entry("should return an error when the namespace name is invalid",
			map[string][]byte{
				"namespace": []byte("%foo"),
				"token":     []byte("foo"),
				"username":  []byte("admin"),
			}, HaveOccurred()),
		Entry("should return an error when the username is missing",
			map[string][]byte{
				"namespace": []byte("foo"),
				"token":     []byte("bar"),
			}, HaveOccurred()),
		Entry("should return no error if the secret is valid",
			map[string][]byte{
				"namespace": []byte("foo"),
				"token":     []byte("foo"),
				"username":  []byte("admin"),
			},
			Not(HaveOccurred())),
	)
})
