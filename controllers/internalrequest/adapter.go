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
	libhandler "github.com/operator-framework/operator-lib/handler"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	"github.com/redhat-appstudio/internal-services/loader"
	"github.com/redhat-appstudio/internal-services/tekton"
	"github.com/redhat-appstudio/operator-goodies/reconciler"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// Adapter holds the objects needed to reconcile an InternalRequest.
type Adapter struct {
	client                 client.Client
	internalServicesConfig *v1alpha1.InternalServicesConfig
	ctx                    context.Context
	internalClient         client.Client
	internalRequest        *v1alpha1.InternalRequest
	loader                 loader.ObjectLoader
	logger                 logr.Logger
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
func (a *Adapter) EnsureConfigIsLoaded() (reconciler.OperationResult, error) {
	namespace := os.Getenv("SERVICE_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	var err error
	a.internalServicesConfig, err = a.loader.GetInternalServicesConfig(a.ctx, a.internalClient, v1alpha1.InternalServicesConfigResourceName, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return reconciler.RequeueWithError(err)
	}

	if err != nil {
		a.internalServicesConfig = a.getDefaultInternalServicesConfig(namespace)
	}

	return reconciler.ContinueProcessing()
}

// EnsurePipelineRunIsCreated is an operation that will ensure that the InternalRequest is handled by creating a new
// PipelineRun for the Pipeline referenced in the Request field.
func (a *Adapter) EnsurePipelineRunIsCreated() (reconciler.OperationResult, error) {
	pipelineRun, err := a.loader.GetInternalRequestPipelineRun(a.ctx, a.internalClient, a.internalRequest)
	if err != nil && !errors.IsNotFound(err) {
		return reconciler.RequeueWithError(err)
	}

	if pipelineRun == nil || !a.internalRequest.HasStarted() {
		if pipelineRun == nil {
			pipelineRun = tekton.NewPipelineRun(a.internalRequest, a.internalServicesConfig)

			err = libhandler.SetOwnerAnnotations(a.internalRequest, pipelineRun)
			if err != nil {
				return reconciler.RequeueWithError(err)
			}

			err = a.internalClient.Create(a.ctx, pipelineRun)
			if err != nil {
				return reconciler.RequeueWithError(err)
			}

			a.logger.Info("Created PipelineRun to handle request",
				"PipelineRun.Name", pipelineRun.Name, "PipelineRun.Namespace", pipelineRun.Namespace)
		}

		return reconciler.RequeueOnErrorOrContinue(a.registerInternalRequestStatus(pipelineRun))
	}

	return reconciler.ContinueProcessing()
}

// EnsureRequestIsAllowed is an operation that will ensure that the request is coming from a namespace allowed
// to execute InternalRequests. If the InternalServicesConfig spec.allowList is empty, any request will be allowed regardless of the
// remote namespace.
func (a *Adapter) EnsureRequestIsAllowed() (reconciler.OperationResult, error) {
	for _, namespace := range a.internalServicesConfig.Spec.AllowList {
		if namespace == a.internalRequest.Namespace {
			return reconciler.ContinueProcessing()
		}
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())
	a.internalRequest.MarkInvalid(v1alpha1.InternalRequestRejected,
		fmt.Sprintf("the internal request namespace (%s) is not in the allow list", a.internalRequest.Namespace))
	return reconciler.RequeueOnErrorOrStop(a.client.Status().Patch(a.ctx, a.internalRequest, patch))
}

// EnsureStatusIsTracked is an operation that will ensure that the release PipelineRun status is tracked
// in the InternalRequest being processed.
func (a *Adapter) EnsureStatusIsTracked() (reconciler.OperationResult, error) {
	pipelineRun, err := a.loader.GetInternalRequestPipelineRun(a.ctx, a.internalClient, a.internalRequest)
	if err != nil && !errors.IsNotFound(err) {
		return reconciler.RequeueWithError(err)
	}

	if pipelineRun != nil {
		return reconciler.RequeueOnErrorOrContinue(a.registerInternalRequestPipelineRunStatus(pipelineRun))
	}

	return reconciler.ContinueProcessing()
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
func (a *Adapter) registerInternalRequestStatus(pipelineRun *v1beta1.PipelineRun) error {
	if pipelineRun == nil {
		return nil
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())

	a.internalRequest.MarkRunning()

	return a.client.Status().Patch(a.ctx, a.internalRequest, patch)
}

// registerInternalRequestPipelineRunStatus keeps track of the PipelineRun status in the InternalRequest being processed.
func (a *Adapter) registerInternalRequestPipelineRunStatus(pipelineRun *v1beta1.PipelineRun) error {
	if pipelineRun == nil || !pipelineRun.IsDone() {
		return nil
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())

	condition := pipelineRun.Status.GetCondition(apis.ConditionSucceeded)
	if condition.IsTrue() {
		a.internalRequest.Status.Results = tekton.GetResultsFromPipelineRun(pipelineRun)
		a.internalRequest.MarkSucceeded()
	} else if strings.Contains(condition.Message, "not found") {
		var endpoint string
		if pipelineRun.Spec.PipelineRef != nil {
			endpoint = pipelineRun.Spec.PipelineRef.Name
		}
		a.internalRequest.MarkFailed(fmt.Sprintf("No endpoint to handle '%s' requests", endpoint))
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
