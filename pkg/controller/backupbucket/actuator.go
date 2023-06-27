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

package backupbucket

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/backupbucket"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
)

type actuator struct {
	common.RESTConfigContext
	backupBucketConfig *controllerconfig.BackupBucketConfig
}

func newActuator(backupBucketConfig *controllerconfig.BackupBucketConfig) backupbucket.Actuator {
	return &actuator{
		backupBucketConfig: backupBucketConfig,
	}
}

// Reconcile implements backupbucket.Actuator
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, backupBucket *extensionsv1alpha1.BackupBucket) error {
	log.V(2).Info("Reconciling backupBucket")

	onmetalClient, namespace, err := onmetal.GetOnmetalClientAndNamespaceFromSecretRef(ctx, a.Client(), &backupBucket.Spec.SecretRef)
	if err != nil {
		return fmt.Errorf("failed to get onmetal client and namespace from cloudprovider secret: %w", err)
	}

	// If the generated secret in the backupbucket status not exists that means
	// no backupbucket exists and it need to be created.
	if backupBucket.Status.GeneratedSecretRef == nil {
		if err := validateConfiguration(a.backupBucketConfig); err != nil {
			return fmt.Errorf("error validating configuration: %w", err)
		}
		onmetalBucket, err := a.ensureBackupBucket(ctx, namespace, onmetalClient, backupBucket)
		if err != nil {
			return fmt.Errorf("failed to create backup bucket: %w", err)
		}

		log.V(2).Info("Successfully reconciled backupBucket")

		secertRef, err := getBucketGeneratedSecretRef(onmetalBucket)

		if err != nil {
			return fmt.Errorf("failed to get access secretRef from backup bucket: %w", err)
		}

		// update status
		return a.updateBackupBucketStatus(backupBucket, secertRef, onmetalBucket.Namespace, ctx)
	}

	return nil
}

func (a *actuator) Delete(ctx context.Context, log logr.Logger, backupBucket *extensionsv1alpha1.BackupBucket) error {
	log.V(2).Info("Deleting Bucket")

	onmetalClient, namespace, err := onmetal.GetOnmetalClientAndNamespaceFromSecretRef(ctx, a.Client(), &backupBucket.Spec.SecretRef)
	if err != nil {
		return fmt.Errorf("failed to get onmetal client and namespace from cloudprovider secret: %w", err)
	}

	bucket := &storagev1alpha1.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupBucket.Name,
			Namespace: namespace,
		},
	}
	if err = onmetalClient.Delete(ctx, bucket); err != nil {
		return fmt.Errorf("failed to delete backup bucket : %v", err)
	}

	log.V(2).Info("Deleted Bucket")
	return nil
}
