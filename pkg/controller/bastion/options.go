// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bastion

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	//maxLengthForBaseName for "base" name due to fact that we use this name to name other openstack resources,
	maxLengthForBaseName = 33
)

// Options contains provider-related information required for setting up
// a bastion instance. This struct combines precomputed values like the
// bastion instance name with the IDs of pre-existing cloud provider
// resources, like the nic name, subnet name etc.
type Options struct {
	BastionInstanceName string
	Region              string
	ShootName           string
	SecretReference     corev1.SecretReference
	//TODO: add securityGroup/networkPolicy once implemented
	UserData []byte
}

// DetermineOptions determines the required information that are required to reconcile a Bastion on onmetal. This
// function does not create any IaaS resources.
func DetermineOptions(bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) (*Options, error) {
	clusterName := cluster.ObjectMeta.Name
	region := cluster.Shoot.Spec.Region

	baseResourceName, err := generateBastionBaseResourceName(clusterName, bastion)
	if err != nil {
		return nil, err
	}

	secretReference := corev1.SecretReference{
		Namespace: clusterName,
		Name:      v1beta1constants.SecretNameCloudProvider,
	}

	return &Options{
		ShootName:           clusterName,
		BastionInstanceName: baseResourceName,
		SecretReference:     secretReference,
		//TODO: add securityGroup/networkPolicy once implemented
		Region:   region,
		UserData: []byte(base64.StdEncoding.EncodeToString(bastion.Spec.UserData)),
	}, nil
}

func generateBastionBaseResourceName(clusterName string, bastion *extensionsv1alpha1.Bastion) (string, error) {
	bastionName := bastion.Name
	if bastionName == "" {
		return "", fmt.Errorf("bastionName can't be empty")
	}
	if clusterName == "" {
		return "", fmt.Errorf("clusterName can't be empty")
	}
	staticName := clusterName + "-" + bastionName
	nameSuffix := strings.Split(string(bastion.UID), "-")[0]
	return fmt.Sprintf("%s-bastion-%s", staticName, nameSuffix), nil
}
