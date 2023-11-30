// Copyright 2023 IronCore authors
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
	"fmt"
	"net"
	"strings"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
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
func getPrivateAndVirtualIPsFromNetworkInterfaces(networkInterfaces []computev1alpha1.NetworkInterfaceStatus) (string, string, error) {
	var privateIP, virtualIP string

	for _, ni := range networkInterfaces {
		if ni.IPs == nil {
			return "", "", fmt.Errorf("no private ip found for network interface: %s", ni.Name)
		}
		for _, ip := range ni.IPs {
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
		if ni.VirtualIP != nil {
			parsedIP := net.ParseIP(ni.VirtualIP.String())
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
