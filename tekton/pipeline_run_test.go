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

		internalRequest *v1alpha1.InternalRequest
		pipelineRun     *tektonv1beta1.PipelineRun
	)

	BeforeAll(func() {
		createResources()
	})

	Context("When calling NewPipelineRun", func() {
		It("should return a PipelineRun named after the InternalRequest", func() {
			newPipelineRun := NewPipelineRun(internalRequest)
			generatedName := strings.ToLower(reflect.TypeOf(v1alpha1.InternalRequest{}).Name()) + "-"

			Expect(newPipelineRun.GenerateName).To(ContainSubstring(generatedName))
		})

		It("should contain the InternalRequest labels", func() {
			newPipelineRun := NewPipelineRun(internalRequest)

			Expect(newPipelineRun.Labels[InternalRequestNameLabel]).To(Equal(internalRequest.Name))
			Expect(newPipelineRun.Labels[InternalRequestNamespaceLabel]).To(Equal(internalRequest.Namespace))
		})

		It("should reference the Pipeline specified in the InternalRequest", func() {
			newPipelineRun := NewPipelineRun(internalRequest)

			Expect(newPipelineRun.Spec.PipelineRef.Name).To(Equal(internalRequest.Spec.Request))
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

		pipelineRun = &tektonv1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline-run",
				Namespace: "default",
			},
		}
	}

})
