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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
// resource.
type CloudProfileConfig struct {
	metav1.TypeMeta `json:",inline"`
	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to provider-specific identifiers.
	MachineImages []MachineImages `json:"machineImages"`
	// RegionConfigs is the list of supported regions.
	RegionConfigs []RegionConfig `json:"regionConfigs,omitempty"`
	// StorageClasses defines the storage classes for the shoot
	// +optional
	StorageClasses []StorageClassDefinition `json:"storageClasses,omitempty"`
}

// StorageClassDefinition is a definition of a storageClass
type StorageClassDefinition struct {
	// Name is the name of the storageclass
	Name string `json:"name"`
	// Default set the storageclass to the default one
	// +optional
	Default *bool `json:"default,omitempty"`
	// Type is referring to the VolumeClass to use for this StorageClass
	Type string `json:"type"`
}

// MachineImages is a mapping from logical names and versions to provider-specific identifiers.
type MachineImages struct {
	// Name is the logical name of the machine image.
	Name string `json:"name"`
	// Versions contains versions and a provider-specific identifier.
	Versions []MachineImageVersion `json:"versions"`
}

// RegionConfig is the definition of a region.
type RegionConfig struct {
	// Name is the name of a region.
	Name string `json:"name"`
	// Server is the server endpoint of this region.
	Server string `json:"server"`
	// CertificateAuthorityData is the CA data of the region server.
	CertificateAuthorityData []byte `json:"certificateAuthorityData"`
}

// MachineImageVersion contains a version and a provider-specific identifier.
type MachineImageVersion struct {
	// Version is the version of the image.
	Version string `json:"version"`
	// Image is the path to the image.
	Image string `json:"image"`
	// Architecture is the CPU architecture of the machine image.
	// +optional
	Architecture *string `json:"architecture,omitempty"`
}
