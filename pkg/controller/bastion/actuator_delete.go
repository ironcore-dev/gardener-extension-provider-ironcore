// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Delete implements bastion.Actuator.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) error {
	log.V(2).Info("Deleting bastion host")

	ironcoreClient, namespace, err := ironcore.GetIroncoreClientAndNamespaceFromCloudProviderSecret(ctx, a.client, cluster.ObjectMeta.Name)
	if err != nil {
		return fmt.Errorf("failed to get ironcore client and namespace from cloudprovider secret: %w", err)
	}

	bastionHostName, err := generateBastionHostResourceName(cluster.ObjectMeta.Name, bastion)
	if err != nil {
		return err
	}
	bastionHost := &computev1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      bastionHostName,
		},
	}
	if err := ironcoreClient.Delete(ctx, bastionHost); err != nil {
		return fmt.Errorf("failed to delete bastion host: %v", err)
	}

	log.V(2).Info("Deleted bastion host")
	return nil
}
