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
	"strings"

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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

// EnsureFinalizerIsAdded is an operation that will ensure that the InternalRequest being processed contains a finalizer.
func (a *Adapter) EnsureFinalizerIsAdded() (controller.OperationResult, error) {
	if !controllerutil.ContainsFinalizer(a.internalRequest, tekton.InternalRequestFinalizer) {
		a.logger.Info("Adding Finalizer to the InternalRequest")
		patch := client.MergeFrom(a.internalRequest.DeepCopy())
		controllerutil.AddFinalizer(a.internalRequest, tekton.InternalRequestFinalizer)
		err := a.client.Patch(a.ctx, a.internalRequest, patch)

		return controller.RequeueOnErrorOrContinue(err)
	}

	return controller.ContinueProcessing()
}

// EnsureFinalizersAreCalled is an operation that will ensure that finalizers are called whenever the InternalRequest
// being processed is marked for deletion. Once finalizers get called, the finalizer will be removed and the
// InternalRequest will go back to the queue, so it gets deleted. If a finalizer function fails its execution or a
// finalizer fails to be removed, the InternalRequest will be requeued with the error attached.
func (a *Adapter) EnsureFinalizersAreCalled() (controller.OperationResult, error) {
	// Check if the InternalRequest is marked for deletion and continue processing other operations otherwise
	if a.internalRequest.GetDeletionTimestamp() == nil {
		return controller.ContinueProcessing()
	}

	if controllerutil.ContainsFinalizer(a.internalRequest, tekton.InternalRequestFinalizer) {
		if err := a.finalizeInternalRequest(); err != nil {
			return controller.RequeueWithError(err)
		}

		patch := client.MergeFrom(a.internalRequest.DeepCopy())
		controllerutil.RemoveFinalizer(a.internalRequest, tekton.InternalRequestFinalizer)
		err := a.client.Patch(a.ctx, a.internalRequest, patch)
		if err != nil {
			return controller.RequeueWithError(err)
		}
	}

	// Requeue the InternalRequest again so it gets deleted and other operations are not executed
	return controller.Requeue()
}

// finalizeInternalRequest cancels the PipelineRun associated with the InternalRequest if it is still running.
// It prefers the authoritative reference in status.pipelineRun for a deterministic Get; when that field is
// not yet set (e.g. the IR was deleted before EnsureStatusIsTracked ran) it falls back to a label-based list.
func (a *Adapter) finalizeInternalRequest() error {
	if a.internalRequest.Status.PipelineRun != "" {
		parts := strings.SplitN(a.internalRequest.Status.PipelineRun, "/", 2)
		if len(parts) == 2 {
			pipelineRun := &tektonv1.PipelineRun{}
			err := a.internalClient.Get(a.ctx, types.NamespacedName{
				Namespace: parts[0],
				Name:      parts[1],
			}, pipelineRun)
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				return err
			}
			return a.cancelPipelineRun(pipelineRun)
		}
	}

	pipelineRun, err := a.loader.GetInternalRequestPipelineRun(a.ctx, a.internalClient, a.internalRequest)
	if err != nil {
		return err
	}

	return a.cancelPipelineRun(pipelineRun)
}

// cancelPipelineRun patches the given PipelineRun to the Cancelled state if it is still running.
func (a *Adapter) cancelPipelineRun(pipelineRun *tektonv1.PipelineRun) error {
	if pipelineRun == nil || pipelineRun.IsDone() {
		return nil
	}

	a.logger.Info("Cancelling PipelineRun due to InternalRequest deletion",
		"PipelineRun.Name", pipelineRun.Name, "PipelineRun.Namespace", pipelineRun.Namespace)
	pipelineRunPatch := client.MergeFrom(pipelineRun.DeepCopy())
	pipelineRun.Spec.Status = tektonv1.PipelineRunSpecStatusCancelled

	return a.internalClient.Patch(a.ctx, pipelineRun, pipelineRunPatch)
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
			return a.ensureGitResolverURLIsAllowed()
		}
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())
	a.internalRequest.MarkRejected(
		fmt.Sprintf("the internal request namespace (%s) is not in the allow list", a.internalRequest.Namespace),
	)
	return controller.RequeueOnErrorOrStop(a.client.Status().Patch(a.ctx, a.internalRequest, patch))
}

// ensureGitResolverURLIsAllowed checks whether the git resolver URL in the InternalRequest
// matches an entry in the AllowedGitResolverURLs list. If the resolver is not "git" or the
// list is empty, all requests are allowed.
func (a *Adapter) ensureGitResolverURLIsAllowed() (controller.OperationResult, error) {
	if a.internalRequest.Spec.Pipeline.PipelineRef.Resolver != "git" ||
		len(a.internalServicesConfig.Spec.AllowedGitResolverURLs) == 0 {
		return controller.ContinueProcessing()
	}

	var url string
	for _, param := range a.internalRequest.Spec.Pipeline.PipelineRef.Params {
		if param.Name == "url" {
			url = param.Value
			break
		}
	}

	for _, allowedURL := range a.internalServicesConfig.Spec.AllowedGitResolverURLs {
		if url == allowedURL {
			return controller.ContinueProcessing()
		}
	}

	patch := client.MergeFrom(a.internalRequest.DeepCopy())
	a.internalRequest.MarkRejected(
		fmt.Sprintf("the pipeline git resolver URL (%s) is not in the allowed list: %v", url, a.internalServicesConfig.Spec.AllowedGitResolverURLs),
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

// failedTaskRunMessage lists the child TaskRuns of pipelineRun and returns a
// combined message from any that have a False Succeeded condition. This gives a
// more specific failure reason (e.g. image pull error, step exit code) than the
// generic PipelineRun-level summary Tekton sets. Returns an empty string when no
// failed TaskRun conditions are found or the list call fails, so the caller can
// fall back to the PipelineRun condition message.
func (a *Adapter) failedTaskRunMessage(pipelineRun *tektonv1.PipelineRun) string {
	taskRuns, err := a.loader.GetInternalRequestPipelineRunTaskRuns(a.ctx, a.internalClient, pipelineRun)
	if err != nil {
		a.logger.Error(err, "Failed to list child TaskRuns", "PipelineRun", pipelineRun.Name)
		return ""
	}

	var messages []string
	for i := range taskRuns.Items {
		c := taskRuns.Items[i].Status.GetCondition(apis.ConditionSucceeded)
		if c != nil && c.IsFalse() && c.Message != "" {
			messages = append(messages,
				fmt.Sprintf("%s (%s): %s", taskRuns.Items[i].Name, c.Reason, c.Message))
		}
	}
	return strings.Join(messages, "; ")
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
			// The PipelineRun-level condition message is a generic Tekton summary
			// (e.g. "Tasks Completed: 1 (Failed: 1, Cancelled 0), Skipped: 0").
			// Try to surface the more actionable child TaskRun condition messages
			// instead, so callers on the tenant cluster can see the real cause
			// (e.g. image pull failure, step exit code) without needing access to
			// the internal-services namespace.
			message := a.failedTaskRunMessage(pipelineRun)
			if message == "" {
				message = condition.Message
			}
			a.internalRequest.MarkFailed(message)
		}
		a.logger.Info("Request execution finished", "Succeeded", a.internalRequest.HasSucceeded())
	}

	err := a.client.Status().Patch(a.ctx, a.internalRequest, patch)
	if err != nil {
		return err
	}

	return nil
}
