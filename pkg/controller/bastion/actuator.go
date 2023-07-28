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
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type actuator struct {
	client        client.Client
	bastionConfig *controllerconfig.BastionConfig
}

// NewActuator creates a new bastion.Actuator.
func NewActuator(mgr manager.Manager, bastionConfig *controllerconfig.BastionConfig) bastion.Actuator {
	return &actuator{
		client:        mgr.GetClient(),
		bastionConfig: bastionConfig,
	}
}
