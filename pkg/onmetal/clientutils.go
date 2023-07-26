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

package onmetal

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
	ipamv1alpha1 "github.com/onmetal/onmetal-api/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
)

var onmetalScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(corev1.AddToScheme(onmetalScheme))
	utilruntime.Must(networkingv1alpha1.AddToScheme(onmetalScheme))
	utilruntime.Must(computev1alpha1.AddToScheme(onmetalScheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(onmetalScheme))
	utilruntime.Must(ipamv1alpha1.AddToScheme(onmetalScheme))
	utilruntime.Must(extensionsv1alpha1.AddToScheme(onmetalScheme))
}

// GetOnmetalClientAndNamespaceFromCloudProviderSecret extracts the <onmetalClient, onmetalNamespace> from the
// cloudprovider secret in the Shoot namespace.
func GetOnmetalClientAndNamespaceFromCloudProviderSecret(ctx context.Context, cl client.Client, shootNamespace string) (client.Client, string, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Namespace: shootNamespace, Name: v1beta1constants.SecretNameCloudProvider}
	if err := cl.Get(ctx, secretKey, secret); err != nil {
		return nil, "", fmt.Errorf("failed to get cloudprovider secret: %w", err)
	}
	kubeconfig, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, "", fmt.Errorf("could not find a kubeconfig in the cloudprovider secret")
	}
	namespace, ok := secret.Data["namespace"]
	if !ok {
		return nil, "", fmt.Errorf("could not find a namespace in the cloudprovider secret")
	}
	clientCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create rest config from cloudprovider secret: %w", err)
	}
	c, err := client.New(clientCfg, client.Options{Scheme: onmetalScheme})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create client from cloudprovider secret: %w", err)
	}

	return c, string(namespace), nil
}

// GetOnmetalClientAndNamespaceFromSecretRef extracts the <onmetalClient, onmetalNamespace> from the
// provided secret
func GetOnmetalClientAndNamespaceFromSecretRef(ctx context.Context, cl client.Client, secretRef *corev1.SecretReference) (client.Client, string, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, cl, secretRef)
	if err != nil {
		return nil, "", err
	}

	if secret.Data == nil {
		return nil, "", fmt.Errorf("secret does not contain any data")
	}
	kubeconfig, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, "", fmt.Errorf("could not find a kubeconfig in the secret")
	}
	namespace, ok := secret.Data["namespace"]
	if !ok {
		return nil, "", fmt.Errorf("could not find a namespace in the secret")
	}
	clientCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create rest config from secret: %w", err)
	}
	c, err := client.New(clientCfg, client.Options{Scheme: onmetalScheme})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create client from secret: %w", err)
	}

	return c, string(namespace), nil
}
