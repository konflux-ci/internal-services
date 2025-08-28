package loader

import (
	"context"

	"github.com/konflux-ci/internal-services/api/v1alpha1"
	"github.com/konflux-ci/internal-services/tekton"
	toolkit "github.com/konflux-ci/operator-toolkit/loader"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectLoader interface {
	GetInternalRequest(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalRequest, error)
	GetInternalRequestPipeline(ctx context.Context, cli client.Client, name, namespace string) (*v1.Pipeline, error)
	GetInternalRequestPipelineRun(ctx context.Context, cli client.Client, internalRequest *v1alpha1.InternalRequest) (*v1.PipelineRun, error)
	GetInternalServicesConfig(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalServicesConfig, error)
}

type loader struct{}

func NewLoader() ObjectLoader {
	return &loader{}
}

// GetInternalRequest returns the InternalRequest with the given name and namespace. If the InternalRequest is not
// found or the Get operation fails, an error will be returned.
func (l *loader) GetInternalRequest(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalRequest, error) {
	internalRequest := &v1alpha1.InternalRequest{}
	return internalRequest, toolkit.GetObject(name, namespace, cli, ctx, internalRequest)
}

// GetInternalRequestPipeline returns the Pipeline with the given name and namespace. If the Pipeline is not
// found or the Get operation fails, an error will be returned.
func (l *loader) GetInternalRequestPipeline(ctx context.Context, cli client.Client, name, namespace string) (*v1.Pipeline, error) {
	pipeline := &v1.Pipeline{}
	return pipeline, toolkit.GetObject(name, namespace, cli, ctx, pipeline)
}

// GetInternalRequestPipelineRun returns the PipelineRun referenced by the given InternalRequest or nil if it's not
// found. In the case the List operation fails, an error will be returned.
func (l *loader) GetInternalRequestPipelineRun(ctx context.Context, cli client.Client, internalRequest *v1alpha1.InternalRequest) (*v1.PipelineRun, error) {
	pipelineRuns := &v1.PipelineRunList{}
	err := cli.List(ctx, pipelineRuns,
		client.Limit(1),
		client.MatchingLabels{
			tekton.InternalRequestNameLabel:      internalRequest.Name,
			tekton.InternalRequestNamespaceLabel: internalRequest.Namespace,
		})

	if err == nil && len(pipelineRuns.Items) > 0 {
		return &pipelineRuns.Items[0], nil
	}

	return nil, err
}

// GetInternalServicesConfig returns the InternalServicesConfig with the given name and namespace. If the
// InternalServicesConfig is not found or the Get operation fails, an error will be returned.
func (l *loader) GetInternalServicesConfig(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalServicesConfig, error) {
	internalServicesConfig := &v1alpha1.InternalServicesConfig{}
	return internalServicesConfig, toolkit.GetObject(name, namespace, cli, ctx, internalServicesConfig)
}
