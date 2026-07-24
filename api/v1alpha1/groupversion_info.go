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

// Package v1alpha1 contains API Schema definitions for the appstudio v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=appstudio.redhat.com
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "appstudio.redhat.com", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &schemeBuilder{}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// schemeBuilder is a minimal stand-in for the now-deprecated
// sigs.k8s.io/controller-runtime/pkg/scheme.Builder. controller-runtime
// recommends api packages depend only on k8s.io/apimachinery, so this keeps
// the same Register(objects...) call sites in this package's *_types.go
// files working without pulling in controller-runtime.
type schemeBuilder struct {
	runtime.SchemeBuilder
}

// Register adds one or more objects to the SchemeBuilder so they can be added to a Scheme.
func (bld *schemeBuilder) Register(object ...runtime.Object) *schemeBuilder {
	bld.SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(GroupVersion, object...)
		metav1.AddToGroupVersion(scheme, GroupVersion)
		return nil
	})
	return bld
}
