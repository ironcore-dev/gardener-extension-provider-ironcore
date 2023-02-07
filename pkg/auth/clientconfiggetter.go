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

package auth

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientConfigGetter interface {
	GetClientConfig(ctx context.Context, region string, secretKey client.ObjectKey) (clientcmd.ClientConfig, error)
}

type clientConfigGetter struct {
	client   client.Client
	registry RegionStubRegistry
}

func NewClientConfigGetter(c client.Client, registry RegionStubRegistry) ClientConfigGetter {
	return &clientConfigGetter{
		client:   c,
		registry: registry,
	}
}

func (c *clientConfigGetter) GetClientConfig(ctx context.Context, region string, secretKey client.ObjectKey) (clientcmd.ClientConfig, error) {
	regionStub, err := c.registry.GetRegionStub(ctx, region)
	if err != nil {
		// TODO: handle not found
		return nil, fmt.Errorf("")
	}

	secret := &corev1.Secret{}
	if err := c.client.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get infrastructure secret %s: %w", secretKey, err)
	}

	namespace, token, err := ParseInfraSecret(secret)
	if err != nil {
		return nil, err
	}

	return regionStub.ClientConfig(namespace, token)
}
