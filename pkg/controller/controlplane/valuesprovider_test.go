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

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/internal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	"github.com/onmetal/onmetal-api/api/common/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
)

var _ = Describe("Valueprovider Reconcile", func() {
	ctx := testutils.SetupContext()
	ns, kubeconfig, valueProvider := SetupTest(ctx)

	var (
		fakeClient         client.Client
		fakeSecretsManager = fakesecretsmanager.New(fakeClient, ns.Name)
		enabledTrue        = map[string]interface{}{"enabled": true}
	)

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
	Describe("#GetControlPlaneShootChartValues", func() {
		cp := &extensionsv1alpha1.ControlPlane{}

		It("should return correct control plane shoot chart values when PodSecurityPolicy admission plugin is not disabled in the shoot", func() {

			cluster := &extensionscontroller.Cluster{
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.24.0",
							VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
								Enabled: true,
							},
							KubeAPIServer: &gardencorev1beta1.KubeAPIServerConfig{
								AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
									{
										Name: "PodSecurityPolicy",
									},
								},
							},
						},
					},
				},
			}

			values, err := valueProvider.GetControlPlaneShootChartValues(ctx, cp, cluster, fakeSecretsManager, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				onmetal.CloudControllerManagerName: enabledTrue,
				onmetal.CSINodeName: utils.MergeMaps(enabledTrue, map[string]interface{}{
					"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
					"vpaEnabled":        true,
					"pspDisabled":       false,
				}),
			}))
		})

		It("should return correct control plane shoot chart values when PodSecurityPolicy admission plugin is disabled in the shoot", func() {

			cluster := &extensionscontroller.Cluster{
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.24.0",
							VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
								Enabled: true,
							},
							KubeAPIServer: &gardencorev1beta1.KubeAPIServerConfig{
								AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
									{
										Name:     "PodSecurityPolicy",
										Disabled: pointer.Bool(true),
									},
								},
							},
						},
					},
				},
			}

			values, err := valueProvider.GetControlPlaneShootChartValues(ctx, cp, cluster, fakeSecretsManager, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				onmetal.CloudControllerManagerName: enabledTrue,
				onmetal.CSINodeName: utils.MergeMaps(enabledTrue, map[string]interface{}{
					"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
					"vpaEnabled":        true,
					"pspDisabled":       true,
				}),
			}))
		})

		It("should return correct control plane shoot chart values when VerticalPodAutoscaler and PodSecurityPolicy admission plugin are disabled in the shoot", func() {

			cluster := &extensionscontroller.Cluster{
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.24.0",
							VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
								Enabled: false,
							},
							KubeAPIServer: &gardencorev1beta1.KubeAPIServerConfig{
								AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
									{
										Name: "PodSecurityPolicy",
									},
								},
							},
						},
					},
				},
			}

			values, err := valueProvider.GetControlPlaneShootChartValues(ctx, cp, cluster, fakeSecretsManager, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				onmetal.CloudControllerManagerName: enabledTrue,
				onmetal.CSINodeName: utils.MergeMaps(enabledTrue, map[string]interface{}{
					"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
					"vpaEnabled":        false,
					"pspDisabled":       false,
				}),
			}))
		})

	})

	Describe("#GetStorageClassesChartValues", func() {
		It("should return correct storage class chart values", func() {
			cp := &extensionsv1alpha1.ControlPlane{}
			storageClasses := []apisonmetal.VolumeClassDefinition{{
				Name:             "testStorage",
				StorageClassName: pointer.String("testStorageClass"),
			},
			}
			cloudProfileConfig := &apisonmetal.CloudProfileConfig{
				VolumeClasses: storageClasses,
			}

			cloudProfileConfigJSON, _ := json.Marshal(cloudProfileConfig)
			cluster := &extensionscontroller.Cluster{
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Raw: cloudProfileConfigJSON,
						},
					},
				},
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.24.0",
							VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
								Enabled: true,
							},
							KubeAPIServer: &gardencorev1beta1.KubeAPIServerConfig{
								AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
									{
										Name: "PodSecurityPolicy",
									},
								},
							},
						},
					},
				},
			}

			decoder := serializer.NewCodecFactory(k8sClient.Scheme(), serializer.EnableStrict).UniversalDecoder()
			common.NewClientContext(k8sClient, k8sClient.Scheme(), decoder)
			clientContext := common.NewClientContext(k8sClient, k8sClient.Scheme(), decoder)
			valueProvider.ClientContext = clientContext

			values, err := valueProvider.GetStorageClassesChartValues(ctx, cp, cluster)
			Expect(err).NotTo(HaveOccurred())

			Expect(values["volumeClasses"]).To(HaveLen(1))
			Expect(values["volumeClasses"].([]map[string]interface{})[0]["name"]).To(Equal("testStorage"))
			Expect(values["volumeClasses"].([]map[string]interface{})[0]["storageClassName"]).To(Equal(pointer.String("testStorageClass")))
		})
	})

})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
