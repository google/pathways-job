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
	corev1 "k8s.io/api/core/v1"
	// corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

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

// PathwaysJob creates a Pathways workload. It sets up the TPU
// workers needed for training or inference, along with Pathways
// resources such as the Pathways Resource Manager(RM) and Proxy
// server at the specifiec controller node location. It provides
// an option to deploy a user workload and other containers within
// a Pod. If this pod is not provided, then the workload is assumed
// to be running in headless mode and the user can connect to Proxy,
// to run their workloads.

type PathwaysAPISpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ColocationPolicy defines whether the user job and the Pathways resources (RM, proxy)
	// must be colocated on TPUs with the Pathways workers or not.
	// Users may opt for best-effort placement where scheduler places the RM and proxy
	// on the CPU nodepools by default.
	// Default is best-effort.
	ColocationPolicy ColocationPolicy `json:"colocationPolicy,omitempty"`

	// PathwaysWorkerNodeSelector is used to specify the nodeSelector for
	// Pathways TPU workers (accelerator type and topology).
	PathwaysWorkerNodeSelector map[string]string `json:"pathwaysWorkerNodeSelector,omitempty"`

	// PathwaysControllerNodeSelector is used to specify where Pathways resources
	// such as RM and proxy should be deployed.
	PathwaysControllerNodeSelector map[string]string `json:"pathwaysControllerNodeSelector,omitempty"`

	// Maximum number of times the JobSet is restarted.
	MaxRestarts int32 `json:"maxRestarts,omitempty"`

	// Number of TPU slices requested for the Pathways workers.
	NumSlices int32 `json:"numSlices,omitempty"`

	// PathwaysDir is a persistent location like GCS at which temporary
	// Pathways artifacts can be stored like HBM state during interruptions.
	// Currently, Pathways supports a precreated GCS directory only.
	PathwaysDir string `json:"pathwaysDir,omitempty"`

	// PathwaysVersion is the version of the Pathways client.
	PathwaysVersion string `json:"pathwaysVersion,omitempty"`

	// UserPodTemplate accepts a pod composed of user's workload
	// (and other) containers.
	// https://pkg.go.dev/k8s.io/api/core/v1#PodTemplateSpec
	// +optional
	UserPodTemplate *corev1.PodTemplateSpec `json:"template" protobuf:"bytes,6,opt,name=template"`
}

// PathwaysAPIStatus defines the observed state of PathwaysAPI
type PathwaysAPIStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Track the state of the Pathways workload, acceptable values are -
	// Running, Suspended, Completed, Failed.
	// Contains a human readable message to provide additional details to the // user.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:validation:Enum=colocate;best-effort
type ColocationPolicy string

const (
	Colocate   ColocationPolicy = "colocate"
	BestEffort ColocationPolicy = "best-effort"
)

func init() {
	SchemeBuilder.Register(&PathwaysAPI{}, &PathwaysAPIList{})
}
