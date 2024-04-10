package v1alpha1

import "github.com/konflux-ci/operator-toolkit/conditions"

const (
	// SucceededConditionType is the type used when setting a status condition
	SucceededConditionType conditions.ConditionType = "Succeeded"
)

const (
	// FailedReason is the reason set when the PipelineRun failed
	FailedReason conditions.ConditionReason = "Failed"

	// RejectedReason is the reason set when the InternalRequest is rejected
	RejectedReason conditions.ConditionReason = "Rejected"

	// RunningReason is the reason set when the PipelineRun starts running
	RunningReason conditions.ConditionReason = "Running"

	// SucceededReason is the reason set when the PipelineRun has succeeded
	SucceededReason conditions.ConditionReason = "Succeeded"
)
