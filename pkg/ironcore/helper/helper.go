// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"fmt"

	"k8s.io/utils/ptr"

	api "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	apiv1alpha1 "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
)

// FindMachineImage takes a list of machine images and tries to find the first entry
// whose name, version, architecture and zone matches with the given name, version, and zone. If no such entry is
// found then an error will be returned.
func FindMachineImage(machineImages []apiv1alpha1.MachineImage, name, version string, architecture *string) (*apiv1alpha1.MachineImage, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == name && machineImage.Version == version && ptr.Equal[string](architecture, machineImage.Architecture) {
			return &machineImage, nil
		}
	}
	return nil, fmt.Errorf("no machine image found with name %q and version %q", name, version)
}

// FindImageFromCloudProfile takes a list of machine images, and the desired image name and version. It tries
// to find the image with the given name, architecture and version in the desired cloud profile. If it cannot be found then an error
// is returned.
func FindImageFromCloudProfile(cloudProfileConfig *api.CloudProfileConfig, imageName, imageVersion string, architecture *string) (string, error) {
	if cloudProfileConfig != nil {
		for _, machineImage := range cloudProfileConfig.MachineImages {
			if machineImage.Name != imageName {
				continue
			}
			for _, version := range machineImage.Versions {
				if imageVersion == version.Version && ptr.Equal[string](architecture, version.Architecture) {
					return version.Image, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find an image for name %q and in version %q", imageName, imageVersion)
}
