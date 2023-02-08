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
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/auth"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
	ipamv1alpha1 "github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	clientConfigGetter auth.ClientConfigGetter
}

func (a *actuator) getClientConfigForInfra(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) (clientcmd.ClientConfig, error) {
	secretKey := client.ObjectKey{Namespace: infra.Spec.SecretRef.Namespace, Name: infra.Spec.SecretRef.Name}
	clientCfg, err := a.clientConfigGetter.GetClientConfig(ctx, infra.Spec.Region, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get client config from infra secret %s: %w", secretKey, err)
	}
	return clientCfg, nil
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

// NewActuator creates a new infrastructure.Actuator.
func NewActuator(configGetter auth.ClientConfigGetter) infrastructure.Actuator {
	return &actuator{
		clientConfigGetter: configGetter,
	}
}
