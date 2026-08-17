package loader

import (
	"github.com/konflux-ci/internal-services/api/v1alpha1"
	"github.com/konflux-ci/internal-services/tekton"
	"github.com/konflux-ci/internal-services/tekton/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Loader", Ordered, func() {
	var (
		loader          ObjectLoader
		createResources func()

		internalServicesConfig *v1alpha1.InternalServicesConfig
		internalRequest        *v1alpha1.InternalRequest
		pipeline               *tektonv1.Pipeline
		pipelineRun            *tektonv1.PipelineRun
		taskRun                *tektonv1.TaskRun
	)

	BeforeAll(func() {
		createResources()

		loader = NewLoader()
	})

	Context("When calling GetInternalRequest", func() {
		It("returns the requested InternalRequest", func() {
			returnedObject, err := loader.GetInternalRequest(ctx, k8sClient, internalRequest.Name, internalRequest.Namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedObject).NotTo(Equal(&v1alpha1.InternalRequest{}))
			Expect(returnedObject.Name).To(Equal(internalRequest.Name))
		})
	})

	Context("When calling GetInternalRequestPipeline", func() {
		It("returns the requested Pipeline", func() {
			returnedObject, err := loader.GetInternalRequestPipeline(ctx, k8sClient, pipeline.Name, pipeline.Namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedObject).NotTo(Equal(&tektonv1.Pipeline{}))
			Expect(returnedObject.Name).To(Equal(pipeline.Name))
		})
	})

	Context("When calling GetInternalRequestPipelineRun", func() {
		It("returns a PipelineRun if the labels match with the internal request data", func() {
			returnedObject, err := loader.GetInternalRequestPipelineRun(ctx, k8sClient, internalRequest)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedObject).NotTo(Equal(&tektonv1.PipelineRun{}))
			Expect(returnedObject.Name).To(Equal(pipelineRun.Name))
		})

		It("fails to return a PipelineRun if the labels don't match with the InternalRequest data", func() {
			modifiedRequest := internalRequest.DeepCopy()
			modifiedRequest.Name = "non-existing-request"

			returnedObject, err := loader.GetInternalRequestPipelineRun(ctx, k8sClient, modifiedRequest)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedObject).To(BeNil())
		})
	})

	Context("When calling GetInternalRequestPipelineRunTaskRuns", func() {
		It("returns TaskRuns whose label matches the PipelineRun name", func() {
			returnedObject, err := loader.GetInternalRequestPipelineRunTaskRuns(ctx, k8sClient, pipelineRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedObject.Items).To(HaveLen(1))
			Expect(returnedObject.Items[0].Name).To(Equal(taskRun.Name))
		})

		It("returns an empty list when no TaskRuns match the PipelineRun name", func() {
			modifiedPipelineRun := pipelineRun.DeepCopy()
			modifiedPipelineRun.Name = "non-existing-pipeline-run"

			returnedObject, err := loader.GetInternalRequestPipelineRunTaskRuns(ctx, k8sClient, modifiedPipelineRun)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedObject.Items).To(BeEmpty())
		})
	})

	Context("When calling GetInternalServicesConfig", func() {
		It("returns the requested InternalServicesConfig", func() {
			returnedObject, err := loader.GetInternalServicesConfig(ctx, k8sClient, internalServicesConfig.Name, internalServicesConfig.Namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedObject).NotTo(Equal(&v1alpha1.InternalServicesConfig{}))
			Expect(returnedObject.Name).To(Equal(internalServicesConfig.Name))
		})
	})

	createResources = func() {
		parameterizedPipeline := utils.ParameterizedPipeline{}
		parameterizedPipeline.PipelineRef = utils.PipelineRef{
			Resolver: "git",
			Params: []utils.Param{
				{Name: "url", Value: "my-url"},
				{Name: "revision", Value: "my-revision"},
				{Name: "pathInRepo", Value: "my-path"},
			},
		}
		internalRequest = &v1alpha1.InternalRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "request",
				Namespace: "default",
			},
			Spec: v1alpha1.InternalRequestSpec{
				Pipeline: &parameterizedPipeline,
			},
		}
		Expect(k8sClient.Create(ctx, internalRequest)).To(Succeed())

		internalServicesConfig = &v1alpha1.InternalServicesConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1alpha1.InternalServicesConfigResourceName,
				Namespace: "default",
			},
		}
		Expect(k8sClient.Create(ctx, internalServicesConfig)).To(Succeed())

		pipeline = &tektonv1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "default",
			},
		}
		Expect(k8sClient.Create(ctx, pipeline)).To(Succeed())

		pipelineRun = &tektonv1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					tekton.InternalRequestNameLabel:      internalRequest.Name,
					tekton.InternalRequestNamespaceLabel: internalRequest.Namespace,
				},
				Name:      "pipeline-run",
				Namespace: "default",
			},
		}
		Expect(k8sClient.Create(ctx, pipelineRun)).To(Succeed())

		taskRun = &tektonv1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"tekton.dev/pipelineRun": pipelineRun.Name,
				},
				Name:      "pipeline-run-task",
				Namespace: "default",
			},
		}
		Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())
	}

})
