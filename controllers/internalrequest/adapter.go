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

package internalrequest

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/konflux-ci/internal-services/api/v1alpha1"
	"github.com/konflux-ci/internal-services/loader"
	"github.com/konflux-ci/internal-services/tekton"
	"github.com/konflux-ci/operator-toolkit/controller"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Adapter holds the objects needed to reconcile an InternalRequest.
type Adapter struct {
	client                  client.Client
	internalServicesConfig  *v1alpha1.InternalServicesConfig
	ctx                     context.Context
	internalClient          client.Client
	internalRequest         *v1alpha1.InternalRequest
	internalRequestPipeline *tektonv1.Pipeline
	loader                  loader.ObjectLoader
	logger                  logr.Logger
}

// NewAdapter creates and returns an Adapter instance.
func NewAdapter(ctx context.Context, client, internalClient client.Client, internalRequest *v1alpha1.InternalRequest, loader loader.ObjectLoader, logger logr.Logger) *Adapter {
	return &Adapter{
		client:          client,
		ctx:             ctx,
		internalRequest: internalRequest,
		internalClient:  internalClient,
		loader:          loader,
		logger:          logger,
	}
}

// EnsureConfigIsLoaded is an operation that will load the service InternalServicesConfig from the manager namespace. If not found,
// a new InternalServicesConfig resource will be generated and attached to the adapter.
//
// Note: This operation sets values in the adapter to be used by other operations, so it should be always enabled.
func (a *Adapter) EnsureConfigIsLoaded() (controller.OperationResult, error) {
	namespace := os.Getenv("SERVICE_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	var err error
	a.internalServicesConfig, err = a.loader.GetInternalServicesConfig(a.ctx, a.internalClient, v1alpha1.InternalServicesConfigResourceName, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return controller.RequeueWithError(err)
	}

	if err != nil {
		a.internalServicesConfig = a.getDefaultInternalServicesConfig(namespace)
	}

	return controller.ContinueProcessing()
}

// EnsurePipelineRunIsCreated is an operation that will ensure that the InternalRequest is handled by creating a new
// PipelineRun for the Pipeline referenced in the Request field.
func (a *Adapter) EnsurePipelineRunIsCreated() (controller.OperationResult, error) {
	pipelineRun, err := a.loader.GetInternalRequestPipelineRun(a.ctx, a.internalClient, a.internalRequest)
	if err != nil && !errors.IsNotFound(err) {
		return controller.RequeueWithError(err)
	}

	if pipelineRun == nil || !a.internalRequest.IsRunning() {
		if pipelineRun == nil {
			pipelineRun, err = a.createInternalRequestPipelineRun()
			if err != nil {
				return controller.RequeueWithError(err)
			}

			a.logger.Info("Created PipelineRun to handle request",
				"PipelineRun.Name", pipelineRun.Name, "PipelineRun.Namespace", pipelineRun.Namespace)
		}

		return controller.RequeueOnErrorOrContinue(a.registerInternalRequestStatus(pipelineRun))
	}

	return controller.ContinueProcessing()
}

// EnsurePipelineRunIsDeleted is an operation that will ensure that the PipelineRun created to handle the InternalRequest
// is deleted once it finishes.
func (a *Adapter) EnsurePipelineRunIsDeleted() (controller.OperationResult, error) {
	if !a.internalRequest.HasCompleted() {
		return controller.ContinueProcessing()
	}

	if a.internalServicesConfig.Spec.Debug {
		a.logger.Info("Running in debug mode. Skipping PipelineRun deletion")

		return controller.ContinueProcessing()
	}

	pipelineRun, err := a.loader.GetInternalRequestPipelineRun(a.ctx, a.internalClient, a.internalRequest)
	if err != nil {
		return controller.RequeueWithError(err)
	}

	return controller.RequeueOnErrorOrContinue(a.internalClient.Delete(a.ctx, pipelineRun))
}

// EnsureRequestIsAllowed is an operation that will ensure that the request is coming from a namespace allowed
// to execute InternalRequests. If the InternalServicesConfig spec.allowList is empty, any request will be allowed regardless of the
// remote namespace.
func (a *Adapter) EnsureRequestIsAllowed() (controller.OperationResult, error) {
	for _, namespace := range a.internalServicesConfig.Spec.AllowList {
		if namespace == a.internalRequest.Namespace {
			return controller.ContinueProcessing()
		}
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())
	a.internalRequest.MarkRejected(
		fmt.Sprintf("the internal request namespace (%s) is not in the allow list", a.internalRequest.Namespace),
	)
	return controller.RequeueOnErrorOrStop(a.client.Status().Patch(a.ctx, a.internalRequest, patch))
}

// EnsureRequestINotCompleted is an operation that will stop processing a request if it was completed already.
func (a *Adapter) EnsureRequestINotCompleted() (controller.OperationResult, error) {
	if a.internalRequest.HasCompleted() {
		return controller.StopProcessing()
	}

	return controller.ContinueProcessing()
}

// EnsureStatusIsTracked is an operation that will ensure that the InternalRequest PipelineRun status is tracked
// in the InternalRequest being processed.
func (a *Adapter) EnsureStatusIsTracked() (controller.OperationResult, error) {
	pipelineRun, err := a.loader.GetInternalRequestPipelineRun(a.ctx, a.internalClient, a.internalRequest)
	if err != nil && !errors.IsNotFound(err) {
		return controller.RequeueWithError(err)
	}

	if pipelineRun != nil {
		return controller.RequeueOnErrorOrContinue(a.registerInternalRequestPipelineRunStatus(pipelineRun))
	}

	return controller.ContinueProcessing()
}

// createInternalRequestPipelineRun creates and returns a new InternalRequest PipelineRun. The new PipelineRun will
// include owner annotations, so it triggers InternalRequest reconciles whenever it changes. The Pipeline information
// and its parameters will be extracted from the InternalRequest.
func (a *Adapter) createInternalRequestPipelineRun() (*tektonv1.PipelineRun, error) {
	pipelineRun := tekton.NewInternalRequestPipelineRun(a.internalServicesConfig).
		WithInternalRequest(a.internalRequest).
		WithOwner(a.internalRequest).
		WithPipelineRef(a.internalRequest, a.internalServicesConfig).
		AsPipelineRun()

	err := a.internalClient.Create(a.ctx, pipelineRun)
	if err != nil {
		return nil, err
	}

	return pipelineRun, nil
}

// getDefaultInternalServicesConfig creates and returns a InternalServicesConfig resource in the given namespace with default values.
func (a *Adapter) getDefaultInternalServicesConfig(namespace string) *v1alpha1.InternalServicesConfig {
	return &v1alpha1.InternalServicesConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      v1alpha1.InternalServicesConfigResourceName,
			Namespace: namespace,
		},
	}
}

// registerInternalRequestStatus sets the InternalRequest to Running.
func (a *Adapter) registerInternalRequestStatus(pipelineRun *tektonv1.PipelineRun) error {
	if pipelineRun == nil {
		return nil
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())

	a.internalRequest.MarkRunning()

	return a.client.Status().Patch(a.ctx, a.internalRequest, patch)
}

// registerInternalRequestPipelineRunStatus keeps track of the PipelineRun status in the InternalRequest being processed.
func (a *Adapter) registerInternalRequestPipelineRunStatus(pipelineRun *tektonv1.PipelineRun) error {
	if pipelineRun == nil {
		return nil
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())

	a.internalRequest.Status.PipelineRun = fmt.Sprintf("%s%c%s",
		pipelineRun.Namespace, types.Separator, pipelineRun.Name)

	if pipelineRun.IsDone() {
		condition := pipelineRun.Status.GetCondition(apis.ConditionSucceeded)
		if condition.IsTrue() {
			a.internalRequest.Status.Results = tekton.GetResultsFromPipelineRun(pipelineRun)
			a.internalRequest.MarkSucceeded()
		} else {
			a.internalRequest.MarkFailed(condition.Message)
		}
		a.logger.Info("Request execution finished", "Succeeded", a.internalRequest.HasSucceeded())
	}

	err := a.client.Status().Patch(a.ctx, a.internalRequest, patch)
	if err != nil {
		return err
	}

	return nil
}
