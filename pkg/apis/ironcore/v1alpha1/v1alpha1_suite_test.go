// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV1alpha1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API V1Alpha1 Suite")
}
