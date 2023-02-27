package loader

import (
	"errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Loader Mock", Ordered, func() {
	var (
		loader ObjectLoader
	)

	BeforeAll(func() {
		loader = NewMockLoader()
	})

	Context("When calling getMockedResourceAndErrorFromContext", func() {
		contextErr := errors.New("error")
		contextResource := &v1alpha1.InternalRequest{
			ObjectMeta: v12.ObjectMeta{
				Name:      "pod",
				Namespace: "default",
			},
		}

		It("returns the resource from the context", func() {
			mockContext := GetMockedContext(ctx, []MockData{
				{
					ContextKey: InternalRequestContextKey,
					Resource:   contextResource,
				},
			})
			resource, err := getMockedResourceAndErrorFromContext(mockContext, InternalRequestContextKey, contextResource)
			Expect(err).To(BeNil())
			Expect(resource).To(Equal(contextResource))
		})

		It("returns the error from the context", func() {
			mockContext := GetMockedContext(ctx, []MockData{
				{
					ContextKey: InternalRequestContextKey,
					Err:        contextErr,
				},
			})
			resource, err := getMockedResourceAndErrorFromContext(mockContext, InternalRequestContextKey, contextResource)
			Expect(err).To(Equal(contextErr))
			Expect(resource).To(BeNil())
		})

		It("returns the resource and the error from the context", func() {
			mockContext := GetMockedContext(ctx, []MockData{
				{
					ContextKey: InternalRequestContextKey,
					Resource:   contextResource,
					Err:        contextErr,
				},
			})
			resource, err := getMockedResourceAndErrorFromContext(mockContext, InternalRequestContextKey, contextResource)
			Expect(err).To(Equal(contextErr))
			Expect(resource).To(Equal(contextResource))
		})

		It("should panic when the mocked data is not present", func() {
			Expect(func() {
				_, _ = getMockedResourceAndErrorFromContext(ctx, InternalRequestContextKey, contextResource)
			}).To(Panic())
		})
	})

	Context("When calling GetInternalRequest", func() {
		It("returns the resource and error from the context", func() {
			internalRequest := &v1alpha1.InternalRequest{}
			mockContext := GetMockedContext(ctx, []MockData{
				{
					ContextKey: InternalRequestContextKey,
					Resource:   internalRequest,
				},
			})
			resource, err := loader.GetInternalRequest(mockContext, nil, "", "")
			Expect(resource).To(Equal(internalRequest))
			Expect(err).To(BeNil())
		})
	})

	Context("When calling GetInternalRequestPipeline", func() {
		It("returns the resource and error from the context", func() {
			pipeline := &tektonv1beta1.Pipeline{}
			mockContext := GetMockedContext(ctx, []MockData{
				{
					ContextKey: InternalRequestPipelineContextKey,
					Resource:   pipeline,
				},
			})
			resource, err := loader.GetInternalRequestPipeline(mockContext, nil, "", "")
			Expect(resource).To(Equal(pipeline))
			Expect(err).To(BeNil())
		})
	})

	Context("When calling GetInternalRequestPipelineRun", func() {
		It("returns the resource and error from the context", func() {
			pipelineRun := &tektonv1beta1.PipelineRun{}
			mockContext := GetMockedContext(ctx, []MockData{
				{
					ContextKey: InternalRequestPipelineRunContextKey,
					Resource:   pipelineRun,
				},
			})
			resource, err := loader.GetInternalRequestPipelineRun(mockContext, nil, &v1alpha1.InternalRequest{})
			Expect(resource).To(Equal(pipelineRun))
			Expect(err).To(BeNil())
		})
	})

	Context("When calling GetInternalServicesConfig", func() {
		It("returns the resource and error from the context", func() {
			internalServicesConfig := &v1alpha1.InternalServicesConfig{}
			mockContext := GetMockedContext(ctx, []MockData{
				{
					ContextKey: InternalServicesConfigContextKey,
					Resource:   internalServicesConfig,
				},
			})
			resource, err := loader.GetInternalServicesConfig(mockContext, nil, "", "")
			Expect(resource).To(Equal(internalServicesConfig))
			Expect(err).To(BeNil())
		})
	})

})
