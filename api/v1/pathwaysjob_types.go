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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// PathwaysJob is the Schema for the PathwaysJobs API
type PathwaysJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PathwaysJobSpec   `json:"spec,omitempty"`
	Status PathwaysJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PathwaysJobList contains a list of PathwaysJob
type PathwaysJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PathwaysJob `json:"items"`
}

// PathwaysJob creates a Pathways workload. It sets up the TPU
// workers needed for training or inference, along with Pathways
// resources such as the Pathways Resource Manager(RM) and Proxy
// server at the specifiec controller node location. It provides
// an option to deploy a user workload and other containers within
// a Pod. If this pod is not provided, then the workload is assumed
// to be running in headless mode and the user can connect to Proxy,
// to run their workloads.

type PathwaysJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Maximum number of times the JobSet is restarted.
	MaxRestarts int32 `json:"maxRestarts,omitempty"`

	// PathwaysDir is a persistent location like GCS at which temporary
	// Pathways artifacts can be stored like HBM state during interruptions.
	// Currently, Pathways supports a precreated GCS directory only.
	PathwaysDir string `json:"pathwaysDir,omitempty"`

	// PathwaysVersion is the version of the Pathways client.
	PathwaysVersion string `json:"pathwaysVersion,omitempty"`

	// The list of worker types created for the Pathways Job. Currently only
	// one type of worker is supported.
	Workers []WorkerSpec `json:"workers"`

	// Pathways single-controller specifications and user workload.
	Controller *ControllerSpec `json:"controller"`
}

// PathwaysJobStatus defines the observed state of PathwaysJob
type PathwaysJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Track the state of the Pathways workload, acceptable values are -
	// Running, Suspended, Completed, Failed.
	// Contains a human readable message to provide additional details to the // user.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:validation:Enum=colocate;default
type DeploymentMode string

const (
	Colocate DeploymentMode = "colocate"
	Default  DeploymentMode = "default"
)

// The WorkerSpec struct lists the specifications for the
// Pathways workers.
type WorkerSpec struct {
	// This will translate to a nodeSelector of the form
	// cloud.google.com/gke-tpu-accelerator: tpu-v5-lite-podslice
	Type string `json:"type"`

	// This will translate to a nodeSelector of the form
	// cloud.google.com/gke-tpu-topology:2x2
	Topology string `json:"topology"`

	// Number of TPU slices requested for the Pathways workers.
	NumSlices int32 `json:"numSlices"`
}

// The ControllerSpec struct lists the specifications for the
// Pathways controller. User workload can also be provided here.
type ControllerSpec struct {
	// DeploymentMode defines whether the user job and the Pathways
	// resources (RM, proxy) must be colocated on TPUs, with the Pathways
	// workers or not. If user chooses to "colocate", then the Pathways RM
	// and proxy run together with the user job as a single pod.
	// Users may opt for "default" placement where scheduler places the
	// RM pod and the proxy pod on the CPU nodepools by default. User
	// workload will be deployed separately, as a pod.
	DeploymentMode DeploymentMode `json:"deploymentMode,omitempty"`

	// UserPodTemplate accepts a pod composed of user's workload
	// (and other) containers.
	// https://pkg.go.dev/k8s.io/api/core/v1#PodTemplateSpec
	// +optional
	UserPodTemplate *corev1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,6,opt,name=template"`
}

func init() {
	SchemeBuilder.Register(&PathwaysJob{}, &PathwaysJobList{})
}
