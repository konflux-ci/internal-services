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
	libhandler "github.com/operator-framework/operator-lib/handler"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "knative.dev/pkg/apis/duck/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Utils", func() {

	Context("when calling GetResultsFromPipelineRun", func() {
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

		It("returns a map with all the PipelineRun results", func() {
			results := GetResultsFromPipelineRun(pipelineRun)
			Expect(results).To(HaveLen(2))
			Expect(results["foo"]).To(Equal("bar"))
			Expect(results["baz"]).To(Equal("qux"))
		})
	})

	Context("when calling isInternalRequestsPipelineRun", func() {
		It("returns true if the PipelineRun is owned by an InternalRequest object", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						libhandler.TypeAnnotation: "InternalRequest",
					},
					Name:      "pipeline-run",
					Namespace: "default",
				},
			}

			Expect(isInternalRequestsPipelineRun(pipelineRun)).To(BeTrue())
		})

		It("returns false if the PipelineRun is not owned by an InternalRequest object", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						libhandler.TypeAnnotation: "Pod",
					},
					Name:      "pipeline-run",
					Namespace: "default",
				},
			}

			Expect(isInternalRequestsPipelineRun(pipelineRun)).To(BeFalse())
		})

		It("returns false if the PipelineRun does not have the owner annotation", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline-run",
					Namespace: "default",
				},
			}

			Expect(isInternalRequestsPipelineRun(pipelineRun)).To(BeFalse())
		})
	})

	Context("when calling hasPipelineSucceeded", func() {
		It("returns false if any of the objects is not a PipelineRun", func() {
			Expect(hasPipelineSucceeded(&v1.Pod{}, &v1.Pod{})).To(BeFalse())
			Expect(hasPipelineSucceeded(&tektonv1beta1.PipelineRun{}, &v1.Pod{})).To(BeFalse())
			Expect(hasPipelineSucceeded(&v1.Pod{}, &tektonv1beta1.PipelineRun{})).To(BeFalse())
		})

		It("returns true if the event is a change where the PipelineRun status changes to done", func() {
			oldPipelineRun := &tektonv1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline-run",
					Namespace: "default",
				},
			}
			newPipelineRun := oldPipelineRun.DeepCopy()
			newPipelineRun.Status.MarkSucceeded("", "")

			Expect(hasPipelineSucceeded(oldPipelineRun, newPipelineRun)).To(BeTrue())
		})

		It("returns false if the PipelineRun event is not a change to done", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline-run",
					Namespace: "default",
				},
			}
			pipelineRun.Status.MarkSucceeded("", "")

			Expect(hasPipelineSucceeded(pipelineRun, pipelineRun)).To(BeFalse())
		})
	})

})
