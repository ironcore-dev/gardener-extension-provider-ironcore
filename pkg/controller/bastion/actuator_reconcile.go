// Copyright 2023 OnMetal authors
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
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	reconcilerutils "github.com/gardener/gardener/pkg/controllerutils/reconciler"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
	ipamv1alpha1 "github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/go-logr/logr"
	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/controller/bastion/ignition"

	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/helper"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// bastionEndpoints collects the endpoints the bastion host provides; the
// private endpoint is important for opening a port on the worker node
// ingress network policy rule to allow SSH from that node, the public endpoint is where
// the end user connects to establish the SSH connection.
type bastionEndpoints struct {
	private *corev1.LoadBalancerIngress
	public  *corev1.LoadBalancerIngress
}

// Reconcile implements bastion.Actuator.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) error {
	err := bastionConfigCheck(a.bastionConfig)
	if err != nil {
		return err
	}

	opt, err := DetermineOptions(bastion, cluster)
	if err != nil {
		return fmt.Errorf("failed to determine options: %w", err)
	}

	// get onmetal credentials from infrastructure config
	onmetalClient, namespace, err := onmetal.GetOnmetalClientAndNamespaceFromCloudProviderSecret(ctx, a.Client(), cluster.ObjectMeta.Name)
	if err != nil {
		return fmt.Errorf("failed to get onmetal client and namespace from cloudprovider secret: %w", err)
	}
	infraStatus, err := getInfrastructureStatus(ctx, onmetalClient, cluster)
	if err != nil {
		return fmt.Errorf("failed to get infrastructure status: %w", err)
	}

	// TODO: Add NetworkPolicy related implementation

	ignitionSecret, err := createIgnitionSecret(namespace, opt)
	if err != nil {
		return fmt.Errorf("failed to create ignition secret: %w", err)
	}

	bastionHost := createMachine(namespace, a.bastionConfig, ignitionSecret, infraStatus, opt)

	machine, err := applyMachine(ctx, onmetalClient, ignitionSecret, bastionHost)
	if err != nil {
		return fmt.Errorf("failed to create machine: %w", err)
	}

	// check if the machine have private and virtual IPs
	endpoints, err := getMachineEndpoints(machine)
	if err != nil {
		return fmt.Errorf("failed to get machine endpoints: %w", err)
	}

	if !endpoints.Ready() {
		return &reconcilerutils.RequeueAfterError{
			// requeue rather soon, so that the user (most likely gardenctl eventually)
			// doesn't have to wait too long for the public endpoint to become available
			RequeueAfter: 5 * time.Second,
			Cause:        fmt.Errorf("bastion instance has no public/private endpoints yet"),
		}
	}

	// once a public endpoint is available, publish the endpoint on the
	// Bastion resource to notify upstream about the ready instance
	patch := client.MergeFrom(bastion.DeepCopy())
	bastion.Status.Ingress = endpoints.public
	return a.Client().Status().Patch(ctx, bastion, patch)
}

func getMachineEndpoints(machine *computev1alpha1.Machine) (*bastionEndpoints, error) {
	if machine == nil {
		return nil, fmt.Errorf("machine can't be nil")
	}

	if machine.Status.State != computev1alpha1.MachineStateRunning {
		return nil, fmt.Errorf("machine not running, status: %s", machine.Status.State)
	}

	endpoints := &bastionEndpoints{}

	if len(machine.Status.NetworkInterfaces) == 0 {
		return nil, fmt.Errorf("no network interface found for machine: %s", machine.Name)
	}

	if len(machine.Status.NetworkInterfaces[0].IPs) == 0 {
		return nil, fmt.Errorf("no private ip found for network interface for machine: %s", machine.Name)
	}

	privateIP := machine.Status.NetworkInterfaces[0].IPs[0].String()
	if ingress := addressToIngress(&machine.Name, &privateIP); ingress != nil {
		endpoints.private = ingress
	}

	if machine.Status.NetworkInterfaces[0].VirtualIP == nil {
		return nil, fmt.Errorf("no virtual ip found for network interface for machine: %s", machine.Name)
	}

	virtualIP := machine.Status.NetworkInterfaces[0].VirtualIP.String()
	if ingress := addressToIngress(&machine.Name, &virtualIP); ingress != nil {
		endpoints.public = ingress
	}

	return endpoints, nil
}

func bastionConfigCheck(bastionConfig *controllerconfig.BastionConfig) error {
	if bastionConfig == nil {
		return errors.New("bastionConfig must not be empty")
	}

	if bastionConfig.MachineClassName == "" {
		return errors.New("bastion not supported as no flavor is configured for the bastion host machine")
	}

	if bastionConfig.Image == "" {
		return errors.New("bastion not supported as no Image is configured for the bastion host machine")
	}
	return nil
}

func getInfrastructureStatus(ctx context.Context, c client.Client, cluster *controller.Cluster) (*api.InfrastructureStatus, error) {
	worker := &extensionsv1alpha1.Worker{}
	err := c.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, worker)
	if err != nil {
		return nil, err
	}

	if worker == nil || worker.Spec.InfrastructureProviderStatus == nil {
		return nil, errors.New("infrastructure provider status must be not empty for worker")
	}

	return helper.InfrastructureStatusFromRaw(worker.Spec.InfrastructureProviderStatus)
}

func applyMachine(ctx context.Context, onmetalClient client.Client, ignitionSecret *corev1.Secret, bastionHost *computev1alpha1.Machine) (*computev1alpha1.Machine, error) {

	if err := controllerutil.SetOwnerReference(bastionHost, ignitionSecret, onmetalClient.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference for ignition secret %s: %w", client.ObjectKeyFromObject(ignitionSecret), err)
	}

	_, err := controllerutil.CreateOrPatch(ctx, onmetalClient, ignitionSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create or patch ignition secret %s for bastion host %s: %w", client.ObjectKeyFromObject(ignitionSecret), client.ObjectKeyFromObject(bastionHost), err)
	}

	_, err = controllerutil.CreateOrPatch(ctx, onmetalClient, bastionHost, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create or patch bastion host machine %s: %w", client.ObjectKeyFromObject(bastionHost), err)
	}

	return bastionHost, nil
}

func createIgnitionSecret(namespace string, opt *Options) (*corev1.Secret, error) {
	// Construct ignition file config
	config := &ignition.Config{
		Hostname:   opt.BastionInstanceName,
		UserData:   string(opt.UserData),
		DnsServers: []netip.Addr{netip.MustParseAddr("8.8.8.8")},
	}

	ignitionContent, err := ignition.File(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ignition file for machine %s: %w", opt.BastionInstanceName, err)
	}

	ignitionData := map[string][]byte{}
	ignitionData[computev1alpha1.DefaultIgnitionKey] = []byte(ignitionContent)
	ignitionSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getIgnitionNameForMachine(opt.BastionInstanceName),
			Namespace: namespace,
		},
		Data: ignitionData,
	}

	return ignitionSecret, nil
}

func createMachine(namespace string, bastionConfig *controllerconfig.BastionConfig, ignitionSecret *corev1.Secret, infraStatus *api.InfrastructureStatus, opt *Options) *computev1alpha1.Machine {
	bastionHost := &computev1alpha1.Machine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: computev1alpha1.SchemeGroupVersion.String(),
			Kind:       "Machine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opt.BastionInstanceName,
			Namespace: namespace,
		},
		Spec: computev1alpha1.MachineSpec{
			MachineClassRef: corev1.LocalObjectReference{
				Name: bastionConfig.MachineClassName,
			},
			Power: computev1alpha1.PowerOn,
			NetworkInterfaces: []computev1alpha1.NetworkInterface{
				{
					Name: "primary",
					NetworkInterfaceSource: computev1alpha1.NetworkInterfaceSource{
						Ephemeral: &computev1alpha1.EphemeralNetworkInterfaceSource{
							NetworkInterfaceTemplate: &networkingv1alpha1.NetworkInterfaceTemplateSpec{
								Spec: networkingv1alpha1.NetworkInterfaceSpec{
									NetworkRef: corev1.LocalObjectReference{
										Name: infraStatus.NetworkRef.Name,
									},
									IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
									IPs: []networkingv1alpha1.IPSource{
										{
											Ephemeral: &networkingv1alpha1.EphemeralPrefixSource{
												PrefixTemplate: &ipamv1alpha1.PrefixTemplateSpec{
													Spec: ipamv1alpha1.PrefixSpec{
														// request single IP
														PrefixLength: 32,
														ParentRef: &corev1.LocalObjectReference{
															Name: infraStatus.PrefixRef.Name,
														},
													},
												},
											},
										},
									},
									VirtualIP: &networkingv1alpha1.VirtualIPSource{
										Ephemeral: &networkingv1alpha1.EphemeralVirtualIPSource{
											VirtualIPTemplate: &networkingv1alpha1.VirtualIPTemplateSpec{
												Spec: networkingv1alpha1.VirtualIPSpec{
													Type:     networkingv1alpha1.VirtualIPTypePublic,
													IPFamily: corev1.IPv4Protocol,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			IgnitionRef: &commonv1alpha1.SecretKeySelector{
				Name: ignitionSecret.Name,
			},
		},
	}

	return bastionHost
}

func getIgnitionNameForMachine(machineName string) string {
	return fmt.Sprintf("%s-%s", machineName, "ignition")
}

// addressToIngress converts the IP address into a
// corev1.LoadBalancerIngress resource. If both arguments are nil, then
// nil is returned.
func addressToIngress(dnsName *string, ipAddress *string) *corev1.LoadBalancerIngress {
	var ingress *corev1.LoadBalancerIngress

	if ipAddress != nil || dnsName != nil {
		ingress = &corev1.LoadBalancerIngress{}
		if dnsName != nil {
			ingress.Hostname = *dnsName
		}

		if ipAddress != nil {
			ingress.IP = *ipAddress
		}
	}

	return ingress
}

// Ready returns true if both public and private interfaces each have either
// an IP or a hostname or both.
func (be *bastionEndpoints) Ready() bool {
	return be != nil && IngressReady(be.private) && IngressReady(be.public)
}

// IngressReady returns true if either an IP or a hostname or both are set.
func IngressReady(ingress *corev1.LoadBalancerIngress) bool {
	return ingress != nil && (ingress.Hostname != "" || ingress.IP != "")
}
