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
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("BackupEntry Delete", func() {
	ns := SetupTest()
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
				Tolerations: []commonv1alpha1.Toleration{
					{
						Key:      "key",
						Operator: "Equal",
						Value:    "value",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, bucket)).Should(Succeed())
		Eventually(Object(bucket)).Should(SatisfyAll(
			HaveField("Spec.BucketClassRef", &corev1.LocalObjectReference{
				Name: "test-bucket-class",
			}),
		))

		By("creating a secret with credentials data to access onmetal bucket")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "test-secret",
			},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("test-access-key"),
				"AWS_SECRET_ACCESS_KEY": []byte("test-secret-access-key"),
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
					Name: "backupprovider",
				},
			},
		}
		Expect(k8sClient.Create(ctx, backupEntry)).To(Succeed())

		Expect(k8sClient.Delete(ctx, backupEntry)).To(Succeed())

		By("waiting for the backupEntry to be gone")
		Eventually(Get(backupEntry)).Should(Satisfy(apierrors.IsNotFound))

	})

})
