/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("OCIClusterAutoscaler Controller", func() {
	const (
		resourceName = "test-autoscaler"
		namespace    = "default"
	)

	var (
		ctx                  context.Context
		typeNamespacedName   types.NamespacedName
		controllerReconciler *OCIClusterAutoscalerReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		controllerReconciler = &OCIClusterAutoscalerReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	Context("When creating a new OCIClusterAutoscaler", func() {
		var instance *capiv1alpha1.OCIClusterAutoscaler

		BeforeEach(func() {
			instance = &capiv1alpha1.OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: capiv1alpha1.OCIClusterAutoscalerSpec{
					Autoscaling: capiv1alpha1.AutoscalingConfig{
						MinNodes: 1,
						MaxNodes: 3,
						ShapeConfig: &capiv1alpha1.ShapeConfig{
							CPUs:   2,
							Memory: 4,
						},
						Shape:   "oc3",
						ImageID: "test-image",
					},
					CAPI: capiv1alpha1.CAPIConfig{
						Namespace:   "capi-system",
						ClusterName: "test-cluster",
					},
					ClusterAutoscaler: capiv1alpha1.ClusterAutoscalerConfig{
						Name:               "test-autoscaler",
						Namespace:          "test-namespace",
						ServiceAccountName: "test-sa",
						CloudProvider:      "oci",
						RepositoryURL:      "https://kubernetes.github.io/autoscaler",
						Version:            "9.29.0",
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			By("Checking the resource status")
			updatedInstance := &capiv1alpha1.OCIClusterAutoscaler{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedInstance)
			Expect(err).NotTo(HaveOccurred())

			// Verify status conditions
			Expect(updatedInstance.Status.Conditions).NotTo(BeEmpty())
			for _, condition := range updatedInstance.Status.Conditions {
				Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			}
		})

		It("should handle invalid min/max nodes configuration", func() {
			By("Updating the resource with invalid configuration")
			instance.Spec.Autoscaling.MinNodes = 5
			instance.Spec.Autoscaling.MaxNodes = 3
			Expect(k8sClient.Update(ctx, instance)).To(Succeed())

			By("Reconciling the updated resource")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("min nodes must be less than max nodes"))
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should handle missing required fields", func() {
			By("Updating the resource with missing fields")
			instance.Spec.CAPI.ClusterName = ""
			Expect(k8sClient.Update(ctx, instance)).To(Succeed())

			By("Reconciling the updated resource")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})

	Context("When deleting an OCIClusterAutoscaler", func() {
		It("should handle resource not found", func() {
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})
})
