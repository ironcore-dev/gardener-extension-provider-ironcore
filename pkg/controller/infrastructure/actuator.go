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
	"fmt"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
	ipamv1alpha1 "github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

var onmetalScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(networkingv1alpha1.AddToScheme(onmetalScheme))
	utilruntime.Must(computev1alpha1.AddToScheme(onmetalScheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(onmetalScheme))
	utilruntime.Must(ipamv1alpha1.AddToScheme(onmetalScheme))
}

type actuator struct {
	common.RESTConfigContext
	disableProjectedTokenMount bool
	regionStubRegistry         RegionStubRegistry
}

type RegionStubRegistry interface {
	GetRegionStub(ctx context.Context, region string) (*RegionStub, error)
}

type SimpleRegionStubRegistry struct {
	mu    sync.RWMutex
	items map[string]clientcmdapi.Config
}

func NewSimpleRegionStubRegistry() *SimpleRegionStubRegistry {
	return &SimpleRegionStubRegistry{items: make(map[string]clientcmdapi.Config)}
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

func (s *SimpleRegionStubRegistry) AddRegionStub(region string, clientCfg clientcmdapi.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[region] = clientCfg
}

type SecretRegionStubRegistry struct {
	client    client.Client
	namespace string
}

func (s *SecretRegionStubRegistry) GetRegionStub(ctx context.Context, region string) (*RegionStub, error) {
	secret := &v1.Secret{}
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
	config clientcmdapi.Config
}

func NewRegionStub(clientCfg *clientcmdapi.Config) (*RegionStub, error) {
	// TODO: Validate client cfg to be non-functional for stub to avoid a customer accidentally (by wrong config)
	// 		 exposing a confidential kubeconfig.
	return &RegionStub{config: *clientCfg}, nil
}

func (r *RegionStub) ClientConfig(namespace string, token string) (clientcmd.ClientConfig, error) {
	return clientcmd.NewDefaultClientConfig(r.config, &clientcmd.ConfigOverrides{
		AuthInfo: clientcmdapi.AuthInfo{
			Token: token,
		},
		Context: clientcmdapi.Context{
			Namespace: namespace,
		},
	}), nil
}

func (a *actuator) getClientConfigForInfra(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) (clientcmd.ClientConfig, error) {
	regionStub, err := a.regionStubRegistry.GetRegionStub(ctx, infra.Spec.Region)
	if err != nil {
		// TODO: handle not found
		return nil, fmt.Errorf("")
	}

	secret := &v1.Secret{}
	secretKey := client.ObjectKey{Namespace: infra.Namespace, Name: infra.Spec.SecretRef.Name}
	if err := a.Client().Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get infrastructure secret %s: %w", secretKey, err)
	}

	namespace, token, err := ParseInfraSecret(secret)
	if err != nil {
		return nil, err
	}

	return regionStub.ClientConfig(namespace, token)
}

func (a *actuator) newClientFromConfig(clientCfg clientcmd.ClientConfig) (client.Client, string, error) {
	cfg, err := clientCfg.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("error getting client config: %w", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: onmetalScheme})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create client: %w", err)
	}
	namespace, _, err := clientCfg.Namespace()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get namespace from client config: %w", err)
	}
	return c, namespace, nil
}

func ParseInfraSecret(secret *v1.Secret) (namespace, token string, err error) {
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

// NewActuator creates a new infrastructure.Actuator.
func NewActuator() infrastructure.Actuator {
	return &actuator{}
}
