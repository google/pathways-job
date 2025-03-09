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

package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	jobsetclient "sigs.k8s.io/jobset/client-go/clientset/versioned"

	pathwaysjob "pathways-job/api/v1"
)

// PathwaysJobReconciler reconciles a PathwaysJob object
type PathwaysJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Public variables to store TPU information
var (
	TPUVersion   string
	TPUTopology  string
	InstanceType string
	NumVMs       int32
)

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PathwaysJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile

// +kubebuilder:rbac:groups="",resources=events,verbs=create;watch;update;patch
// +kubebuilder:rbac:groups=pathways-job.pathways.domain,resources=pathwaysjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pathways-job.pathways.domain,resources=pathwaysjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pathways-job.pathways.domain,resources=pathwaysjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=jobset.x-k8s.io,resources=jobsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jobset.x-k8s.io,resources=jobsets/status,verbs=get;update;patch
func (r *PathwaysJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := &pathwaysjob.PathwaysJob{}
	log := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("PathwaysJob: CONTROLLER WORKING...", "req.NamespacedName", req.NamespacedName.String(), "req.Namespace", req.Namespace)

	// 1. Fetch the Pathways object
	if err := r.Get(ctx, req.NamespacedName, pw); err != nil {
		log.Info("PathwaysJob: Unable to fetch Pathways ")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Process the Pathways object and build a JobSet client
	kubeconfig := ctrl.GetConfigOrDie()
	// log.Info("PathwaysJob: config established...")

	jobSetClient := jobsetclient.NewForConfigOrDie(kubeconfig)
	// log.Info("PathwaysJob: client built for JobSet...")

	// 2.1 Figure out if PathwaysJob is already present
	// if it is the case, there is nothing to do.
	// (ToDo) check states in "Suspended / Completed / Failed states",

	childJobSet, err := r.getChildJobSet(ctx, pw, jobSetClient)
	if err != nil {
		log.Info("PathwaysJob: can't find JobSet, may create one!")
		// return ctrl.Result{}, err
	} else if childJobSet != nil {
		log.Info("PathwaysJob: JobSet exists, not creating")
		// 2.2 Find out JobSet's status
		r.findJobSetStatus(ctx, childJobSet)
		return ctrl.Result{}, nil
	}

	// 3. Update the cluster - create update and delete other resources
	log.Info("PathwaysJob: creating JobSet \n")
	if err := r.createJobSet(ctx, pw, jobSetClient); err != nil {
		log.Error(err, "PathwaysJob: failed to create JobSet")
		return ctrl.Result{}, err
	}

	//4. Update the object's status using Conditions (?)
	childJobSet, _ = r.getChildJobSet(ctx, pw, jobSetClient)
	r.findJobSetStatus(ctx, childJobSet)

	//5. Return a result
	log.Info("PathwaysJob: DONE DONE DONE!")
	return ctrl.Result{}, nil
}

func (r *PathwaysJobReconciler) getChildJobSet(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) (*jobsetv1alpha2.JobSet, error) {
	log3 := ctrl.LoggerFrom(ctx)
	// .WithValues("pathwaysjob", klog.KObj(pw))
	// ctx = ctrl.LoggerInto(ctx, log3)

	log3.Info("PathwaysJob: in getChildJobSet", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())

	var js *jobsetv1alpha2.JobSet
	js, err := jobSetClient.JobsetV1alpha2().JobSets(pw.GetObjectMeta().GetNamespace()).Get(ctx, pw.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return js, nil
}

// Create the JobSet for 'colocated' or 'default' deployment modes.
func (r *PathwaysJobReconciler) createJobSet(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) error {
	log2 := ctrl.LoggerFrom(ctx)
	// .WithValues("pathwaysjob", klog.KObj(pw))
	// ctx = ctrl.LoggerInto(ctx, log2)

	log2.Info("PathwaysJob: in createJobSet", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())

	var jobs []jobsetv1alpha2.ReplicatedJob
	var rmJobName string

	calculateTPUInfo(ctx, pw)
	log2.Info("PathwaysJob: in createJobSet TPU variables", "TPUVersion", TPUVersion, "TPUTopology", TPUTopology, "InstanceType", InstanceType, "NumVMs", NumVMs)

	// Pathways Spec + JobSet for training or batch inference ------
	if pw.Spec.Controller.DeploymentMode == pathwaysjob.Colocate {
		rmJobName = "leader"
		jobs, _ = MakeLeaderJobForColocatedDeployment(ctx, pw, rmJobName)
	} else {
		rmJobName = "rm"
		jobs, _ = MakeJobsForDefaultDeployment(ctx, pw, rmJobName)
	}

	workerJob, _ := MakeWorkerJob(ctx, pw, rmJobName)

	mainJobSetConfig := jobsetv1alpha2.JobSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pw.GetName(),
			Namespace: pw.GetNamespace(),
		},
		Spec: jobsetv1alpha2.JobSetSpec{
			FailurePolicy: &jobsetv1alpha2.FailurePolicy{
				MaxRestarts: pw.Spec.MaxRestarts,
			},
			SuccessPolicy:  &jobsetv1alpha2.SuccessPolicy{Operator: jobsetv1alpha2.OperatorAll, TargetReplicatedJobs: []string{rmJobName}}, // ToDo: change to user job
			ReplicatedJobs: append(jobs, workerJob),
		},
	}

	// Set Pathways controller as the owner of the JobSet for garbage collection.
	if err := ctrl.SetControllerReference(pw, &mainJobSetConfig, r.Scheme); err != nil {
		log2.Info("PathwaysJob: failed to set Pathways as owner of JobSet.", "error ", err)
	} else {
		log2.Info("PathwaysJob: successfully set Pathways as owner of JobSet.")
	}

	js, err := jobSetClient.JobsetV1alpha2().JobSets(pw.GetObjectMeta().GetNamespace()).Create(ctx, &mainJobSetConfig, metav1.CreateOptions{})

	if err != nil {
		return err
	} else {
		log2.Info("PathwaysJob: successfully created JobSet: ", "JobSet name", js.Name)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PathwaysJobReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := jobsetv1alpha2.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&pathwaysjob.PathwaysJob{}).
		Owns(&jobsetv1alpha2.JobSet{}). // PathwaysJob owns the underlying JobSet object
		Complete(r)
}

// Find the status of the underlying JobSet for 'colocated' or 'default' deployment modes.
func (r *PathwaysJobReconciler) findJobSetStatus(ctx context.Context, js *jobsetv1alpha2.JobSet) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("PathwaysJob findJobSetStatus", "Jobset name", js.ObjectMeta.Name)

	for _, c := range js.Status.Conditions {
		log.Info("\n\n PathwaysJob CONDITION", "CONDITION", c.Type, "Status", c.Status, "Message", c.Message)
		if (c.Type == string(jobsetv1alpha2.JobSetSuspended) || c.Type == string(jobsetv1alpha2.JobSetCompleted) || c.Type == string(jobsetv1alpha2.JobSetFailed)) && c.Status == metav1.ConditionTrue {
			log.Info("\n\n PathwaysJob: JobSet in TERMINAL STATE", "Condition ", c.Type)
		}
		if (c.Type == string(jobsetv1alpha2.JobSetStartupPolicyCompleted) ||
			c.Type == string(jobsetv1alpha2.JobSetStartupPolicyInProgress)) && c.Status == metav1.ConditionTrue {
			log.Info("\n\n PathwaysJob: JobSet in TERMINAL STATE", "Condition ", c.Type)
		}
	}
}

// ---------------------- PATHWAYS HELPERS --------------------------
// Find TPU version from the worker's type (- used to determine Pathways instance_type)
func extractTPUVersionFromWorkerType(ctx context.Context, tpuGKEAcceleratorType pathwaysjob.WorkerType) string {
	log := ctrl.LoggerFrom(ctx)
	parts := strings.Split(string(tpuGKEAcceleratorType), "-")
	if len(parts) >= 2 && strings.HasPrefix(parts[1], "v") {
		log.Info("TPU type and version", "value ", parts[0]+parts[1])
		return parts[0] + parts[1] // example tpuv4 / tpuv5
	}
	return ""
}

// Validate that topology provided is valid for the provided worker type.
func validateTPUTopologyWithType(tpuGKEAcceleratorType pathwaysjob.WorkerType, topology string) string {
	// ToDo: validate topology based on the TPU type
	return topology
}

// Calculate the number of VMs based on the Topology (- used in completions/parallelisms)
func calculateVMsFromTopology(topology string) int32 {
	parts := strings.Split(topology, "x") // Examples - 2x2x4 or 4x4
	if len(parts) < 2 {
		return 0
	}
	// Calculate the number of chips based on the Topology.
	chips := 1
	for _, part := range parts {
		num, _ := strconv.Atoi(part)
		chips *= num
	}
	vms := 1
	chipsperVM := 4
	if chips >= chipsperVM {
		vms = chips / chipsperVM
	}
	return int32(vms)
}

// Calculate all TPU related information
func calculateTPUInfo(ctx context.Context, pw *pathwaysjob.PathwaysJob) {
	TPUVersion := extractTPUVersionFromWorkerType(ctx, pw.Spec.Workers[0].Type)                      // setting public variable
	TPUTopology := validateTPUTopologyWithType(pw.Spec.Workers[0].Type, pw.Spec.Workers[0].Topology) // setting public variable
	InstanceType = TPUVersion + ":" + TPUTopology                                                    // setting public variable
	NumVMs = calculateVMsFromTopology(pw.Spec.Workers[0].Topology)                                   // setting public variable
}

// Constructs the Pathways resource manager container spec for the underlying JobSet
func MakeResourceManagerContainer(pw *pathwaysjob.PathwaysJob, rmJobName string) (*corev1.Container, error) {
	truth := true

	rmContainerSpec := corev1.Container{
		Name:            "pathways-rm",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/sanitized_server:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{Privileged: &truth},
		Args: []string{
			"--server_port=29001",
			fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
			"--node_type=resource_manager",
			fmt.Sprintf("--instance_count=%d", int32(pw.Spec.Workers[0].NumSlices)),
			fmt.Sprintf("--instance_type=%s", InstanceType),
		},
		Env: []corev1.EnvVar{
			{Name: "REPLICATED_JOB_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/replicatedjob-name']"}}},
			{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
			{Name: "HOST_ADDRESS", Value: fmt.Sprintf("%s-%s-0-0.%s", pw.GetName(), rmJobName, pw.GetName())},
			{Name: "TPU_SKIP_MDS_QUERY", Value: "true"},
		},
		Ports: []corev1.ContainerPort{{ContainerPort: 29001}, {ContainerPort: 29002}},
		// Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(8000000000, resource.DecimalSI)}},
	}
	return &rmContainerSpec, nil
}

// Constructs the Pathways proxy container spec for the underlying JobSet
func MakeProxyContainer(pw *pathwaysjob.PathwaysJob, rmJobName string) (*corev1.Container, error) {
	truth := true

	proxyContainerSpec := corev1.Container{
		Name:            "pathways-proxy",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/sanitized_proxy_server:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{Privileged: &truth},
		Args: []string{
			"--server_port=29008",
			fmt.Sprintf("--resource_manager_address=%s-%s-0-0.%s:29001", pw.GetName(), rmJobName, pw.GetName()),
			fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
		},
		Ports: []corev1.ContainerPort{{ContainerPort: 29008}},
		// Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(10000000000, resource.DecimalSI)}},
	}
	return &proxyContainerSpec, nil
}

// Constructs Pathways worker replicated job for both 'colocated' and 'default' deployment modes.
func MakeWorkerJob(ctx context.Context, pw *pathwaysjob.PathwaysJob, rmJobName string) (jobsetv1alpha2.ReplicatedJob, error) {
	truth := true
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	workerJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "worker",
		Replicas: int32(pw.Spec.Workers[0].NumSlices),
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: ptr.To(int32(4)),
				Completions:  ptr.To(int32(NumVMs)), // number of workers remember to change
				Parallelism:  ptr.To(int32(NumVMs)), // number of workers  remember to change
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "pathways-worker",
								Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/sanitized_server:latest",
								ImagePullPolicy: "Always",
								SecurityContext: &corev1.SecurityContext{Privileged: &truth},
								Args: []string{
									"--server_port=29005",
									fmt.Sprintf("--resource_manager_address=%s-%s-0-0.%s:29001", pw.GetName(), rmJobName, pw.GetName()),
									fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
								},
								Env: []corev1.EnvVar{
									{Name: "TPU_MIN_LOG_LEVEL", Value: "0"},
									{Name: "TF_CPP_MIN_LOG_LEVEL", Value: "0"},
									{Name: "XCLOUD_ENVIRONMENT", Value: "GCP"},
								},
								Ports: []corev1.ContainerPort{{ContainerPort: 29005}, {ContainerPort: 29006}, {ContainerPort: 8471}, {ContainerPort: 8080}},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "shared-tmp",
										MountPath: "/tmp",
									},
								},
								Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"google.com/tpu": *resource.NewQuantity(4, resource.DecimalSI)}},
							}, // end Pathways worker container
						},
						NodeSelector: map[string]string{
							"cloud.google.com/gke-tpu-accelerator": string(pw.Spec.Workers[0].Type),
							"cloud.google.com/gke-tpu-topology":    pw.Spec.Workers[0].Topology,
						},
						Volumes: []corev1.Volume{
							{
								Name: "shared-tmp",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/tmp",
										Type: &volumeSourceType,
									},
								},
							},
						}, // end Volumes
						HostNetwork: true,                              // For performance == McJAX
						DNSPolicy:   corev1.DNSClusterFirstWithHostNet, // For performance == McJAX
					},
				},
			},
		},
	} // end worker replicated job
	return workerJob, nil
}

// Affinity rules to allow the leader pod to coexist with worker pod in the 'colocate' mode.
func MakePodAffinityRules(pw *pathwaysjob.PathwaysJob) (*corev1.Affinity, error) {
	affinity := corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "jobset.sigs.k8s.io/jobset-name",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{pw.GetName()},
							},
						},
					},
					TopologyKey: "cloud.google.com/gke-nodepool",
				},
			},
		}, // end PodAffinity
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "jobset.sigs.k8s.io/jobset-name",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{pw.GetName()},
							},
							{
								Key:      "job-name",
								Operator: metav1.LabelSelectorOpExists,
							},
						},
					},
					TopologyKey: "cloud.google.com/gke-nodepool",
				},
			},
		}, // end PodAntiAffinity
	} // end Affinity
	return &affinity, nil
}

// Get the containers (main workload and any sidecars) from the user's pod spec.
// This is used to inject the containers into the leader pod in the 'colocate' deployment mode.
func GetUserContainerList(pw *pathwaysjob.PathwaysJob) ([]corev1.Container, error) {
	containerList := pw.Spec.Controller.UserPodTemplate.Spec.Containers
	return containerList, nil
}

// Construct the "leader" replicated job containing the Pathways RM, Pathways Proxy and User job
// as containers within a pod for the 'colocate' deployment mode.
func MakeLeaderJobForColocatedDeployment(ctx context.Context, pw *pathwaysjob.PathwaysJob, rmJobName string) ([]jobsetv1alpha2.ReplicatedJob, error) {
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	RMContainerSpec, _ := MakeResourceManagerContainer(pw, rmJobName)
	ProxyContainerSpec, _ := MakeProxyContainer(pw, rmJobName)
	affinitySpec, _ := MakePodAffinityRules(pw)
	containerList, _ := GetUserContainerList(pw)
	containerList = append(containerList, *RMContainerSpec, *ProxyContainerSpec)

	leaderJob := jobsetv1alpha2.ReplicatedJob{
		Name:     rmJobName,
		Replicas: 1,
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: ptr.To(int32(0)),
				Completions:  ptr.To(int32(1)),
				Parallelism:  ptr.To(int32(1)),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Affinity: affinitySpec,
						NodeSelector: map[string]string{
							"cloud.google.com/gke-tpu-accelerator": string(pw.Spec.Workers[0].Type),
							"cloud.google.com/gke-tpu-topology":    pw.Spec.Workers[0].Topology,
						},
						HostNetwork: true,                              // For performance == McJAX
						DNSPolicy:   corev1.DNSClusterFirstWithHostNet, // For performance == McJAX
						Tolerations: []corev1.Toleration{
							{
								Key:      "google.com/tpu",
								Operator: "Exists",
								Effect:   "NoSchedule",
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "shared-tmp",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/tmp",
										Type: &volumeSourceType,
									},
								},
							},
						}, // end Volumes
						Containers: containerList, // end leader []containers
					}, // end PodSpec
				},
			},
		},
	} // end replicated Job
	return []jobsetv1alpha2.ReplicatedJob{leaderJob}, nil
}

// Construct replicated jobs for Pathways RM, Pathways Proxy and the user job for the 'default' deployment mode.
func MakeJobsForDefaultDeployment(ctx context.Context, pw *pathwaysjob.PathwaysJob, rmJobName string) ([]jobsetv1alpha2.ReplicatedJob, error) {
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	RMContainerSpec, _ := MakeResourceManagerContainer(pw, rmJobName)
	ProxyContainerSpec, _ := MakeProxyContainer(pw, rmJobName)
	userContainerList, _ := GetUserContainerList(pw)

	rmJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "rm",
		Replicas: 1,
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: ptr.To(int32(0)),
				Completions:  ptr.To(int32(1)),
				Parallelism:  ptr.To(int32(1)),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						HostNetwork: true,                              // For performance == McJAX
						DNSPolicy:   corev1.DNSClusterFirstWithHostNet, // For performance == McJAX
						Tolerations: []corev1.Toleration{
							{
								Key:      "google.com/tpu",
								Operator: "Exists",
								Effect:   "NoSchedule",
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "shared-tmp",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/tmp",
										Type: &volumeSourceType,
									},
								},
							},
						}, // end Volumes
						Containers: []corev1.Container{*RMContainerSpec}, // end leader []containers
					}, // end PodSpec
				},
			},
		},
	} // end replicated Job

	proxyJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "proxy",
		Replicas: 1,
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: ptr.To(int32(0)),
				Completions:  ptr.To(int32(1)),
				Parallelism:  ptr.To(int32(1)),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						HostNetwork: true,                              // For performance == McJAX
						DNSPolicy:   corev1.DNSClusterFirstWithHostNet, // For performance == McJAX
						Tolerations: []corev1.Toleration{
							{
								Key:      "google.com/tpu",
								Operator: "Exists",
								Effect:   "NoSchedule",
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "shared-tmp",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/tmp",
										Type: &volumeSourceType,
									},
								},
							},
						}, // end Volumes
						Containers: []corev1.Container{*ProxyContainerSpec}, // end leader []containers
					}, // end PodSpec
				},
			},
		},
	} // end replicated Job

	// Adding user job conditionally for headless mode, if the user has provided containers in PodSpec.
	// ToDo: Add other things in PodSpec
	if len(userContainerList) > 0 {
		userJob := jobsetv1alpha2.ReplicatedJob{
			Name:     "user-job",
			Replicas: 1,
			Template: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: ptr.To(int32(0)),
					Completions:  ptr.To(int32(1)),
					Parallelism:  ptr.To(int32(1)),
					// Template: corev1.PodTemplateSpec{
					// 	Spec: pw.Spec.Controller.UserPodTemplate.Spec,
					// },
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							HostNetwork: true,                              // For performance == McJAX
							DNSPolicy:   corev1.DNSClusterFirstWithHostNet, // For performance == McJAX
							Tolerations: []corev1.Toleration{ // tolerations are important here to not run this job on TPUs
								{
									Key:      "google.com/tpu",
									Operator: "Exists",
									Effect:   "NoSchedule",
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "shared-tmp",
									VolumeSource: corev1.VolumeSource{
										HostPath: &corev1.HostPathVolumeSource{
											Path: "/tmp",
											Type: &volumeSourceType,
										},
									},
								},
							}, // end Volumes
							Containers: userContainerList,
						}, // end PodSpec
					},
				},
			},
		} // end replicated Job
		// Default mode jobs, when user pod is provided
		return []jobsetv1alpha2.ReplicatedJob{rmJob, proxyJob, userJob}, nil
	}
	// Default mode jobs, headless mode
	return []jobsetv1alpha2.ReplicatedJob{rmJob, proxyJob}, nil
}
