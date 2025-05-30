/*
Copyright 2023.

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

package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"reflect"
)

var _ = Describe("Pipeline", func() {
	var (
		clusterRef PipelineRef
		gitRef     PipelineRef
		bundleRef  PipelineRef
	)

	BeforeEach(func() {
		clusterRef = PipelineRef{
			Resolver: "cluster",
			Params: []Param{
				{Name: "kind", Value: "pipeline"},
				{Name: "name", Value: "my-cluster-pipeline"},
				{Name: "namespace", Value: "my-namespace"},
			},
		}
		gitRef = PipelineRef{
			Resolver: "git",
			Params: []Param{
				{Name: "url", Value: "my-git-url"},
				{Name: "revision", Value: "my-revision"},
				{Name: "pathInRepo", Value: "my-path-in-repo"},
			},
		}
		bundleRef = PipelineRef{
			Resolver: "bundles",
			Params: []Param{
				{Name: "bundle", Value: "my-bundle"},
				{Name: "name", Value: "my-pipeline"},
				{Name: "kind", Value: "pipeline"},
			},
		}
	})

	When("ToTektonPipelineRef method is called", func() {
		It("should return Tekton PipelineRef representation of the PipelineRef", func() {
			ref := clusterRef.ToTektonPipelineRef()
			Expect(string(ref.ResolverRef.Resolver)).To(Equal("cluster"))
			params := ref.ResolverRef.Params
			Expect(params[0].Name).To(Equal("kind"))
			Expect(params[0].Value.StringVal).To(Equal("pipeline"))
			Expect(params[1].Name).To(Equal("name"))
			Expect(params[1].Value.StringVal).To(Equal("my-cluster-pipeline"))
			Expect(params[2].Name).To(Equal("namespace"))
			Expect(params[2].Value.StringVal).To(Equal("my-namespace"))

			ref = gitRef.ToTektonPipelineRef()
			Expect(string(ref.ResolverRef.Resolver)).To(Equal("git"))
			params = ref.ResolverRef.Params
			Expect(params[0].Name).To(Equal("url"))
			Expect(params[0].Value.StringVal).To(Equal("my-git-url"))
			Expect(params[1].Name).To(Equal("revision"))
			Expect(params[1].Value.StringVal).To(Equal("my-revision"))
			Expect(params[2].Name).To(Equal("pathInRepo"))
			Expect(params[2].Value.StringVal).To(Equal("my-path-in-repo"))

			ref = bundleRef.ToTektonPipelineRef()
			Expect(string(ref.ResolverRef.Resolver)).To(Equal("bundles"))
			params = ref.ResolverRef.Params
			Expect(params[0].Name).To(Equal("bundle"))
			Expect(params[0].Value.StringVal).To(Equal("my-bundle"))
			Expect(params[1].Name).To(Equal("name"))
			Expect(params[1].Value.StringVal).To(Equal("my-pipeline"))
			Expect(params[2].Name).To(Equal("kind"))
			Expect(params[2].Value.StringVal).To(Equal("pipeline"))
		})
	})

	When("GetPipelineNameFromGitResolver method is called", func() {
		It("should return the empty string if there is no pathInRepo parameter", func() {
			parameterizedPipeline := ParameterizedPipeline{}
			parameterizedPipeline.Params = []Param{
				{Name: "parameter1", Value: "value1"},
			}

			name := parameterizedPipeline.GetPipelineNameFromGitResolver()
			Expect(name).To(Equal(""))
		})

		It("should return the proper name when passed a nested yaml", func() {
			parameterizedPipeline := ParameterizedPipeline{}
			parameterizedPipeline.Params = []Param{
				{Name: "parameter1", Value: "value1"},
				{Name: "pathInRepo", Value: "pipelines/internal/my-pipeline/my-pipeline.yaml"},
			}

			name := parameterizedPipeline.GetPipelineNameFromGitResolver()
			Expect(name).To(Equal("my-pipeline"))
		})

		It("should return the proper name when passed just a filename", func() {
			parameterizedPipeline := ParameterizedPipeline{}
			parameterizedPipeline.Params = []Param{
				{Name: "parameter1", Value: "value1"},
				{Name: "pathInRepo", Value: "my-pipeline.yaml"},
			}

			name := parameterizedPipeline.GetPipelineNameFromGitResolver()
			Expect(name).To(Equal("my-pipeline"))
		})
	})

	When("GetTektonParams method is called", func() {
		It("should return a tekton Param list", func() {
			parameterizedPipeline := ParameterizedPipeline{}
			parameterizedPipeline.Params = []Param{
				{Name: "parameter1", Value: "value1"},
				{Name: "parameter2", Value: "value2"},
			}

			params := parameterizedPipeline.GetTektonParams()
			Expect(reflect.TypeOf(params[0])).To(Equal(reflect.TypeOf(tektonv1.Param{})))
			Expect(params[0].Name).To(Equal("parameter1"))
			Expect(params[0].Value.StringVal).To(Equal("value1"))

			Expect(reflect.TypeOf(params[1])).To(Equal(reflect.TypeOf(tektonv1.Param{})))
			Expect(params[1].Name).To(Equal("parameter2"))
			Expect(params[1].Value.StringVal).To(Equal("value2"))
		})
	})

	When("IsClusterScoped method is called", func() {
		It("should return true for a cluster pipeline", func() {
			Expect(clusterRef.IsClusterScoped()).To(BeTrue())
		})

		It("should return false for non-cluster pipelines", func() {
			Expect(gitRef.IsClusterScoped()).To(BeFalse())
			Expect(bundleRef.IsClusterScoped()).To(BeFalse())
		})
	})

})
