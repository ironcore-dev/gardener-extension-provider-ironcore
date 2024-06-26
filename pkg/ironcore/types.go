// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ironcore

import (
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

const (
	// ProviderName is the name of the ironcore provider.
	ProviderName = "provider-ironcore"

	// CloudControllerManagerImageName is the name of the cloud-controller-manager image.
	CloudControllerManagerImageName = "cloud-controller-manager"
	// CSIDriverImageName is the name of the csi-driver image.
	CSIDriverImageName = "csi-driver"
	// CSIProvisionerImageName is the name of the csi-provisioner image.
	CSIProvisionerImageName = "csi-provisioner"
	// CSIAttacherImageName is the name of the csi-attacher image.
	CSIAttacherImageName = "csi-attacher"
	// CSIResizerImageName is the name of the csi-resizer image.
	CSIResizerImageName = "csi-resizer"
	// CSINodeDriverRegistrarImageName is the name of the csi-node-driver-registrar image.
	CSINodeDriverRegistrarImageName = "csi-node-driver-registrar"
	// CSILivenessProbeImageName is the name of the csi-liveness-probe image.
	CSILivenessProbeImageName = "csi-liveness-probe"
	// MachineControllerManagerImageName is the name of the MachineControllerManager image.
	MachineControllerManagerImageName = "machine-controller-manager"
	// MachineControllerManagerProviderIroncoreImageName is the name of the MachineController ironcore image.
	MachineControllerManagerProviderIroncoreImageName = "machine-controller-manager-provider-ironcore"

	// AccessKeyID is a constant for the key in a ironcore bucket access secret that holds the Bucket access key id.
	BucketAccessKeyID = "AWS_ACCESS_KEY_ID"
	// AwsSecretAccessKey is a constant for the key in a ironcore bucket access secret that holds the Bucket secret access key.
	BucketSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	// AccessKeyID is a constant for the key in a cloud provider secret and backup secret that holds the Bucket access key id.
	AccessKeyID = "accessKeyID"
	// SecretAccessKey is a constant for the key in a cloud provider secret and backup secret that holds the Bucket secret access key.
	SecretAccessKey = "secretAccessKey"
	//Endpoint
	Endpoint = "endpoint"
	// UsernameFieldName is the field in a secret where the namespace is stored at.
	UsernameFieldName = "username"
	// NamespaceFieldName is the field in a secret where the namespace is stored at.
	NamespaceFieldName = "namespace"
	// KubeConfigFieldName is containing the effective kubeconfig to access an ironcore cluster.
	KubeConfigFieldName = "kubeconfig"
	// TokenFieldName is containing the token to access an ironcore cluster.
	TokenFieldName = "token"
	// NetworkFieldName is the name of network field
	NetworkFieldName = "networkName"
	// PrefixFieldName is the name of the prefix field
	PrefixFieldName = "prefixName"
	// ClusterFieldName is the name of the cluster field
	ClusterFieldName = "clusterName"
	// LabelsFieldName is the name of the labels field
	LabelsFieldName = "labels"
	// UserDataFieldName is the name of the user data field
	UserDataFieldName = "userData"
	// ImageFieldName is the name of the image field
	ImageFieldName = "image"
	// RootDiskFieldName is the name of the root disk field
	RootDiskFieldName = "rootDisk"
	// SizeFieldName is the name of the size field
	SizeFieldName = "size"
	// VolumeClassFieldName is the name of the volume class field
	VolumeClassFieldName = "volumeClassName"
	// ClusterNameLabel is the name is the label key of the cluster name
	ClusterNameLabel = "extension.ironcore.dev/cluster-name"

	// CloudProviderConfigName is the name of the secret containing the cloud provider config.
	CloudProviderConfigName = "cloud-provider-config"
	// CloudControllerManagerName is a constant for the name of the CloudController deployed by the worker controller.
	CloudControllerManagerName = "cloud-controller-manager"
	// CSIControllerName is a constant for the name of the CSI controller deployment in the seed.
	CSIControllerName = "csi-driver-controller"
	// CSIControllerObservabilityConfigName is the name of the ConfigMap containing monitoring and logging stack configurations for csi-driver.
	CSIControllerObservabilityConfigName = "csi-driver-controller-observability-config"
	// CSINodeName is a constant for the name of the CSI node deployment in the shoot.
	CSINodeName = "csi-driver-node"
	// CSIDriverName is a constant for the name of the csi-driver component.
	CSIDriverName = "csi-driver"
	// CSIProvisionerName is a constant for the name of the csi-provisioner component.
	CSIProvisionerName = "csi-provisioner"
	// CSIAttacherName is a constant for the name of the csi-attacher component.
	CSIAttacherName = "csi-attacher"
	// CSIResizerName is a constant for the name of the csi-resizer component.
	CSIResizerName = "csi-resizer"
	// CSINodeDriverRegistrarName is a constant for the name of the csi-node-driver-registrar component.
	CSINodeDriverRegistrarName = "csi-node-driver-registrar"
	// CSILivenessProbeName is a constant for the name of the csi-liveness-probe component.
	CSILivenessProbeName = "csi-liveness-probe"
	// CSIStorageProvisioner is a constant with the storage provisioner name which is used in storageclasses.
	CSIStorageProvisioner = "ironcore-csi-driver"
	// MachineControllerManagerName is a constant for the name of the machine-controller-manager.
	MachineControllerManagerName = "machine-controller-manager"
	// MachineControllerManagerVpaName is the name of the VerticalPodAutoscaler of the machine-controller-manager deployment.
	MachineControllerManagerVpaName = "machine-controller-manager-vpa"
	// MachineControllerManagerMonitoringConfigName is the name of the ConfigMap containing monitoring stack configurations for machine-controller-manager.
	MachineControllerManagerMonitoringConfigName = "machine-controller-manager-monitoring-config"
	// MaxAvailableNATPortsPerNetworkInterface defines the maximum number of ports per network interface the NAT gateway should use.
	MaxAvailableNATPortsPerNetworkInterface = 64512
)

var (
	// UsernamePrefix is a constant for the username prefix of components deployed by ironcore.
	UsernamePrefix = extensionsv1alpha1.SchemeGroupVersion.Group + ":" + ProviderName + ":"
)
