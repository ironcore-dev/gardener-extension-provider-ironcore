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

package config

import (
	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/apis/config"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfig "k8s.io/component-base/config"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the onmetal provider.
type ControllerConfiguration struct {
	metav1.TypeMeta

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection *componentbaseconfig.ClientConnectionConfiguration
	// ETCD is the etcd configuration.
	ETCD ETCD
	// HealthCheckConfig is the config for the health check controller
	HealthCheckConfig *healthcheckconfig.HealthCheckConfig
	// FeatureGates is a map of feature names to bools that enable
	// or disable alpha/experimental features.
	// Default: nil
	FeatureGates map[string]bool
	// Regions is a list of supported region mappings of the onmetal extension controller.
	Regions []Region
}

type Region struct {
	// Name is the name of the region
	Name string
	// KubeConfigStub is the kubeconfig go template to use as a base for establishing the connection
	// to the onmetal cluster using the user access token and namespace.
	KubeConfigStub string
}

// ETCD is an etcd configuration.
type ETCD struct {
	// ETCDStorage is the etcd storage configuration.
	Storage ETCDStorage
	// ETCDBackup is the etcd backup configuration.
	Backup ETCDBackup
}

// ETCDStorage is an etcd storage configuration.
type ETCDStorage struct {
	// ClassName is the name of the storage class used in etcd-main volume claims.
	ClassName *string
	// Capacity is the storage capacity used in etcd-main volume claims.
	Capacity *resource.Quantity
}

// ETCDBackup is an etcd backup configuration.
type ETCDBackup struct {
	// Schedule is the etcd backup schedule.
	Schedule *string
}
