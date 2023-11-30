// Copyright 2023 IronCore authors
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
	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

// Options contains provider-related information required for setting up
// a bastion instance. This struct combines precomputed values like the
// bastion instance name with the IDs of pre-existing cloud provider
// resources, like the nic name, subnet name etc.
type Options struct {
	BastionInstanceName string
	UserData            []byte
}

// DetermineOptions determines the required information that are required to reconcile a Bastion on ironcore. This
// function does not create any IaaS resources.
func DetermineOptions(bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) (*Options, error) {
	clusterName := cluster.ObjectMeta.Name
	baseResourceName, err := generateBastionHostResourceName(clusterName, bastion)
	if err != nil {
		return nil, err
	}
	return &Options{
		BastionInstanceName: baseResourceName,
		UserData:            bastion.Spec.UserData,
	}, nil
}
