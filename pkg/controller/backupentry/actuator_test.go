// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package backupentry

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gardener/gardener/extensions/pkg/controller/backupentry/genericactuator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BackupEntry Delete", func() {
	mgr, ns := SetupTest()

	var (
		ctrl         *gomock.Controller
		a            genericactuator.BackupEntryDelegate
		log          logr.Logger
		mockS3Client *MockS3Client
	)

	BeforeEach(func(ctx SpecContext) {
		a = newActuator(*mgr)
		Expect(a).NotTo(BeNil())

		ctrl = gomock.NewController(GinkgoT())
		mockS3Client = NewMockS3Client(ctrl)

		NewS3ClientFromConfig = func(cfg aws.Config, optFns ...func(*s3.Options)) S3Client {
			return mockS3Client
		}
	})

	It("should delete Backupentry", func(ctx SpecContext) {

		By("creating an Ironcore bucket resource")
		bucketName := "test-bucket"
		bucket := &storagev1alpha1.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      bucketName,
			},
			Spec: storagev1alpha1.BucketSpec{
				BucketClassRef: &corev1.LocalObjectReference{
					Name: "test-bucket-class",
				},
				BucketPoolSelector: map[string]string{
					"key": "value",
				},
				BucketPoolRef: &corev1.LocalObjectReference{
					Name: "my-bucket-pool",
				},
			},
		}
		Expect(k8sClient.Create(ctx, bucket)).Should(Succeed())

		By("creating a secret with credentials data to access ironcore bucket")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "test-secret",
			},
			Data: map[string][]byte{
				"accessKeyID":     []byte("test-access-key"),
				"secretAccessKey": []byte("test-secret-access-key"),
				"endpoint":        []byte("endpoint-efef-ihfbd-ssadd.storage"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		DeferCleanup(k8sClient.Delete, secret)

		By("patching ironcore bucket with available state and credentials secret")
		bucketBase := bucket.DeepCopy()
		bucket.Status.State = storagev1alpha1.BucketStateAvailable
		bucket.Status.Access = &storagev1alpha1.BucketAccess{
			SecretRef: &corev1.LocalObjectReference{
				Name: secret.Name,
			},
			Endpoint: "s3-storage-endpoint",
		}
		Expect(k8sClient.Status().Patch(ctx, bucket, client.MergeFrom(bucketBase))).To(Succeed())
		DeferCleanup(k8sClient.Delete, bucket)

		By("creating a BackupEntry")
		backupEntry := &extensionsv1alpha1.BackupEntry{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "test-backup-entry",
			},
			Spec: extensionsv1alpha1.BackupEntrySpec{
				Region:     "foo",
				BucketName: bucketName,
				SecretRef: corev1.SecretReference{
					Name:      secret.Name,
					Namespace: ns.Name,
				},
			},
		}
		Expect(k8sClient.Create(ctx, backupEntry)).To(Succeed())

		listIn := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(fmt.Sprintf("%s/", backupEntry.Name)),
		}
		listOut := &s3.ListObjectsV2Output{
			Contents: []types.Object{
				{
					Key: aws.String(fmt.Sprintf("%s/test-obj", backupEntry.Name)),
				},
			},
		}
		deleteIn := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &types.Delete{
				Objects: []types.ObjectIdentifier{
					{
						Key: aws.String(fmt.Sprintf("%s/test-obj", backupEntry.Name)),
					},
				},
				Quiet: aws.Bool(true),
			},
		}
		mockS3Client.EXPECT().ListObjectsV2(ctx, listIn, gomock.Any()).Return(listOut, nil)
		mockS3Client.EXPECT().DeleteObjects(ctx, deleteIn).Return(&s3.DeleteObjectsOutput{}, nil)

		By("deleting the BackupEntry")
		Expect(a.Delete(ctx, log, backupEntry)).Should(Succeed())
	})

})
