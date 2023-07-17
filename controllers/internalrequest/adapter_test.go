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
	"github.com/redhat-appstudio/internal-services/loader"
	toolkit "github.com/redhat-appstudio/operator-toolkit/loader"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libhandler "github.com/operator-framework/operator-lib/handler"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("PipelineRun", Ordered, func() {

	var (
		createInternalRequestAndAdapter func() *adapter
		createResources                 func()
		deleteResources                 func()

		internalServicesConfig *v1alpha1.InternalServicesConfig
		pipeline               *tektonv1beta1.Pipeline
	)

	AfterAll(func() {
		deleteResources()
	})

	BeforeAll(func() {
		createResources()
	})

	When("newAdapter is called", func() {
		It("creates a new InternalRequest adapter", func() {
			Expect(reflect.TypeOf(newAdapter(ctx, k8sClient, k8sClient, nil, nil, &ctrl.Log))).To(Equal(reflect.TypeOf(&adapter{})))
		})
	})

	When("EnsureConfigIsLoaded is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("loads the InternalServicesConfig and assigns it to the adapter", func() {
			result, err := adapter.EnsureConfigIsLoaded()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
			Expect(adapter.internalServicesConfig).NotTo(BeNil())
		})

		It("creates and assigns a new InternalServicesConfig if none is found", func() {
			adapter.ctx = toolkit.GetMockedContext(ctx, []toolkit.MockData{
				{
					ContextKey: loader.InternalServicesConfigContextKey,
					Err:        errors.NewNotFound(schema.GroupResource{}, ""),
				},
			})

			result, err := adapter.EnsureConfigIsLoaded()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
			Expect(adapter.internalServicesConfig).NotTo(BeNil())
		})
	})

	When("EnsurePipelineExists is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("should continue if the Pipeline exists", func() {
			result, err := adapter.EnsurePipelineExists()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should set a 'No endpoint to handle' if the Pipeline was not found", func() {
			adapter.ctx = toolkit.GetMockedContext(ctx, []toolkit.MockData{
				{
					ContextKey: loader.InternalRequestPipelineContextKey,
					Err:        fmt.Errorf("not found"),
				},
			})
			adapter.internalRequest.MarkRunning()

			result, err := adapter.EnsurePipelineExists()
			Expect(result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())

			condition := meta.FindStatusCondition(adapter.internalRequest.Status.Conditions, v1alpha1.SucceededConditionType.String())
			Expect(condition.Message).To(Equal(fmt.Sprintf("No endpoint to handle '%s' requests", adapter.internalRequest.Spec.Request)))
		})
	})

	When("EnsurePipelineRunIsCreated is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
			adapter.internalRequestPipeline = pipeline
		})

		It("ensures a PipelineRun exists", func() {
			result, err := adapter.EnsurePipelineRunIsCreated()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())

			pipelineRun, err := adapter.loader.GetInternalRequestPipelineRun(ctx, k8sClient, adapter.internalRequest)
			Expect(pipelineRun).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("ensures the InternalRequest is marked as running", func() {
			result, err := adapter.EnsurePipelineRunIsCreated()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())
			Expect(adapter.internalRequest.IsRunning()).To(BeTrue())
		})
	})

	When("EnsurePipelineRunIsDeleted is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("should continue if the InternalRequest has not completed", func() {
			result, err := adapter.EnsurePipelineRunIsDeleted()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should continue if the operator is set to run in debug mode", func() {
			adapter.internalServicesConfig = &v1alpha1.InternalServicesConfig{
				Spec: v1alpha1.InternalServicesConfigSpec{
					AllowList: []string{"default"},
					Debug:     true,
				},
			}
			result, err := adapter.EnsurePipelineRunIsDeleted()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should requeue if it fails to load the InternalRequest PipelineRun", func() {
			adapter.ctx = toolkit.GetMockedContext(ctx, []toolkit.MockData{
				{
					ContextKey: loader.InternalRequestPipelineRunContextKey,
					Err:        fmt.Errorf("not found"),
				},
			})
			adapter.internalRequest.MarkRunning()
			adapter.internalRequest.MarkSucceeded()
			result, err := adapter.EnsurePipelineRunIsDeleted()
			Expect(!result.CancelRequest && result.RequeueRequest).To(BeTrue())
			Expect(err).NotTo(BeNil())
		})

		It("should delete the InternalRequest PipelineRun", func() {
			adapter.internalRequestPipeline = pipeline
			adapter.internalRequest.MarkRunning()
			adapter.internalRequest.MarkSucceeded()

			pipelineRun, err := adapter.createInternalRequestPipelineRun()
			Expect(pipelineRun).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())

			result, err := adapter.EnsurePipelineRunIsDeleted()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
		})
	})

	When("EnsureRequestIsAllowed is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("should deny any request when the spec.allowList is empty", func() {
			result, err := adapter.EnsureRequestIsAllowed()
			Expect(result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
			Expect(adapter.internalRequest.Status.Conditions).To(HaveLen(1))
			Expect(adapter.internalRequest.Status.Conditions[0].Reason).To(Equal(string(v1alpha1.RejectedReason)))
			Expect(adapter.internalRequest.Status.Conditions[0].Message).To(ContainSubstring("not in the allow list"))
		})

		It("should allow any request from a namespace in the spec.allowList", func() {
			adapter.internalServicesConfig = &v1alpha1.InternalServicesConfig{
				Spec: v1alpha1.InternalServicesConfigSpec{
					AllowList: []string{"default"},
				},
			}

			result, err := adapter.EnsureRequestIsAllowed()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should deny any request from a namespace not in the spec.allowList", func() {
			adapter.ctx = toolkit.GetMockedContext(ctx, []toolkit.MockData{
				{
					ContextKey: loader.InternalServicesConfigContextKey,
					Resource: &v1alpha1.InternalServicesConfig{
						Spec: v1alpha1.InternalServicesConfigSpec{
							AllowList: []string{"foo"},
						},
					},
				},
			})
			result, err := adapter.EnsureRequestIsAllowed()
			Expect(result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
			Expect(adapter.internalRequest.Status.Conditions).To(HaveLen(1))
			Expect(adapter.internalRequest.Status.Conditions[0].Reason).To(Equal(string(v1alpha1.RejectedReason)))
			Expect(adapter.internalRequest.Status.Conditions[0].Message).To(ContainSubstring("not in the allow list"))
		})
	})

	When("EnsureRequestINotCompleted is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("should stop processing when the InternalRequest is completed", func() {
			adapter.internalRequest.MarkRunning()
			adapter.internalRequest.MarkSucceeded()
			result, err := adapter.EnsureRequestINotCompleted()
			Expect(result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
		})

		It("should continue processing when the InternalRequest is not completed", func() {
			result, err := adapter.EnsureRequestINotCompleted()
			Expect(!result.CancelRequest && !result.RequeueRequest).To(BeTrue())
			Expect(err).To(BeNil())
		})
	})

	When("EnsureStatusIsTracked is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("should succeed when a PipelineRun exists", func() {
			result, err := adapter.EnsurePipelineRunIsCreated()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())

			result, err = adapter.EnsureStatusIsTracked()
			Expect(!result.CancelRequest && err == nil).Should(BeTrue())
		})
	})

	When("createInternalRequestPipelineRun is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
			adapter.internalRequestPipeline = pipeline
		})

		It("creates a PipelineRun with the InternalRequest params and labels", func() {
			pipelineRun, err := adapter.createInternalRequestPipelineRun()
			Expect(pipelineRun).NotTo(BeNil())
			Expect(err).To(BeNil())
			Expect(pipelineRun.Labels).To(HaveLen(2))
			Expect(pipelineRun.Spec.Params).To(HaveLen(len(adapter.internalRequest.Spec.Params)))
		})

		It("creates a PipelineRun owned by the InternalRequest", func() {
			pipelineRun, err := adapter.createInternalRequestPipelineRun()
			Expect(pipelineRun).NotTo(BeNil())
			Expect(err).To(BeNil())
			Expect(pipelineRun.Annotations).To(HaveLen(2))
			Expect(pipelineRun.Annotations[libhandler.NamespacedNameAnnotation]).To(
				Equal(adapter.internalRequest.Namespace + "/" + adapter.internalRequest.Name),
			)
			Expect(pipelineRun.Annotations[libhandler.TypeAnnotation]).To(Equal(adapter.internalRequest.Kind))
		})

		It("creates a PipelineRun referencing the Pipeline requested in the InternalRequest", func() {
			pipelineRun, err := adapter.createInternalRequestPipelineRun()
			Expect(pipelineRun).NotTo(BeNil())
			Expect(err).To(BeNil())
			Expect(pipelineRun.Spec.PipelineRef.Name).To(Equal(adapter.internalRequestPipeline.Name))
		})
	})

	When("getDefaultInternalServicesConfig is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("should return a InternalServicesConfig without Spec and with the right ObjectMeta", func() {
			internalServicesConfig := adapter.getDefaultInternalServicesConfig("namespace")
			Expect(internalServicesConfig).NotTo(BeNil())
			Expect(internalServicesConfig.Name).To(Equal(v1alpha1.InternalServicesConfigResourceName))
			Expect(internalServicesConfig.Namespace).To(Equal("namespace"))
		})
	})

	When("registerInternalRequestStatus is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
		})

		It("should return nil if the PipelineRun is nil", func() {
			Expect(adapter.registerInternalRequestStatus(nil)).To(BeNil())
		})

		It("should mark the InternalRequest as running", func() {
			Expect(adapter.internalRequest.IsRunning()).To(BeFalse())
			Expect(adapter.registerInternalRequestStatus(&tektonv1beta1.PipelineRun{})).To(BeNil())
			Expect(adapter.internalRequest.IsRunning()).To(BeTrue())
		})
	})

	When("registerInternalRequestPipelineRunStatus is called", func() {
		var adapter *adapter

		AfterEach(func() {
			_ = adapter.client.Delete(ctx, adapter.internalRequest)
		})

		BeforeEach(func() {
			adapter = createInternalRequestAndAdapter()
			adapter.internalRequest.MarkRunning()
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
			adapter.internalRequest.MarkRunning()
			pipelineRun := &tektonv1beta1.PipelineRun{}
			pipelineRun.Status.MarkSucceeded("", "")
			Expect(adapter.registerInternalRequestPipelineRunStatus(pipelineRun)).To(BeNil())
			Expect(adapter.internalRequest.HasSucceeded()).To(BeTrue())
		})

		It("should set the InternalRequest as failed if the PipelineRun failed", func() {
			adapter.internalRequest.MarkRunning()
			pipelineRun := &tektonv1beta1.PipelineRun{}
			pipelineRun.Status.MarkFailed("", "")
			Expect(adapter.registerInternalRequestPipelineRunStatus(pipelineRun)).To(BeNil())
			Expect(adapter.internalRequest.HasSucceeded()).To(BeFalse())
		})
	})

	createInternalRequestAndAdapter = func() *adapter {
		internalRequest := &v1alpha1.InternalRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "request",
				Namespace: "default",
			},
			Spec: v1alpha1.InternalRequestSpec{
				Request: pipeline.Name,
			},
		}
		Expect(k8sClient.Create(ctx, internalRequest)).To(Succeed())

		// Set a proper Kind
		internalRequest.Kind = "InternalRequest"

		adapter := newAdapter(ctx, k8sClient, k8sClient, internalRequest, loader.NewMockLoader(), &ctrl.Log)
		adapter.internalServicesConfig = internalServicesConfig

		return adapter
	}

	createResources = func() {
		internalServicesConfig = &v1alpha1.InternalServicesConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1alpha1.InternalServicesConfigResourceName,
				Namespace: "default",
			},
		}
		Expect(k8sClient.Create(ctx, internalServicesConfig)).To(Succeed())

		pipeline = &tektonv1beta1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "default",
			},
		}
		Expect(k8sClient.Create(ctx, pipeline)).To(Succeed())
	}

	deleteResources = func() {
		Expect(k8sClient.Delete(ctx, internalServicesConfig)).To(Succeed())
		Expect(k8sClient.Delete(ctx, pipeline)).To(Succeed())
		err := k8sClient.DeleteAllOf(ctx, &tektonv1beta1.PipelineRun{})
		Expect(err == nil || errors.IsNotFound(err)).To(BeTrue())
	}

})
