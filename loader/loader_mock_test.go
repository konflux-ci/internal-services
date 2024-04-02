package loader

import (
	"github.com/konflux-ci/internal-services/api/v1alpha1"
	toolkit "github.com/konflux-ci/operator-toolkit/loader"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var _ = Describe("Loader Mock", Ordered, func() {
	var (
		loader ObjectLoader
	)

	BeforeAll(func() {
		loader = NewMockLoader()
	})

	Context("When calling GetInternalRequest", func() {
		It("returns the resource and error from the context", func() {
			internalRequest := &v1alpha1.InternalRequest{}
			mockContext := toolkit.GetMockedContext(ctx, []toolkit.MockData{
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
			mockContext := toolkit.GetMockedContext(ctx, []toolkit.MockData{
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
			mockContext := toolkit.GetMockedContext(ctx, []toolkit.MockData{
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
			mockContext := toolkit.GetMockedContext(ctx, []toolkit.MockData{
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
