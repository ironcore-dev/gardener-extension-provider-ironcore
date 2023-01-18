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
	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/helper"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	"github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	networkFieldOwner    = client.FieldOwner("extension.onmetal.de/network")
	natGatewayFieldOwner = client.FieldOwner("extension.onmetal.de/natgateway")
	prefixFieldOwner     = client.FieldOwner("extension.onmetal.de/prefix")
)

// Reconcile implements infrastructure.Actuator.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	return a.reconcile(ctx, log, infra, cluster)
}

func (a *actuator) reconcile(ctx context.Context, logger logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	config, err := helper.InfrastructureConfigFromInfrastructure(infra)
	if err != nil {
		return err
	}

	// get onmetal credentials from infrastructure config
	onmetalClient, namespace, err := a.getOnmetalClientAndNamespaceFromInfraConfig(infra)
	if err != nil {
		return fmt.Errorf("failed to extract credentials from infrastructure config for shoot %s/%s: %w", cluster.Shoot.Namespace, cluster.Shoot.Name, err)
	}

	// apply infrastructure components in onmetal cluster
	network := &networkingv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			// TODO: use generated Name and store the UIDs in the infrastatus
			// Additional add lables to the created resources to identity resources if no status is present.
			Name: fmt.Sprintf("network-%s", cluster.Shoot.Name),
		},
	}
	if err := onmetalClient.Patch(ctx, network, client.Apply, networkFieldOwner); err != nil {
		return fmt.Errorf("failed to apply network %s: %w", client.ObjectKeyFromObject(network), err)
	}

	natGateway := &networkingv1alpha1.NATGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      cluster.Shoot.Name,
		},
		Spec: networkingv1alpha1.NATGatewaySpec{
			Type:       networkingv1alpha1.NATGatewayTypePublic,
			IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
			NetworkRef: corev1.LocalObjectReference{
				Name: network.Name,
			},
			NetworkInterfaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					// TODO: use a proper label selector to identify the correct shoot worker nodes
					"worker": cluster.Shoot.Spec.Provider.Workers[0].Name,
				},
			},
		},
	}
	if err := onmetalClient.Patch(ctx, natGateway, client.Apply, natGatewayFieldOwner); err != nil {
		return fmt.Errorf("failed to apply natgateway %s: %w", client.ObjectKeyFromObject(natGateway), err)
	}

	prefix := &v1alpha1.Prefix{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      cluster.Shoot.Name,
		},
		Spec: v1alpha1.PrefixSpec{
			IPFamily: corev1.IPv4Protocol,
			Prefix:   commonv1alpha1.MustParseNewIPPrefix(*cluster.Shoot.Spec.Networking.Nodes),
		},
	}
	if err := onmetalClient.Patch(ctx, prefix, client.Apply, prefixFieldOwner); err != nil {
		return fmt.Errorf("failed to apply prefix %s: %w", client.ObjectKeyFromObject(prefix), err)
	}

	// update status
	return a.updateProviderStatus(ctx, infra, config)
}

func (a *actuator) updateProviderStatus(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, config *api.InfrastructureConfig) error {
	return nil
}
