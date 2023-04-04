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

package v1alpha1

import (
	"time"

	"github.com/redhat-appstudio/internal-services/metrics"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HandlingReason represents a reason for the InternalRequest "InternalRequestSucceeded" condition.
type HandlingReason string

const (
	// InternalRequestSucceededConditionType is the type used when setting a status condition
	InternalRequestSucceededConditionType string = "InternalRequestSucceeded"

	// InternalRequestFailed is the reason set when the PipelineRun failed
	InternalRequestFailed HandlingReason = "Failed"

	// InternalRequestRejected is the reason set when the InternalRequest is rejected
	InternalRequestRejected HandlingReason = "Rejected"

	// InternalRequestRunning is the reason set when the PipelineRun starts running
	InternalRequestRunning HandlingReason = "Running"

	// InternalRequestSucceeded is the reason set when the PipelineRun has succeeded
	InternalRequestSucceeded HandlingReason = "Succeeded"
)

func (rr HandlingReason) String() string {
	return string(rr)
}

// InternalRequestSpec defines the desired state of InternalRequest.
type InternalRequestSpec struct {
	// Request is the name of the internal internalrequest which will be translated into a Tekton pipeline
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +required
	Request string `json:"request"`

	// Params is the list of optional parameters to pass to the Tekton pipeline
	// kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Params map[string]string `json:"params,omitempty"`
}

// InternalRequestStatus defines the observed state of InternalRequest.
type InternalRequestStatus struct {
	// StartTime is the time when the InternalRequest PipelineRun was created and set to run
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the InternalRequest PipelineRun completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Conditions represent the latest available observations for the internalrequest
	// +optional
	Conditions []metav1.Condition `json:"conditions"`

	// Results is the list of optional results as seen in the Tekton PipelineRun
	// kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Results map[string]string `json:"results,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Succeeded",type=string,JSONPath=`.status.conditions[?(@.type=="InternalRequestSucceeded")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="InternalRequestSucceeded")].reason`

// InternalRequest is the Schema for the internalrequests API.
type InternalRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InternalRequestSpec   `json:"spec,omitempty"`
	Status InternalRequestStatus `json:"status,omitempty"`
}

// HasCompleted checks whether the InternalRequest has been completed.
func (ir *InternalRequest) HasCompleted() bool {
	condition := meta.FindStatusCondition(ir.Status.Conditions, InternalRequestSucceededConditionType)
	return condition != nil && condition.Status != metav1.ConditionUnknown && ir.Status.CompletionTime != nil
}

// HasFailed checks whether the InternalRequest has failed.
func (ir *InternalRequest) HasFailed() bool {
	return meta.IsStatusConditionFalse(ir.Status.Conditions, InternalRequestSucceededConditionType)
}

// HasStarted checks whether the InternalRequest has started.
func (ir *InternalRequest) HasStarted() bool {
	condition := meta.FindStatusCondition(ir.Status.Conditions, InternalRequestSucceededConditionType)
	return condition != nil && ir.Status.StartTime != nil
}

// HasSucceeded checks whether the InternalRequest has succeeded.
func (ir *InternalRequest) HasSucceeded() bool {
	return meta.IsStatusConditionTrue(ir.Status.Conditions, InternalRequestSucceededConditionType)
}

// MarkFailed registers the completion time and changes the Succeeded condition to False with the provided message.
func (ir *InternalRequest) MarkFailed(message string) {
	if !ir.HasStarted() || ir.HasCompleted() {
		return
	}

	ir.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	ir.setStatusConditionWithMessage(InternalRequestSucceededConditionType, metav1.ConditionFalse, InternalRequestFailed, message)

	go metrics.RegisterCompletedInternalRequest(ir.Spec.Request, ir.Namespace, InternalRequestFailed.String(),
		ir.Status.StartTime, ir.Status.CompletionTime, false)
}

// MarkInvalid changes the Succeeded condition to False with the provided reason and message.
func (ir *InternalRequest) MarkInvalid(reason HandlingReason, message string) {
	if ir.HasCompleted() {
		return
	}

	ir.setStatusConditionWithMessage(InternalRequestSucceededConditionType, metav1.ConditionFalse, reason, message)
}

// MarkRunning registers the start time and changes the Succeeded condition to Unknown.
func (ir *InternalRequest) MarkRunning() {
	if ir.HasStarted() {
		return
	}

	ir.Status.StartTime = &metav1.Time{Time: time.Now()}
	ir.setStatusCondition(InternalRequestSucceededConditionType, metav1.ConditionUnknown, InternalRequestRunning)
}

// MarkSucceeded registers the completion time and changes the Succeeded condition to True.
func (ir *InternalRequest) MarkSucceeded() {
	if !ir.HasStarted() || ir.HasCompleted() {
		return
	}

	ir.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	ir.setStatusCondition(InternalRequestSucceededConditionType, metav1.ConditionTrue, InternalRequestSucceeded)

	go metrics.RegisterCompletedInternalRequest(ir.Spec.Request, ir.Namespace, InternalRequestSucceeded.String(), ir.Status.StartTime, ir.Status.CompletionTime, true)
}

// setStatusCondition creates a new condition with the given InternalRequestSucceededConditionType, status and reason. Then, it sets this new condition,
// unsetting previous conditions with the same type as necessary.
func (ir *InternalRequest) setStatusCondition(conditionType string, status metav1.ConditionStatus, reason HandlingReason) {
	ir.setStatusConditionWithMessage(conditionType, status, reason, "")
}

// setStatusConditionWithMessage creates a new condition with the given InternalRequestSucceededConditionType, status, reason and message. Then, it sets this new condition,
// unsetting previous conditions with the same type as necessary.
func (ir *InternalRequest) setStatusConditionWithMessage(conditionType string, status metav1.ConditionStatus, reason HandlingReason, message string) {
	meta.SetStatusCondition(&ir.Status.Conditions, metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason.String(),
		Message: message,
	})
}

// +kubebuilder:object:root=true

// InternalRequestList contains a list of InternalRequest.
type InternalRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternalRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternalRequest{}, &InternalRequestList{})
}
