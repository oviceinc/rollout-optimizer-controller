/*
Copyright 2023.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RolloutScaleDownSpec defines the desired state of RolloutScaleDown
type RolloutScaleDownSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of RolloutScaleDown. Edit rolloutscaledown_types.go to remove/update
	TargetRollout    string `json:"targetRollout,omitempty"`
	TerminatePerOnce int    `json:"terminatePerOnce,1"`
	CoolTimeSeconds  int    `json:"coolTimeSeconds,30"`
}

// RolloutScaleDownStatus defines the observed state of RolloutScaleDown
type RolloutScaleDownStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RolloutScaleDown is the Schema for the rolloutscaledowns API
type RolloutScaleDown struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RolloutScaleDownSpec   `json:"spec,omitempty"`
	Status RolloutScaleDownStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RolloutScaleDownList contains a list of RolloutScaleDown
type RolloutScaleDownList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RolloutScaleDown `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RolloutScaleDown{}, &RolloutScaleDownList{})
}
