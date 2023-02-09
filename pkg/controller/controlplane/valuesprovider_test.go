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

package controlplane

import (
	"os"
	"path/filepath"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/internal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	"github.com/onmetal/onmetal-api/api/common/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Valueprovider Reconcile", func() {
	ctx := testutils.SetupContext()
	ns, kubeconfig := SetupTest(ctx)

	BeforeEach(func() {
		curDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(filepath.Join("..", "..", ".."))).To(Succeed())
		DeferCleanup(os.Chdir, curDir)
	})

	Describe("#GetConfigChartValues", func() {
		It("should return correct config chart values", func() {
			cp := &extensionsv1alpha1.ControlPlane{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "control-plane",
					Namespace: ns.Name,
				},
				Spec: extensionsv1alpha1.ControlPlaneSpec{
					Region: "foo",
					SecretRef: corev1.SecretReference{
						Name:      "my-infra-creds",
						Namespace: ns.Name,
					},
					DefaultSpec: extensionsv1alpha1.DefaultSpec{
						Type: onmetal.Type,
						ProviderConfig: &runtime.RawExtension{
							Raw: encode(&apisonmetal.ControlPlaneConfig{
								CloudControllerManager: &apisonmetal.CloudControllerManagerConfig{
									FeatureGates: map[string]bool{
										"CustomResourceValidation": true,
									},
								},
							}),
						},
					},
					InfrastructureProviderStatus: &runtime.RawExtension{
						Raw: encode(&apisonmetal.InfrastructureStatus{
							NetworkRef: v1alpha1.LocalUIDReference{
								Name: "my-network",
								UID:  "1234",
							},
						}),
					},
				},
			}

			Expect(k8sClient.Create(ctx, cp)).To(Succeed())

			By("ensuring that the provider secret has been created")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      internal.CloudProviderSecretName,
				},
			}
			Eventually(Object(secret)).Should(SatisfyAll(
				HaveField("Data", HaveKeyWithValue(onmetal.NamespaceFieldName, []byte(ns.Name))),
				HaveField("Data", HaveKeyWithValue(onmetal.KubeConfigFieldName, *kubeconfig))),
			)
		})
	})

})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
