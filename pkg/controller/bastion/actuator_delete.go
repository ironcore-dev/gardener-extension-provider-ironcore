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
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	computev1alpha1 "github.com/onmetal/onmetal-api/api/compute/v1alpha1"
)

// Delete implements bastion.Actuator.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *controller.Cluster) error {
	log.V(2).Info("Deleting Bastion Host")

	// get onmetal credentials from infrastructure config
	onmetalClient, namespace, err := onmetal.GetOnmetalClientAndNamespaceFromCloudProviderSecret(ctx, a.Client(), cluster.ObjectMeta.Name)
	if err != nil {
		return fmt.Errorf("failed to get onmetal client and namespace from cloudprovider secret: %w", err)
	}

	if err := a.deleteBastionMachine(ctx, onmetalClient, namespace, bastion.Name); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete infrastructure: %w", err)
	}

	log.V(2).Info("Successfully deleted Bastion host")
	return nil
}

func (a *actuator) deleteBastionMachine(ctx context.Context, onmetalClient client.Client, namespace, name string) error {
	prefix := &computev1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	return onmetalClient.Delete(ctx, prefix)
}
