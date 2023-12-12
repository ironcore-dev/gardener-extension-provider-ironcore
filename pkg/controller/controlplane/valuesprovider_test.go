// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"os"
	"path/filepath"

	"github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore/api/core/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	apisironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/internal"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

var _ = Describe("Valueprovider Reconcile", func() {
	ns, vp, cluster := SetupTest()

	var (
		fakeClient         client.Client
		fakeSecretsManager secretsmanager.Interface
	)

	BeforeEach(func(ctx SpecContext) {
		curDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(filepath.Join("..", "..", ".."))).To(Succeed())
		DeferCleanup(os.Chdir, curDir)

		fakeClient = fakeclient.NewClientBuilder().Build()
		fakeSecretsManager = fakesecretsmanager.New(fakeClient, ns.Name)
		Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-controller-manager-server", Namespace: ns.Name}})).To(Succeed())
	})

	Describe("#GetConfigChartValues", func() {
		It("should return correct config chart values", func(ctx SpecContext) {
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
						Type: ironcore.Type,
						ProviderConfig: &runtime.RawExtension{
							Raw: encode(&apisironcore.ControlPlaneConfig{
								CloudControllerManager: &apisironcore.CloudControllerManagerConfig{
									FeatureGates: map[string]bool{
										"CustomResourceValidation": true,
									},
								},
							}),
						},
					},
					InfrastructureProviderStatus: &runtime.RawExtension{
						Raw: encode(&apisironcore.InfrastructureStatus{
							NetworkRef: v1alpha1.LocalUIDReference{
								Name: "my-network",
								UID:  "1234",
							},
							PrefixRef: v1alpha1.LocalUIDReference{
								Name: "my-prefix",
								UID:  "6789",
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
			Expect(cloudProviderConfig["prefixName"]).To(Equal("my-prefix"))
			Expect(cloudProviderConfig["clusterName"]).To(Equal(cluster.Name))
		})
	})

	Describe("#GetStorageClassesChartValues", func() {
		BeforeEach(func(ctx SpecContext) {
			By("creating an expand only VolumeClass")
			volumeClassExpandOnly := &storagev1alpha1.VolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-expandable",
				},
				Capabilities: corev1alpha1.ResourceList{
					corev1alpha1.ResourceIOPS: resource.MustParse("100"),
					corev1alpha1.ResourceTPS:  resource.MustParse("100"),
				},
				ResizePolicy: storagev1alpha1.ResizePolicyExpandOnly,
			}
			Expect(k8sClient.Create(ctx, volumeClassExpandOnly)).To(Succeed())
			DeferCleanup(k8sClient.Delete, volumeClassExpandOnly)

			By("creating an static VolumeClass")
			volumeClassStatic := &storagev1alpha1.VolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-static",
				},
				Capabilities: corev1alpha1.ResourceList{
					corev1alpha1.ResourceIOPS: resource.MustParse("100"),
					corev1alpha1.ResourceTPS:  resource.MustParse("100"),
				},
				ResizePolicy: storagev1alpha1.ResizePolicyStatic,
			}
			Expect(k8sClient.Create(ctx, volumeClassStatic)).To(Succeed())
			DeferCleanup(k8sClient.Delete, volumeClassStatic)
		})
		It("should return an empty config chart value map if not storageclasses are present in the cloudprofile", func(ctx SpecContext) {
			providerCloudProfile := &apisironcore.CloudProfileConfig{}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())

			cluster := &controller.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns.Name,
				},
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

		It("should return correct config chart values if default and additional storage classes are present in the cloudprofile", func(ctx SpecContext) {
			providerCloudProfile := &apisironcore.CloudProfileConfig{
				StorageClasses: apisironcore.StorageClasses{
					Default: &apisironcore.StorageClass{
						Name: "foo",
						Type: "volume-expandable",
					},
					Additional: []apisironcore.StorageClass{
						{
							Name: "foo1",
							Type: "volume-expandable",
						},
						{
							Name: "foo2",
							Type: "volume-static",
						},
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())

			cluster := &controller.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns.Name,
				},
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
						"name":       "foo",
						"type":       "volume-expandable",
						"default":    true,
						"expandable": true,
					},
					{
						"name":       "foo1",
						"type":       "volume-expandable",
						"expandable": true,
					},
					{
						"name":       "foo2",
						"type":       "volume-static",
						"expandable": false,
					},
				},
			}))
		})

		It("should return correct config chart values if only additional storage classes are present in the cloudprofile", func(ctx SpecContext) {
			providerCloudProfile := &apisironcore.CloudProfileConfig{
				StorageClasses: apisironcore.StorageClasses{
					Additional: []apisironcore.StorageClass{
						{
							Name: "foo1",
							Type: "volume-expandable",
						},
						{
							Name: "foo2",
							Type: "volume-static",
						},
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())

			cluster := &controller.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns.Name,
				},
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
						"name":       "foo1",
						"type":       "volume-expandable",
						"expandable": true,
					},
					{
						"name":       "foo2",
						"type":       "volume-static",
						"expandable": false,
					},
				},
			}))
		})

		It("should return error if volumeClass is not available", func(ctx SpecContext) {
			providerCloudProfile := &apisironcore.CloudProfileConfig{
				StorageClasses: apisironcore.StorageClasses{
					Default: &apisironcore.StorageClass{
						Name: "foo",
						Type: "volume-non-existing",
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())

			cluster := &controller.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns.Name,
				},
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Raw: providerCloudProfileJson,
						},
					},
				},
			}

			_, err = vp.GetStorageClassesChartValues(ctx, nil, cluster)
			Expect(err).To(MatchError("could not get resize policy from volumeclass : VolumeClass not found"))
		})

	})

	Describe("#GetControlPlaneShootCRDsChartValues", func() {
		It("should return correct config chart values", func(ctx SpecContext) {
			values, err := vp.GetControlPlaneShootCRDsChartValues(ctx, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(map[string]interface{}{}))
		})
	})

	Describe("#GetControlPlaneShootChartValues", func() {
		It("should return correct config chart values", func(ctx SpecContext) {
			providerCloudProfile := &apisironcore.CloudProfileConfig{
				StorageClasses: apisironcore.StorageClasses{
					Default: &apisironcore.StorageClass{
						Name: "foo",
						Type: "volume-expandable",
					},
					Additional: []apisironcore.StorageClass{
						{
							Name: "bar",
							Type: "volume-static",
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
		It("should return correct config chart values", func(ctx SpecContext) {
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
						Type: ironcore.Type,
						ProviderConfig: &runtime.RawExtension{
							Raw: encode(&apisironcore.ControlPlaneConfig{
								CloudControllerManager: &apisironcore.CloudControllerManagerConfig{
									FeatureGates: map[string]bool{
										"CustomResourceValidation": true,
									},
								},
							}),
						},
					},
					InfrastructureProviderStatus: &runtime.RawExtension{
						Raw: encode(&apisironcore.InfrastructureStatus{
							NetworkRef: v1alpha1.LocalUIDReference{
								Name: "my-network",
								UID:  "1234",
							},
						}),
					},
				},
			}
			providerCloudProfile := &apisironcore.CloudProfileConfig{
				StorageClasses: apisironcore.StorageClasses{
					Default: &apisironcore.StorageClass{
						Name: "foo",
						Type: "volumeTypeFoo",
					},
					Additional: []apisironcore.StorageClass{
						{
							Name: "bar",
							Type: "volumeTypeBar",
						},
					},
				},
			}
			providerCloudProfileJson, err := json.Marshal(providerCloudProfile)
			Expect(err).NotTo(HaveOccurred())
			networkProviderConfig := &unstructured.Unstructured{Object: map[string]any{
				"kind":       "FooNetworkConfig",
				"apiVersion": "v1alpha1",
				"overlay": map[string]any{
					"enabled": false,
				},
			}}
			networkProviderConfigData, err := runtime.Encode(unstructured.UnstructuredJSONScheme, networkProviderConfig)
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
						Networking: &gardencorev1beta1.Networking{
							ProviderConfig: &runtime.RawExtension{Raw: networkProviderConfigData},
							Pods:           pointer.String("10.0.0.0/16"),
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
				ironcore.CloudProviderConfigName: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
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
					"podNetwork":           "10.0.0.0/16",
					"configureCloudRoutes": true,
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
