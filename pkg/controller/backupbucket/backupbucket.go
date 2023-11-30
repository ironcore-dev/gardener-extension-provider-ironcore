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
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	controllerconfig "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/config"
	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
)

const (
	waitBucketInitDelay   = 1 * time.Second
	waitBucketFactor      = 1.2
	waitBucketActiveSteps = 19
)

// ensureBackupBucket creates ironcore backupBucket object and returns access to bucket
func (a *actuator) ensureBackupBucket(ctx context.Context, namespace string, ironcoreClient client.Client, backupBucket *extensionsv1alpha1.BackupBucket) error {
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
	//create ironcore bucket
	if _, err := controllerutil.CreateOrPatch(ctx, ironcoreClient, bucket, nil); err != nil {
		return fmt.Errorf("failed to create or patch backup bucket %s: %w", client.ObjectKeyFromObject(bucket), err)
	}
	//wait for bucket creation
	if err := waitBackupBucketToAvailable(ctx, ironcoreClient, bucket); err != nil {
		return fmt.Errorf("could not determine status of backup bucket %w", err)
	}

	accessSecret := &corev1.Secret{}
	if err := ironcoreClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: bucket.Status.Access.SecretRef.Name}, accessSecret); err != nil {
		return fmt.Errorf("failed to get bucket access secret %s: %w", client.ObjectKeyFromObject(accessSecret), err)
	}
	//update backupBucket secret
	if err := a.patchBackupBucketStatus(ctx, backupBucket, accessSecret.Data, bucket.Status.Access.Endpoint); err != nil {
		return fmt.Errorf("failed to patch backupbucket status %s: %w", client.ObjectKeyFromObject(bucket), err)
	}
	return nil
}

func waitBackupBucketToAvailable(ctx context.Context, ironcoreClient client.Client, bucket *storagev1alpha1.Bucket) error {
	backoff := wait.Backoff{
		Duration: waitBucketInitDelay,
		Factor:   waitBucketFactor,
		Steps:    waitBucketActiveSteps,
	}

	err := wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (bool, error) {
		err := ironcoreClient.Get(ctx, client.ObjectKey{Namespace: bucket.Namespace, Name: bucket.Name}, bucket)
		if err == nil && bucket.Status.State == storagev1alpha1.BucketStateAvailable && isBucketAccessDetailsAvailable(bucket) {
			return true, nil
		}
		return false, err
	})

	if wait.Interrupted(err) {
		return fmt.Errorf("timeout waiting for the ironcore Bucket %s status: %w", client.ObjectKeyFromObject(bucket), err)
	}

	return err
}

func isBucketAccessDetailsAvailable(bucket *storagev1alpha1.Bucket) bool {
	return bucket.Status.Access != nil && bucket.Status.Access.SecretRef != nil && bucket.Status.Access.Endpoint != ""
}

// patchBackupBucketStatus updates backupBucket status with access secretRef
func (a *actuator) patchBackupBucketStatus(ctx context.Context, backupBucket *extensionsv1alpha1.BackupBucket, secretData map[string][]byte, endpoint string) error {
	if secretData == nil {
		return fmt.Errorf("secret does not contain any data")
	}

	accessKeyID, ok := secretData[ironcore.BucketAccessKeyID]
	if !ok {
		return fmt.Errorf("missing %q field in secret", ironcore.BucketAccessKeyID)
	}

	secretAccessKey, ok := secretData[ironcore.BucketSecretAccessKey]
	if !ok {
		return fmt.Errorf("missing %q field in secret", ironcore.BucketSecretAccessKey)
	}

	accessSecretData := map[string][]byte{}
	accessSecretData[ironcore.AccessKeyID] = []byte(accessKeyID)
	accessSecretData[ironcore.SecretAccessKey] = []byte(secretAccessKey)
	accessSecretData[ironcore.Endpoint] = []byte(endpoint)

	patch := client.MergeFrom(backupBucket.DeepCopy())

	backupBucketSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1constants.SecretPrefixGeneratedBackupBucket + backupBucket.Name,
			Namespace: backupBucket.Spec.SecretRef.Namespace,
		},
		Data: accessSecretData,
	}

	if err := controllerutil.SetOwnerReference(backupBucket, backupBucketSecret, a.client.Scheme()); err != nil {
		return fmt.Errorf("failed to set owner reference for bucket generated secret %s: %w", client.ObjectKeyFromObject(backupBucketSecret), err)
	}

	if _, err := controllerutil.CreateOrPatch(ctx, a.client, backupBucketSecret, nil); err != nil {
		return fmt.Errorf("failed to create backup bucket generated secret %s: %w", client.ObjectKeyFromObject(backupBucketSecret), err)
	}

	backupBucket.Status.GeneratedSecretRef = &corev1.SecretReference{
		Name:      backupBucketSecret.Name,
		Namespace: backupBucketSecret.Namespace,
	}
	return a.client.Status().Patch(ctx, backupBucket, patch)
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
