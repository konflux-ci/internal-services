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
	libhandler "github.com/operator-framework/operator-lib/handler"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/event"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Predicates", func() {

	Context("when testing InternalRequestPipelineRunSucceededPredicate predicate", func() {
		instance := InternalRequestPipelineRunSucceededPredicate()

		pipelineRun := &tektonv1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline-run",
				Namespace: "default",
			},
		}

		It("should ignore creating events", func() {
			contextEvent := event.CreateEvent{
				Object: pipelineRun,
			}

			Expect(instance.Create(contextEvent)).To(BeFalse())
		})

		It("should ignore deleting events", func() {
			contextEvent := event.DeleteEvent{
				Object: pipelineRun,
			}

			Expect(instance.Delete(contextEvent)).To(BeFalse())
		})

		It("should ignore generic events", func() {
			contextEvent := event.GenericEvent{
				Object: pipelineRun,
			}

			Expect(instance.Generic(contextEvent)).To(BeFalse())
		})

		It("should return true when an updated event is received for a succeeded InternalRequest PipelineRun", func() {
			donePipelineRun := pipelineRun.DeepCopy()
			donePipelineRun.Status.MarkSucceeded("", "")
			donePipelineRun.Annotations = map[string]string{
				libhandler.TypeAnnotation: "InternalRequest",
			}

			contextEvent := event.UpdateEvent{
				ObjectOld: pipelineRun,
				ObjectNew: donePipelineRun,
			}

			Expect(instance.Update(contextEvent)).To(BeTrue())
		})
	})

})
