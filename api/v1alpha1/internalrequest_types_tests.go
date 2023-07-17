package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/redhat-appstudio/operator-toolkit/conditions"
)

var _ = Describe("Release type", func() {

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

})
