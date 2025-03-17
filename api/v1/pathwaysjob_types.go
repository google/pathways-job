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

	// PathwaysDir is a persistent GCS location at which temporary
	// Pathways artifacts can be stored like HBM state during interruptions.
	// Currently, Pathways supports a precreated GCS directory only.
	PathwaysDir string `json:"pathwaysDir,omitempty"`

	// PathwaysVersion is the version of the Pathways cluster.
	// This indicates the version of the Pathways RM, Proxy and Workers.
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

	// Aggregate  of the PathwaysJob workload, based on worker and
	// controller statuses.
	// One of - Pending, Running, Suspended, Completed, Failed.
	// Contains a human readable message to provide additional details to the
	// user. Conditions are mentioned in PathwaysConditionType.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Track the  of the Pathways TPU workers -
	WorkersStatus *WorkersStatus `json:"workersStatus,omitempty"`

	// Tracks the  of the Pathways controller -
	// 1. derived from "leader" replicatedJob in colocated mode
	// (leader job contains "rm", "proxy" and "user" as containers)
	// 2. derived from "rm" and "proxy" replicatedJobs in
	// default + headless mode.
	// 3. derived from "rm", "proxy" and "user-job" replicatedJobs in
	// default + container mode.
	ControllerStatus *PathwaysComponentStatus `json:"controllerStatus,omitempty"`
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
	DeploymentMode DeploymentMode `json:"deploymentMode,omitempty"`

	// UserPodTemplate accepts a pod composed of user's workload
	// (and other) containers.
	// https://pkg.go.dev/k8s.io/api/core/v1#PodTemplateSpec
	// +optional
	UserPodTemplate *corev1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,6,opt,name=template"`
}

type PathwaysConditionType string

// These are built-in conditions for PathwaysJob.
const (
	// PathwaysJobPending means the PathwaysJob is constructed and/or may be
	// deployed, but pods are yet to be scheduled on nodes.
	PathwaysJobPending PathwaysConditionType = "Pending"
	// PathwaysJobRunning means PathwaysJob has been scheduled and
	// Pathways servers have started running.
	PathwaysJobRunning PathwaysConditionType = "Running"
	// PathwaysJobCompleted means the underlying JobSet has completed its
	// execution.
	PathwaysJobCompleted PathwaysConditionType = "Completed"
	// PathwaysJobFailed means the JobSet has failed its execution.
	// Reason for failure may be found in Condition.Message
	PathwaysJobFailed PathwaysConditionType = "Failed"
	// PathwaysJobSuspended means the underlying Jobset is suspended.
	PathwaysJobSuspended PathwaysConditionType = "Suspended"
)

type ControllerStatus struct {
	// Status of the Pathways Controller
	CurrentStatus *PathwaysComponentStatus `json:"currentStatus,omitempty"`
}

// ReplicatedJob Status in JobSet
type WorkersStatus struct {
	// Status aggregated over all TPU slices.
	// One of - Pending, Running, Suspended, Completed, Failed.
	AggregateWorkersStatus *PathwaysComponentStatus `json:"aggregateWorkersStatus,omitempty"`
	// Status details on each TPU worker slice
	WorkersSliceStatus []WorkerSliceStatus `json:"workersSliceStatus,omitempty"`
}

// Job Status in JobSet
type WorkerSliceStatus struct {
	// Individual TPU slice's status.
	SliceStatus *PathwaysComponentStatus `json:"sliceStatus,omitempty"`
	// Number of workers in the slice that are ready.
	Ready int32 `json:"ready,omitempty"`
}

type PathwaysComponentStatus string

// Pending - one of more jobs ready but not active.
// Running - all jobs active.
// Suspended - all jobs suspended.
// Completed - all jobs completed successfully.
// Failed - one or more jobs failed.
const (
	PathwaysComponentStatusPending PathwaysComponentStatus = "Pending"
	// Running will be based on a readiness probe
	PathwaysComponentStatusRunning   PathwaysComponentStatus = "Running"
	PathwaysComponentStatusCompleted PathwaysComponentStatus = "Completed"
	PathwaysComponentStatusFailed    PathwaysComponentStatus = "Failed"
	PathwaysComponentStatusSuspended PathwaysComponentStatus = "Suspended"
)

func init() {
	SchemeBuilder.Register(&PathwaysJob{}, &PathwaysJobList{})
}
