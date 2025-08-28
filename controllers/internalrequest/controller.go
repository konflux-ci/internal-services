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

	"github.com/go-logr/logr"
	"github.com/konflux-ci/internal-services/api/v1alpha1"
	"github.com/konflux-ci/internal-services/loader"
	"github.com/konflux-ci/internal-services/tekton"
	"github.com/konflux-ci/operator-toolkit/controller"
	"github.com/konflux-ci/operator-toolkit/predicates"
	libhandler "github.com/operator-framework/operator-lib/handler"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Reconciler reconciles an InternalRequest object
type Reconciler struct {
	Client         client.Client
	Log            logr.Logger
	InternalClient client.Client
	Scheme         *runtime.Scheme
}

// +kubebuilder:rbac:groups=appstudio.redhat.com,resources=internalrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=appstudio.redhat.com,resources=internalrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=appstudio.redhat.com,resources=internalrequests/finalizers,verbs=update
// +kubebuilder:rbac:groups=appstudio.redhat.com,resources=internalservicesconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("InternalRequest", req.NamespacedName)

	internalRequest := &v1alpha1.InternalRequest{}
	err := r.Client.Get(ctx, req.NamespacedName, internalRequest)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	adapter := NewAdapter(ctx, r.Client, r.InternalClient, internalRequest, loader.NewLoader(), logger)

	return controller.ReconcileHandler([]controller.Operation{
		adapter.EnsureRequestINotCompleted,
		adapter.EnsureConfigIsLoaded, // This operation sets the config in the adapter to be used in other operations.
		adapter.EnsureRequestIsAllowed,
		adapter.EnsurePipelineRunIsCreated,
		adapter.EnsureStatusIsTracked,
		adapter.EnsurePipelineRunIsDeleted,
	})
}

// Register registers the controller with the passed manager and log. This controller monitors new InternalRequests and
// filters out status updates. It also watches for PipelineRuns created by this controller and owned by the
// InternalRequests so the owner gets reconciled on PipelineRun changes.
func (r *Reconciler) Register(mgr ctrl.Manager, log *logr.Logger, remoteCluster cluster.Cluster) error {
	r.Client = remoteCluster.GetClient()
	r.InternalClient = mgr.GetClient()
	r.Log = log.WithName("internalRequest")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1alpha1.InternalRequest{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}, predicates.IgnoreAllPredicate{}),
		).
		WatchesRawSource(
			source.TypedKind(remoteCluster.GetCache(), &v1alpha1.InternalRequest{},
				&handler.TypedEnqueueRequestForObject[*v1alpha1.InternalRequest]{},
				predicates.TypedGenerationChangedPredicate[*v1alpha1.InternalRequest]{},
				predicates.TypedNewObjectsPredicate[*v1alpha1.InternalRequest]{},
			),
		).
		Watches(&tektonv1.PipelineRun{}, &libhandler.EnqueueRequestForAnnotation[client.Object]{
			Type: schema.GroupKind{
				Kind:  "InternalRequest",
				Group: "appstudio.redhat.com",
			},
		}, builder.WithPredicates(tekton.InternalRequestPipelineRunSucceededPredicate())).
		Complete(r)
}
