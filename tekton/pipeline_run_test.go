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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strings"
)

var _ = Describe("PipelineRun", Ordered, func() {
	var (
		createResources func()

		internalServicesConfig *v1alpha1.InternalServicesConfig
		internalRequest        *v1alpha1.InternalRequest
		pipelineRun            *tektonv1beta1.PipelineRun
	)

	BeforeAll(func() {
		createResources()
	})

	Context("When calling NewPipelineRun", func() {
		It("should return a PipelineRun named after the InternalRequest", func() {
			newPipelineRun := NewPipelineRun(internalRequest, internalServicesConfig)
			generatedName := strings.ToLower(reflect.TypeOf(v1alpha1.InternalRequest{}).Name()) + "-"

			Expect(newPipelineRun.GenerateName).To(ContainSubstring(generatedName))
		})

		It("should contain the InternalRequest labels", func() {
			newPipelineRun := NewPipelineRun(internalRequest, internalServicesConfig)

			Expect(newPipelineRun.Labels[InternalRequestNameLabel]).To(Equal(internalRequest.Name))
			Expect(newPipelineRun.Labels[InternalRequestNamespaceLabel]).To(Equal(internalRequest.Namespace))
		})

		It("should target the InternalServicesConfig namespace if spec.catalog.namespace is not set", func() {
			newInternalServicesConfig := &v1alpha1.InternalServicesConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "internalServicesConfig",
					Namespace: "internalServicesConfig-namespace",
				},
			}
			newPipelineRun := NewPipelineRun(internalRequest, newInternalServicesConfig)

			Expect(newPipelineRun.Namespace).To(Equal("internalServicesConfig-namespace"))
		})

		It("should target the InternalServicesConfig spec.catalog.namespace if set", func() {
			newPipelineRun := NewPipelineRun(internalRequest, internalServicesConfig)

			Expect(newPipelineRun.Namespace).To(Equal(internalServicesConfig.Namespace))
		})

		It("should reference the Pipeline specified in the InternalRequest", func() {
			newPipelineRun := NewPipelineRun(internalRequest, internalServicesConfig)

			Expect(newPipelineRun.Spec.PipelineRef.Name).To(Equal(internalRequest.Spec.Request))
		})

		It("should reference a bundle if set in the InternalServicesConfig", func() {
			newInternalServicesConfig := &v1alpha1.InternalServicesConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "internalServicesConfig",
					Namespace: "internalServicesConfig-namespace",
				},
				Spec: v1alpha1.InternalServicesConfigSpec{
					Catalog: v1alpha1.Catalog{
						Bundle: "quay.io/foo/bar:baz",
					},
				},
			}
			newPipelineRun := NewPipelineRun(internalRequest, newInternalServicesConfig)

			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef).NotTo(Equal(tektonv1beta1.ResolverRef{}))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Resolver).To(Equal(tektonv1beta1.ResolverName("bundles")))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Params).To(HaveLen(3))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Params[0].Name).To(Equal("bundle"))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Params[0].Value.StringVal).To(Equal(newInternalServicesConfig.Spec.Catalog.Bundle))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Params[1].Name).To(Equal("kind"))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Params[1].Value.StringVal).To(Equal("pipeline"))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Params[2].Name).To(Equal("name"))
			Expect(newPipelineRun.Spec.PipelineRef.ResolverRef.Params[2].Value.StringVal).To(Equal(internalRequest.Spec.Request))
		})
	})

	Context("When calling appendInternalRequestParams", func() {
		It("should append to the PipelineRun the parameters specified in the InternalRequest", func() {
			newPipelineRun := pipelineRun.DeepCopy()
			appendInternalRequestParams(newPipelineRun, internalRequest)

			Expect(newPipelineRun.Spec.Params).To(HaveLen(len(internalRequest.Spec.Params)))
			for _, param := range newPipelineRun.Spec.Params {
				Expect(param.Value.StringVal).To(Equal(internalRequest.Spec.Params[param.Name]))
			}
		})
	})

	Context("When calling getPipelineRef", func() {
		It("should return a PipelineRef without resolver if the internalServicesConfig contains no bundle", func() {
			pipelineRef := getPipelineRef(internalRequest, internalServicesConfig)
			Expect(pipelineRef.Name).To(Equal(internalRequest.Name))
			Expect(pipelineRef.ResolverRef).To(Equal(tektonv1beta1.ResolverRef{}))
		})

		It("should return a PipelineRef with a bundle resolver if the internalServicesConfig contains a bundle", func() {
			newInternalServicesConfig := &v1alpha1.InternalServicesConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "internalServicesConfig",
					Namespace: "internalServicesConfig-namespace",
				},
				Spec: v1alpha1.InternalServicesConfigSpec{
					Catalog: v1alpha1.Catalog{
						Bundle: "quay.io/foo/bar:baz",
					},
				},
			}
			pipelineRef := getPipelineRef(internalRequest, newInternalServicesConfig)
			Expect(pipelineRef.Name).To(BeEmpty())
			Expect(pipelineRef.ResolverRef).NotTo(Equal(tektonv1beta1.ResolverRef{}))
			Expect(pipelineRef.ResolverRef.Resolver).To(Equal(tektonv1beta1.ResolverName("bundles")))
			Expect(pipelineRef.ResolverRef.Params).To(HaveLen(3))
			Expect(pipelineRef.ResolverRef.Params[0].Name).To(Equal("bundle"))
			Expect(pipelineRef.ResolverRef.Params[0].Value.StringVal).To(Equal(newInternalServicesConfig.Spec.Catalog.Bundle))
			Expect(pipelineRef.ResolverRef.Params[1].Name).To(Equal("kind"))
			Expect(pipelineRef.ResolverRef.Params[1].Value.StringVal).To(Equal("pipeline"))
			Expect(pipelineRef.ResolverRef.Params[2].Name).To(Equal("name"))
			Expect(pipelineRef.ResolverRef.Params[2].Value.StringVal).To(Equal(internalRequest.Spec.Request))
		})
	})

	Context("When calling getBundleResolver", func() {
		It("should return a bundle resolver referencing the InternalServicesConfig bundle and Pipeline", func() {
			resolver := getBundleResolver("quay.io/foo/bar:baz", "pipeline")
			Expect(resolver).NotTo(Equal(tektonv1beta1.ResolverRef{}))
			Expect(resolver.Resolver).To(Equal(tektonv1beta1.ResolverName("bundles")))
			Expect(resolver.Params).To(HaveLen(3))
			Expect(resolver.Params[0].Name).To(Equal("bundle"))
			Expect(resolver.Params[0].Value.StringVal).To(Equal("quay.io/foo/bar:baz"))
			Expect(resolver.Params[1].Name).To(Equal("kind"))
			Expect(resolver.Params[1].Value.StringVal).To(Equal("pipeline"))
			Expect(resolver.Params[2].Name).To(Equal("name"))
			Expect(resolver.Params[2].Value.StringVal).To(Equal("pipeline"))
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

		internalServicesConfig = &v1alpha1.InternalServicesConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1alpha1.InternalServicesConfigResourceName,
				Namespace: "default",
			},
		}

		pipelineRun = &tektonv1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline-run",
				Namespace: "default",
			},
		}
	}

})
