// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	api "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/helper"
	apiv1alpha1 "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

const (
	shootPrefix = "shoot"
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

	// get ironcore credentials from infrastructure config
	ironcoreClient, namespace, err := ironcore.GetIroncoreClientAndNamespaceFromCloudProviderSecret(ctx, a.client, cluster.ObjectMeta.Name)
	if err != nil {
		return fmt.Errorf("failed to get ironcore client and namespace from cloudprovider secret: %w", err)
	}

	network, err := a.applyNetwork(ctx, ironcoreClient, namespace, config, cluster)
	if err != nil {
		return err
	}

	natGateway, err := a.applyNATGateway(ctx, ironcoreClient, namespace, cluster, network)
	if err != nil {
		return err
	}

	prefix, err := a.applyPrefix(ctx, ironcoreClient, namespace, cluster)
	if err != nil {
		return err
	}

	log.V(2).Info("Successfully reconciled infrastructure")

	// update status
	return a.updateProviderStatus(ctx, infra, network, natGateway, prefix)
}

func (a *actuator) applyPrefix(ctx context.Context, ironcoreClient client.Client, namespace string, cluster *controller.Cluster) (*ipamv1alpha1.Prefix, error) {
	prefix := &ipamv1alpha1.Prefix{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Prefix",
			APIVersion: "ipam.ironcore.dev/v1alpha1",
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
		prefix.Spec.Prefix = v1alpha1.MustParseNewIPPrefix(ptr.Deref[string](nodeCIDR, ""))
	}

	if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, prefix, nil); err != nil {
		return nil, fmt.Errorf("failed to apply prefix %s: %w", client.ObjectKeyFromObject(prefix), err)
	}

	return prefix, nil
}

func (a *actuator) applyNATGateway(ctx context.Context, ironcoreClient client.Client, namespace string, cluster *controller.Cluster, network *networkingv1alpha1.Network) (*networkingv1alpha1.NATGateway, error) {
	natGateway := &networkingv1alpha1.NATGateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NATGateway",
			APIVersion: "networking.ironcore.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
		Spec: networkingv1alpha1.NATGatewaySpec{
			Type: networkingv1alpha1.NATGatewayTypePublic,
			// TODO: for now we only support IPv4 until Gardener has support for IPv6 based Shoots
			IPFamily: corev1.IPv4Protocol,
			NetworkRef: corev1.LocalObjectReference{
				Name: network.Name,
			},
		},
	}

	if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, natGateway, nil); err != nil {
		return nil, fmt.Errorf("failed to apply natgateway %s: %w", client.ObjectKeyFromObject(natGateway), err)
	}
	return natGateway, nil
}

func (a *actuator) applyNetwork(ctx context.Context, ironcoreClient client.Client, namespace string, config *api.InfrastructureConfig, cluster *controller.Cluster) (*networkingv1alpha1.Network, error) {
	if config != nil && config.NetworkRef != nil {
		network := &networkingv1alpha1.Network{}
		networkKey := client.ObjectKey{Namespace: namespace, Name: config.NetworkRef.Name}
		if err := ironcoreClient.Get(ctx, networkKey, network); err != nil {
			return nil, fmt.Errorf("failed to get network %s: %w", networkKey, err)
		}
		return network, nil
	}

	network := &networkingv1alpha1.Network{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Network",
			APIVersion: "networking.ironcore.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
	}

	if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, network, nil); err != nil {
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
	return a.client.Status().Patch(ctx, infra, client.MergeFrom(infraBase))
}
