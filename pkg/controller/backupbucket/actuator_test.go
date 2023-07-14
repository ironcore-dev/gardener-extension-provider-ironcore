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

package backupbucket

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	controllerconfig "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/config"
	"github.com/onmetal/gardener-extension-provider-onmetal/pkg/onmetal"
	storagev1alpha1 "github.com/onmetal/onmetal-api/api/storage/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Backupbucket Reconcile", Ordered, func() {
	var (
		backupBucket *extensionsv1alpha1.BackupBucket
		bucket       *storagev1alpha1.Bucket
	)
	ns := SetupTest()

	It("should create backup bucket for a given bucket configuration", func(ctx SpecContext) {
		By("creating backup bucket resource")
		backupBucket = &extensionsv1alpha1.BackupBucket{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-backup-bucket",
			},
			Spec: extensionsv1alpha1.BackupBucketSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           onmetal.Type,
					ProviderConfig: nil,
				},
				Region: "europe-central",
				SecretRef: corev1.SecretReference{
					Name:      "backupprovider",
					Namespace: ns.Name,
				},
			},
		}
		Expect(k8sClient.Create(ctx, backupBucket)).Should(Succeed())

		Eventually(Object(backupBucket)).Should(SatisfyAll(
			HaveField("Status.LastOperation.Type", gardencorev1beta1.LastOperationTypeCreate),
		))

		By("ensuring backup bucket is created with correct spec")

		bucket = &storagev1alpha1.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-backup-bucket",
			},
		}

		Eventually(Object(bucket)).Should(SatisfyAll(
			HaveField("Spec.BucketClassRef", &corev1.LocalObjectReference{
				Name: "my-bucket-class",
			}),
		))

		bucketAccesSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "my-bucket-secret",
			},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("YXdzQWNjZXNzS2V5Cg=="),
				"AWS_SECRET_ACCESS_KEY": []byte("YXdzQWNjZXNzU2VjcmV0S2V5Cg==ss"),
			},
		}
		Expect(k8sClient.Create(ctx, bucketAccesSecret)).To(Succeed())

		By("patching backup bucket with available state and access details")
		bucketBase := bucket.DeepCopy()
		bucket.Status.State = storagev1alpha1.BucketStateAvailable
		bucket.Status.Access = &storagev1alpha1.BucketAccess{
			SecretRef: &corev1.LocalObjectReference{
				Name: "my-bucket-secret",
			},
			Endpoint: "endpoint-efef-ihfbd-ssadd.storage",
		}
		Expect(k8sClient.Status().Patch(ctx, bucket, client.MergeFrom(bucketBase))).To(Succeed())

		By("ensuring that bucket is created and Available")
		Eventually(Object(bucket)).Should(SatisfyAll(
			HaveField("Status.State", storagev1alpha1.BucketStateAvailable),
			HaveField("Status.Access.SecretRef.Name", "my-bucket-secret"),
			HaveField("Status.Access.Endpoint", "endpoint-efef-ihfbd-ssadd.storage"),
		))

		Eventually(Get(bucket)).Should(Succeed())

		By("ensuring that bucket updated with access secret and endpoint")
		Eventually(Object(backupBucket)).Should(SatisfyAll(
			HaveField("Status.GeneratedSecretRef.Name", v1beta1constants.SecretPrefixGeneratedBackupBucket+backupBucket.Name),
		))

		generatedSecretRef := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: backupBucket.Status.GeneratedSecretRef.Namespace,
				Name:      backupBucket.Status.GeneratedSecretRef.Name,
			},
		}

		Eventually(Object(generatedSecretRef)).Should(SatisfyAll(
			HaveField("ObjectMeta.OwnerReferences", ContainElement(SatisfyAll(
				HaveField("Name", backupBucket.Name),
				HaveField("Kind", "BackupBucket"),
				HaveField("UID", backupBucket.UID),
			))),
			HaveField("Data", map[string][]byte{
				"endpoint":        []byte("endpoint-efef-ihfbd-ssadd.storage"),
				"secretAccessKey": []byte("YXdzQWNjZXNzU2VjcmV0S2V5Cg==ss"),
				"accessKeyID":     []byte("YXdzQWNjZXNzS2V5Cg=="),
			}),
		))

		By("ensuring backupbucket is delete successfully")
		Expect(k8sClient.Delete(ctx, backupBucket)).To(Succeed())

		By("waiting for the bucket to be gone")
		Eventually(Get(bucket)).Should(Satisfy(apierrors.IsNotFound))
	})

	It("should check backup bucket configuration", func(ctx SpecContext) {

		By("creating backupbucketconfig")
		config := &controllerconfig.BackupBucketConfig{
			BucketClassName: "",
		}

		By("validating backupbucket config")
		err := validateConfiguration(nil)
		Expect(err).To(MatchError("backupBucketConfig must not be empty"))

		By("validating bucketclassname is not empty")
		err = validateConfiguration(config)
		Expect(err).To(MatchError("BucketClassName is mandatory"))

		config.BucketClassName = "foo"

		By("validating backupbucketconfig is valid")
		ret := validateConfiguration(config)
		Expect(ret).To(BeNil())

	})

})
