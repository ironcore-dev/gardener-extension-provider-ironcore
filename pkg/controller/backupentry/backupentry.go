// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package backupentry

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3ObjectLister interface {
	ListObjectsPages(ctx context.Context, s3Client S3Client, input *s3.ListObjectsV2Input, bucketName string) error
}

type s3ObjectListerImpl struct{}

var objectLister s3ObjectLister = s3ObjectListerImpl{}

func (o s3ObjectListerImpl) ListObjectsPages(ctx context.Context, s3Client S3Client, input *s3.ListObjectsV2Input, bucketName string) error {
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
func DeleteObjectsWithPrefix(ctx context.Context, s3Client S3Client, bucketName, prefix string) error {
	in := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	}

	if err := objectLister.ListObjectsPages(ctx, s3Client, in, bucketName); err != nil {
		return fmt.Errorf("failed to list objects pages: %w", err)
	}

	return nil
}
