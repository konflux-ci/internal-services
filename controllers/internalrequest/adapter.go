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
	"github.com/go-logr/logr"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	"github.com/redhat-appstudio/internal-services/loader"
	"github.com/redhat-appstudio/internal-services/tekton"
	"github.com/redhat-appstudio/operator-toolkit/controller"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// adapter holds the objects needed to reconcile an InternalRequest.
type adapter struct {
	client                  client.Client
	internalServicesConfig  *v1alpha1.InternalServicesConfig
	ctx                     context.Context
	internalClient          client.Client
	internalRequest         *v1alpha1.InternalRequest
	internalRequestPipeline *v1beta1.Pipeline
	loader                  loader.ObjectLoader
	logger                  logr.Logger
}

// newAdapter creates and returns an adapter instance.
func newAdapter(ctx context.Context, client, internalClient client.Client, internalRequest *v1alpha1.InternalRequest, loader loader.ObjectLoader, logger logr.Logger) *adapter {
	return &adapter{
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
func (a *adapter) EnsureConfigIsLoaded() (controller.OperationResult, error) {
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

// EnsurePipelineExists is an operation that will ensure the Pipeline referenced by the InternalRequest exists and add it
// to the adapter, so it can be used in other operations. If the Pipeline doesn't exist, the InternalRequest will be
// marked as failed.
//
// Note: This operation sets values in the adapter to be used by other operations, so it should be always enabled.
func (a *adapter) EnsurePipelineExists() (controller.OperationResult, error) {
	var err error
	a.internalRequestPipeline, err = a.loader.GetInternalRequestPipeline(a.ctx, a.internalClient,
		a.internalRequest.Spec.Request, a.internalServicesConfig.Namespace)

	if err != nil {
		patch := client.MergeFrom(a.internalRequest.DeepCopy())
		a.internalRequest.MarkFailed(fmt.Sprintf("No endpoint to handle '%s' requests", a.internalRequest.Spec.Request))
		return controller.RequeueOnErrorOrStop(a.client.Status().Patch(a.ctx, a.internalRequest, patch))
	}

	return controller.ContinueProcessing()
}

// EnsurePipelineRunIsCreated is an operation that will ensure that the InternalRequest is handled by creating a new
// PipelineRun for the Pipeline referenced in the Request field.
func (a *adapter) EnsurePipelineRunIsCreated() (controller.OperationResult, error) {
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
func (a *adapter) EnsurePipelineRunIsDeleted() (controller.OperationResult, error) {
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
func (a *adapter) EnsureRequestIsAllowed() (controller.OperationResult, error) {
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
func (a *adapter) EnsureRequestINotCompleted() (controller.OperationResult, error) {
	if a.internalRequest.HasCompleted() {
		return controller.StopProcessing()
	}

	return controller.ContinueProcessing()
}

// EnsureStatusIsTracked is an operation that will ensure that the InternalRequest PipelineRun status is tracked
// in the InternalRequest being processed.
func (a *adapter) EnsureStatusIsTracked() (controller.OperationResult, error) {
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
func (a *adapter) createInternalRequestPipelineRun() (*v1beta1.PipelineRun, error) {
	pipelineRun := tekton.NewInternalRequestPipelineRun(a.internalServicesConfig).
		WithInternalRequest(a.internalRequest).
		WithOwner(a.internalRequest).
		WithPipeline(a.internalRequestPipeline, a.internalServicesConfig).
		AsPipelineRun()

	err := a.internalClient.Create(a.ctx, pipelineRun)
	if err != nil {
		return nil, err
	}

	return pipelineRun, nil
}

// getDefaultInternalServicesConfig creates and returns a InternalServicesConfig resource in the given namespace with default values.
func (a *adapter) getDefaultInternalServicesConfig(namespace string) *v1alpha1.InternalServicesConfig {
	return &v1alpha1.InternalServicesConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      v1alpha1.InternalServicesConfigResourceName,
			Namespace: namespace,
		},
	}
}

// registerInternalRequestStatus sets the InternalRequest to Running.
func (a *adapter) registerInternalRequestStatus(pipelineRun *v1beta1.PipelineRun) error {
	if pipelineRun == nil {
		return nil
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())

	a.internalRequest.MarkRunning()

	return a.client.Status().Patch(a.ctx, a.internalRequest, patch)
}

// registerInternalRequestPipelineRunStatus keeps track of the PipelineRun status in the InternalRequest being processed.
func (a *adapter) registerInternalRequestPipelineRunStatus(pipelineRun *v1beta1.PipelineRun) error {
	if pipelineRun == nil || !pipelineRun.IsDone() {
		return nil
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())

	condition := pipelineRun.Status.GetCondition(apis.ConditionSucceeded)
	if condition.IsTrue() {
		a.internalRequest.Status.Results = tekton.GetResultsFromPipelineRun(pipelineRun)
		a.internalRequest.MarkSucceeded()
	} else {
		a.internalRequest.MarkFailed(condition.Message)
	}

	err := a.client.Status().Patch(a.ctx, a.internalRequest, patch)
	if err != nil {
		return err
	}

	a.logger.Info("Request execution finished", "Succeeded", a.internalRequest.HasSucceeded())

	return nil
}
