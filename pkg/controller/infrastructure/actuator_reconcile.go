// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"slices"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
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

	natGateway, err := a.applyNATGateway(ctx, config, ironcoreClient, namespace, cluster, network)
	if err != nil {
		return err
	}

	prefixes, servicePrefix, err := a.applyPrefixes(ctx, ironcoreClient, namespace, cluster)
	if err != nil {
		return err
	}

	networkPolicy, err := a.applyNetworkPolicy(ctx, ironcoreClient, namespace, config, cluster, network)
	if err != nil {
		return err
	}

	log.V(2).Info("Successfully reconciled infrastructure")

	return a.updateProviderStatus(ctx, infra, network, natGateway, prefixes, networkPolicy, servicePrefix, cluster)
}

func (a *actuator) applyPrefixes(ctx context.Context, ironcoreClient client.Client, namespace string, cluster *controller.Cluster) ([]ipamv1alpha1.Prefix, *ipamv1alpha1.Prefix, error) {
	prefixIPV4 := &ipamv1alpha1.Prefix{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Prefix",
			APIVersion: "ipam.ironcore.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster) + "-v4",
		},
		Spec: ipamv1alpha1.PrefixSpec{
			// TODO: for now we only support IPv4 until Gardener has support for IPv6 based Shoots
			IPFamily: corev1.IPv4Protocol,
		},
	}

	if nodeCIDR := cluster.Shoot.Spec.Networking.Nodes; nodeCIDR != nil {
		prefixIPV4.Spec.Prefix = v1alpha1.MustParseNewIPPrefix(ptr.Deref[string](nodeCIDR, ""))
	}

	if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, prefixIPV4, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to apply prefix %s: %w", client.ObjectKeyFromObject(prefixIPV4), err)
	}

	prefixes := []ipamv1alpha1.Prefix{*prefixIPV4}
	var servicePrefix *ipamv1alpha1.Prefix
	if slices.Contains(cluster.Shoot.Spec.Networking.IPFamilies, v1beta1.IPFamilyIPv6) {
		// TODO: Get overlay IPv6 Block from Malte
		nodesIPV6Prefix, err := netip.ParsePrefix("2a10:afc0:e010:cafe::/64")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse IPv6 prefix: %w", err)
		}
		prefixIPV6 := &ipamv1alpha1.Prefix{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Prefix",
				APIVersion: "ipam.ironcore.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      generateResourceNameFromCluster(cluster) + "-v6",
			},
			Spec: ipamv1alpha1.PrefixSpec{
				IPFamily: corev1.IPv6Protocol,
				Prefix: &commonv1alpha1.IPPrefix{
					Prefix: nodesIPV6Prefix,
				},
			},
		}
		if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, prefixIPV6, nil); err != nil {
			return nil, nil, fmt.Errorf("failed to apply prefix %s: %w", client.ObjectKeyFromObject(prefixIPV6), err)
		}

		prefixes = append(prefixes, *prefixIPV6)

		maxPrefixLength := 128
		servicePrefixLength := int32((maxPrefixLength-nodesIPV6Prefix.Bits())/2 + nodesIPV6Prefix.Bits())
		servicePrefix = &ipamv1alpha1.Prefix{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Prefix",
				APIVersion: "ipam.ironcore.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      generateResourceNameFromCluster(cluster) + "-services-v6",
			},
			Spec: ipamv1alpha1.PrefixSpec{
				IPFamily:     corev1.IPv6Protocol,
				PrefixLength: servicePrefixLength,
				ParentRef: &corev1.LocalObjectReference{
					Name: prefixIPV6.Name,
				},
			},
		}
		if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, servicePrefix, nil); err != nil {
			return nil, nil, fmt.Errorf("failed to apply prefix %s: %w", client.ObjectKeyFromObject(servicePrefix), err)
		}
	}

	return prefixes, servicePrefix, nil
}

func (a *actuator) applyNATGateway(ctx context.Context, config *api.InfrastructureConfig, ironcoreClient client.Client, namespace string, cluster *controller.Cluster, network *networkingv1alpha1.Network) (*networkingv1alpha1.NATGateway, error) {

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

	if portsPerNetworkInterface := config.NATPortsPerNetworkInterface; natGateway.Spec.IPFamily == corev1.IPv4Protocol && portsPerNetworkInterface != nil {
		if nodeCIDR := cluster.Shoot.Spec.Networking.Nodes; nodeCIDR != nil {
			_, ipv4Net, err := net.ParseCIDR(*nodeCIDR)
			if err != nil {
				return nil, fmt.Errorf("failed to parse node cidr %s: %w", *nodeCIDR, err)
			}

			// determines how many IP addresses reside within nodeCIDR.
			// The first and the last IPs are NOT excluded.
			// see reference https://github.com/cilium/cilium/blob/main/pkg/ip/ip.go#L27
			subnet, size := ipv4Net.Mask.Size()
			amount := big.NewInt(0).Sub(big.NewInt(2).Exp(big.NewInt(2), big.NewInt(int64(size-subnet)), nil), big.NewInt(0))
			maxPorts := big.NewInt(int64(ironcore.MaxAvailableNATPortsPerNetworkInterface))
			ports := big.NewInt(0).Div(maxPorts, amount)

			if ports.Int64() < int64(*portsPerNetworkInterface) {
				natGateway.Spec.PortsPerNetworkInterface = portsPerNetworkInterface
			} else {
				natGateway.Spec.PortsPerNetworkInterface = ptr.To(previousPowOf2(int32(ports.Int64())))
			}
		}
	}

	if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, natGateway, nil); err != nil {
		return nil, fmt.Errorf("failed to apply natgateway %s: %w", client.ObjectKeyFromObject(natGateway), err)
	}
	return natGateway, nil
}

func previousPowOf2(n int32) int32 {
	n = n | (n >> 1)
	n = n | (n >> 2)
	n = n | (n >> 4)
	n = n | (n >> 8)
	n = n | (n >> 16)
	return n - (n >> 1)
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

func (a *actuator) applyNetworkPolicy(ctx context.Context, ironcoreClient client.Client, namespace string, config *api.InfrastructureConfig, cluster *controller.Cluster, network *networkingv1alpha1.Network) (*networkingv1alpha1.NetworkPolicy, error) {
	if config != nil && config.NetworkPolicyRef != nil {
		networkPolicy := &networkingv1alpha1.NetworkPolicy{}
		networkKey := client.ObjectKey{Namespace: namespace, Name: config.NetworkRef.Name}
		if err := ironcoreClient.Get(ctx, networkKey, networkPolicy); err != nil {
			return nil, fmt.Errorf("failed to get network policy %s: %w", networkKey, err)
		}
		return networkPolicy, nil
	}

	networkPolicy := &networkingv1alpha1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.ironcore.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      generateResourceNameFromCluster(cluster),
		},
		Spec: networkingv1alpha1.NetworkPolicySpec{
			NetworkRef: corev1.LocalObjectReference{
				Name: network.Name,
			},
			NetworkInterfaceSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					ironcore.ClusterNameLabel: cluster.ObjectMeta.Name,
				},
			},
			Ingress: []networkingv1alpha1.NetworkPolicyIngressRule{},
			Egress:  []networkingv1alpha1.NetworkPolicyEgressRule{},
			PolicyTypes: []networkingv1alpha1.PolicyType{
				networkingv1alpha1.PolicyTypeIngress,
				networkingv1alpha1.PolicyTypeEgress,
			},
		},
	}

	if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, networkPolicy, nil); err != nil {
		return nil, fmt.Errorf("failed to apply network policy %s: %w", client.ObjectKeyFromObject(networkPolicy), err)
	}
	return networkPolicy, nil
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
	prefixes []ipamv1alpha1.Prefix,
	networkPolicy *networkingv1alpha1.NetworkPolicy,
	servicePrefix *ipamv1alpha1.Prefix,
	cluster *controller.Cluster,
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
		NetworkPolicyRef: v1alpha1.LocalUIDReference{
			Name: networkPolicy.Name,
			UID:  networkPolicy.UID,
		},
	}
	var (
		nodes    []string
		pods     []string
		services []string
	)
	if cluster.Shoot.Spec.Networking.Pods != nil {
		pods = []string{*cluster.Shoot.Spec.Networking.Pods}
	}
	if cluster.Shoot.Spec.Networking.Services != nil {
		services = []string{*cluster.Shoot.Spec.Networking.Services}
	}
	for _, prefix := range prefixes {
		infraStatus.PrefixRefs = append(infraStatus.PrefixRefs,
			v1alpha1.LocalUIDReference{
				Name: prefix.Name,
				UID:  prefix.UID,
			},
		)
		nodes = append(nodes, prefix.Spec.Prefix.Prefix.String())
		if prefix.Spec.IPFamily == corev1.IPv6Protocol {
			// for IPv6 the pods and the nodes share the same prefix
			pods = append(pods, prefix.Spec.Prefix.Prefix.String())
		}
	}
	if servicePrefix != nil {
		if servicePrefix.Status.Phase != ipamv1alpha1.PrefixPhaseAllocated {
			return fmt.Errorf("service prefix not yet allocated")
		}
		if servicePrefix.Spec.Prefix != nil {
			services = append(services, servicePrefix.Spec.Prefix.Prefix.String())
		}
	}
	infra.Status.Networking = &extensionsv1alpha1.InfrastructureStatusNetworking{
		Nodes:    nodes,
		Pods:     pods,
		Services: services,
	}
	infraBase := infra.DeepCopy()
	infra.Status.ProviderStatus = &runtime.RawExtension{
		Object: infraStatus,
	}
	return a.client.Status().Patch(ctx, infra, client.MergeFrom(infraBase))
}
