// Copyright 2023 OnMetal authors
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

package infrastructure

import (
	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Infrastructure Reconcile", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)

	It("should create a network, natgateway and prefix for a given infrastructure configuration", func() {
		By("creating a cluster resource")
		cluster := controller.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Namespace,
				Name:      "foo",
			},
			CloudProfile: nil,
			Seed:         nil,
			Shoot: &v1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      "foo",
				},
				Spec: v1beta1.ShootSpec{},
			},
		}

		By("creating an infrastructure configuration")
		infra := &extensionsv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-infra",
				Annotations: map[string]string{
					constants.GardenerOperation: constants.GardenerOperationReconcile,
				},
			},
			Spec: extensionsv1alpha1.InfrastructureSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           onmetal.Type,
					ProviderConfig: nil,
				},
				Region: "foo",
				SecretRef: corev1.SecretReference{
					Name: "my-infra-creds",
				},
				SSHPublicKey: nil,
			},
		}
		Expect(k8sClient.Create(ctx, infra)).Should(Succeed())

		network := &networkingv1alpha1.Network{}
		networkKey := client.ObjectKey{Namespace: ns.Name, Name: generateResourceNameFromCluster(&cluster)}
		By("expecting a network being created")
		Eventually(func(g Gomega) {
			err := k8sClient.Get(ctx, networkKey, network)
			Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())
	})
})
