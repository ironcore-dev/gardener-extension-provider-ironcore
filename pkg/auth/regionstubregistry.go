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
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NewRegistryFunc func(client client.Client) (RegionStubRegistry, error)

type RegionStubRegistry interface {
	GetRegionStub(ctx context.Context, region string) (*RegionStub, error)
}

type SimpleRegionStubRegistry struct {
	mu    sync.RWMutex
	items map[string]api.Config
}

func NewSimpleRegionStubRegistry() *SimpleRegionStubRegistry {
	return &SimpleRegionStubRegistry{items: make(map[string]api.Config)}
}

func (s *SimpleRegionStubRegistry) GetRegionStub(ctx context.Context, region string) (*RegionStub, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[region]
	if !ok {
		return nil, errors.NewNotFound(schema.GroupResource{}, fmt.Sprintf("no stub for region %s", region))
	}
	return NewRegionStub(&item)
}

func (s *SimpleRegionStubRegistry) AddRegionStub(region string, clientCfg api.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[region] = clientCfg
}

type SecretRegionStubRegistry struct {
	client    client.Client
	namespace string
}

func NewSecretRegionStubRegistry(c client.Client, namespace string) *SecretRegionStubRegistry {
	return &SecretRegionStubRegistry{
		client:    c,
		namespace: namespace,
	}
}

func (s *SecretRegionStubRegistry) GetRegionStub(ctx context.Context, region string) (*RegionStub, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: s.namespace,
		Name:      fmt.Sprintf("onmetal-region-%s", region),
	}
	if err := s.client.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret for region %s: %w", region, err)
	}
	kubeconfig, ok := secret.Data["kubeconfigStub"]
	if !ok {
		return nil, fmt.Errorf("region %s secret does not contain a kubeconfig stub", region)
	}
	clientCfg, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error loading region %s kubeconfig stub: %w", region, err)
	}
	return NewRegionStub(clientCfg)
}

type RegionStub struct {
	config api.Config
}

func NewRegionStub(clientCfg *api.Config) (*RegionStub, error) {
	// TODO: Validate client cfg to be non-functional for stub to avoid a customer accidentally (by wrong config)
	// 		 exposing a confidential kubeconfig.
	return &RegionStub{config: *clientCfg}, nil
}

func (r *RegionStub) ClientConfig(namespace string, token string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewDefaultClientConfig(r.config, &clientcmd.ConfigOverrides{
		AuthInfo: api.AuthInfo{
			Token: token,
		},
		Context: api.Context{
			Namespace: namespace,
		},
	}), nil
}

func ParseInfraSecret(secret *corev1.Secret) (namespace, token string, err error) {
	namespaceData, ok := secret.Data["namespace"]
	if !ok {
		return "", "", fmt.Errorf("namespace needs to be set")
	}
	tokenData, ok := secret.Data["token"]
	if !ok {
		return "", "", fmt.Errorf("token needs to be set")
	}
	return string(namespaceData), string(tokenData), nil
}
