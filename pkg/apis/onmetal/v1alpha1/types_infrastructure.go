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

package v1alpha1

import (
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
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
	// NATPortsPerNetworkInterface defines the number of ports per network interface the NAT gateway should use.
	// Has to be a power of 2. If empty, 2048 is the default.
	NATPortsPerNetworkInterface *int32 `json:"natPortsPerNetworkInterface,omitempty"`
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
