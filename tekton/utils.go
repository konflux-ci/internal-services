/*
Copyright 2022.

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
	libhandler "github.com/operator-framework/operator-lib/handler"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetResultsFromPipelineRun returns a map with all the results emitted by the given PipelineRun. Only string results
// are supported. Other type of results will be silently omitted.
func GetResultsFromPipelineRun(pipelineRun *tektonv1.PipelineRun) map[string]string {
	results := map[string]string{}

	for _, pipelineResult := range pipelineRun.Status.Results {
		if pipelineResult.Value.Type == tektonv1.ParamTypeString {
			results[pipelineResult.Name] = pipelineResult.Value.StringVal
		}
	}

	return results
}

// isInternalRequestsPipelineRun returns a boolean indicating whether the object passed is an internal request
// PipelineRun or not.
func isInternalRequestsPipelineRun(object client.Object) bool {
	_, ok := object.(*tektonv1.PipelineRun)
	if !ok {
		return false
	}

	internalRequestKind := reflect.TypeOf(v1alpha1.InternalRequest{}).Name()

	if ownerType, ok := object.GetAnnotations()[libhandler.TypeAnnotation]; ok {
		return strings.Contains(ownerType, internalRequestKind)
	}

	return false
}

// hasPipelineSucceeded returns a boolean that is true if objectOld has not yet succeeded and objectNew has.
// If any of the objects passed to this function are not a PipelineRun, the function will return false.
func hasPipelineSucceeded(objectOld, objectNew client.Object) bool {
	if oldPipelineRun, ok := objectOld.(*tektonv1.PipelineRun); ok {
		if newPipelineRun, ok := objectNew.(*tektonv1.PipelineRun); ok {
			return !oldPipelineRun.IsDone() && newPipelineRun.IsDone()
		}
	}

	return false
}
