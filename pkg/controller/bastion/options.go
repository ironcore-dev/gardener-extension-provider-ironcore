// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
