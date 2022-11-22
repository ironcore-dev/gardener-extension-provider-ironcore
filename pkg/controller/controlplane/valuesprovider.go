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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/chart"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	"github.com/gardener/gardener/pkg/utils/version"
	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/internal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	caNameControlPlane                   = "ca-" + onmetal.ProviderName + "-controlplane"
	cloudControllerManagerDeploymentName = "cloud-controller-manager"
	cloudControllerManagerServerName     = "cloud-controller-manager-server"
	csiSnapshotValidationServerName      = onmetal.CSISnapshotValidation + "-server"
)

func secretConfigsFunc(namespace string) []extensionssecretsmanager.SecretConfigWithOptions {
	return []extensionssecretsmanager.SecretConfigWithOptions{
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:       caNameControlPlane,
				CommonName: caNameControlPlane,
				CertType:   secrets.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        cloudControllerManagerServerName,
				CommonName:                  onmetal.CloudControllerManagerName,
				DNSNames:                    kutil.DNSNamesForService(onmetal.CloudControllerManagerName, namespace),
				CertType:                    secretutils.ServerCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        csiSnapshotValidationServerName,
				CommonName:                  onmetal.UsernamePrefix + onmetal.CSISnapshotValidation,
				DNSNames:                    kutil.DNSNamesForService(onmetal.CSISnapshotValidation, namespace),
				CertType:                    secretutils.ServerCert,
				SkipPublishingCACertificate: true,
			},
			// use current CA for signing server cert to prevent mismatches when dropping the old CA from the webhook
			// config in phase Completing
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane, secretsmanager.UseCurrentCA)},
		},
	}
}

func shootAccessSecretsFunc(namespace string) []*gutil.ShootAccessSecret {
	return []*gutil.ShootAccessSecret{
		gutil.NewShootAccessSecret(cloudControllerManagerDeploymentName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSIProvisionerName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSIAttacherName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSISnapshotterName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSIResizerName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSISnapshotControllerName, namespace),
	}
}

var (
	configChart = &chart.Chart{
		Name: "cloud-provider-config",
		Path: filepath.Join(onmetal.InternalChartsPath, "cloud-provider-config"),
		Objects: []*chart.Object{
			{Type: &corev1.ConfigMap{}, Name: internal.CloudProviderConfigName},
		},
	}

	controlPlaneChart = &chart.Chart{
		Name: "seed-controlplane",
		Path: filepath.Join(onmetal.InternalChartsPath, "seed-controlplane"),
		SubCharts: []*chart.Chart{
			{
				Name:   onmetal.CloudControllerManagerName,
				Images: []string{onmetal.CloudControllerManagerImageName},
				Objects: []*chart.Object{
					{Type: &corev1.Service{}, Name: "cloud-controller-manager"},
					{Type: &appsv1.Deployment{}, Name: "cloud-controller-manager"},
					{Type: &corev1.ConfigMap{}, Name: "cloud-controller-manager-observability-config"},
					{Type: &autoscalingv1.VerticalPodAutoscaler{}, Name: "cloud-controller-manager-vpa"},
				},
			},
			{
				Name: onmetal.CSIControllerName,
				Images: []string{
					onmetal.CSIDriverImageName,
					onmetal.CSIProvisionerImageName,
					onmetal.CSIAttacherImageName,
					onmetal.CSISnapshotterImageName,
					onmetal.CSIResizerImageName,
					onmetal.CSILivenessProbeImageName,
					onmetal.CSISnapshotControllerImageName,
					onmetal.CSISnapshotValidationWebhookImageName,
				},
				Objects: []*chart.Object{
					// csi-driver-controller
					{Type: &appsv1.Deployment{}, Name: onmetal.CSIControllerName},
					{Type: &corev1.ConfigMap{}, Name: onmetal.CSIControllerConfigName},
					{Type: &corev1.ConfigMap{}, Name: onmetal.CSIControllerObservabilityConfigName},
					{Type: &autoscalingv1.VerticalPodAutoscaler{}, Name: onmetal.CSIControllerName + "-vpa"},
					// csi-snapshot-controller
					{Type: &appsv1.Deployment{}, Name: onmetal.CSISnapshotControllerName},
					{Type: &autoscalingv1.VerticalPodAutoscaler{}, Name: onmetal.CSISnapshotControllerName + "-vpa"},
					// csi-snapshot-validation-webhook
					{Type: &appsv1.Deployment{}, Name: onmetal.CSISnapshotValidation},
					{Type: &corev1.Service{}, Name: onmetal.CSISnapshotValidation},
					{Type: &networkingv1.NetworkPolicy{}, Name: "allow-kube-apiserver-to-csi-snapshot-validation"},
				},
			},
		},
	}

	controlPlaneShootChart = &chart.Chart{
		Name: "shoot-system-components",
		Path: filepath.Join(onmetal.InternalChartsPath, "shoot-system-components"),
		SubCharts: []*chart.Chart{
			{
				Name: "cloud-controller-manager",
				Path: filepath.Join(onmetal.InternalChartsPath, "cloud-controller-manager"),
				Objects: []*chart.Object{
					{Type: &rbacv1.ClusterRole{}, Name: "system:controller:cloud-node-controller"},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:controller:cloud-node-controller"},
					{Type: &rbacv1.ClusterRole{}, Name: "onmetal:cloud-provider"},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: "onmetal:cloud-provider"},
				},
			},
			{
				Name: onmetal.CSINodeName,
				Images: []string{
					onmetal.CSIDriverImageName,
					onmetal.CSINodeDriverRegistrarImageName,
					onmetal.CSILivenessProbeImageName,
				},
				Objects: []*chart.Object{
					// csi-driver
					{Type: &appsv1.DaemonSet{}, Name: onmetal.CSINodeName},
					{Type: &storagev1.CSIDriver{}, Name: "pd.csi.storage.gke.io"},
					{Type: &corev1.ServiceAccount{}, Name: onmetal.CSIDriverName},
					{Type: &rbacv1.ClusterRole{}, Name: onmetal.UsernamePrefix + onmetal.CSIDriverName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIDriverName},
					{Type: &policyv1beta1.PodSecurityPolicy{}, Name: strings.Replace(onmetal.UsernamePrefix+onmetal.CSIDriverName, ":", ".", -1)},
					{Type: extensionscontroller.GetVerticalPodAutoscalerObject(), Name: onmetal.CSINodeName},
					// csi-provisioner
					{Type: &rbacv1.ClusterRole{}, Name: onmetal.UsernamePrefix + onmetal.CSIProvisionerName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIProvisionerName},
					{Type: &rbacv1.Role{}, Name: onmetal.UsernamePrefix + onmetal.CSIProvisionerName},
					{Type: &rbacv1.RoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIProvisionerName},
					// csi-attacher
					{Type: &rbacv1.ClusterRole{}, Name: onmetal.UsernamePrefix + onmetal.CSIAttacherName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIAttacherName},
					{Type: &rbacv1.Role{}, Name: onmetal.UsernamePrefix + onmetal.CSIAttacherName},
					{Type: &rbacv1.RoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIAttacherName},
					// csi-snapshot-controller
					{Type: &rbacv1.ClusterRole{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotControllerName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotControllerName},
					{Type: &rbacv1.Role{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotControllerName},
					{Type: &rbacv1.RoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotControllerName},
					// csi-snapshotter
					{Type: &rbacv1.ClusterRole{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotterName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotterName},
					{Type: &rbacv1.Role{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotterName},
					{Type: &rbacv1.RoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSISnapshotterName},
					// csi-resizer
					{Type: &rbacv1.ClusterRole{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
					{Type: &rbacv1.Role{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
					{Type: &rbacv1.RoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
					// csi-snapshot-validation-webhook
					{Type: &admissionregistrationv1.ValidatingWebhookConfiguration{}, Name: onmetal.CSISnapshotValidation},
				},
			},
		},
	}

	controlPlaneShootCRDsChart = &chart.Chart{
		Name: "shoot-crds",
		Path: filepath.Join(onmetal.InternalChartsPath, "shoot-crds"),
		SubCharts: []*chart.Chart{
			{
				Name: "volumesnapshots",
				Objects: []*chart.Object{
					{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: "volumesnapshotclasses.snapshot.storage.k8s.io"},
					{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: "volumesnapshotcontents.snapshot.storage.k8s.io"},
					{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: "volumesnapshots.snapshot.storage.k8s.io"},
				},
			},
		},
	}

	storageClassChart = &chart.Chart{
		Name: "shoot-storageclasses",
		Path: filepath.Join(onmetal.InternalChartsPath, "shoot-storageclasses"),
	}
)

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider() genericactuator.ValuesProvider {
	return &valuesProvider{}
}

// valuesProvider is a ValuesProvider that provides onmetal-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	genericactuator.NoopValuesProvider
	common.ClientContext
}

// GetConfigChartValues returns the values for the config chart applied by the generic actuator.
func (vp *valuesProvider) GetConfigChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	// Decode providerConfig
	cpConfig := &apisonmetal.ControlPlaneConfig{}
	if cp.Spec.ProviderConfig != nil {
		if _, _, err := vp.Decoder().Decode(cp.Spec.ProviderConfig.Raw, nil, cpConfig); err != nil {
			return nil, fmt.Errorf("could not decode providerConfig of controlplane '%s': %w", kutil.ObjectName(cp), err)
		}
	}

	provicerSecret := &corev1.Secret{}
	providerSecretKey := types.NamespacedName{Namespace: cp.Namespace, Name: cp.Spec.SecretRef.Name}
	if cp.Spec.SecretRef.Name != "" {
		if err := vp.Client().Get(ctx, providerSecretKey, provicerSecret); err != nil {
			return nil, fmt.Errorf("could not get provider secret %s: %w", providerSecretKey, err)
		}
	}

	// Decode infrastructureProviderStatus
	infraStatus := &apisonmetal.InfrastructureStatus{}
	if _, _, err := vp.Decoder().Decode(cp.Spec.InfrastructureProviderStatus.Raw, nil, infraStatus); err != nil {
		return nil, fmt.Errorf("could not decode infrastructureProviderStatus of controlplane '%s': %w", kutil.ObjectName(cp), err)
	}

	// Get config chart values
	return getConfigChartValues(cpConfig, infraStatus, cp, provicerSecret)
}

// GetControlPlaneChartValues returns the values for the control plane chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	checksums map[string]string,
	scaledDown bool,
) (
	map[string]interface{},
	error,
) {
	cpConfig := &apisonmetal.ControlPlaneConfig{}
	if cp.Spec.ProviderConfig != nil {
		if _, _, err := vp.Decoder().Decode(cp.Spec.ProviderConfig.Raw, nil, cpConfig); err != nil {
			return nil, fmt.Errorf("could not decode providerConfig of controlplane '%s': %w", kutil.ObjectName(cp), err)
		}
	}

	return getControlPlaneChartValues(cpConfig, cp, cluster, secretsReader, checksums, scaledDown)
}

// GetControlPlaneShootChartValues returns the values for the control plane shoot chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneShootChartValues(
	_ context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	_ map[string]string,
) (
	map[string]interface{},
	error,
) {
	return getControlPlaneShootChartValues(cluster, cp, secretsReader)
}

// GetControlPlaneShootCRDsChartValues returns the values for the control plane shoot CRDs chart applied by the generic actuator.
// Currently the provider extension does not specify a control plane shoot CRDs chart. That's why we simply return empty values.
func (vp *valuesProvider) GetControlPlaneShootCRDsChartValues(
	_ context.Context,
	_ *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	k8sVersionLessThan118, err := version.CompareVersions(cluster.Shoot.Spec.Kubernetes.Version, "<", onmetal.CSIMigrationKubernetesVersion)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"volumesnapshots": map[string]interface{}{
			"enabled": !k8sVersionLessThan118,
		},
	}, nil
}

// GetStorageClassesChartValues returns the values for the storage classes chart applied by the generic actuator.
func (vp *valuesProvider) GetStorageClassesChartValues(
	_ context.Context,
	_ *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	k8sVersionLessThan118, err := version.CompareVersions(cluster.Shoot.Spec.Kubernetes.Version, "<", onmetal.CSIMigrationKubernetesVersion)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"useLegacyProvisioner": k8sVersionLessThan118,
	}, nil
}

// getConfigChartValues collects and returns the configuration chart values.
func getConfigChartValues(cpConfig *apisonmetal.ControlPlaneConfig, infraStatus *apisonmetal.InfrastructureStatus, cp *extensionsv1alpha1.ControlPlane, secret *corev1.Secret) (map[string]interface{}, error) {
	namespace, ok := secret.Data[onmetal.NamespaceFieldName]
	if !ok {
		return nil, fmt.Errorf("no namespace found in provider secret %s", client.ObjectKeyFromObject(secret))
	}
	token, ok := secret.Data[onmetal.TokenFieldName]
	if !ok {
		return nil, fmt.Errorf("no token found in provider secret %s", client.ObjectKeyFromObject(secret))
	}

	// Collect config chart values
	return map[string]interface{}{
		onmetal.NamespaceFieldName: string(namespace),
		onmetal.TokenFieldName:     string(token),
	}, nil
}

// getControlPlaneChartValues collects and returns the control plane chart values.
func getControlPlaneChartValues(
	cpConfig *apisonmetal.ControlPlaneConfig,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	checksums map[string]string,
	scaledDown bool,
) (
	map[string]interface{},
	error,
) {
	ccm, err := getCCMChartValues(cpConfig, cp, cluster, secretsReader, checksums, scaledDown)
	if err != nil {
		return nil, err
	}

	csi, err := getCSIControllerChartValues(cpConfig, cp, cluster, secretsReader, checksums, scaledDown)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"global": map[string]interface{}{
			"genericTokenKubeconfigSecretName": extensionscontroller.GenericTokenKubeconfigSecretNameFromCluster(cluster),
		},
		onmetal.CloudControllerManagerName: ccm,
		onmetal.CSIControllerName:          csi,
	}, nil
}

// getCCMChartValues collects and returns the CCM chart values.
func getCCMChartValues(
	cpConfig *apisonmetal.ControlPlaneConfig,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	kubeVersion, err := semver.NewVersion(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return nil, err
	}

	serverSecret, found := secretsReader.Get(cloudControllerManagerServerName)
	if !found {
		return nil, fmt.Errorf("secret %q not found", cloudControllerManagerServerName)
	}

	values := map[string]interface{}{
		"enabled":           true,
		"replicas":          extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
		"clusterName":       cp.Namespace,
		"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		"podNetwork":        extensionscontroller.GetPodNetwork(cluster),
		"podAnnotations": map[string]interface{}{
			"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: checksums[v1beta1constants.SecretNameCloudProvider],
			"checksum/configmap-" + internal.CloudProviderConfigName:      checksums[internal.CloudProviderConfigName],
		},
		"podLabels": map[string]interface{}{
			v1beta1constants.LabelPodMaintenanceRestart: "true",
		},
		"tlsCipherSuites": kutil.TLSCipherSuites(kubeVersion),
		"secrets": map[string]interface{}{
			"server": serverSecret.Name,
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
	}

	return values, nil
}

// getCSIControllerChartValues collects and returns the CSIController chart values.
func getCSIControllerChartValues(
	cpConfig *apisonmetal.ControlPlaneConfig,
	_ *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	k8sVersionLessThan118, err := version.CompareVersions(cluster.Shoot.Spec.Kubernetes.Version, "<", onmetal.CSIMigrationKubernetesVersion)
	if err != nil {
		return nil, err
	}

	if k8sVersionLessThan118 {
		return map[string]interface{}{"enabled": false}, nil
	}

	serverSecret, found := secretsReader.Get(csiSnapshotValidationServerName)
	if !found {
		return nil, fmt.Errorf("secret %q not found", csiSnapshotValidationServerName)
	}

	return map[string]interface{}{
		"enabled":  true,
		"replicas": extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
		"podAnnotations": map[string]interface{}{
			"checksum/secret-" + v1beta1constants.SecretNameCloudProvider: checksums[v1beta1constants.SecretNameCloudProvider],
		},
		"csiSnapshotController": map[string]interface{}{
			"replicas": extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
		},
		"csiSnapshotValidationWebhook": map[string]interface{}{
			"replicas": extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"secrets": map[string]interface{}{
				"server": serverSecret.Name,
			},
		},
	}, nil
}

// getControlPlaneShootChartValues collects and returns the control plane shoot chart values.
func getControlPlaneShootChartValues(
	cluster *extensionscontroller.Cluster,
	cp *extensionsv1alpha1.ControlPlane,
	secretsReader secretsmanager.Reader,
) (map[string]interface{}, error) {
	kubernetesVersion := cluster.Shoot.Spec.Kubernetes.Version
	k8sVersionLessThan118, err := version.CompareVersions(kubernetesVersion, "<", onmetal.CSIMigrationKubernetesVersion)
	if err != nil {
		return nil, err
	}

	caSecret, found := secretsReader.Get(caNameControlPlane)
	if !found {
		return nil, fmt.Errorf("secret %q not found", caNameControlPlane)
	}

	return map[string]interface{}{
		onmetal.CloudControllerManagerName: map[string]interface{}{"enabled": true},
		onmetal.CSINodeName: map[string]interface{}{
			"enabled":           !k8sVersionLessThan118,
			"kubernetesVersion": kubernetesVersion,
			"vpaEnabled":        gardencorev1beta1helper.ShootWantsVerticalPodAutoscaler(cluster.Shoot),
			"webhookConfig": map[string]interface{}{
				"url":      "https://" + onmetal.CSISnapshotValidation + "." + cp.Namespace + "/volumesnapshot",
				"caBundle": string(caSecret.Data[secretutils.DataKeyCertificateBundle]),
			},
			"pspDisabled": gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
		},
	}, nil
}
