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
	apiv1alpha1 "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	"github.com/onmetal/onmetal-api/api/common/v1alpha1"
	ipamv1alpha1 "github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	shootPrefix = "shoot"
)

var (
	networkFieldOwner    = client.FieldOwner("extension.gardener.onmetal.de/network")
	natGatewayFieldOwner = client.FieldOwner("extension.gardener.onmetal.de/natgateway")
	prefixFieldOwner     = client.FieldOwner("extension.gardener.onmetal.de/prefix")
)

// Reconcile implements infrastructure.Actuator.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	return a.reconcile(ctx, log, infra, cluster)
}

func (a *actuator) reconcile(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	log.V(2).Info("Reconciling infrastructure")

	config, err := helper.InfrastructureConfigFromInfrastructure(infra)
	if err != nil {
		return err
	}

	// get onmetal credentials from infrastructure config
	onmetalClient, namespace, err := a.getOnmetalClientAndNamespaceFromCloudProviderSecret(ctx, infra)
	if err != nil {
		return fmt.Errorf("failed to get onmetal client and namespace from cloudprovider secret: %w", err)
	}

	network, err := a.applyNetwork(ctx, onmetalClient, namespace, config, cluster)
	if err != nil {
		return err
	}

	natGateway, err := a.applyNATGateway(ctx, onmetalClient, namespace, cluster, network)
	if err != nil {
		return err
	}

	prefix, err := a.applyPrefix(ctx, onmetalClient, namespace, cluster)
	if err != nil {
		return err
	}

	log.V(2).Info("Successfully reconciled infrastructure")

	// update status
	return a.updateProviderStatus(ctx, infra, network, natGateway, prefix)
}

func (a *actuator) applyPrefix(ctx context.Context, onmetalClient client.Client, namespace string, cluster *controller.Cluster) (*ipamv1alpha1.Prefix, error) {
	prefix := &ipamv1alpha1.Prefix{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Prefix",
			APIVersion: "ipam.api.onmetal.de/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
		Spec: ipamv1alpha1.PrefixSpec{
			// TODO: for now we only support IPv4 until Gardener has support for IPv6 based Shoots
			IPFamily: corev1.IPv4Protocol,
		},
	}

	if nodeCIDR := cluster.Shoot.Spec.Networking.Nodes; nodeCIDR != nil {
		prefix.Spec.Prefix = v1alpha1.MustParseNewIPPrefix(pointer.StringDeref(nodeCIDR, ""))
	}

	if err := onmetalClient.Patch(ctx, prefix, client.Apply, prefixFieldOwner); err != nil {
		return nil, fmt.Errorf("failed to apply prefix %s: %w", client.ObjectKeyFromObject(prefix), err)
	}
	return prefix, nil
}

func (a *actuator) applyNATGateway(ctx context.Context, onmetalClient client.Client, namespace string, cluster *controller.Cluster, network *networkingv1alpha1.Network) (*networkingv1alpha1.NATGateway, error) {
	natGateway := &networkingv1alpha1.NATGateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NATGateway",
			APIVersion: "networking.api.onmetal.de/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
		Spec: networkingv1alpha1.NATGatewaySpec{
			Type: networkingv1alpha1.NATGatewayTypePublic,
			// TODO: for now we only support IPv4 until Gardener has support for IPv6 based Shoots
			IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
			IPs: []networkingv1alpha1.NATGatewayIP{
				{
					Name: "primary",
				},
			},
			NetworkRef: corev1.LocalObjectReference{
				Name: network.Name,
			},
		},
	}

	natGateway.Spec.NetworkInterfaceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			onmetal.ClusterNameLabel: cluster.ObjectMeta.Name,
		},
	}

	if err := onmetalClient.Patch(ctx, natGateway, client.Apply, natGatewayFieldOwner); err != nil {
		return nil, fmt.Errorf("failed to apply natgateway %s: %w", client.ObjectKeyFromObject(natGateway), err)
	}
	return natGateway, nil
}

func (a *actuator) applyNetwork(ctx context.Context, onmetalClient client.Client, namespace string, config *api.InfrastructureConfig, cluster *controller.Cluster) (*networkingv1alpha1.Network, error) {
	if config != nil && config.NetworkRef != nil {
		network := &networkingv1alpha1.Network{}
		networkKey := client.ObjectKey{Namespace: namespace, Name: config.NetworkRef.Name}
		if err := onmetalClient.Get(ctx, networkKey, network); err != nil {
			return nil, fmt.Errorf("failed to get network %s: %w", networkKey, err)
		}
		return network, nil
	}

	network := &networkingv1alpha1.Network{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Network",
			APIVersion: "networking.api.onmetal.de/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}

	if err := onmetalClient.Patch(ctx, network, client.Apply, networkFieldOwner); err != nil {
		return nil, fmt.Errorf("failed to apply network %s: %w", client.ObjectKeyFromObject(network), err)
	}
	return network, nil
}

func generateResourceNameFromCluster(cluster *controller.Cluster) string {
	// TODO: use cluster.Name
	// alternatively shoot.status.technicalID
	return fmt.Sprintf("%s--%s--%s", shootPrefix, cluster.Shoot.Namespace, cluster.Shoot.Name)
}

func (a *actuator) updateProviderStatus(
	ctx context.Context,
	infra *extensionsv1alpha1.Infrastructure,
	network *networkingv1alpha1.Network,
	natGateway *networkingv1alpha1.NATGateway,
	prefix *ipamv1alpha1.Prefix,
) error {
	infraStatus := &apiv1alpha1.InfrastructureStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureStatus",
		},
		NetworkRef: v1alpha1.LocalUIDReference{
			Name: network.Name,
			UID:  network.UID,
		},
		NATGatewayRef: v1alpha1.LocalUIDReference{
			Name: natGateway.Name,
			UID:  natGateway.UID,
		},
		PrefixRef: v1alpha1.LocalUIDReference{
			Name: prefix.Name,
			UID:  prefix.UID,
		},
	}
	infraBase := infra.DeepCopy()
	infra.Status.ProviderStatus = &runtime.RawExtension{
		Object: infraStatus,
	}
	return a.Client().Status().Patch(ctx, infra, client.MergeFrom(infraBase))
}
