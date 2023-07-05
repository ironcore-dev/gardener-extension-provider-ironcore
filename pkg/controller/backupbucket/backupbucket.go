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

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
)

// ensureBackupBucket creates onmetal backupBucket object
func (a *actuator) ensureBackupBucket(ctx context.Context, namespace string, onmetalClient client.Client, backupBucket *extensionsv1alpha1.BackupBucket) error {
	bucket := &storagev1alpha1.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupBucket.Name,
			Namespace: namespace,
		},
		Spec: storagev1alpha1.BucketSpec{
			BucketClassRef: &corev1.LocalObjectReference{
				Name: a.backupBucketConfig.BucketClassName,
			},
		},
	}

	if _, err := controllerutil.CreateOrPatch(ctx, onmetalClient, bucket, nil); err != nil {
		return fmt.Errorf("failed to create or patch backup bucket %s: %w", client.ObjectKeyFromObject(bucket), err)
	}
	return nil
}

// validateConfiguration checks whether a backup bucket configuration is valid.
func validateConfiguration(config *controllerconfig.BackupBucketConfig) error {
	if config == nil {
		return fmt.Errorf("backupBucketConfig must not be empty")
	}

	if config.BucketClassName == "" {
		return fmt.Errorf("BucketClassName is mandatory")
	}

	return nil
}
