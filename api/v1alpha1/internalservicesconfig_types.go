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

const InternalServicesConfigResourceName string = "config"

// InternalServicesConfigSpec defines the desired state of InternalServicesConfig.
type InternalServicesConfigSpec struct {
	// AllowList is the list of remote namespaces that are allowed to execute InternalRequests
	// +required
	AllowList []string `json:"allowList,omitempty"`

	// Catalog holds the information about the Tekton catalog
	// +optional
	Catalog Catalog `json:"catalog,omitempty"`
}

type Catalog struct {
	// Namespace where to find the Tekton Pipelines
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Bundle where to find the Tekton Pipelines
	// +optional
	Bundle string `json:"bundle,omitempty"`
}

// InternalServicesConfigStatus defines the observed state of InternalServicesConfig.
type InternalServicesConfigStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InternalServicesConfig is the Schema for the internalservicesconfigs API
type InternalServicesConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InternalServicesConfigSpec   `json:"spec,omitempty"`
	Status InternalServicesConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InternalServicesConfigList contains a list of InternalServicesConfig.
type InternalServicesConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternalServicesConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternalServicesConfig{}, &InternalServicesConfigList{})
}
