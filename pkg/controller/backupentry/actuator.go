// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type actuator struct {
	client client.Client
}

func newActuator(mgr manager.Manager) genericactuator.BackupEntryDelegate {
	return &actuator{
		client: mgr.GetClient(),
	}
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
	s3Client, err := GetS3ClientFromS3ClientSecret(ctx, s3ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to get s3 client from s3 client secret: %w", err)
	}

	return DeleteObjectsWithPrefix(ctx, s3Client, backupEntry.Spec.BucketName, fmt.Sprintf("%s/", backupEntry.Name))
}
