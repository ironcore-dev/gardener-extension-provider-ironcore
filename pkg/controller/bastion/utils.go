// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"context"
	"fmt"
	"net"
	"strings"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// generateBastionHostResourceName returns a unique name for the Bastion host in
// the Gardener Kubernetes cluster, based on the cluster name, Bastion name, and
// UID. The function concatenates these values and truncates the resulting
// string to 63 characters, if necessary, to comply with the Kubernetes naming
// convention. In rare cases, this truncation may result in non-unique names,
// but the likelihood of this happening is extremely low.
func generateBastionHostResourceName(clusterName string, bastion *extensionsv1alpha1.Bastion) (string, error) {
	bastionName := bastion.Name
	if bastionName == "" {
		return "", fmt.Errorf("bastionName can not be empty")
	}
	if clusterName == "" {
		return "", fmt.Errorf("clusterName can not be empty")
	}
	staticName := clusterName + "-" + bastionName
	nameSuffix := strings.Split(string(bastion.UID), "-")[0]
	name := fmt.Sprintf("%s-bastion-%s", staticName, nameSuffix)
	if len(name) > 63 {
		name = name[:63]
	}
	return name, nil
}

func getIgnitionNameForMachine(machineName string) string {
	return fmt.Sprintf("%s-%s", machineName, "ignition")
}

// getPrivateAndVirtualIPsFromNetworkInterfaces extracts the private IPv4 and
// virtual IPv4 addresses from the given slice of NetworkInterfaceStatus
// objects.
//
// If a network interface has multiple private IPs, only the first one will be
// returned. If multiple network interfaces have a virtual IP, only the first
// one will be returned.
//
// TODO: IPv6 addresses are ignored for now and will be
// added in the future once Gardener extension supports IPv6.
func getPrivateAndVirtualIPsFromNetworkInterfaces(ctx context.Context, networkInterfaces []computev1alpha1.NetworkInterfaceStatus, irocoreClient client.Client, namespace string) (string, string, error) {
	var privateIP, virtualIP string
	for _, machineStatusNetworkInterface := range networkInterfaces {
		nicName := machineStatusNetworkInterface.NetworkInterfaceRef.Name
		// Fetch the NetworkInterface object
		nic := &networkingv1alpha1.NetworkInterface{}
		if err := irocoreClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: nicName}, nic); err != nil {
			return "", "", fmt.Errorf("failed to get NetworkInterface %s/%s: %v", namespace, nicName, err)
		}

		for _, ip := range nic.Status.IPs {
			parsedIP := net.ParseIP(ip.String())
			if parsedIP == nil {
				continue // skip invalid IP
			}
			if parsedIP.To4() != nil {
				privateIP = parsedIP.String()
				break
			} else {
				// IPv6 case
				continue
			}
		}
		if nic.Status.VirtualIP != nil {
			parsedIP := net.ParseIP(nic.Status.VirtualIP.String())
			if parsedIP == nil {
				continue // skip invalid IP
			}
			if parsedIP.To4() != nil {
				virtualIP = parsedIP.String()
				break
			} else {
				// IPv6 case
				continue
			}
		}
	}
	if privateIP == "" {
		return "", "", fmt.Errorf("private IPv4 address not found")
	}
	if virtualIP == "" {
		return "", "", fmt.Errorf("virtual IPv4 address not found")
	}

	return privateIP, virtualIP, nil
}
