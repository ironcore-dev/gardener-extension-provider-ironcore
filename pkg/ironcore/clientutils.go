// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ironcore

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	computev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ironcoreScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(corev1.AddToScheme(ironcoreScheme))
	utilruntime.Must(networkingv1alpha1.AddToScheme(ironcoreScheme))
	utilruntime.Must(computev1alpha1.AddToScheme(ironcoreScheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(ironcoreScheme))
	utilruntime.Must(ipamv1alpha1.AddToScheme(ironcoreScheme))
	utilruntime.Must(extensionsv1alpha1.AddToScheme(ironcoreScheme))
}

// GetIroncoreClientAndNamespaceFromCloudProviderSecret extracts the <ironcoreClient, ironcoreNamespace> from the
// cloudprovider secret in the Shoot namespace.
func GetIroncoreClientAndNamespaceFromCloudProviderSecret(ctx context.Context, cl client.Client, shootNamespace string) (client.Client, string, error) {
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
	c, err := client.New(clientCfg, client.Options{Scheme: ironcoreScheme})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create client from cloudprovider secret: %w", err)
	}

	return c, string(namespace), nil
}

// GetIroncoreClientAndNamespaceFromSecretRef extracts the <ironcoreClient, ironcoreNamespace> from the
// provided secret
func GetIroncoreClientAndNamespaceFromSecretRef(ctx context.Context, cl client.Client, secretRef *corev1.SecretReference) (client.Client, string, error) {
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
	c, err := client.New(clientCfg, client.Options{Scheme: ironcoreScheme})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create client from secret: %w", err)
	}

	return c, string(namespace), nil
}
