// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/chart"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/charts"
	apisironcore "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/internal"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

const (
	caNameControlPlane                   = "ca-" + ironcore.ProviderName + "-controlplane"
	cloudControllerManagerDeploymentName = "cloud-controller-manager"
	cloudControllerManagerServerName     = "cloud-controller-manager-server"
)

func secretConfigsFunc(namespace string) []extensionssecretsmanager.SecretConfigWithOptions {
	return []extensionssecretsmanager.SecretConfigWithOptions{
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:       caNameControlPlane,
				CommonName: caNameControlPlane,
				CertType:   secretutils.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        cloudControllerManagerServerName,
				CommonName:                  ironcore.CloudControllerManagerName,
				DNSNames:                    kutil.DNSNamesForService(ironcore.CloudControllerManagerName, namespace),
				CertType:                    secretutils.ServerCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
	}
}

func shootAccessSecretsFunc(namespace string) []*gutil.AccessSecret {
	return []*gutil.AccessSecret{
		gutil.NewShootAccessSecret(cloudControllerManagerDeploymentName, namespace),
		gutil.NewShootAccessSecret(ironcore.CSIProvisionerName, namespace),
		gutil.NewShootAccessSecret(ironcore.CSIAttacherName, namespace),
		gutil.NewShootAccessSecret(ironcore.CSIResizerName, namespace),
		// TODO: This needs to be fixed!!!
		//		 Since the csi controller needs to access the Node resources in the Shoot cluster,
		//		 it should use the same ServiceAccount as the csi-driver-node in the Shoot. That way
		//		 the correct ClusterRolebindings will be used for both components.
		gutil.NewShootAccessSecret(ironcore.CSINodeName, namespace),
	}
}

var (
	configChart = &chart.Chart{
		Name:       "cloud-provider-config",
		EmbeddedFS: charts.InternalChart,
		Path:       filepath.Join(charts.InternalChartsPath, "cloud-provider-config"),
		Objects: []*chart.Object{
			{Type: &corev1.ConfigMap{}, Name: internal.CloudProviderConfigMapName},
		},
	}

	controlPlaneChart = &chart.Chart{
		Name:       "seed-controlplane",
		EmbeddedFS: charts.InternalChart,
		Path:       filepath.Join(charts.InternalChartsPath, "seed-controlplane"),
		SubCharts: []*chart.Chart{
			{
				Name:   ironcore.CloudControllerManagerName,
				Images: []string{ironcore.CloudControllerManagerImageName},
				Objects: []*chart.Object{
					{Type: &corev1.Service{}, Name: "cloud-controller-manager"},
					{Type: &appsv1.Deployment{}, Name: "cloud-controller-manager"},
					{Type: &corev1.ConfigMap{}, Name: "cloud-controller-manager-observability-config"},
					{Type: &autoscalingv1.VerticalPodAutoscaler{}, Name: "cloud-controller-manager-vpa"},
				},
			},
			{
				Name: ironcore.CSIControllerName,
				Images: []string{
					ironcore.CSIDriverImageName,
					ironcore.CSIProvisionerImageName,
					ironcore.CSIAttacherImageName,
					ironcore.CSIResizerImageName,
					ironcore.CSILivenessProbeImageName,
				},
				Objects: []*chart.Object{
					// csi-driver-controller
					{Type: &appsv1.Deployment{}, Name: ironcore.CSIControllerName},
					{Type: &corev1.ConfigMap{}, Name: ironcore.CSIControllerObservabilityConfigName},
					{Type: &autoscalingv1.VerticalPodAutoscaler{}, Name: ironcore.CSIControllerName + "-vpa"},
				},
			},
		},
	}

	controlPlaneShootChart = &chart.Chart{
		Name:       "shoot-system-components",
		EmbeddedFS: charts.InternalChart,
		Path:       filepath.Join(charts.InternalChartsPath, "shoot-system-components"),
		SubCharts: []*chart.Chart{
			{
				Name: "cloud-controller-manager",
				Path: filepath.Join(charts.InternalChartsPath, "cloud-controller-manager"),
				Objects: []*chart.Object{
					{Type: &rbacv1.ClusterRole{}, Name: "system:controller:cloud-node-controller"},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:controller:cloud-node-controller"},
					{Type: &rbacv1.ClusterRole{}, Name: "ironcore:cloud-provider"},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: "ironcore:cloud-provider"},
				},
			},
			{
				Name: ironcore.CSINodeName,
				Images: []string{
					ironcore.CSIDriverImageName,
					ironcore.CSINodeDriverRegistrarImageName,
					ironcore.CSILivenessProbeImageName,
				},
				Objects: []*chart.Object{
					// csi-driver
					{Type: &appsv1.DaemonSet{}, Name: ironcore.CSINodeName},
					{Type: &storagev1.CSIDriver{}, Name: ironcore.CSIStorageProvisioner},
					{Type: &corev1.ServiceAccount{}, Name: ironcore.CSIDriverName},
					{Type: &rbacv1.ClusterRole{}, Name: ironcore.UsernamePrefix + ironcore.CSIDriverName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: ironcore.UsernamePrefix + ironcore.CSIDriverName},
					// csi-provisioner
					{Type: &rbacv1.ClusterRole{}, Name: ironcore.UsernamePrefix + ironcore.CSIProvisionerName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: ironcore.UsernamePrefix + ironcore.CSIProvisionerName},
					{Type: &rbacv1.Role{}, Name: ironcore.UsernamePrefix + ironcore.CSIProvisionerName},
					{Type: &rbacv1.RoleBinding{}, Name: ironcore.UsernamePrefix + ironcore.CSIProvisionerName},
					// csi-attacher
					{Type: &rbacv1.ClusterRole{}, Name: ironcore.UsernamePrefix + ironcore.CSIAttacherName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: ironcore.UsernamePrefix + ironcore.CSIAttacherName},
					{Type: &rbacv1.Role{}, Name: ironcore.UsernamePrefix + ironcore.CSIAttacherName},
					{Type: &rbacv1.RoleBinding{}, Name: ironcore.UsernamePrefix + ironcore.CSIAttacherName},
					// csi-resizer
					{Type: &rbacv1.ClusterRole{}, Name: ironcore.UsernamePrefix + ironcore.CSIResizerName},
					{Type: &rbacv1.ClusterRoleBinding{}, Name: ironcore.UsernamePrefix + ironcore.CSIResizerName},
					{Type: &rbacv1.Role{}, Name: ironcore.UsernamePrefix + ironcore.CSIResizerName},
					{Type: &rbacv1.RoleBinding{}, Name: ironcore.UsernamePrefix + ironcore.CSIResizerName},
				},
			},
		},
	}

	storageClassChart = &chart.Chart{
		Name:       "shoot-storageclasses",
		EmbeddedFS: charts.InternalChart,
		Path:       filepath.Join(charts.InternalChartsPath, "shoot-storageclasses"),
	}
)

// valuesProvider is a ValuesProvider that provides ironcore-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	client  client.Client
	decoder runtime.Decoder
}

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider(mgr manager.Manager) genericactuator.ValuesProvider {
	return &valuesProvider{
		client:  mgr.GetClient(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

func (vp *valuesProvider) GetControlPlaneExposureChartValues(ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	checksums map[string]string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// GetConfigChartValues returns the values for the config chart applied by the generic actuator.
func (vp *valuesProvider) GetConfigChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	infrastructureStatus := &apisironcore.InfrastructureStatus{}
	if _, _, err := vp.decoder.Decode(cp.Spec.InfrastructureProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return nil, fmt.Errorf("failed to decode infrastructure status: %w", err)
	}
	// Collect config chart values
	return map[string]interface{}{
		ironcore.NetworkFieldName: infrastructureStatus.NetworkRef.Name,
		ironcore.PrefixFieldName:  infrastructureStatus.PrefixRef.Name,
		ironcore.ClusterFieldName: cluster.ObjectMeta.Name,
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
	cpConfig := &apisironcore.ControlPlaneConfig{}
	if cp.Spec.ProviderConfig != nil {
		if _, _, err := vp.decoder.Decode(cp.Spec.ProviderConfig.Raw, nil, cpConfig); err != nil {
			return nil, fmt.Errorf("could not decode providerConfig of controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
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
	ctx context.Context,
	controlPlane *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	providerConfig := apisironcore.CloudProfileConfig{}
	if config := cluster.CloudProfile.Spec.ProviderConfig; config != nil {
		if _, _, err := vp.decoder.Decode(config.Raw, nil, &providerConfig); err != nil {
			return nil, fmt.Errorf("could not decode cloudprofile providerConfig for controlplane '%s': %w", client.ObjectKeyFromObject(controlPlane), err)
		}
	}

	values := make(map[string]interface{})
	var defaultStorageClass int
	if providerConfig.StorageClasses.Default != nil {
		defaultStorageClass++
	}

	// get ironcore credentials from infrastructure config
	ironcoreClient, _, err := ironcore.GetIroncoreClientAndNamespaceFromCloudProviderSecret(ctx, vp.client, cluster.ObjectMeta.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get ironcore client and namespace from cloudprovider secret: %w", err)
	}

	var expandable bool
	storageClasses := make([]map[string]interface{}, 0, len(providerConfig.StorageClasses.Additional)+defaultStorageClass)
	if providerConfig.StorageClasses.Default != nil {
		if expandable, err = isVolumeClassExpandable(ctx, ironcoreClient, providerConfig.StorageClasses.Default); err != nil {
			return nil, fmt.Errorf("could not get resize policy from volumeclass : %w", err)
		}

		storageClasses = append(storageClasses, map[string]interface{}{
			StorageClassNameKeyName:       providerConfig.StorageClasses.Default.Name,
			StorageClassTypeKeyName:       providerConfig.StorageClasses.Default.Type,
			StorageClassDefaultKeyName:    true,
			StorageClassExpandableKeyName: expandable,
		})
	}
	for _, sc := range providerConfig.StorageClasses.Additional {
		if expandable, err = isVolumeClassExpandable(ctx, ironcoreClient, &sc); err != nil {
			return nil, fmt.Errorf("could not get resize policy from volumeclass : %w", err)
		}
		storageClasses = append(storageClasses, map[string]interface{}{
			StorageClassNameKeyName:       sc.Name,
			StorageClassTypeKeyName:       sc.Type,
			StorageClassExpandableKeyName: expandable,
		})
	}

	values["storageClasses"] = storageClasses

	return values, nil
}

func isVolumeClassExpandable(ctx context.Context, ironcoreClient client.Client, storageClass *apisironcore.StorageClass) (bool, error) {
	volumeClass := &storagev1alpha1.VolumeClass{}
	if err := ironcoreClient.Get(ctx, client.ObjectKey{Name: storageClass.Type}, volumeClass); err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("VolumeClass not found")
		}
		return false, fmt.Errorf("could not get volumeclass: %w", err)
	}
	return volumeClass.ResizePolicy == storagev1alpha1.ResizePolicyExpandOnly, nil
}

// getControlPlaneChartValues collects and returns the control plane chart values.
func getControlPlaneChartValues(
	cpConfig *apisironcore.ControlPlaneConfig,
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
		ironcore.CloudControllerManagerName: ccm,
		ironcore.CSIControllerName:          csi,
	}, nil
}

// getCCMChartValues collects and returns the CCM chart values.
func getCCMChartValues(
	cpConfig *apisironcore.ControlPlaneConfig,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	serverSecret, found := secretsReader.Get(cloudControllerManagerServerName)
	if !found {
		return nil, fmt.Errorf("secret %q not found", cloudControllerManagerServerName)
	}

	values := map[string]interface{}{
		"enabled":     true,
		"replicas":    extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
		"clusterName": cp.Namespace,
		"podNetwork":  strings.Join(extensionscontroller.GetPodNetwork(cluster), ","),
		"podAnnotations": map[string]interface{}{
			"checksum/secret-" + internal.CloudProviderConfigMapName: checksums[internal.CloudProviderConfigMapName],
		},
		"podLabels": map[string]interface{}{
			v1beta1constants.LabelPodMaintenanceRestart: "true",
		},
		"tlsCipherSuites": kutil.TLSCipherSuites,
		"secrets": map[string]interface{}{
			"server": serverSecret.Name,
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
	}

	overlayEnabled, err := isOverlayEnabled(cluster.Shoot.Spec.Networking)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if overlay is enabled: %w", err)
	}
	values["configureCloudRoutes"] = !overlayEnabled

	return values, nil
}

func isOverlayEnabled(networking *gardencorev1beta1.Networking) (bool, error) {
	if networking == nil || networking.ProviderConfig == nil {
		return false, nil
	}

	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, networking.ProviderConfig.Raw)
	if err != nil {
		return false, err
	}

	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return false, fmt.Errorf("object %T is not an unstructured.Unstructured", obj)
	}

	enabled, ok, err := unstructured.NestedBool(u.UnstructuredContent(), "overlay", "enabled")
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	return enabled, nil
}

// getCSIControllerChartValues collects and returns the CSIController chart values.
func getCSIControllerChartValues(
	_ *apisironcore.ControlPlaneConfig,
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
func (vp *valuesProvider) getControlPlaneShootChartValues(cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	if cluster.Shoot == nil {
		return nil, fmt.Errorf("cluster %s does not contain a shoot object", cluster.ObjectMeta.Name)
	}
	csiNodeDriverValues := map[string]interface{}{
		"enabled": true,
	}

	return map[string]interface{}{
		ironcore.CloudControllerManagerName: map[string]interface{}{"enabled": true},
		ironcore.CSINodeName:                csiNodeDriverValues,
	}, nil

}
