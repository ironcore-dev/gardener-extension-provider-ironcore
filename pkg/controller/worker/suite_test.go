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

package worker

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerextensionv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/client/kubernetes"
	machinescheme "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned/scheme"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
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

	"github.com/onmetal/controller-utils/modutils"
	apiv1alpha1 "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	envtestutils "github.com/onmetal/onmetal-api/utils/envtest"
)

const (
	pollingInterval      = 50 * time.Millisecond
	eventuallyTimeout    = 10 * time.Second
	consistentlyDuration = 1 * time.Second
)

var (
	testEnv    *envtest.Environment
	testEnvExt *envtestutils.EnvironmentExtensions
	cfg        *rest.Config
	k8sClient  client.Client
)

// global Gardener resources used by delegates
var (
	shootVersionMajorMinor = "1.2"
	shootVersion           = shootVersionMajorMinor + ".3"

	pool               gardenerextensionv1alpha1.WorkerPool
	cloudProfileConfig *apiv1alpha1.CloudProfileConfig

	clusterWithoutImages   *extensionscontroller.Cluster
	cluster                *extensionscontroller.Cluster
	cloudProfileConfigJSON []byte

	w *gardenerextensionv1alpha1.Worker
)

func TestAPIs(t *testing.T) {
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.InfoLevel)))

	var err error

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			modutils.Dir("github.com/gardener/machine-controller-manager", "kubernetes", "crds", "machine.sapcloud.io_machineclasses.yaml"),
			modutils.Dir("github.com/gardener/machine-controller-manager", "kubernetes", "crds", "machine.sapcloud.io_machinedeployments.yaml"),
			modutils.Dir("github.com/gardener/machine-controller-manager", "kubernetes", "crds", "machine.sapcloud.io_machines.yaml"),
			modutils.Dir("github.com/gardener/machine-controller-manager", "kubernetes", "crds", "machine.sapcloud.io_machinesets.yaml"),
			modutils.Dir("github.com/gardener/machine-controller-manager", "kubernetes", "crds", "machine.sapcloud.io_scales.yaml"),
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_controlplanes.yaml"),
			filepath.Join("..", "..", "..", "example", "20-crd-extensions.gardener.cloud_workers.yaml"),
		},
		ErrorIfCRDPathMissing: true,
	}
	testEnvExt = &envtestutils.EnvironmentExtensions{
		APIServiceDirectoryPaths: []string{
			modutils.Dir("github.com/onmetal/onmetal-api", "config", "apiserver", "apiservice", "bases"),
		},
		ErrorIfAPIServicePathIsMissing: true,
	}

	cfg, err = envtestutils.StartWithExtensions(testEnv, testEnvExt)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	DeferCleanup(envtestutils.StopWithExtensions, testEnv, testEnvExt)

	Expect(networkingv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(apiextensionsscheme.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(machinescheme.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(gardenerextensionv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(apiv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	// Init package-level k8sClient
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	komega.SetClient(k8sClient)
})

func SetupTest(ctx context.Context) (*corev1.Namespace, *gardener.ChartApplier) {
	var (
		chartApplier gardener.ChartApplier
	)
	ns := &corev1.Namespace{}

	BeforeEach(func() {
		var err error
		*ns = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "testns-",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create test namespace")

		chartApplier, err = gardener.NewChartApplierForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		volumeName := "test-volume"
		volumeType := "fast"

		// define test resources
		pool = gardenerextensionv1alpha1.WorkerPool{
			MachineType:    "foo",
			Maximum:        10,
			MaxSurge:       intstr.IntOrString{IntVal: 5},
			MaxUnavailable: intstr.IntOrString{IntVal: 2},
			Annotations:    map[string]string{"foo": "bar"},
			Labels:         map[string]string{"foo": "bar"},
			MachineImage: gardenerextensionv1alpha1.MachineImage{
				Name:    "my-os",
				Version: "1.0",
			},
			Minimum:  0,
			Name:     "pool",
			UserData: []byte("some-data"),
			Volume: &gardenerextensionv1alpha1.Volume{
				Name: &volumeName,
				Type: &volumeType,
				Size: "10Gi",
			},
			Zones:        []string{"zone1", "zone2"},
			Architecture: pointer.String("amd64"),
			NodeTemplate: &gardenerextensionv1alpha1.NodeTemplate{
				Capacity: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		}
		cloudProfileConfig = &apiv1alpha1.CloudProfileConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
				Kind:       "CloudProfileConfig",
			},
			MachineImages: []apiv1alpha1.MachineImages{
				{
					Name: "my-os",
					Versions: []apiv1alpha1.MachineImageVersion{
						{
							Version:      "1.0",
							Image:        "registry/my-os",
							Architecture: pointer.String("amd64"),
						},
					},
				},
			},
		}
		shootVersionMajorMinor = "1.2"
		shootVersion = shootVersionMajorMinor + ".3"
		clusterWithoutImages = &extensionscontroller.Cluster{
			Shoot: &gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: shootVersion,
					},
				},
			},
		}
		cloudProfileConfigJSON, _ = json.Marshal(cloudProfileConfig)
		cluster = &extensionscontroller.Cluster{
			CloudProfile: &gardencorev1beta1.CloudProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name: "onmetal",
				},
				Spec: gardencorev1beta1.CloudProfileSpec{
					ProviderConfig: &runtime.RawExtension{
						Raw: cloudProfileConfigJSON,
					},
				},
			},
			Shoot: clusterWithoutImages.Shoot,
		}
		w = &gardenerextensionv1alpha1.Worker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pool",
				Namespace: ns.Name,
			},
			Spec: gardenerextensionv1alpha1.WorkerSpec{
				DefaultSpec: gardenerextensionv1alpha1.DefaultSpec{},
				Region:      "foo",
				SecretRef: corev1.SecretReference{
					Name: "my-secret",
				},
				SSHPublicKey: nil,
				Pools: []gardenerextensionv1alpha1.WorkerPool{
					pool,
				},
			},
		}
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed(), "failed to delete test namespace")
	})

	return ns, &chartApplier
}
