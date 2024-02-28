// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	apiv1alpha1 "github.com/ironcore-dev/gardener-extension-provider-ironcore/pkg/apis/ironcore/v1alpha1"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MachinesImages", func() {
	ns, _ := SetupTest()

	It("should update the worker status", func(ctx SpecContext) {
		By("defining and setting infrastructure status for worker")
		infraStatus := &apiv1alpha1.InfrastructureStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiv1alpha1.SchemeGroupVersion.String(),
				Kind:       "InfrastructureStatus",
			},
			NetworkRef: commonv1alpha1.LocalUIDReference{
				Name: "my-network",
				UID:  "1234",
			},
		}
		w.Spec.InfrastructureProviderStatus = &runtime.RawExtension{Object: infraStatus}

		By("creating the worker")
		Expect(k8sClient.Create(ctx, w)).To(Succeed())

		By("creating a worker delegate")
		decoder := serializer.NewCodecFactory(k8sClient.Scheme(), serializer.EnableStrict).UniversalDecoder()
		workerDelegate, err := NewWorkerDelegate(k8sClient, decoder, k8sClient.Scheme(), "", w, testCluster)
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
						Architecture: ptr.To[string]("amd64"),
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
