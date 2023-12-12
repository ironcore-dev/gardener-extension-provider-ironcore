// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type actuator struct {
	client client.Client
}

// NewActuator creates a new infrastructure.Actuator.
func NewActuator(mgr manager.Manager) infrastructure.Actuator {
	return &actuator{
		client: mgr.GetClient(),
	}
}
