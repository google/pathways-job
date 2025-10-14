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
// +kubebuilder:printcolumn:name="TerminalState",priority=0,JSONPath=".status.terminalState",type=string,description="Final state of the PathwaysJob"
// +kubebuilder:printcolumn:name="Completed",type="string",JSONPath=".status.conditions[?(@.type==\"Completed\")].status"
// +kubebuilder:printcolumn:name="Failed",type="string",JSONPath=".status.conditions[?(@.type==\"Failed\")].status"
// +kubebuilder:printcolumn:name="Suspended",type="string",JSONPath=".status.conditions[?(@.type==\"Suspended\")].status"
// +kubebuilder:printcolumn:name="Age",JSONPath=".metadata.creationTimestamp",type=date,description="Creation time for the PathwaysJob"
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
	// +listType=map
	// +listMapKey=type
	Workers []WorkerSpec `json:"workers"`

	// Pathways single-controller specifications and user workload.
	Controller *ControllerSpec `json:"controller"`

	// PathwaysJob components that can be customized.
	// +optional
	// +kubebuilder:validation:MaxItems=4
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="custom components are immutable"
	// +listType=map
	// +listMapKey=componentType
	CustomComponents []CustomComponentsSpec `json:"customComponents,omitempty"`
}

// +kubebuilder:validation:Enum=colocate_head_with_workers;default
type DeploymentMode string

const (
	ColocateHeadWithWorkers DeploymentMode = "colocate_head_with_workers"
	Default                 DeploymentMode = "default"
)

// +kubebuilder:validation:Enum=tpu7x-standard-4t;ct6e-standard-4t;ct6e-standard-8t;ct5p-hightpu-4t;ct5lp-hightpu-4t;ct5lp-hightpu-8t;ct4p-hightpu-4t
type MachineType string

const (
	// 7x
	Tpu7x_standard_4t MachineType = "tpu7x-standard-4t"
	// v6e
	Ct6e_standard_4t MachineType = "ct6e-standard-4t"
	Ct6e_standard_8t MachineType = "ct6e-standard-8t"
	// v5p
	Ct5p_hightpu_4t MachineType = "ct5p-hightpu-4t"
	// v5e
	Ct5lp_hightpu_4t MachineType = "ct5lp-hightpu-4t"
	Ct5lp_hightpu_8t MachineType = "ct5lp-hightpu-8t"
	// v4
	Ct4p_hightpu_4t MachineType = "ct4p-hightpu-4t"
)

// The WorkerSpec struct lists the specifications for the
// Pathways workers.
type WorkerSpec struct {
	// MachineType is GKE machine type.
	// It will translate to a nodeSelector of the form
	// cloud.google.com/gke-tpu-accelerator: tpu-v5-lite-podslice
	Type MachineType `json:"type"`

	// Topology will translate to a nodeSelector of the form
	// cloud.google.com/gke-tpu-topology:2x2
	Topology string `json:"topology"`

	// Number of TPU slices requested for the Pathways workers.
	NumSlices int32 `json:"numSlices"`

	// Maximum times the workers in a slice can be restarted.
	// Used with elasticSlices.
	// +kubebuilder:default=1
	MaxSliceRestarts int32 `json:"maxSliceRestarts,omitempty"`

	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// This is set for Pathways workers only.
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// Priority class for the PathwaysJob workers if kueue is configured on the cluster.
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// NodeSelector is a selector which must be true for the worker to fit on a node.
	// Selector which must match a node's labels for the worker to be scheduled on that node.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// The ControllerSpec struct lists the specifications for the
// Pathways controller. User workload can also be provided here.
type ControllerSpec struct {
	// DeploymentMode defines whether the user job and the Pathways
	// resources (RM, proxy) must be colocated on TPUs, with the Pathways
	// workers or not. If user chooses to "colocate_head_with_workers", then the Pathways RM
	// and proxy run together with the user job as a single pod.
	// Users may opt for "default" placement where scheduler places the
	// RM pod and the proxy pod on the CPU nodepools by default. User
	// workload will be deployed separately, as a pod.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="deploymentMode is immutable"
	DeploymentMode DeploymentMode `json:"deploymentMode"`

	// +kubebuilder:default="main"
	MainContainerName string `json:"mainContainerName,omitempty"`

	// Enables metrics collection for the PathwaysJob
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="enableMetricsCollection is immutable"
	EnableMetricsCollection bool `json:"enableMetricsCollection,omitempty"`

	// UserPodTemplate accepts a pod composed of user's workload
	// (and other) containers.
	// https://pkg.go.dev/k8s.io/api/core/v1#PodTemplateSpec
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="userPodTemplate is immutable"
	// +kubebuilder:pruning:PreserveUnknownFields
	// +crd:generateEmbeddedObjectMeta=true
	UserPodTemplate *corev1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,6,opt,name=template"`

	// Enables elasticity and sets the maximum number of slices
	// that can be missing at any given time.
	ElasticSlices int32 `json:"elasticSlices,omitempty"`
}

// CustomComponents struct lists the Pathways server/ Pathways Proxy/ Pathways worker
// attributes that can be customized by the user.
type CustomComponentsSpec struct {

	// Component type - one of pathways_server or worker or proxy_server
	ComponentType PathwaysComponentType `json:"componentType"`

	// Image for this component.
	// +optional
	Image string `json:"image,omitempty"`

	// Custom flags to be provided to this component.
	// +optional
	CustomFlags []string `json:"customFlags,omitempty"`

	// Custom environment variables to be provided to this component.
	// +optional
	CustomEnv []corev1.EnvVar `json:"customEnv,omitempty"`
}

// +kubebuilder:validation:Enum=pathways_server;worker;proxy_server;colocated_python_sidecar
type PathwaysComponentType string

const (
	// Pathways resource manager component
	PathwaysServer PathwaysComponentType = "pathways_server"
	// Pathways proxy component
	PathwaysProxy PathwaysComponentType = "proxy_server"
	// Pathways worker component
	PathwaysWorker PathwaysComponentType = "worker"
	//Pathways colocated python sidecar component, hosted on the workers.
	PathwaysColocatedPythonSidecar PathwaysComponentType = "colocated_python_sidecar"
)

// PathwaysJobStatus defines the observed state of PathwaysJob
type PathwaysJobStatus struct {

	// Conditions help track the state of the PathwaysJob.
	// PathwaysJob can be one of Pending, Running, Suspended, Completed, Failed.
	// They contain a human readable message to provide additional details
	// to the user. Condition types are mentioned in PathwaysConditionType.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Restarts tracks the number of times the PathwaysJob has restarted.
	Restarts int32 `json:"restarts,omitempty"`

	// TerminalState the state of the PathwaysJob when it finishes execution.
	// It can be either Complete or Failed. Otherwise, it is empty by default.
	TerminalState string `json:"terminalState,omitempty"`
}

type PathwaysConditionType string

// These are built-in conditions for PathwaysJob.
const (
	// PathwaysJobPending means the PathwaysJob is being constructed and
	// pods are yet to be scheduled on nodes.
	PathwaysJobPending PathwaysConditionType = "Pending"
	// PathwaysJobRunning means the PathwaysJob has started running and
	// the underlying JobSet is in progress.
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

func init() {
	SchemeBuilder.Register(&PathwaysJob{}, &PathwaysJobList{})
}
