// Copyright 2023 IronCore authors
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

package backupbucket

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/backupbucket"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	controllerconfig "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/config"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type actuator struct {
	backupBucketConfig *controllerconfig.BackupBucketConfig
	client             client.Client
}

func newActuator(mgr manager.Manager, backupBucketConfig *controllerconfig.BackupBucketConfig) backupbucket.Actuator {
	return &actuator{
		client:             mgr.GetClient(),
		backupBucketConfig: backupBucketConfig,
	}
}

// Reconcile implements backupbucket.Actuator
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, backupBucket *extensionsv1alpha1.BackupBucket) error {
	log.V(2).Info("Reconciling BackupBucket")

	ironcoreClient, namespace, err := ironcore.GetIroncoreClientAndNamespaceFromSecretRef(ctx, a.client, &backupBucket.Spec.SecretRef)
	if err != nil {
		return fmt.Errorf("failed to get ironcore client and namespace from cloudprovider secret: %w", err)
	}

	// If the generated secret in the backupbucket status not exists that means
	// no backupbucket exists, and it needs to be created.
	if backupBucket.Status.GeneratedSecretRef == nil {
		if err := validateConfiguration(a.backupBucketConfig); err != nil {
			return fmt.Errorf("failed to validate configuration: %w", err)
		}

		if err := a.ensureBackupBucket(ctx, namespace, ironcoreClient, backupBucket); err != nil {
			return fmt.Errorf("failed to ensure backupbucket: %w", err)
		}
	}
	log.V(2).Info("Reconciled BackupBucket")
	return nil
}

func (a *actuator) Delete(ctx context.Context, log logr.Logger, backupBucket *extensionsv1alpha1.BackupBucket) error {
	log.V(2).Info("Deleting BackupBucket")
	ironcoreClient, namespace, err := ironcore.GetIroncoreClientAndNamespaceFromSecretRef(ctx, a.client, &backupBucket.Spec.SecretRef)
	if err != nil {
		return fmt.Errorf("failed to get ironcore client and namespace from cloudprovider secret: %w", err)
	}

	bucket := &storagev1alpha1.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupBucket.Name,
			Namespace: namespace,
		},
	}
	if err = ironcoreClient.Delete(ctx, bucket); err != nil {
		return fmt.Errorf("failed to delete backup bucket: %v", err)
	}

	log.V(2).Info("Deleted BackupBucket")
	return nil
}
