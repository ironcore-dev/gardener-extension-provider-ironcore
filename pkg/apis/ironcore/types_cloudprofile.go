// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ironcore

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
// resource.
type CloudProfileConfig struct {
	metav1.TypeMeta
	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to provider-specific identifiers.
	MachineImages []MachineImages
	// RegionConfigs is the list of supported regions.
	RegionConfigs []RegionConfig
	// StorageClasses defines the DefaultStrorageClass and AdditionalStoreClasses for the shoot
	StorageClasses StorageClasses
}

// StorageClasses is a definition of a storageClasses
type StorageClasses struct {
	// Default defines the default storage class for the shoot
	Default *StorageClass
	// Additional defines the additional storage classes for the shoot
	Additional []StorageClass
}

// StorageClass is the definition of a storageClass
type StorageClass struct {
	// Name is the name of the storageclass
	Name string
	// Type is referring to the VolumeClass to use for this StorageClass
	Type string
}

// MachineImages is a mapping from logical names and versions to provider-specific identifiers.
type MachineImages struct {
	// Name is the logical name of the machine image.
	Name string
	// Versions contains versions and a provider-specific identifier.
	Versions []MachineImageVersion
}

// RegionConfig is the definition of a region.
type RegionConfig struct {
	// Name is the name of a region.
	Name string
	// Server is the server endpoint of this region.
	Server string
	// CertificateAuthorityData is the CA data of the region server.
	CertificateAuthorityData []byte
}

// MachineImageVersion contains a version and a provider-specific identifier.
type MachineImageVersion struct {
	// Version is the version of the image.
	Version string
	// Image is the path to the image.
	Image string
	// Architecture is the CPU architecture of the machine image.
	Architecture *string
}
