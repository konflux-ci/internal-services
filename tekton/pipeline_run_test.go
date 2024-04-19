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
package tekton

import (
	"reflect"
	"strings"

	"github.com/konflux-ci/internal-services/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-lib/handler"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PipelineRun", Ordered, func() {
	var (
		createResources func()

		internalServicesConfig *v1alpha1.InternalServicesConfig
		internalRequest        *v1alpha1.InternalRequest
		pipeline               *tektonv1beta1.Pipeline
	)

	BeforeAll(func() {
		createResources()
	})

	Context("When calling NewInternalRequestPipelineRun", func() {
		It("should return a PipelineRun named after the InternalRequest", func() {
			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			generatedName := strings.ToLower(reflect.TypeOf(v1alpha1.InternalRequest{}).Name()) + "-"

			Expect(newInternalRequestPipelineRun.GenerateName).To(ContainSubstring(generatedName))
		})

		It("should target the InternalServicesConfig namespace", func() {
			newInternalServicesConfig := &v1alpha1.InternalServicesConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "internalServicesConfig",
					Namespace: "internalServicesConfig-namespace",
				},
			}
			newPipelineRun := NewInternalRequestPipelineRun(newInternalServicesConfig)

			Expect(newPipelineRun.Namespace).To(Equal("internalServicesConfig-namespace"))
		})
	})

	Context("When calling AsPipelineRun", func() {
		It("should return a PipelineRun representing the InternalRequest PipelineRun", func() {
			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			pipelineRun := newInternalRequestPipelineRun.AsPipelineRun()

			Expect(reflect.TypeOf(pipelineRun)).To(Equal(reflect.TypeOf(&tektonv1beta1.PipelineRun{})))
			generatedName := strings.ToLower(reflect.TypeOf(v1alpha1.InternalRequest{}).Name()) + "-"
			Expect(pipelineRun.GenerateName).To(ContainSubstring(generatedName))
			Expect(pipelineRun.Namespace).To(Equal(internalServicesConfig.Namespace))
		})
	})

	Context("When calling WithInternalRequest", func() {
		It("should append to the PipelineRun the parameters specified in the InternalRequest", func() {
			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			newInternalRequestPipelineRun.WithInternalRequest(internalRequest)

			Expect(newInternalRequestPipelineRun.Spec.Params).To(HaveLen(len(internalRequest.Spec.Params)))
			for _, param := range newInternalRequestPipelineRun.Spec.Params {
				Expect(param.Value.StringVal).To(Equal(internalRequest.Spec.Params[param.Name]))
			}
		})

		It("should contain the InternalRequest labels", func() {
			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			newInternalRequestPipelineRun.WithInternalRequest(internalRequest)

			Expect(newInternalRequestPipelineRun.Labels[InternalRequestNameLabel]).To(Equal(internalRequest.Name))
			Expect(newInternalRequestPipelineRun.Labels[InternalRequestNamespaceLabel]).To(Equal(internalRequest.Namespace))
		})
	})

	Context("When calling WithOwner", func() {
		It("should add ownership annotations", func() {
			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			newInternalRequestPipelineRun.WithOwner(internalRequest)

			Expect(newInternalRequestPipelineRun.Annotations).To(HaveLen(2))
			Expect(newInternalRequestPipelineRun.Annotations[handler.NamespacedNameAnnotation]).To(
				Equal(internalRequest.Namespace + "/" + internalRequest.Name),
			)
			Expect(newInternalRequestPipelineRun.Annotations[handler.TypeAnnotation]).To(Equal(internalRequest.Kind))
		})
	})

	Context("When calling WithPipeline", func() {
		It("should reference the Pipeline passed as an argument", func() {
			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			newInternalRequestPipelineRun.WithPipeline(pipeline, internalServicesConfig)

			Expect(newInternalRequestPipelineRun.Spec.PipelineRef).NotTo(BeNil())
			Expect(newInternalRequestPipelineRun.Spec.PipelineRef.Name).To(Equal(pipeline.Name))
		})

		It("should not contain a workspace if the Pipeline doesn't specify one", func() {
			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			newInternalRequestPipelineRun.WithPipeline(pipeline, internalServicesConfig)

			Expect(newInternalRequestPipelineRun.Spec.Workspaces).To(HaveLen(0))
		})

		It("should not contain a workspace if the Pipeline specify one with a different name from the one in the InternalServicesConfig", func() {
			newPipeline := &tektonv1beta1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: tektonv1beta1.PipelineSpec{
					Workspaces: []tektonv1beta1.PipelineWorkspaceDeclaration{
						{Name: "foo"},
					},
				},
			}

			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			newInternalRequestPipelineRun.WithPipeline(newPipeline, internalServicesConfig)

			Expect(newInternalRequestPipelineRun.Spec.Workspaces).To(HaveLen(0))
		})

		It("should contain a workspace if the Pipeline specify one with the same name seen in the InternalServicesConfig", func() {
			newPipeline := &tektonv1beta1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: "default",
				},
				Spec: tektonv1beta1.PipelineSpec{
					Workspaces: []tektonv1beta1.PipelineWorkspaceDeclaration{
						{Name: internalServicesConfig.Spec.VolumeClaim.Name},
					},
				},
			}

			newInternalRequestPipelineRun := NewInternalRequestPipelineRun(internalServicesConfig)
			newInternalRequestPipelineRun.WithPipeline(newPipeline, internalServicesConfig)

			Expect(newInternalRequestPipelineRun.Spec.Workspaces).To(HaveLen(1))
			Expect(newInternalRequestPipelineRun.Spec.Workspaces[0].Name).To(Equal(internalServicesConfig.Spec.VolumeClaim.Name))
			Expect(newInternalRequestPipelineRun.Spec.Workspaces[0].VolumeClaimTemplate).NotTo(BeNil())
		})
	})

	createResources = func() {
		internalRequest = &v1alpha1.InternalRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "request",
				Namespace: "default",
			},
			Spec: v1alpha1.InternalRequestSpec{
				Request: "request",
				Params: map[string]string{
					"foo": "bar",
					"baz": "qux",
				},
			},
		}
		internalRequest.Kind = "InternalRequest"

		internalServicesConfig = &v1alpha1.InternalServicesConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1alpha1.InternalServicesConfigResourceName,
				Namespace: "default",
			},
			Spec: v1alpha1.InternalServicesConfigSpec{
				VolumeClaim: v1alpha1.VolumeClaim{
					Name: "workspace",
					Size: "1Gi",
				},
			},
		}

		pipeline = &tektonv1beta1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "default",
			},
		}
	}

})
