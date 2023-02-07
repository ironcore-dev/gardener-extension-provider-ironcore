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
	ns := SetupTest(ctx)

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
				HaveField("Data", HaveKey(onmetal.NamespaceFieldName)),
				HaveField("Data", HaveKey(onmetal.KubeConfigFieldName)),
			))
		})
	})

})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

//
//var _ = Describe("ValuesProvider", func() {
//	var (
//		ctrl *gomock.Controller
//		ctx  = testutils.SetupContext()
//
//		scheme = runtime.NewScheme()
//		_      = apisonmetal.AddToScheme(scheme)
//
//		vp genericactuator.ValuesProvider
//		c  *mockclient.MockClient
//
//		cp = &extensionsv1alpha1.ControlPlane{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      "control-plane",
//				Namespace: namespace,
//			},
//			Spec: extensionsv1alpha1.ControlPlaneSpec{
//				SecretRef: corev1.SecretReference{
//					Name:      v1beta1constants.SecretNameCloudProvider,
//					Namespace: namespace,
//				},
//				DefaultSpec: extensionsv1alpha1.DefaultSpec{
//					ProviderConfig: &runtime.RawExtension{
//						Raw: encode(&apisonmetal.ControlPlaneConfig{
//							CloudControllerManager: &apisonmetal.CloudControllerManagerConfig{
//								FeatureGates: map[string]bool{
//									"CustomResourceValidation": true,
//								},
//							},
//						}),
//					},
//				},
//				InfrastructureProviderStatus: &runtime.RawExtension{
//					Raw: encode(&apisonmetal.InfrastructureStatus{
//						NetworkRef: v1alpha1.LocalUIDReference{
//							Name: "my-network",
//							UID:  "1234",
//						},
//					}),
//				},
//			},
//		}
//
//		cidr                  = "10.250.0.0/19"
//		clusterK8sLessThan118 = &extensionscontroller.Cluster{
//			ObjectMeta: metav1.ObjectMeta{
//				Annotations: map[string]string{
//					"generic-token-kubeconfig.secret.gardener.cloud/name": genericTokenKubeconfigSecretName,
//				},
//			},
//			Shoot: &gardencorev1beta1.Shoot{
//				Spec: gardencorev1beta1.ShootSpec{
//					Networking: gardencorev1beta1.Networking{
//						Pods: &cidr,
//					},
//					Kubernetes: gardencorev1beta1.Kubernetes{
//						Version: "1.17.1",
//					},
//				},
//			},
//		}
//		clusterK8sAtLeast118 = &extensionscontroller.Cluster{
//			ObjectMeta: metav1.ObjectMeta{
//				Annotations: map[string]string{
//					"generic-token-kubeconfig.secret.gardener.cloud/name": genericTokenKubeconfigSecretName,
//				},
//			},
//			Shoot: &gardencorev1beta1.Shoot{
//				Spec: gardencorev1beta1.ShootSpec{
//					Networking: gardencorev1beta1.Networking{
//						Pods: &cidr,
//					},
//					Kubernetes: gardencorev1beta1.Kubernetes{
//						Version: "1.18.0",
//						VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
//							Enabled: true,
//						},
//					},
//				},
//			},
//		}
//
//		cpSecretKey = types.NamespacedName{Namespace: namespace, Name: v1beta1constants.SecretNameCloudProvider}
//		cpSecret    = &corev1.Secret{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      v1beta1constants.SecretNameCloudProvider,
//				Namespace: namespace,
//			},
//			Type: corev1.SecretTypeOpaque,
//			Data: map[string][]byte{
//				onmetal.NamespaceFieldName:  []byte("default"),
//				onmetal.KubeConfigFieldName: []byte("abcd"),
//			},
//		}
//
//		checksums = map[string]string{
//			v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
//			internal.CloudProviderConfigName:         "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
//		}
//
//		enabledTrue = map[string]interface{}{"enabled": true}
//
//		fakeClient         client.Client
//		fakeSecretsManager secretsmanager.Interface
//	)
//
//	BeforeEach(func() {
//		ctrl = gomock.NewController(GinkgoT())
//
//		registry := auth.NewSimpleRegionStubRegistry()
//		registry.AddRegionStub("foo", cfg)
//		clientConfigGetter := auth.NewClientConfigGetter(c, registry)
//		vp = NewValuesProvider(clientConfigGetter)
//		err := vp.(inject.Scheme).InjectScheme(scheme)
//		Expect(err).NotTo(HaveOccurred())
//		err = vp.(inject.Client).InjectClient(c)
//		Expect(err).NotTo(HaveOccurred())
//
//		fakeClient = fakeclient.NewClientBuilder().Build()
//		fakeSecretsManager = fakesecretsmanager.New(fakeClient, namespace)
//	})
//
//	AfterEach(func() {
//		ctrl.Finish()
//	})
//
//	Describe("#GetConfigChartValues", func() {
//		It("should return correct config chart values", func() {
//			c.EXPECT().Get(ctx, cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))
//
//			values, err := vp.GetConfigChartValues(ctx, cp, clusterK8sLessThan118)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(values).To(Equal(map[string]interface{}{
//				"namespace": "default",
//				"token":     "abcd",
//			}))
//		})
//	})
//
//	Describe("#GetControlPlaneChartValues", func() {
//		ccmChartValues := utils.MergeMaps(enabledTrue, map[string]interface{}{
//			"replicas":    1,
//			"clusterName": namespace,
//			"podNetwork":  cidr,
//			"podAnnotations": map[string]interface{}{
//				"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
//				"checksum/configmap-" + internal.CloudProviderConfigName:      "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
//			},
//			"podLabels": map[string]interface{}{
//				"maintenance.gardener.cloud/restart": "true",
//			},
//			"featureGates": map[string]bool{
//				"CustomResourceValidation": true,
//			},
//			"tlsCipherSuites": []string{
//				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
//				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
//				"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
//				"TLS_RSA_WITH_AES_128_CBC_SHA",
//				"TLS_RSA_WITH_AES_256_CBC_SHA",
//				"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
//			},
//			"secrets": map[string]interface{}{
//				"server": "cloud-controller-manager-server",
//			},
//		})
//
//		BeforeEach(func() {
//			//c.EXPECT().Get(ctx, cpSecretKey, &corev1.Secret{}).DoAndReturn(clientGet(cpSecret))
//
//			By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
//			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-provider-onmetal-controlplane", Namespace: namespace}})).To(Succeed())
//			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "csi-snapshot-validation-server", Namespace: namespace}})).To(Succeed())
//			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-controller-manager-server", Namespace: namespace}})).To(Succeed())
//		})
//
//		It("should return correct control plane chart values (k8s >= 1.18)", func() {
//			values, err := vp.GetControlPlaneChartValues(ctx, cp, clusterK8sAtLeast118, fakeSecretsManager, checksums, false)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(values).To(Equal(map[string]interface{}{
//				"global": map[string]interface{}{
//					"genericTokenKubeconfigSecretName": genericTokenKubeconfigSecretName,
//				},
//				onmetal.CloudControllerManagerName: utils.MergeMaps(ccmChartValues, map[string]interface{}{
//					"kubernetesVersion": clusterK8sAtLeast118.Shoot.Spec.Kubernetes.Version,
//				}),
//				onmetal.CSIControllerName: utils.MergeMaps(enabledTrue, map[string]interface{}{
//					"replicas": 1,
//					"podAnnotations": map[string]interface{}{
//						"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: checksums[v1beta1constants.SecretNameCloudProvider],
//					},
//					"csiSnapshotController": map[string]interface{}{
//						"replicas": 1,
//					},
//					"csiSnapshotValidationWebhook": map[string]interface{}{
//						"replicas": 1,
//						"secrets": map[string]interface{}{
//							"server": "csi-snapshot-validation-server",
//						},
//					},
//				}),
//			}))
//		})
//	})
//
//	Describe("#GetControlPlaneShootChartValues", func() {
//		BeforeEach(func() {
//			By("creating secrets managed outside of this package for whose secretsmanager.Get() will be called")
//			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-provider-onmetal-controlplane", Namespace: namespace}})).To(Succeed())
//			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "csi-snapshot-validation-server", Namespace: namespace}})).To(Succeed())
//			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cloud-controller-manager-server", Namespace: namespace}})).To(Succeed())
//		})
//
//		Context("shoot control plane chart values (k8s >= 1.18)", func() {
//			It("should return correct shoot control plane chart when ca is secret found", func() {
//				values, err := vp.GetControlPlaneShootChartValues(ctx, cp, clusterK8sAtLeast118, fakeSecretsManager, nil)
//				Expect(err).NotTo(HaveOccurred())
//				Expect(values).To(Equal(map[string]interface{}{
//					onmetal.CloudControllerManagerName: enabledTrue,
//					onmetal.CSINodeName: utils.MergeMaps(enabledTrue, map[string]interface{}{
//						"kubernetesVersion": "1.18.0",
//						"vpaEnabled":        true,
//						"webhookConfig": map[string]interface{}{
//							"url":      "https://" + onmetal.CSISnapshotValidation + "." + cp.Namespace + "/volumesnapshot",
//							"caBundle": "",
//						},
//						"pspDisabled": false,
//					}),
//				}))
//			})
//		})
//
//		Context("podSecurityPolicy", func() {
//			It("should return correct shoot control plane chart when PodSecurityPolicy admission plugin is not disabled in the shoot", func() {
//				clusterK8sAtLeast118.Shoot.Spec.Kubernetes.KubeAPIServer = &gardencorev1beta1.KubeAPIServerConfig{
//					AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
//						{
//							Name: "PodSecurityPolicy",
//						},
//					},
//				}
//				values, err := vp.GetControlPlaneShootChartValues(ctx, cp, clusterK8sAtLeast118, fakeSecretsManager, nil)
//				Expect(err).NotTo(HaveOccurred())
//				Expect(values).To(Equal(map[string]interface{}{
//					onmetal.CloudControllerManagerName: enabledTrue,
//					onmetal.CSINodeName: utils.MergeMaps(enabledTrue, map[string]interface{}{
//						"kubernetesVersion": "1.18.0",
//						"vpaEnabled":        true,
//						"webhookConfig": map[string]interface{}{
//							"url":      "https://" + onmetal.CSISnapshotValidation + "." + cp.Namespace + "/volumesnapshot",
//							"caBundle": "",
//						},
//						"pspDisabled": false,
//					}),
//				}))
//			})
//			It("should return correct shoot control plane chart when PodSecurityPolicy admission plugin is disabled in the shoot", func() {
//				clusterK8sAtLeast118.Shoot.Spec.Kubernetes.KubeAPIServer = &gardencorev1beta1.KubeAPIServerConfig{
//					AdmissionPlugins: []gardencorev1beta1.AdmissionPlugin{
//						{
//							Name:     "PodSecurityPolicy",
//							Disabled: pointer.Bool(true),
//						},
//					},
//				}
//				values, err := vp.GetControlPlaneShootChartValues(ctx, cp, clusterK8sAtLeast118, fakeSecretsManager, nil)
//				Expect(err).NotTo(HaveOccurred())
//				Expect(values).To(Equal(map[string]interface{}{
//					onmetal.CloudControllerManagerName: enabledTrue,
//					onmetal.CSINodeName: utils.MergeMaps(enabledTrue, map[string]interface{}{
//						"kubernetesVersion": "1.18.0",
//						"vpaEnabled":        true,
//						"webhookConfig": map[string]interface{}{
//							"url":      "https://" + onmetal.CSISnapshotValidation + "." + cp.Namespace + "/volumesnapshot",
//							"caBundle": "",
//						},
//						"pspDisabled": true,
//					}),
//				}))
//			})
//		})
//	})
//
//	Describe("#GetControlPlaneShootCRDsChartValues", func() {
//		It("should return correct control plane shoot CRDs chart values (k8s < 1.18)", func() {
//			values, err := vp.GetControlPlaneShootCRDsChartValues(ctx, cp, clusterK8sLessThan118)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(values).To(Equal(map[string]interface{}{"volumesnapshots": map[string]interface{}{"enabled": false}}))
//		})
//
//		It("should return correct control plane shoot CRDs chart values (k8s >= 1.18)", func() {
//			values, err := vp.GetControlPlaneShootCRDsChartValues(ctx, cp, clusterK8sAtLeast118)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(values).To(Equal(map[string]interface{}{"volumesnapshots": map[string]interface{}{"enabled": true}}))
//		})
//	})
//
//	Describe("#GetStorageClassesChartValues", func() {
//		It("should return correct storage class chart values (k8s < 1.18)", func() {
//			values, err := vp.GetStorageClassesChartValues(ctx, cp, clusterK8sLessThan118)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(values).To(Equal(map[string]interface{}{"useLegacyProvisioner": true}))
//		})
//
//		It("should return correct storage class chart values (k8s >= 1.18)", func() {
//			values, err := vp.GetStorageClassesChartValues(ctx, cp, clusterK8sAtLeast118)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(values).To(Equal(map[string]interface{}{"useLegacyProvisioner": false}))
//		})
//	})
//})
//
//func encode(obj runtime.Object) []byte {
//	data, _ := json.Marshal(obj)
//	return data
//}
//
//func clientGet(result runtime.Object) interface{} {
//	return func(ctx context.Context, key client.ObjectKey, obj runtime.Object, _ ...client.GetOption) error {
//		switch obj.(type) {
//		case *corev1.Secret:
//			*obj.(*corev1.Secret) = *result.(*corev1.Secret)
//		}
//		return nil
//	}
//}
