// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package backupentry

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	corev1 "k8s.io/api/core/v1"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

//go:generate $MOCKGEN -copyright_file ../../../hack/license-header.txt -package backupentry -destination=mock_s3_client.go -source s3_client.go Client

type S3Client interface {
	s3.ListObjectsV2APIClient
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}

// GetS3ClientFromS3ClientSecret creates s3Client from bucket access key ID
// and secret access key.
func GetS3ClientFromS3ClientSecret(ctx context.Context, secret *corev1.Secret) (S3Client, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	accessKeyID, ok := secret.Data[ironcore.AccessKeyID]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", ironcore.AccessKeyID)
	}

	secretAccessKey, ok := secret.Data[ironcore.SecretAccessKey]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", ironcore.SecretAccessKey)
	}

	endpoint, ok := secret.Data[ironcore.Endpoint]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", ironcore.Endpoint)
	}

	awsCredentials := credentials.NewStaticCredentialsProvider(string(accessKeyID), string(secretAccessKey), "")
	endpointStr := string(endpoint)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(awsCredentials), config.WithBaseEndpoint(endpointStr))
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}
	s3Client := NewS3ClientFromConfig(cfg)

	return s3Client, nil
}

var NewS3ClientFromConfig = func(cfg aws.Config, optFns ...func(*s3.Options)) S3Client {
	return s3.NewFromConfig(cfg, optFns...)
}
