// Copyright 2022 OnMetal authors
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

package worker

import (
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	api "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal"
	apiv1alpha1 "github.com/onmetal/gardener-extension-provider-onmetal/pkg/apis/onmetal/v1alpha1"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	testutils "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MachinesImages", func() {
	ctx := testutils.SetupContext()
	ns, _ := SetupTest(ctx)

	It("should update the worker status", func() {
		By("defining and setting infrastructure status for worker")
		infraStatus := &api.InfrastructureStatus{
			TypeMeta: metav1.TypeMeta{
				Kind:       "InfrastructureStatus",
				APIVersion: "onmetal.provider.extensions.gardener.cloud/v1alpha1",
			},
			NetworkRef: commonv1alpha1.LocalUIDReference{
				Name: "my-network",
				UID:  "1234",
			},
		}
		w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{Raw: encodeObject(infraStatus)}

		By("creating the worker")
		Expect(k8sClient.Create(ctx, w)).To(Succeed())

		By("creating a worker delegate")
		decoder := serializer.NewCodecFactory(k8sClient.Scheme(), serializer.EnableStrict).UniversalDecoder()
		workerDelegate, err := NewWorkerDelegate(common.NewClientContext(k8sClient, k8sClient.Scheme(), decoder), "", w, cluster)
		Expect(err).NotTo(HaveOccurred())

		By("calling the updating machine image status")
		err = workerDelegate.UpdateMachineImagesStatus(ctx)
		Expect(err).NotTo(HaveOccurred())

		By("ensuring that the worker infrastructure status has been updated")
		Eventually(func(g Gomega) {
			workerKey := types.NamespacedName{Namespace: ns.Name, Name: w.Name}
			err := k8sClient.Get(ctx, workerKey, w)
			Expect(client.IgnoreNotFound(err)).To(Succeed())
			g.Expect(err).NotTo(HaveOccurred())

			expectedWorkerStatus := &apiv1alpha1.WorkerStatus{
				TypeMeta: metav1.TypeMeta{
					APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
					Kind:       "WorkerStatus",
				},
				MachineImages: []apiv1alpha1.MachineImage{
					{
						Name:         "my-os",
						Version:      "1.0",
						Image:        "registry/my-os",
						Architecture: pointer.String(constants.ArchitectureAMD64),
					},
				},
			}
			g.Expect(err).ToNot(HaveOccurred())

			workerStatus := &apiv1alpha1.WorkerStatus{}
			_, _, err = decoder.Decode(w.Status.ProviderStatus.Raw, nil, workerStatus)
			Expect(err).NotTo(HaveOccurred())
			g.Expect(workerStatus).To(Equal(expectedWorkerStatus))
		}).Should(Succeed())
	})
})
