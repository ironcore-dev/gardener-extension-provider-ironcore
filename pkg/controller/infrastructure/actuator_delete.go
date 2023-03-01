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

package infrastructure

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	ipamv1alpha1 "github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete implements infrastructure.Actuator.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	log.V(2).Info("Deleting infrastructure")

	// get onmetal credentials from infrastructure config
	onmetalClient, namespace, err := a.getOnmetalClientAndNamespaceFromCloudProviderSecret(ctx, infra)
	if err != nil {
		return fmt.Errorf("failed to get onmetal client and namespace from cloudprovider secret: %w", err)
	}

	if err := a.deletePrefix(ctx, onmetalClient, namespace, cluster); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete infrastructure: %w", err)
	}

	if err := a.deleteNATGateway(ctx, onmetalClient, namespace, cluster); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete infrastructure: %w", err)
	}

	if err := a.deleteNetwork(ctx, onmetalClient, namespace, cluster); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete infrastructure: %w", err)
	}

	log.V(2).Info("Successfully deleted infrastructure")
	return nil
}

func (a *actuator) deletePrefix(ctx context.Context, onmetalClient client.Client, namespace string, cluster *controller.Cluster) error {
	prefix := &ipamv1alpha1.Prefix{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}
	return onmetalClient.Delete(ctx, prefix)
}

func (a *actuator) deleteNATGateway(ctx context.Context, onmetalClient client.Client, namespace string, cluster *controller.Cluster) error {
	natGateway := &networkingv1alpha1.NATGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}
	return onmetalClient.Delete(ctx, natGateway)
}

func (a *actuator) deleteNetwork(ctx context.Context, onmetalClient client.Client, namespace string, cluster *controller.Cluster) error {
	network := &networkingv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}
	return onmetalClient.Delete(ctx, network)
}
