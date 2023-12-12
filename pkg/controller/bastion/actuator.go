// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	controllerconfig "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/config"
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
