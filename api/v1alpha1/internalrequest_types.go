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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InternalRequestSpec defines the desired state of InternalRequest
type InternalRequestSpec struct {
	// Request is the name of the internal internalrequest which will be translated into a Tekton pipeline
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +required
	Request string `json:"internalrequest"`

	// Params is the list of optional parameters to pass to the Tekton pipeline
	// +optional
	Params map[string]string `json:"params,omitempty"`
}

// InternalRequestStatus defines the observed state of InternalRequest
type InternalRequestStatus struct {
	// Conditions represent the latest available observations for the internalrequest
	// +optional
	Conditions []metav1.Condition `json:"conditions"`

	// Results is the list of optional results as seen in the Tekton pipeline
	// +optional
	Results map[string]string `json:"results,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InternalRequest is the Schema for the internalrequests API
type InternalRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InternalRequestSpec   `json:"spec,omitempty"`
	Status InternalRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InternalRequestList contains a list of InternalRequest
type InternalRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternalRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternalRequest{}, &InternalRequestList{})
}
