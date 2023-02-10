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
	"encoding/json"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/helper"
	"github.com/onmetal/onmetal-api/api/common/v1alpha1"
	ipamv1alpha1 "github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	workerNameKey = "worker-name"
	shootPrefix   = "shoot"
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
	log.V(2).Info("Reconciling infrastructure for Shoot", "Shoot", client.ObjectKeyFromObject(cluster.Shoot))

	config, err := helper.InfrastructureConfigFromInfrastructure(infra)
	if err != nil {
		return err
	}

	// get onmetal credentials from infrastructure config
	onmetalClientCfg, err := a.getClientConfigForInfra(ctx, infra)
	if err != nil {
		return fmt.Errorf("failed to extract credentials from infrastructure config for shoot %s/%s: %w", cluster.Shoot.Namespace, cluster.Shoot.Name, err)
	}

	onmetalClient, namespace, err := a.newClientFromConfig(onmetalClientCfg)
	if err != nil {
		return err
	}

	network, err := a.applyNetwork(ctx, onmetalClient, namespace, infra, config, cluster)
	if err != nil {
		return err
	}

	natGateway, err := a.applyNATGateway(ctx, onmetalClient, namespace, infra, cluster, network)
	if err != nil {
		return err
	}

	prefix, err := a.applyPrefix(ctx, onmetalClient, namespace, infra, cluster)
	if err != nil {
		return err
	}

	log.V(2).Info("Successfully reconciled infrastructure for Shoot", "Shoot", client.ObjectKeyFromObject(cluster.Shoot))

	// update status
	return a.updateProviderStatus(ctx, infra, network, natGateway, prefix)
}

func (a *actuator) applyPrefix(ctx context.Context, onmetalClient client.Client, namespace string, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) (*ipamv1alpha1.Prefix, error) {
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

	if err := controllerutil.SetControllerReference(infra, prefix, a.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on prefix %s: %w", client.ObjectKeyFromObject(prefix), err)
	}

	if err := onmetalClient.Patch(ctx, prefix, client.Apply, prefixFieldOwner); err != nil {
		return nil, fmt.Errorf("failed to apply prefix %s: %w", client.ObjectKeyFromObject(prefix), err)
	}
	return prefix, nil
}

func (a *actuator) applyNATGateway(ctx context.Context, onmetalClient client.Client, namespace string, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster, network *networkingv1alpha1.Network) (*networkingv1alpha1.NATGateway, error) {
	// find all worker names to use them as a label selector for the NATGateway
	workerNames := make([]string, 0, len(cluster.Shoot.Spec.Provider.Workers))
	for _, w := range cluster.Shoot.Spec.Provider.Workers {
		workerNames = append(workerNames, w.Name)
	}

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
			NetworkRef: corev1.LocalObjectReference{
				Name: network.Name,
			},
		},
	}

	if len(workerNames) > 0 {
		natGateway.Spec.NetworkInterfaceSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      workerNameKey,
					Operator: metav1.LabelSelectorOpIn,
					Values:   workerNames,
				},
			},
		}
	}

	if err := controllerutil.SetControllerReference(infra, natGateway, a.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on natgateway %s: %w", client.ObjectKeyFromObject(natGateway), err)
	}

	if err := onmetalClient.Patch(ctx, natGateway, client.Apply, natGatewayFieldOwner); err != nil {
		return nil, fmt.Errorf("failed to apply natgateway %s: %w", client.ObjectKeyFromObject(natGateway), err)
	}
	return natGateway, nil
}

func (a *actuator) applyNetwork(ctx context.Context, onmetalClient client.Client, namespace string, infra *extensionsv1alpha1.Infrastructure, config *api.InfrastructureConfig, cluster *controller.Cluster) (*networkingv1alpha1.Network, error) {
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

	if err := controllerutil.SetControllerReference(infra, network, a.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on network %s: %w", client.ObjectKeyFromObject(network), err)
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
	// TODO: use api/infrastatus -> serialize
	providerStatus := map[string]interface{}{
		"networkRef": map[string]interface{}{
			"name": network.Name,
			"uid":  network.UID,
		},
		"natGatewayRef": map[string]interface{}{
			"name": natGateway.Name,
			"uid":  natGateway.UID,
		},
		"prefixRef": map[string]interface{}{
			"name": prefix.Name,
			"uid":  prefix.UID,
		},
	}
	providerStatusJSON, err := json.Marshal(providerStatus)
	if err != nil {
		return fmt.Errorf("failed to encode provider status for infra %s: %w", client.ObjectKeyFromObject(infra), err)
	}
	infraBase := infra.DeepCopy()
	infra.Status.ProviderStatus = &runtime.RawExtension{
		Raw: providerStatusJSON,
	}
	return a.Client().Status().Patch(ctx, infra, client.MergeFrom(infraBase))
}
