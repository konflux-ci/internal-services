package v1alpha1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"k8s.io/apimachinery/pkg/api/meta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konflux-ci/internal-services/metrics"
	tektonutils "github.com/konflux-ci/internal-services/tekton/utils"
	"github.com/konflux-ci/operator-toolkit/conditions"
)

// deletionAttempts reads the 'internal_request_attempt_total' counter for a deleted InternalRequest in the
// given namespace. RegisterDeletion only emits a metric, so the counter is the sole observable effect.
func deletionAttempts(request, namespace string) float64 {
	return testutil.ToFloat64(metrics.InternalRequestAttemptTotal.WithLabelValues(
		request, namespace, DeletedReason.String(), "false"))
}

var _ = Describe("Internal Request type", func() {

	When("HasCompleted is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should return false if the condition is missing", func() {
			Expect(internalRequest.HasCompleted()).To(BeFalse())
		})

		It("should return true if the condition status is True", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionTrue, SucceededReason)
			Expect(internalRequest.HasCompleted()).To(BeTrue())
		})

		It("should return false if the condition status is Unknown", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionUnknown, SucceededReason)
			Expect(internalRequest.HasCompleted()).To(BeFalse())
		})

		It("should return false if the condition status is False and the reason is Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionFalse, RunningReason)
			Expect(internalRequest.HasCompleted()).To(BeFalse())
		})

		It("should return true if the condition status is False and the reason is not Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionFalse, SucceededReason)
			Expect(internalRequest.HasCompleted()).To(BeTrue())
		})
	})

	When("HasFailed is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should return false if the condition is missing", func() {
			Expect(internalRequest.HasFailed()).To(BeFalse())
		})

		It("should return false if the condition status is True", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionTrue, SucceededReason)
			Expect(internalRequest.HasFailed()).To(BeFalse())
		})

		It("should return false if the condition status is Unknown", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionUnknown, SucceededReason)
			Expect(internalRequest.HasFailed()).To(BeFalse())
		})

		It("should return false if the condition status is False and the reason is Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionFalse, RunningReason)
			Expect(internalRequest.HasFailed()).To(BeFalse())
		})

		It("should return true if the condition status is False and the reason is not Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionFalse, SucceededReason)
			Expect(internalRequest.HasFailed()).To(BeTrue())
		})
	})

	When("HasSucceeded is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should return false if the condition is missing", func() {
			Expect(internalRequest.HasSucceeded()).To(BeFalse())
		})

		It("should return false if the condition status is False", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionFalse, SucceededReason)
			Expect(internalRequest.HasSucceeded()).To(BeFalse())
		})

		It("should return true if the condition status is True", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionTrue, SucceededReason)
			Expect(internalRequest.HasSucceeded()).To(BeTrue())
		})

		It("should return false if the condition status is Unknown", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionUnknown, SucceededReason)
			Expect(internalRequest.HasSucceeded()).To(BeFalse())
		})
	})

	When("IsRunning is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should return false if the condition is missing", func() {
			Expect(internalRequest.IsRunning()).To(BeFalse())
		})

		It("should return false if the condition status is True", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionTrue, SucceededReason)
			Expect(internalRequest.IsRunning()).To(BeFalse())
		})

		It("should return false if the condition status is True and the reason is Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionTrue, RunningReason)
			Expect(internalRequest.IsRunning()).To(BeFalse())
		})

		It("should return true if the condition status is False and the reason is Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionFalse, RunningReason)
			Expect(internalRequest.IsRunning()).To(BeTrue())
		})

		It("should return true if the condition status is Unknown and the reason is Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionUnknown, RunningReason)
			Expect(internalRequest.IsRunning()).To(BeTrue())
		})

		It("should return false if the condition status is False and the reason is not Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionFalse, SucceededReason)
			Expect(internalRequest.IsRunning()).To(BeFalse())
		})

		It("should return false if the condition status is Unknown and the reason is not Running", func() {
			conditions.SetCondition(&internalRequest.Status.Conditions, SucceededConditionType, metav1.ConditionUnknown, SucceededReason)
			Expect(internalRequest.IsRunning()).To(BeFalse())
		})
	})

	When("MarkFailed method is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should do nothing if it finished", func() {
			internalRequest.MarkRunning()
			internalRequest.MarkSucceeded()
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeFalse())
			internalRequest.Status.CompletionTime = &metav1.Time{}
			internalRequest.MarkFailed("")
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeTrue())
		})

		It("should register the completion time", func() {
			internalRequest.MarkRunning()
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeTrue())
			internalRequest.MarkFailed("")
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeFalse())
		})

		It("should register the condition", func() {
			Expect(internalRequest.Status.Conditions).To(HaveLen(0))
			internalRequest.MarkRunning()
			internalRequest.MarkFailed("foo")

			condition := meta.FindStatusCondition(internalRequest.Status.Conditions, SucceededConditionType.String())
			Expect(condition).NotTo(BeNil())
			Expect(*condition).To(MatchFields(IgnoreExtras, Fields{
				"Message": Equal("foo"),
				"Reason":  Equal(FailedReason.String()),
				"Status":  Equal(metav1.ConditionFalse),
			}))
		})
	})

	When("MarkRejected method is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should do nothing if it finished", func() {
			internalRequest.MarkRunning()
			internalRequest.MarkSucceeded()
			Expect(internalRequest.HasFailed()).To(BeFalse())
			internalRequest.MarkRejected("")
			Expect(internalRequest.HasFailed()).To(BeFalse())
		})

		It("should register the condition", func() {
			Expect(internalRequest.Status.Conditions).To(HaveLen(0))
			internalRequest.MarkRunning()
			internalRequest.MarkRejected("foo")

			condition := meta.FindStatusCondition(internalRequest.Status.Conditions, SucceededConditionType.String())
			Expect(condition).NotTo(BeNil())
			Expect(*condition).To(MatchFields(IgnoreExtras, Fields{
				"Message": Equal("foo"),
				"Reason":  Equal(RejectedReason.String()),
				"Status":  Equal(metav1.ConditionFalse),
			}))
		})
	})

	When("MarkRunning method is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should do nothing if it finished", func() {
			internalRequest.MarkRunning()
			internalRequest.MarkSucceeded()
			Expect(internalRequest.Status.StartTime.IsZero()).To(BeFalse())
			internalRequest.Status.StartTime = &metav1.Time{}
			internalRequest.MarkRunning()
			Expect(internalRequest.Status.StartTime.IsZero()).To(BeTrue())
		})

		It("should not register the start time it it's running already", func() {
			internalRequest.MarkRunning()
			Expect(internalRequest.Status.StartTime.IsZero()).To(BeFalse())
			internalRequest.Status.StartTime = &metav1.Time{}
			internalRequest.MarkRunning()
			Expect(internalRequest.Status.StartTime.IsZero()).To(BeTrue())
		})

		It("should register the start time", func() {
			Expect(internalRequest.Status.StartTime.IsZero()).To(BeTrue())
			internalRequest.MarkRunning()
			Expect(internalRequest.Status.StartTime.IsZero()).To(BeFalse())
		})

		It("should register the condition", func() {
			Expect(internalRequest.Status.Conditions).To(HaveLen(0))
			internalRequest.MarkRunning()

			condition := meta.FindStatusCondition(internalRequest.Status.Conditions, SucceededConditionType.String())
			Expect(condition).NotTo(BeNil())
			Expect(*condition).To(MatchFields(IgnoreExtras, Fields{
				"Reason": Equal(RunningReason.String()),
				"Status": Equal(metav1.ConditionFalse),
			}))
		})
	})

	When("MarkSucceeded method is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should do nothing if it finished", func() {
			internalRequest.MarkRunning()
			internalRequest.MarkSucceeded()
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeFalse())
			internalRequest.Status.CompletionTime = &metav1.Time{}
			internalRequest.MarkSucceeded()
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeTrue())
		})

		It("should register the completion time", func() {
			internalRequest.MarkRunning()
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeTrue())
			internalRequest.MarkSucceeded()
			Expect(internalRequest.Status.CompletionTime.IsZero()).To(BeFalse())
		})

		It("should register the condition", func() {
			Expect(internalRequest.Status.Conditions).To(HaveLen(0))
			internalRequest.MarkRunning()
			internalRequest.MarkSucceeded()

			condition := meta.FindStatusCondition(internalRequest.Status.Conditions, SucceededConditionType.String())
			Expect(condition).NotTo(BeNil())
			Expect(*condition).To(MatchFields(IgnoreExtras, Fields{
				"Reason": Equal(SucceededReason.String()),
				"Status": Equal(metav1.ConditionTrue),
			}))
		})
	})

	When("RegisterDeletion method is called", func() {
		var internalRequest *InternalRequest

		BeforeEach(func() {
			internalRequest = &InternalRequest{}
		})

		It("should not account for a request that already completed", func() {
			internalRequest.Namespace = "deletion-completed"
			internalRequest.MarkRunning()
			internalRequest.MarkSucceeded()

			before := deletionAttempts("", internalRequest.Namespace)
			internalRequest.RegisterDeletion()
			Consistently(func() float64 {
				return deletionAttempts("", internalRequest.Namespace)
			}).Should(Equal(before))
		})

		It("should account for a request deleted while it was running", func() {
			internalRequest.Namespace = "deletion-running"
			internalRequest.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			internalRequest.MarkRunning()

			before := deletionAttempts("", internalRequest.Namespace)
			internalRequest.RegisterDeletion()
			Eventually(func() float64 {
				return deletionAttempts("", internalRequest.Namespace)
			}).Should(Equal(before + 1))
		})

		It("should account for a request deleted before it started, with no deletion timestamp set", func() {
			internalRequest.Namespace = "deletion-not-started"
			Expect(internalRequest.DeletionTimestamp).To(BeNil())

			before := deletionAttempts("", internalRequest.Namespace)
			internalRequest.RegisterDeletion()
			Eventually(func() float64 {
				return deletionAttempts("", internalRequest.Namespace)
			}).Should(Equal(before + 1))
		})

		It("should label the metric with the pipeline name when the request carries a pipeline", func() {
			internalRequest.Namespace = "deletion-with-pipeline"
			internalRequest.Spec.Pipeline = &tektonutils.ParameterizedPipeline{}
			internalRequest.Spec.Pipeline.Params = []tektonutils.Param{
				{Name: "pathInRepo", Value: "pipelines/internal/my-pipeline/my-pipeline.yaml"},
			}

			before := deletionAttempts("my-pipeline", internalRequest.Namespace)
			internalRequest.RegisterDeletion()
			Eventually(func() float64 {
				return deletionAttempts("my-pipeline", internalRequest.Namespace)
			}).Should(Equal(before + 1))
		})
	})

})
