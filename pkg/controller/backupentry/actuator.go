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

package backupentry

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/backupentry/genericactuator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type actuator struct {
	client client.Client
}

func newActuator() genericactuator.BackupEntryDelegate {
	return &actuator{}
}

func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

func (a *actuator) GetETCDSecretData(_ context.Context, _ logr.Logger, _ *extensionsv1alpha1.BackupEntry, backupSecretData map[string][]byte) (map[string][]byte, error) {
	return backupSecretData, nil
}

func (a *actuator) Delete(ctx context.Context, log logr.Logger, backupEntry *extensionsv1alpha1.BackupEntry) error {

	// get s3Client credentials from secret reference
	s3ClientSecret := &corev1.Secret{}
	s3ClientSecretKey := client.ObjectKey{Namespace: backupEntry.Spec.SecretRef.Namespace, Name: backupEntry.Spec.SecretRef.Name}
	if err := a.client.Get(ctx, s3ClientSecretKey, s3ClientSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("s3 client secret not found: %s", backupEntry.Spec.SecretRef.Name)
		}
		return fmt.Errorf("could not get s3 client secret: %w", err)
	}

	// get s3 client from s3 client secret
	s3Client, err := GetS3ClientFromS3ClientSecret(s3ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to get s3 client from s3 client secret: %w", err)
	}

	return DeleteObjectsWithPrefix(ctx, s3Client, backupEntry.Spec.Region, backupEntry.Spec.BucketName, fmt.Sprintf("%s/", backupEntry.Name))
}
