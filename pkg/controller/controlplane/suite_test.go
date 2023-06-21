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
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/onmetal/controller-utils/buildutils"
	"github.com/onmetal/controller-utils/modutils"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
	utilsenvtest "github.com/onmetal/onmetal-api/utils/envtest"
	"github.com/onmetal/onmetal-api/utils/envtest/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	onmetalextensionv1alpha1 "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1"
)

const (
	pollingInterval      = 50 * time.Millisecond
	eventuallyTimeout    = 10 * time.Second
	consistentlyDuration = 1 * time.Second
	apiServiceTimeout    = 5 * time.Minute
)

var (
	testEnv    *envtest.Environment
	testEnvExt *utilsenvtest.EnvironmentExtensions
	cfg        *rest.Config
	k8sClient  client.Client
)

func TestAPIs(t *testing.T) {
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Infrastructure Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var err error

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "example", "20-crd-resources.gardener.cloud_managedresources.yaml"),
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_clusters.yaml"),
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_infrastructures.yaml"),
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_controlplanes.yaml"),
		},
		ErrorIfCRDPathMissing: true,
	}

	testEnvExt = &utilsenvtest.EnvironmentExtensions{
		APIServiceDirectoryPaths: []string{
			modutils.Dir("github.com/onmetal/onmetal-api", "config", "apiserver", "apiservice", "bases"),
		},
		ErrorIfAPIServicePathIsMissing: true,
	}

	cfg, err = utilsenvtest.StartWithExtensions(testEnv, testEnvExt)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	DeferCleanup(utilsenvtest.StopWithExtensions, testEnv, testEnvExt)

	Expect(extensionsv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(onmetalextensionv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(storagev1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	// Init package-level k8sClient
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	komega.SetClient(k8sClient)

	apiSrv, err := apiserver.New(cfg, apiserver.Options{
		MainPath:     "github.com/onmetal/onmetal-api/cmd/onmetal-apiserver",
		BuildOptions: []buildutils.BuildOption{buildutils.ModModeMod},
		ETCDServers:  []string{testEnv.ControlPlane.Etcd.URL.String()},
		Host:         testEnvExt.APIServiceInstallOptions.LocalServingHost,
		Port:         testEnvExt.APIServiceInstallOptions.LocalServingPort,
		CertDir:      testEnvExt.APIServiceInstallOptions.LocalServingCertDir,
	})
	Expect(err).NotTo(HaveOccurred())

	Expect(apiSrv.Start()).To(Succeed())
	DeferCleanup(apiSrv.Stop)

	err = utilsenvtest.WaitUntilAPIServicesReadyWithTimeout(apiServiceTimeout, testEnvExt, k8sClient, scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
})

func SetupTest() (*corev1.Namespace, *valuesProvider) {
	var (
		namespace = &corev1.Namespace{}
		cluster   = &extensionsv1alpha1.Cluster{}
		vp        = &valuesProvider{}
	)

	BeforeEach(func(ctx SpecContext) {
		var mgrCtx context.Context
		mgrCtx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		*namespace = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "testns-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed(), "failed to create test namespace")
		DeferCleanup(k8sClient.Delete, namespace)

		shoot := v1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace.Name,
				Name:      "foo",
			},
			Spec: v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					Version: "1.26.0",
				},
				Region: "foo",
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
				Networking: &v1beta1.Networking{
					Nodes: pointer.String("10.0.0.0/24"),
				},
			},
		}
		shootJson, err := json.Marshal(shoot)
		Expect(err).NotTo(HaveOccurred())

		*cluster = extensionsv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace.Name,
			},
			Spec: extensionsv1alpha1.ClusterSpec{
				CloudProfile: runtime.RawExtension{Raw: []byte("{}")},
				Seed:         runtime.RawExtension{Raw: []byte("{}")},
				Shoot:        runtime.RawExtension{Raw: shootJson},
			},
		}
		Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
		DeferCleanup(k8sClient.Delete, cluster)

		mgr, err := manager.New(cfg, manager.Options{
			Scheme:             scheme.Scheme,
			MetricsBindAddress: "0",
			CertDir:            testEnv.WebhookInstallOptions.LocalServingCertDir,
		})
		Expect(err).NotTo(HaveOccurred())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace.Name,
				Name:      "my-infra-creds",
			},
			Data: map[string][]byte{
				"namespace": []byte(namespace.Name),
				"token":     []byte("foo"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		user, err := testEnv.AddUser(envtest.User{
			Name:   "dummy",
			Groups: []string{"system:authenticated", "system:masters"},
		}, cfg)
		Expect(err).NotTo(HaveOccurred())

		kubeconfig, err := user.KubeConfig()
		Expect(err).NotTo(HaveOccurred())

		By("creating a test cloudprovider secret")
		cloudproviderSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace.Name,
				Name:      "cloudprovider",
			},
			Data: map[string][]byte{
				"namespace":  []byte(namespace.Name),
				"token":      []byte("foo"),
				"kubeconfig": kubeconfig,
			},
		}
		Expect(k8sClient.Create(ctx, cloudproviderSecret)).To(Succeed())
		DeferCleanup(k8sClient.Delete, cloudproviderSecret)

		Expect(AddToManagerWithOptions(mgr, AddOptions{
			IgnoreOperationAnnotation: true,
		})).NotTo(HaveOccurred())

		*vp = valuesProvider{}
		Expect(vp.InjectScheme(scheme.Scheme)).NotTo(HaveOccurred())
		Expect(vp.InjectClient(k8sClient)).NotTo(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			Expect(mgr.Start(mgrCtx)).To(Succeed(), "failed to start manager")
		}()
	})

	return namespace, vp
}
