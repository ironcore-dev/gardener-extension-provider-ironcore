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
	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/internal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

const (
	caNameControlPlane                   = "ca-" + onmetal.ProviderName + "-controlplane"
	cloudControllerManagerDeploymentName = "cloud-controller-manager"
	cloudControllerManagerServerName     = "cloud-controller-manager-server"
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
	}
}

func shootAccessSecretsFunc(namespace string) []*gutil.ShootAccessSecret {
	return []*gutil.ShootAccessSecret{
		gutil.NewShootAccessSecret(cloudControllerManagerDeploymentName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSIProvisionerName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSIAttacherName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSIResizerName, namespace),
		gutil.NewShootAccessSecret(onmetal.CSIDriverImageName, namespace),
	}
}

var (
	configChart = &chart.Chart{
		Name: "cloud-provider-config",
		Path: filepath.Join(onmetal.InternalChartsPath, "cloud-provider-config"),
		Objects: []*chart.Object{
			{Type: &corev1.ConfigMap{}, Name: internal.CloudProviderSecretName},
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
					onmetal.CSIResizerImageName,
					onmetal.CSILivenessProbeImageName,
				},
				Objects: []*chart.Object{
					// csi-driver-controller
					{Type: &appsv1.Deployment{}, Name: onmetal.CSIControllerName},
					{Type: &corev1.ConfigMap{}, Name: onmetal.CSIControllerObservabilityConfigName},
					{Type: &autoscalingv1.VerticalPodAutoscaler{}, Name: onmetal.CSIControllerName + "-vpa"},
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
					{Type: &storagev1.CSIDriver{}, Name: onmetal.CSIStorageProvisioner},
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
					// csi-resizer
					{Type: &rbacv1.ClusterRole{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
					{Type: &rbacv1.Role{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
					{Type: &rbacv1.RoleBinding{}, Name: onmetal.UsernamePrefix + onmetal.CSIResizerName},
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
	infrastructureStatus := &apisonmetal.InfrastructureStatus{}
	if _, _, err := vp.Decoder().Decode(cp.Spec.InfrastructureProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return nil, fmt.Errorf("failed to decode infrastructure status: %w", err)
	}
	// Collect config chart values
	return map[string]interface{}{
		onmetal.NetworkFieldName: infrastructureStatus.NetworkRef.Name,
	}, nil
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
	_ *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	_ secretsmanager.Reader,
	_ map[string]string,
) (
	map[string]interface{},
	error,
) {
	return vp.getControlPlaneShootChartValues(cluster)
}

// GetControlPlaneShootCRDsChartValues returns the values for the control plane shoot CRDs chart applied by the generic actuator.
// Currently, the provider extension does not specify a control plane shoot CRDs chart. That's why we simply return empty values.
func (vp *valuesProvider) GetControlPlaneShootCRDsChartValues(
	_ context.Context,
	_ *extensionsv1alpha1.ControlPlane,
	_ *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// GetStorageClassesChartValues returns the values for the storage classes chart applied by the generic actuator.
func (vp *valuesProvider) GetStorageClassesChartValues(
	_ context.Context,
	controlPlane *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	providerConfig := apisonmetal.CloudProfileConfig{}
	if config := cluster.CloudProfile.Spec.ProviderConfig; config != nil {
		if _, _, err := vp.Decoder().Decode(config.Raw, nil, &providerConfig); err != nil {
			return nil, fmt.Errorf("could not decode cloudprofile providerConfig for controlplane '%s': %w", kutil.ObjectName(controlPlane), err)
		}
	}
	values := make(map[string]interface{})
	if providerConfig.StorageClasses != nil && len(providerConfig.StorageClasses) != 0 {
		allSc := make([]map[string]interface{}, len(providerConfig.StorageClasses))
		for i, sc := range providerConfig.StorageClasses {
			var storageClassValues = map[string]interface{}{
				"name": sc.Name,
				"type": sc.Type,
			}

			if sc.Default != nil && *sc.Default {
				storageClassValues["default"] = true
			}

			allSc[i] = storageClassValues
		}
		values["storageClasses"] = allSc
	}

	return values, nil
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
		"enabled":     true,
		"replicas":    extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
		"clusterName": cp.Namespace,
		"podNetwork":  extensionscontroller.GetPodNetwork(cluster),
		"podAnnotations": map[string]interface{}{
			"checksum/secret-" + internal.CloudProviderSecretName: checksums[internal.CloudProviderSecretName],
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
	_ *apisonmetal.ControlPlaneConfig,
	_ *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	_ secretsmanager.Reader,
	_ map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	return map[string]interface{}{
		"enabled":  true,
		"replicas": extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
	}, nil
}

// getControlPlaneShootChartValues collects and returns the control plane shoot chart values.
func (vp *valuesProvider) getControlPlaneShootChartValues(
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	if cluster.Shoot == nil {
		return nil, fmt.Errorf("cluster %s does not contain a shoot object", cluster.ObjectMeta.Name)
	}
	csiNodeDriverValues := map[string]interface{}{
		"enabled":     true,
		"vpaEnabled":  gardencorev1beta1helper.ShootWantsVerticalPodAutoscaler(cluster.Shoot),
		"pspDisabled": gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
	}

	return map[string]interface{}{
		onmetal.CloudControllerManagerName: map[string]interface{}{"enabled": true},
		onmetal.CSINodeName:                csiNodeDriverValues,
	}, nil

}
