/*
Copyright 2022 Red Hat Inc.

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

package internalrequest

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libhandler "github.com/operator-framework/operator-lib/handler"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	"github.com/redhat-appstudio/internal-services/loader"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("PipelineRun", Ordered, func() {
	var (
		createResources func()
		deleteResources func()

		adapter *Adapter
	)

	Context("When calling NewAdapter", func() {
		It("creates a new InternalRequest adapter", func() {
			Expect(reflect.TypeOf(NewAdapter(ctx, k8sClient, k8sClient, nil, nil, ctrl.Log))).To(Equal(reflect.TypeOf(&Adapter{})))
		})
	})

	Context("When calling EnsurePipelineRunIsCreated", func() {
		AfterEach(func() {
			deleteResources()
		})

		BeforeEach(func() {
			createResources()
		})

		It("ensures a PipelineRun exists", func() {
			result, err := adapter.EnsurePipelineRunIsCreated()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())

			pipelineRun, err := adapter.loader.GetInternalRequestPipelineRun(ctx, k8sClient, adapter.internalRequest)
			Expect(pipelineRun).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("ensures the pipelineRun is owned by the InternalRequest", func() {
			result, err := adapter.EnsurePipelineRunIsCreated()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())

			pipelineRun, err := adapter.loader.GetInternalRequestPipelineRun(ctx, k8sClient, adapter.internalRequest)
			Expect(pipelineRun).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
			Expect(pipelineRun.Annotations).To(HaveLen(2))
			Expect(pipelineRun.Annotations[libhandler.NamespacedNameAnnotation]).To(
				Equal(adapter.internalRequest.Namespace + "/" + adapter.internalRequest.Name),
			)
			Expect(pipelineRun.Annotations[libhandler.TypeAnnotation]).To(Equal(adapter.internalRequest.Kind))
		})

		It("ensures the InternalRequest is marked as running", func() {
			result, err := adapter.EnsurePipelineRunIsCreated()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())
			Expect(adapter.internalRequest.HasStarted()).To(BeTrue())
		})
	})

	Context("When calling EnsureStatusIsTracked", func() {
		AfterEach(func() {
			deleteResources()
		})

		BeforeEach(func() {
			createResources()
		})

		It("should succeed when a PipelineRun exists", func() {
			result, err := adapter.EnsurePipelineRunIsCreated()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())

			result, err = adapter.EnsureStatusIsTracked()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())
		})
	})

	Context("When calling registerInternalRequestStatus", func() {
		AfterEach(func() {
			deleteResources()
		})

		BeforeEach(func() {
			createResources()
		})

		It("should return nil if the PipelineRun is nil", func() {
			Expect(adapter.registerInternalRequestStatus(nil)).To(BeNil())
		})

		It("should mark the InternalRequest as running", func() {
			Expect(adapter.internalRequest.HasStarted()).To(BeFalse())
			Expect(adapter.registerInternalRequestStatus(&tektonv1beta1.PipelineRun{})).To(BeNil())
			Expect(adapter.internalRequest.HasStarted()).To(BeTrue())
		})
	})

	Context("When calling registerInternalRequestPipelineRunStatus", func() {
		AfterEach(func() {
			deleteResources()
		})

		BeforeEach(func() {
			createResources()
		})

		It("should return nil if the PipelineRun is nil", func() {
			Expect(adapter.registerInternalRequestPipelineRunStatus(nil)).To(BeNil())
		})

		It("should return nil if the PipelineRun is not done", func() {
			Expect(adapter.registerInternalRequestPipelineRunStatus(&tektonv1beta1.PipelineRun{})).To(BeNil())
		})

		It("should copy the results emitted by a successful PipelineRun", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline-run",
					Namespace: "default",
				},
				Status: tektonv1beta1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
						PipelineResults: []tektonv1beta1.PipelineRunResult{
							{
								Name: "foo",
								Value: tektonv1beta1.ResultValue{
									StringVal: "bar",
									Type:      tektonv1beta1.ParamTypeString,
								},
							},
							{
								Name: "baz",
								Value: tektonv1beta1.ResultValue{
									StringVal: "qux",
									Type:      tektonv1beta1.ParamTypeString,
								},
							},
						},
					},
				},
			}
			pipelineRun.Status.MarkSucceeded("", "")
			Expect(adapter.registerInternalRequestPipelineRunStatus(pipelineRun)).To(BeNil())
			Expect(adapter.internalRequest.Status.Results).To(HaveLen(2))
			Expect(adapter.internalRequest.Status.Results["foo"]).To(Equal("bar"))
			Expect(adapter.internalRequest.Status.Results["baz"]).To(Equal("qux"))
		})

		It("should set the InternalRequest as succeeded if the PipelineRun succeeded", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{}
			pipelineRun.Status.MarkSucceeded("", "")
			Expect(adapter.registerInternalRequestPipelineRunStatus(pipelineRun)).To(BeNil())
			Expect(adapter.internalRequest.HasSucceeded()).To(BeTrue())
		})

		It("should set the Release as failed if the PipelineRun failed", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{}
			pipelineRun.Status.MarkFailed("", "")
			Expect(adapter.registerInternalRequestPipelineRunStatus(pipelineRun)).To(BeNil())
			Expect(adapter.internalRequest.HasSucceeded()).To(BeFalse())
		})

		It("should set a 'No endpoint to handle' if the Pipeline was not found", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline-run",
					Namespace: "default",
				},
				Spec: tektonv1beta1.PipelineRunSpec{
					PipelineRef: &tektonv1beta1.PipelineRef{
						Name: "not-found",
					},
				},
			}
			pipelineRun.Status.MarkFailed("", "", "not found")
			Expect(adapter.registerInternalRequestPipelineRunStatus(pipelineRun)).To(BeNil())
			Expect(adapter.internalRequest.HasSucceeded()).To(BeFalse())

			condition := meta.FindStatusCondition(adapter.internalRequest.Status.Conditions, v1alpha1.InternalRequestSucceededConditionType)
			Expect(condition.Message).To(Equal(fmt.Sprintf("No endpoint to handle '%s' requests", pipelineRun.Spec.PipelineRef.Name)))
		})
	})

	createResources = func() {
		internalRequest := &v1alpha1.InternalRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "request",
				Namespace: "default",
			},
			Spec: v1alpha1.InternalRequestSpec{
				Request: "request",
			},
		}
		Expect(k8sClient.Create(ctx, internalRequest)).To(Succeed())

		// Set a proper Kind
		internalRequest.TypeMeta = metav1.TypeMeta{
			Kind: "InternalRequest",
		}

		adapter = NewAdapter(ctx, k8sClient, k8sClient, internalRequest, loader.NewLoader(), ctrl.Log)
	}

	deleteResources = func() {
		Expect(k8sClient.Delete(ctx, adapter.internalRequest)).To(Succeed())
		err := k8sClient.DeleteAllOf(ctx, &tektonv1beta1.PipelineRun{})
		Expect(err == nil || errors.IsNotFound(err)).To(BeTrue())
	}

})
