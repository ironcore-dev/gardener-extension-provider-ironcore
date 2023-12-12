// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete implements infrastructure.Actuator.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	log.V(2).Info("Deleting infrastructure")

	// get ironcore credentials from infrastructure config
	ironcoreClient, namespace, err := ironcore.GetIroncoreClientAndNamespaceFromCloudProviderSecret(ctx, a.client, cluster.ObjectMeta.Name)
	if err != nil {
		return fmt.Errorf("failed to get ironcore client and namespace from cloudprovider secret: %w", err)
	}

	if err := a.deletePrefix(ctx, ironcoreClient, namespace, cluster); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete infrastructure: %w", err)
	}

	if err := a.deleteNATGateway(ctx, ironcoreClient, namespace, cluster); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete infrastructure: %w", err)
	}

	if err := a.deleteNetwork(ctx, ironcoreClient, namespace, cluster); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete infrastructure: %w", err)
	}

	log.V(2).Info("Successfully deleted infrastructure")
	return nil
}

func (a *actuator) deletePrefix(ctx context.Context, ironcoreClient client.Client, namespace string, cluster *controller.Cluster) error {
	prefix := &ipamv1alpha1.Prefix{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}
	return ironcoreClient.Delete(ctx, prefix)
}

func (a *actuator) deleteNATGateway(ctx context.Context, ironcoreClient client.Client, namespace string, cluster *controller.Cluster) error {
	natGateway := &networkingv1alpha1.NATGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}
	return ironcoreClient.Delete(ctx, natGateway)
}

func (a *actuator) deleteNetwork(ctx context.Context, ironcoreClient client.Client, namespace string, cluster *controller.Cluster) error {
	network := &networkingv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}
	return ironcoreClient.Delete(ctx, network)
}
