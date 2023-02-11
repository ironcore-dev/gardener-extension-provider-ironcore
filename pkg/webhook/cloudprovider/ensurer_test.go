// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudprovider

import (
	"context"
	"testing"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/webhook/cloudprovider"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/golang/mock/gomock"
	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/install"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const namespace = "test"

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudProvider Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	var (
		ctrl   *gomock.Controller
		ctx    = context.TODO()
		scheme *runtime.Scheme

		cloudProfileConfig = &api.CloudProfileConfig{
			TypeMeta:      metav1.TypeMeta{},
			MachineImages: []api.MachineImages{},
			RegionConfigs: []api.RegionConfig{
				{
					Name:                     "foo",
					Server:                   "https://localhost",
					CertificateAuthorityData: []byte("abcd1234"),
				},
			},
		}

		eContextK8s = gcontext.NewInternalGardenContext(
			&extensionscontroller.Cluster{
				CloudProfile: &gardencorev1beta1.CloudProfile{
					Spec: gardencorev1beta1.CloudProfileSpec{
						ProviderConfig: &runtime.RawExtension{
							Object: cloudProfileConfig,
						},
					},
				},
				Shoot: &gardencorev1beta1.Shoot{
					Spec: gardencorev1beta1.ShootSpec{
						Region: "foo",
						Kubernetes: gardencorev1beta1.Kubernetes{
							Version: "1.26.0",
						},
					},
				},
			},
		)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#EnsureCloudProviderSecret", func() {
		var (
			secret                 *corev1.Secret
			secretWithoutToken     *corev1.Secret
			secretWithoutNamespace *corev1.Secret
			ensurer                cloudprovider.Ensurer
		)

		BeforeEach(func() {
			scheme = runtime.NewScheme()
			install.Install(scheme)

			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "cloudprovider",
				},
				Data: map[string][]byte{
					"namespace": []byte("foo"),
					"token":     []byte("bar"),
				},
			}

			secretWithoutToken = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "cloudprovider",
				},
				Data: map[string][]byte{
					"token": []byte("bar"),
				},
			}

			secretWithoutNamespace = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "cloudprovider",
				},
				Data: map[string][]byte{
					"namespace": []byte("foo"),
				},
			}

			ensurer = NewEnsurer(logger)
			err := ensurer.(inject.Scheme).InjectScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should add a kubeconfig to the cloudprovider secret", func() {
			err := ensurer.EnsureCloudProviderSecret(ctx, eContextK8s, secret, nil)
			Expect(err).To(Not(HaveOccurred()))

			Expect(secret.Data).To(HaveKey("kubeconfig"))
			config, err := clientcmd.Load(secret.Data["kubeconfig"])
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Clusters[config.CurrentContext].Server).To(Equal("https://localhost"))
			Expect(config.Clusters[config.CurrentContext].CertificateAuthorityData).To(Equal([]byte("abcd1234")))
			Expect(config.AuthInfos[config.CurrentContext].Token).To(Equal("bar"))
		})

		It("should fail if the cloudprovider secret has no token", func() {
			err := ensurer.EnsureCloudProviderSecret(ctx, eContextK8s, secretWithoutToken, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the cloudprovider secret has no namespace", func() {
			err := ensurer.EnsureCloudProviderSecret(ctx, eContextK8s, secretWithoutNamespace, nil)
			Expect(err).To(HaveOccurred())
		})
	})
})
