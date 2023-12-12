// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta `json:",inline"`

	// NetworkRef references the network to use for the Shoot creation.
	NetworkRef *corev1.LocalObjectReference `json:"networkRef,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta `json:",inline"`

	// NetworkRef is the reference to the networked used
	NetworkRef commonv1alpha1.LocalUIDReference `json:"networkRef,omitempty"`
	// NATGatewayRef is the reference to the NAT gateway used
	NATGatewayRef commonv1alpha1.LocalUIDReference `json:"natGatewayRef,omitempty"`
	// PrefixRef is the reference to the Prefix used
	PrefixRef commonv1alpha1.LocalUIDReference `json:"prefixRef,omitempty"`
}
