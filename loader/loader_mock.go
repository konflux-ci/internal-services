package loader

import (
	"context"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	contextKey int
	MockData   struct {
		ContextKey contextKey
		Err        error
		Resource   any
	}
	mockLoader struct {
		loader ObjectLoader
	}
)

const (
	InternalRequestContextKey            contextKey = iota
	InternalRequestPipelineContextKey    contextKey = iota
	InternalRequestPipelineRunContextKey contextKey = iota
	InternalServicesConfigContextKey     contextKey = iota
)

func GetMockedContext(ctx context.Context, data []MockData) context.Context {
	for _, mockData := range data {
		ctx = context.WithValue(ctx, mockData.ContextKey, mockData)
	}

	return ctx
}

func NewMockLoader() ObjectLoader {
	return &mockLoader{
		loader: NewLoader(),
	}
}

// getMockedResourceAndErrorFromContext returns the mocked data found in the context passed as an argument. The data is
// to be found in the contextDataKey key. If not there, a panic will be raised.
func getMockedResourceAndErrorFromContext[T any](ctx context.Context, contextKey contextKey, _ T) (T, error) {
	var resource T
	var err error

	value := ctx.Value(contextKey)
	if value == nil {
		panic("Mocked data not found in the context")
	}

	data, _ := value.(MockData)

	if data.Resource != nil {
		resource = data.Resource.(T)
	}

	if data.Err != nil {
		err = data.Err
	}

	return resource, err
}

// GetInternalRequest returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalRequest(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalRequest, error) {
	if ctx.Value(InternalRequestContextKey) == nil {
		return l.loader.GetInternalRequest(ctx, cli, name, namespace)
	}
	return getMockedResourceAndErrorFromContext(ctx, InternalRequestContextKey, &v1alpha1.InternalRequest{})
}

// GetInternalRequestPipeline returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalRequestPipeline(ctx context.Context, cli client.Client, name, namespace string) (*v1beta1.Pipeline, error) {
	if ctx.Value(InternalRequestPipelineContextKey) == nil {
		return l.loader.GetInternalRequestPipeline(ctx, cli, name, namespace)
	}
	return getMockedResourceAndErrorFromContext(ctx, InternalRequestPipelineContextKey, &v1beta1.Pipeline{})
}

// GetInternalRequestPipelineRun returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalRequestPipelineRun(ctx context.Context, cli client.Client, internalRequest *v1alpha1.InternalRequest) (*v1beta1.PipelineRun, error) {
	if ctx.Value(InternalRequestPipelineRunContextKey) == nil {
		return l.loader.GetInternalRequestPipelineRun(ctx, cli, internalRequest)
	}
	return getMockedResourceAndErrorFromContext(ctx, InternalRequestPipelineRunContextKey, &v1beta1.PipelineRun{})
}

// GetInternalServicesConfig returns the resource and error passed as values of the context.
func (l *mockLoader) GetInternalServicesConfig(ctx context.Context, cli client.Client, name, namespace string) (*v1alpha1.InternalServicesConfig, error) {
	if ctx.Value(InternalServicesConfigContextKey) == nil {
		return l.loader.GetInternalServicesConfig(ctx, cli, name, namespace)
	}
	return getMockedResourceAndErrorFromContext(ctx, InternalServicesConfigContextKey, &v1alpha1.InternalServicesConfig{})
}
