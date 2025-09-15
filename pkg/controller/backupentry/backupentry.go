// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package backupentry

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	corev1 "k8s.io/api/core/v1"

	"github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/ironcore"
)

//go:generate $MOCKGEN -copyright_file ../../../hack/license-header.txt -package backupentry -destination=mock_backupentry.go -source backupentry.go S3ClientGetter,S3ObjectLister

type s3ObjectLister interface {
	ListObjectsPages(ctx context.Context, s3Client *s3.Client, input *s3.ListObjectsV2Input, bucketName string) error
}

type s3ObjectListerImpl struct{}

var objectLister s3ObjectLister = s3ObjectListerImpl{}

func (o s3ObjectListerImpl) ListObjectsPages(ctx context.Context, s3Client *s3.Client, input *s3.ListObjectsV2Input, bucketName string) error {
	paginator := s3.NewListObjectsV2Paginator(s3Client, input)
	for paginator.HasMorePages() {
		objectIDs := make([]s3types.ObjectIdentifier, 0)
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, object := range output.Contents {
			identifier := s3types.ObjectIdentifier{Key: object.Key}
			objectIDs = append(objectIDs, identifier)
		}
		if len(objectIDs) != 0 {
			if _, err = s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &s3types.Delete{
					Objects: objectIDs,
					Quiet:   aws.Bool(true),
				},
			}); err != nil {
				var nsk *s3types.NoSuchKey
				if errors.As(err, &nsk) {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

// DeleteObjectsWithPrefix deletes the s3 objects with the specific <prefix>
// from <bucket>. If it does not exist, no error is returned.
func DeleteObjectsWithPrefix(ctx context.Context, s3Client *s3.Client, bucketName, prefix string) error {
	in := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	}

	if err := objectLister.ListObjectsPages(ctx, s3Client, in, bucketName); err != nil {
		return fmt.Errorf("failed to list objects pages: %w", err)
	}

	return nil
}

// GetS3ClientFromS3ClientSecret creates s3Client from bucket access key ID
// and secret access key.
func GetS3ClientFromS3ClientSecret(ctx context.Context, secret *corev1.Secret) (*s3.Client, error) {
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
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = "region" //TODO: hardcoded the region for now, consider making it configurable if necessary
	})

	return s3Client, nil
}
