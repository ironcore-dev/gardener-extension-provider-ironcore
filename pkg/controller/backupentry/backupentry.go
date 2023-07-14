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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	corev1 "k8s.io/api/core/v1"

	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
)

//go:generate $MOCKGEN -package backupentry -destination=mock_backupentry.go -source backupentry.go S3ClientGetter,S3ObjectLister

type s3ObjectLister interface {
	ListObjectsPages(ctx aws.Context, s3Client *s3.S3, input *s3.ListObjectsInput, bucketName string) error
}

type s3ObjectListerImpl struct{}

var objectLister s3ObjectLister = s3ObjectListerImpl{}

func (o s3ObjectListerImpl) ListObjectsPages(ctx aws.Context, s3Client *s3.S3, input *s3.ListObjectsInput, bucketName string) error {
	var delErr error
	if err := s3Client.ListObjectsPagesWithContext(ctx, input, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		objectIDs := make([]*s3.ObjectIdentifier, 0)
		for _, key := range page.Contents {
			obj := &s3.ObjectIdentifier{
				Key: key.Key,
			}
			objectIDs = append(objectIDs, obj)
		}

		if len(objectIDs) != 0 {
			if _, delErr = s3Client.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &s3.Delete{
					Objects: objectIDs,
					Quiet:   aws.Bool(true),
				},
			}); delErr != nil {
				return false
			}
		}
		return !lastPage
	}); err != nil {
		return fmt.Errorf("error listing objects pages from bucket %s: %w", bucketName, err)
	}

	if delErr != nil {
		if aerr, ok := delErr.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil
		}
		return delErr
	}
	return nil
}

// DeleteObjectsWithPrefix deletes the s3 objects with the specific <prefix>
// from <bucket>. If it does not exist, no error is returned.
func DeleteObjectsWithPrefix(ctx context.Context, s3Client *s3.S3, region, bucketName, prefix string) error {
	in := &s3.ListObjectsInput{
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
func GetS3ClientFromS3ClientSecret(secret *corev1.Secret) (*s3.S3, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	accessKeyID, ok := secret.Data[onmetal.AccessKeyID]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", onmetal.AccessKeyID)
	}

	secretAccessKey, ok := secret.Data[onmetal.SecretAccessKey]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", onmetal.SecretAccessKey)
	}

	endpoint, ok := secret.Data[onmetal.Endpoint]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", onmetal.Endpoint)
	}

	endpointStr := string(endpoint)
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(string(accessKeyID), string(secretAccessKey), ""),
		Endpoint:    &endpointStr,
	}

	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	config := &aws.Config{Region: aws.String("region")} //TODO: hardcoded the region for now, consider making it configurable if necessary
	s3Client := s3.New(s, config)

	return s3Client, nil
}
