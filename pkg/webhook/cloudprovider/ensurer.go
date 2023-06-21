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

package cloudprovider

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/webhook/cloudprovider"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisonmetal "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
)

// NewEnsurer creates cloudprovider ensurer.
func NewEnsurer(logger logr.Logger) cloudprovider.Ensurer {
	return &ensurer{
		logger: logger,
	}
}

type ensurer struct {
	logger  logr.Logger
	client  client.Client
	decoder runtime.Decoder
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

func (e *ensurer) InjectScheme(scheme *runtime.Scheme) error {
	e.decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	return nil
}

// EnsureCloudProviderSecret ensures that cloudprovider secret contains
// the shared credentials file.
func (e *ensurer) EnsureCloudProviderSecret(ctx context.Context, gctx gcontext.GardenContext, newCloudProviderSecret, _ *corev1.Secret) error {
	token, ok := newCloudProviderSecret.Data[onmetal.TokenFieldName]
	if !ok {
		return fmt.Errorf("could not mutate cloudprovider secret as %q field is missing", onmetal.TokenFieldName)
	}
	namespace, ok := newCloudProviderSecret.Data[onmetal.NamespaceFieldName]
	if !ok {
		return fmt.Errorf("could not mutate cloudprovider secret as %q field is missing", onmetal.NamespaceFieldName)
	}
	username, ok := newCloudProviderSecret.Data[onmetal.UsernameFieldName]
	if !ok {
		return fmt.Errorf("could not mutate cloud provider secret as %q fied is missing", onmetal.UsernameFieldName)
	}

	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	cloudProfileConfig := &apisonmetal.CloudProfileConfig{}
	raw, err := cluster.CloudProfile.Spec.ProviderConfig.MarshalJSON()
	if err != nil {
		return fmt.Errorf("could not decode cluster object's providerConfig: %w", err)
	}
	if _, _, err := e.decoder.Decode(raw, nil, cloudProfileConfig); err != nil {
		return fmt.Errorf("could not decode cluster object's providerConfig: %w", err)
	}

	kubeconfig := &clientcmdv1.Config{
		CurrentContext: cluster.Shoot.Spec.Region,
		Clusters: []clientcmdv1.NamedCluster{{
			Name: cluster.Shoot.Spec.Region,
		}},
		AuthInfos: []clientcmdv1.NamedAuthInfo{{
			Name: string(username),
			AuthInfo: clientcmdv1.AuthInfo{
				Token: string(token),
			},
		}},
		Contexts: []clientcmdv1.NamedContext{{
			Name: cluster.Shoot.Spec.Region,
			Context: clientcmdv1.Context{
				Cluster:   cluster.Shoot.Spec.Region,
				AuthInfo:  string(username),
				Namespace: string(namespace),
			},
		}},
	}

	var regionFound bool
	for _, region := range cloudProfileConfig.RegionConfigs {
		if region.Name == cluster.Shoot.Spec.Region {
			kubeconfig.Clusters[0].Cluster.Server = region.Server
			kubeconfig.Clusters[0].Cluster.CertificateAuthorityData = region.CertificateAuthorityData
			regionFound = true
			break
		}
	}
	if !regionFound {
		return fmt.Errorf("faild to find region %s in cloudprofile", cluster.Shoot.Spec.Region)
	}

	raw, err = runtime.Encode(clientcmdlatest.Codec, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to encode kubeconfig: %w", err)
	}

	newCloudProviderSecret.Data[onmetal.KubeConfigFieldName] = raw
	return nil
}
