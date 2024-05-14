// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ironcore

import (
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta

	// NetworkRef references the network to use for the Shoot creation.
	NetworkRef *corev1.LocalObjectReference
	// NATConfig references the configuration to use for the NATGateway
	NATConfig *NATConfig
}

// NATConfig contains configuration about the NATGateway resource
type NATConfig struct {
	// PortsPerNetworkInterface defines the number of ports per network interface the NAT gateway should use.
	// Has to be a power of 2. If empty, 2048 is the default.
	PortsPerNetworkInterface *int32
	// MaxPortsPerNetworkInterface is the maximum number of ports per network interface the NAT gateway should use.
	// If set will be used to auto determine number of ports.
	MaxPortsPerNetworkInterface *int32
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta

	// NetworkRef is the reference to the networked used
	NetworkRef commonv1alpha1.LocalUIDReference
	// NATGatewayRef is the reference to the NAT gateway used
	NATGatewayRef commonv1alpha1.LocalUIDReference
	// PrefixRef is the reference to the Prefix used
	PrefixRef commonv1alpha1.LocalUIDReference
}
