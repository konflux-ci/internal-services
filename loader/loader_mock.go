package loader

import (
	"context"

	"github.com/konflux-ci/internal-services/api/v1alpha1"
	toolkit "github.com/konflux-ci/operator-toolkit/loader"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InternalRequestContextKey            toolkit.ContextKey = iota
	InternalRequestPipelineContextKey    toolkit.ContextKey = iota
	InternalRequestPipelineRunContextKey toolkit.ContextKey = iota
	InternalServicesConfigContextKey     toolkit.ContextKey = iota
)

type mockLoader struct {
	loader ObjectLoader
}

func NewMockLoader() ObjectLoader {
	return &mockLoader{
		loader: NewLoader(),
	}
}

// GetInternalRequest returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalRequest(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalRequest, error) {
	if ctx.Value(InternalRequestContextKey) == nil {
		return l.loader.GetInternalRequest(ctx, cli, name, namespace)
	}
	return toolkit.GetMockedResourceAndErrorFromContext(ctx, InternalRequestContextKey, &v1alpha1.InternalRequest{})
}

// GetInternalRequestPipeline returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalRequestPipeline(ctx context.Context, cli client.Client, name, namespace string) (*v1.Pipeline, error) {
	if ctx.Value(InternalRequestPipelineContextKey) == nil {
		return l.loader.GetInternalRequestPipeline(ctx, cli, name, namespace)
	}
	return toolkit.GetMockedResourceAndErrorFromContext(ctx, InternalRequestPipelineContextKey, &v1.Pipeline{})
}

// GetInternalRequestPipelineRun returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalRequestPipelineRun(ctx context.Context, cli client.Client, internalRequest *v1alpha1.InternalRequest) (*v1.PipelineRun, error) {
	if ctx.Value(InternalRequestPipelineRunContextKey) == nil {
		return l.loader.GetInternalRequestPipelineRun(ctx, cli, internalRequest)
	}
	return toolkit.GetMockedResourceAndErrorFromContext(ctx, InternalRequestPipelineRunContextKey, &v1.PipelineRun{})
}

// GetInternalServicesConfig returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalServicesConfig(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalServicesConfig, error) {
	if ctx.Value(InternalServicesConfigContextKey) == nil {
		return l.loader.GetInternalServicesConfig(ctx, cli, name, namespace)
	}
	return toolkit.GetMockedResourceAndErrorFromContext(ctx, InternalServicesConfigContextKey, &v1alpha1.InternalServicesConfig{})
}
