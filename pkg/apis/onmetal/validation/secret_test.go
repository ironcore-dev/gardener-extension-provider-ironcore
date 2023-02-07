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
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
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
			map[string][]byte{}, HaveOccurred()),
		Entry("should return error when the namespace field is missing",
			map[string][]byte{}, HaveOccurred()),
		Entry("should return an error when the namespace name is invalid",
			map[string][]byte{onmetal.NamespaceFieldName: []byte("%foo")},
			HaveOccurred()),
		Entry("should return error when the namespace name is valid",
			map[string][]byte{
				onmetal.NamespaceFieldName:  []byte("foo"),
				onmetal.KubeConfigFieldName: []byte("foo"),
			},
			Not(HaveOccurred())),
	)
})
