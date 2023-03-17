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

	"github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/internal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	"github.com/onmetal/onmetal-api/api/common/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
)

var _ = Describe("Valueprovider Reconcile", func() {
	ctx := testutils.SetupContext()
	ns, vp := SetupTest(ctx)

	var (
		fakeClient         client.Client
		fakeSecretsManager secretsmanager.Interface
	)

	BeforeEach(func() {
		curDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(filepath.Join("..", "..", ".."))).To(Succeed())
		DeferCleanup(os.Chdir, curDir)

		fakeClient = fakeclient.NewClientBuilder().Build()
		fakeSecretsManager = fakesecretsmanager.New(fakeClient, ns.Name)
		Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-controller-manager-server", Namespace: ns.Name}})).To(Succeed())
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
			config := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      internal.CloudProviderSecretName,
				},
			}
			Eventually(Get(config)).Should(Succeed())
			Expect(config.Data).To(HaveKey("cloudprovider.conf"))
			cloudProviderConfig := map[string]interface{}{}
			Expect(yaml.Unmarshal([]byte(config.Data["cloudprovider.conf"]), &cloudProviderConfig)).NotTo(HaveOccurred())
			Expect(cloudProviderConfig["networkName"]).To(Equal("my-network"))
		})
	})

	Describe("#GetStorageClassesChartValues", func() {
		It("should return an empty config chart value map if not storageclasses are present in the cloudprofile", func() {
			providerCloudProfile := &apisonmetal.CloudProfileConfig{}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())

			cluster := &controller.Cluster{
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Raw: providerCloudProfileJson,
						},
					},
				},
			}

			values, err := vp.GetStorageClassesChartValues(ctx, nil, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				"storageClasses": []map[string]interface{}{},
			}))
		})

		It("should return correct config chart values if default and additional storage classes are present in the cloudprofile", func() {
			providerCloudProfile := &apisonmetal.CloudProfileConfig{
				StorageClasses: apisonmetal.StorageClasses{
					Default: &apisonmetal.StorageClass{
						Name: "foo",
						Type: "volumeTypeFoo",
					},
					Additional: []apisonmetal.StorageClass{
						{
							Name: "foo1",
							Type: "volumeTypeFoo1",
						},
						{
							Name: "foo2",
							Type: "volumeTypeFoo2",
						},
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())

			cluster := &controller.Cluster{
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Raw: providerCloudProfileJson,
						},
					},
				},
			}

			values, err := vp.GetStorageClassesChartValues(ctx, nil, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				"storageClasses": []map[string]interface{}{
					{
						"name":    "foo",
						"type":    "volumeTypeFoo",
						"default": true,
					},
					{
						"name": "foo1",
						"type": "volumeTypeFoo1",
					},
					{
						"name": "foo2",
						"type": "volumeTypeFoo2",
					},
				},
			}))
		})

		It("should return correct config chart values if only additional storage classes are present in the cloudprofile", func() {
			providerCloudProfile := &apisonmetal.CloudProfileConfig{
				StorageClasses: apisonmetal.StorageClasses{
					Additional: []apisonmetal.StorageClass{
						{
							Name: "foo1",
							Type: "volumeTypeFoo1",
						},
						{
							Name: "foo2",
							Type: "volumeTypeFoo2",
						},
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())

			cluster := &controller.Cluster{
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Raw: providerCloudProfileJson,
						},
					},
				},
			}

			values, err := vp.GetStorageClassesChartValues(ctx, nil, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				"storageClasses": []map[string]interface{}{
					{
						"name": "foo1",
						"type": "volumeTypeFoo1",
					},
					{
						"name": "foo2",
						"type": "volumeTypeFoo2",
					},
				},
			}))
		})
	})

	Describe("#GetControlPlaneShootCRDsChartValues", func() {
		It("should return correct config chart values", func() {
			values, err := vp.GetControlPlaneShootCRDsChartValues(ctx, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{}))
		})
	})

	Describe("#GetControlPlaneShootChartValues", func() {
		It("should return correct config chart values", func() {
			providerCloudProfile := &apisonmetal.CloudProfileConfig{
				StorageClasses: apisonmetal.StorageClasses{
					Default: &apisonmetal.StorageClass{
						Name: "foo",
						Type: "volumeTypeFoo",
					},
					Additional: []apisonmetal.StorageClass{
						{
							Name: "bar",
							Type: "volumeTypeBar",
						},
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())
			cluster := &controller.Cluster{
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Raw: providerCloudProfileJson,
						},
					},
				},
				Shoot: &gardencorev1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ns.Name,
						Name:      "my-shoot",
					},
					Spec: gardencorev1beta1.ShootSpec{
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.26.0",
							VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
								Enabled: true,
							},
						},
					},
				},
			}
			values, err := vp.GetControlPlaneShootChartValues(ctx, nil, cluster, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				"cloud-controller-manager": map[string]interface{}{
					"enabled": true,
				},
				"csi-driver-node": map[string]interface{}{
					"enabled":     true,
					"vpaEnabled":  true,
					"pspDisabled": true,
				},
			}))
		})
	})

	Describe("#GetControlPlaneChartValues", func() {
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
			providerCloudProfile := &apisonmetal.CloudProfileConfig{
				StorageClasses: apisonmetal.StorageClasses{
					Default: &apisonmetal.StorageClass{
						Name: "foo",
						Type: "volumeTypeFoo",
					},
					Additional: []apisonmetal.StorageClass{
						{
							Name: "bar",
							Type: "volumeTypeBar",
						},
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())
			cluster := &controller.Cluster{
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Raw: providerCloudProfileJson,
						},
					},
				},
				Shoot: &gardencorev1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ns.Name,
						Name:      "my-shoot",
					},
					Spec: gardencorev1beta1.ShootSpec{
						Networking: gardencorev1beta1.Networking{
							Pods: pointer.String("10.0.0.0/16"),
						},
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.26.0",
							VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
								Enabled: true,
							},
						},
					},
				},
			}

			checksums := map[string]string{
				onmetal.CloudProviderConfigName: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
			}
			values, err := vp.GetControlPlaneChartValues(ctx, cp, cluster, fakeSecretsManager, checksums, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{
				"global": map[string]interface{}{
					"genericTokenKubeconfigSecretName": "generic-token-kubeconfig",
				},
				"cloud-controller-manager": map[string]interface{}{
					"enabled":     true,
					"replicas":    1,
					"clusterName": ns.Name,
					"podAnnotations": map[string]interface{}{
						"checksum/secret-cloud-provider-config": "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
					},
					"podLabels": map[string]interface{}{
						"maintenance.gardener.cloud/restart": "true",
					},
					"tlsCipherSuites": []string{
						"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
						"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
						"TLS_AES_128_GCM_SHA256",
						"TLS_AES_256_GCM_SHA384",
						"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
						"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
						"TLS_CHACHA20_POLY1305_SHA256",
						"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
						"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
					},
					"secrets": map[string]interface{}{
						"server": "cloud-controller-manager-server",
					},
					"featureGates": map[string]bool{
						"CustomResourceValidation": true,
					},
					"podNetwork": "10.0.0.0/16",
				},
				"csi-driver-controller": map[string]interface{}{
					"enabled":  true,
					"replicas": 1,
				},
			}))
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
