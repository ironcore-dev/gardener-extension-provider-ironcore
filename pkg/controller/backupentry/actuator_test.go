// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backupentry

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gardener/gardener/extensions/pkg/controller/backupentry/genericactuator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
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
		ctrl               *gomock.Controller
		a                  genericactuator.BackupEntryDelegate
		log                logr.Logger
		mockS3ObjectLister *Mocks3ObjectLister
	)

	BeforeEach(func(ctx SpecContext) {
		a = newActuator(*mgr)
		Expect(a).NotTo(BeNil())

		ctrl = gomock.NewController(GinkgoT())
		mockS3ObjectLister = NewMocks3ObjectLister(ctrl)

		objectLister = mockS3ObjectLister
	})

	It("should delete Backupentry", func(ctx SpecContext) {

		By("creating an Onmetal bucket resource")
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

		By("creating a secret with credentials data to access onmetal bucket")
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

		By("patching onmetal bucket with available state and credentials secret")
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

		in := &s3.ListObjectsInput{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(fmt.Sprintf("%s/", backupEntry.Name)),
		}
		mockS3ObjectLister.EXPECT().ListObjectsPages(ctx, gomock.Any(), in, backupEntry.Spec.BucketName).Return(nil)

		By("deleting the BackupEntry")
		Expect(a.Delete(ctx, log, backupEntry)).Should(Succeed())
	})

})
