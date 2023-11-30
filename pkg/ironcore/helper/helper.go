// Copyright 2022 IronCore authors
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

package helper

import (
	"fmt"

	"k8s.io/utils/pointer"

	api "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	apiv1alpha1 "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
)

// FindMachineImage takes a list of machine images and tries to find the first entry
// whose name, version, architecture and zone matches with the given name, version, and zone. If no such entry is
// found then an error will be returned.
func FindMachineImage(machineImages []apiv1alpha1.MachineImage, name, version string, architecture *string) (*apiv1alpha1.MachineImage, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == name && machineImage.Version == version && pointer.StringEqual(architecture, machineImage.Architecture) {
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
				if imageVersion == version.Version && pointer.StringEqual(architecture, version.Architecture) {
					return version.Image, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find an image for name %q and in version %q", imageName, imageVersion)
}
