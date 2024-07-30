/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PathwaysAPISpec defines the desired state of PathwaysAPI
type PathwaysAPISpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Text message is an example field of PathwaysAPI. Edit pathwaysapi_types.go to remove/update
	TextMessage  string `json:"textMessage,omitempty"`
	TpuType      string `json:"tpuType,omitempty"`
	NumSlices    int32  `json:"numSlices,omitempty"`
	WorkloadMode string `json:"workloadMode,omitempty"`
	// JobSetSpec   jobsetv1alpha2.JobSet `json:"jobSetSpec"`
}

// tpuType: v4-8
// numSlices: 12
// workloadMode: headless
// backoffLimit: 4 # pass this down to JobSet
// workloadImage: <AR location of the workload image>
// workloadName: <Identifier for this workload>
// workloadType: inference # training or inference

// PathwaysAPIStatus defines the observed state of PathwaysAPI
type PathwaysAPIStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PathwaysAPI is the Schema for the pathwaysapis API
type PathwaysAPI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PathwaysAPISpec   `json:"spec,omitempty"`
	Status PathwaysAPIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PathwaysAPIList contains a list of PathwaysAPI
type PathwaysAPIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PathwaysAPI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PathwaysAPI{}, &PathwaysAPIList{})
}
