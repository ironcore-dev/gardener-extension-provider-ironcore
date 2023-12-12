// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/ironcore-dev/controller-utils/buildutils"
	"github.com/ironcore-dev/controller-utils/modutils"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore/api/core/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	utilsenvtest "github.com/ironcore-dev/ironcore/utils/envtest"
	"github.com/ironcore-dev/ironcore/utils/envtest/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	controllerconfig "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/config"
	apiv1alpha1 "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
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
	RunSpecs(t, "Bastion Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var err error

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_clusters.yaml"),
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_workers.yaml"),
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_bastions.yaml"),
		},
		ErrorIfCRDPathMissing: true,
	}
	testEnvExt = &utilsenvtest.EnvironmentExtensions{
		APIServiceDirectoryPaths: []string{
			modutils.Dir("github.com/ironcore-dev/ironcore", "config", "apiserver", "apiservice", "bases"),
		},
		ErrorIfAPIServicePathIsMissing: true,
	}

	cfg, err = utilsenvtest.StartWithExtensions(testEnv, testEnvExt)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	DeferCleanup(utilsenvtest.StopWithExtensions, testEnv, testEnvExt)

	Expect(computev1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(storagev1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(ipamv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(networkingv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(extensionsv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(apiv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	// Init package-level k8sClient
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	komega.SetClient(k8sClient)

	apiSrv, err := apiserver.New(cfg, apiserver.Options{
		MainPath:     "github.com/ironcore-dev/ironcore/cmd/ironcore-apiserver",
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

func SetupTest() *corev1.Namespace {
	namespace := &corev1.Namespace{}
	cluster := &extensionsv1alpha1.Cluster{}

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

		By("creating a test shoot")
		shoot := v1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace.Name,
				Name:      "foo",
			},
			Spec: v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
				Networking: &v1beta1.Networking{
					Nodes: pointer.String("10.0.0.0/24"),
				},
				Region: "abc",
			},
		}
		shootJson, err := json.Marshal(shoot)
		Expect(err).NotTo(HaveOccurred())

		By("creating a test cluster")
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

		By("creating a test machine class")
		machineClass := &computev1alpha1.MachineClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-machine-class",
			},
			Capabilities: map[corev1alpha1.ResourceName]resource.Quantity{
				corev1alpha1.ResourceCPU:    resource.MustParse("100m"),
				corev1alpha1.ResourceMemory: resource.MustParse("8Gi"),
			},
		}
		Expect(k8sClient.Create(ctx, machineClass)).To(Succeed())
		DeferCleanup(k8sClient.Delete, machineClass)

		By("creating a test volume class")
		volumeClass := &storagev1alpha1.VolumeClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-volume-class",
			},
			Capabilities: map[corev1alpha1.ResourceName]resource.Quantity{
				corev1alpha1.ResourceIOPS: resource.MustParse("100"),
				corev1alpha1.ResourceTPS:  resource.MustParse("100"),
			},
		}
		Expect(k8sClient.Create(ctx, volumeClass)).To(Succeed())
		DeferCleanup(k8sClient.Delete, volumeClass)

		By("creating a test worker")
		volumeName := "test-volume"
		volumeType := "fast"
		worker := &extensionsv1alpha1.Worker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: namespace.Name,
			},
			Spec: extensionsv1alpha1.WorkerSpec{
				Pools: []extensionsv1alpha1.WorkerPool{
					{
						MachineType:    "foo",
						Maximum:        10,
						MaxSurge:       intstr.IntOrString{IntVal: 5},
						MaxUnavailable: intstr.IntOrString{IntVal: 2},
						Annotations:    map[string]string{"foo": "bar"},
						Labels:         map[string]string{"foo": "bar"},
						MachineImage: extensionsv1alpha1.MachineImage{
							Name:    "my-os",
							Version: "1.0",
						},
						Minimum:  0,
						Name:     "pool",
						UserData: []byte("some-data"),
						Volume: &extensionsv1alpha1.Volume{
							Name: &volumeName,
							Type: &volumeType,
							Size: "10Gi",
						},
						Zones:        []string{"zone1", "zone2"},
						Architecture: pointer.String("amd64"),
						NodeTemplate: &extensionsv1alpha1.NodeTemplate{
							Capacity: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceCPU: resource.MustParse("100m"),
							},
						},
					},
				},
			},
		}
		infraStatus := &apiv1alpha1.InfrastructureStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
				Kind:       "InfrastructureStatus",
			},
			NetworkRef: commonv1alpha1.LocalUIDReference{
				Name: "my-network",
				UID:  "1234",
			},
			PrefixRef: commonv1alpha1.LocalUIDReference{
				Name: "my-prefix",
				UID:  "4321",
			},
		}
		worker.Spec.InfrastructureProviderStatus = &runtime.RawExtension{Object: infraStatus}
		Expect(k8sClient.Create(ctx, worker)).Should(Succeed())

		mgr, err := manager.New(cfg, manager.Options{
			Scheme:  scheme.Scheme,
			Metrics: metricsserver.Options{BindAddress: "0"},
		})
		Expect(err).NotTo(HaveOccurred())

		user, err := testEnv.AddUser(envtest.User{
			Name:   "dummy",
			Groups: []string{"system:authenticated", "system:masters"},
		}, cfg)
		Expect(err).NotTo(HaveOccurred())

		kubeconfig, err := user.KubeConfig()
		Expect(err).NotTo(HaveOccurred())

		By("creating a test cloudprovider secret")
		secret := &corev1.Secret{
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
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		bastionConfig := controllerconfig.BastionConfig{
			Image:            "my-image",
			MachineClassName: machineClass.Name,
			VolumeClassName:  volumeClass.Name,
		}
		Expect(AddToManagerWithOptions(mgr, AddOptions{
			IgnoreOperationAnnotation: true,
			BastionConfig:             bastionConfig,
		})).NotTo(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			Expect(mgr.Start(mgrCtx)).To(Succeed(), "failed to start manager")
		}()
	})

	return namespace
}
