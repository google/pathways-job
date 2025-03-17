/*
Copyright 2025.

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

// PathwaysJob is the Schema for the PathwaysJobs API
// Print column specifies the details seen on kubectl get pathwaysjob
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="TerminalState",JSONPath=".status.terminalState",type=string,description="Final state of PathwaysJob"
// +kubebuilder:printcolumn:name="Status",type="string",priority=0,JSONPath=".status.condition.type"
// +kubebuilder:printcolumn:name="Age",JSONPath=".metadata.creationTimestamp",type=date,description="Time this PathwaysJob was created"
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

	// Maximum number of times the JobSet is restarted.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="maxRestarts is immutable"
	MaxRestarts int32 `json:"maxRestarts,omitempty"`

	// PathwaysDir is a persistent GCS location at which temporary
	// Pathways artifacts can be stored like HBM state during interruptions.
	// Currently, Pathways supports a precreated GCS directory only.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="pathwaysDir is immutable"
	PathwaysDir string `json:"pathwaysDir,omitempty"`

	// PathwaysVersion is the version of the Pathways cluster.
	// This indicates the version of the Pathways RM, Proxy and Workers.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="pathwaysVersion is immutable"
	PathwaysVersion string `json:"pathwaysVersion,omitempty"`

	// The list of worker types created for the Pathways Job. Currently only
	// one type of worker is supported.
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="workers are immutable"
	Workers []WorkerSpec `json:"workers"`

	// Pathways single-controller specifications and user workload.
	Controller *ControllerSpec `json:"controller"`
}

// +kubebuilder:validation:Enum=colocate;default
type DeploymentMode string

const (
	Colocate DeploymentMode = "colocate"
	Default  DeploymentMode = "default"
)

// +kubebuilder:validation:Enum=tpu-v4-podslice;tpu-v5p-slice;tpu-v5-lite-podslice;tpu-v6e-slice
type WorkerType string

const (
	tpu_v4_podslice      WorkerType = "tpu-v4-podslice"
	tpu_v5p_slice        WorkerType = "tpu-v5p-slice"
	tpu_v5_lite_podslice WorkerType = "tpu-v5-lite-podslice"
	tpu_v6e_slice        WorkerType = "tpu-v6e-slice"
)

// The WorkerSpec struct lists the specifications for the
// Pathways workers.
type WorkerSpec struct {
	// This will translate to a nodeSelector of the form
	// cloud.google.com/gke-tpu-accelerator: tpu-v5-lite-podslice
	Type WorkerType `json:"type"`

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
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="deploymentMode is immutable"
	DeploymentMode DeploymentMode `json:"deploymentMode,omitempty"`

	// UserPodTemplate accepts a pod composed of user's workload
	// (and other) containers.
	// https://pkg.go.dev/k8s.io/api/core/v1#PodTemplateSpec
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="userPodTemplate is immutable"
	UserPodTemplate *corev1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,6,opt,name=template"`
}

// PathwaysJobStatus defines the observed state of PathwaysJob
type PathwaysJobStatus struct {

	// Aggregate  of the PathwaysJob workload, based on worker and
	// controller statuses.
	// One of - Pending, Running, Suspended, Completed, Failed.
	// Contains a human readable message to provide additional details to the
	// user. Conditions are mentioned in PathwaysConditionType.
	// +optional
	Condition metav1.Condition `json:"condition,omitempty"`

	// Restarts tracks the number of times the PathwaysJob has restarted.
	Restarts int32 `json:"restarts,omitempty"`

	// TerminalState the state of the PathwaysJob when it finishes execution.
	// It can be either Complete or Failed. Otherwise, it is empty by default.
	TerminalState string `json:"terminalState,omitempty"`
}

type PathwaysConditionType string

// These are built-in conditions for PathwaysJob.
const (
	// PathwaysJobPendingOrRunning means the PathwaysJob is constructed and/or may be
	// deployed, but pods are yet to be scheduled on nodes.
	PathwaysJobPendingOrRunning PathwaysConditionType = "PendingOrRunning"
	// PathwaysJobCompleted means the underlying JobSet has completed its
	// execution.
	PathwaysJobCompleted PathwaysConditionType = "Completed"
	// PathwaysJobFailed means the JobSet has failed its execution.
	// Reason for failure may be found in Condition.Message
	PathwaysJobFailed PathwaysConditionType = "Failed"
	// PathwaysJobSuspended means the underlying Jobset is suspended.
	PathwaysJobSuspended PathwaysConditionType = "Suspended"
)

func init() {
	SchemeBuilder.Register(&PathwaysJob{}, &PathwaysJobList{})
}
