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

package controller

import (
	"context"
	"fmt"

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
		log.Info("PathwaysJob: can't find JobSet")
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
		// log3.Info("PathwaysJob: can't get JobSets: ", "error ", err)
		return nil, err
	}
	return js, nil
}

func (r *PathwaysJobReconciler) createJobSet(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) error {
	log2 := ctrl.LoggerFrom(ctx)
	// .WithValues("pathwaysjob", klog.KObj(pw))
	// ctx = ctrl.LoggerInto(ctx, log2)

	log2.Info("PathwaysJob: in createJobSet", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())

	// truth := true

	var jobs []jobsetv1alpha2.ReplicatedJob
	var rmJobName string

	//		// Pathways Spec + JobSet for training or batch inference ------
	if pw.Spec.Controller.DeploymentMode == pathwaysjob.Colocate {
		rmJobName = "leader"
		jobs, _ = MakeLeaderJobForColocatedDeployment(ctx, pw, rmJobName)
	} else {
		rmJobName = "rm"
		jobs, _ = MakeJobsForDefaultDeployment(ctx, pw, rmJobName)
	}

	workerJob, _ := MakeWorkerJob(ctx, pw, rmJobName)

	log2.Info("Length of jobs - ", "HERERERERERE", len(append(jobs, workerJob)))

	mainJobSetConfig := jobsetv1alpha2.JobSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pw.GetName(),
			Namespace: pw.GetNamespace(),
		},
		Spec: jobsetv1alpha2.JobSetSpec{
			FailurePolicy: &jobsetv1alpha2.FailurePolicy{
				MaxRestarts: pw.Spec.MaxRestarts,
			},
			SuccessPolicy:  &jobsetv1alpha2.SuccessPolicy{Operator: jobsetv1alpha2.OperatorAny, TargetReplicatedJobs: []string{rmJobName}}, // change this when needed
			ReplicatedJobs: append(jobs, workerJob),
			// Suspend:        &truth,
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
		Owns(&jobsetv1alpha2.JobSet{}). // For JobSet
		Complete(r)
}

func (r *PathwaysJobReconciler) findJobSetStatus(ctx context.Context, js *jobsetv1alpha2.JobSet) {
	// (bool, jobsetv1alpha2.JobSetConditionType)
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

	// for _, condition := range js.Status.Conditions {
	// 	log.Info("PathwaysJob findJobSetStatus Jobset ", "condition ", condition.Type)
	// 	if condition.Type == string(jobsetv1alpha2.JobSetStartupPolicyCompleted) ||
	// 		condition.Type == string(jobsetv1alpha2.JobSetStartupPolicyInProgress) {
	// 		// && c.Status == corev1.ConditionTrue
	// 		// return true, condition.Type
	// 	}
	// }
	// return false, ""
	// for _, status := range js.Status.ReplicatedJobsStatus {
	// 	log.Info("PathwaysJob RJ status ", "Name ", status.Name, "Ready ", status.Ready, "Succeeded ", status.Succeeded, "Failed ", status.Failed, "Active ", status.Active, "Suspended ", status.Suspended)
	// }

	// updateWorkerStatus(ctx, js)

}

// ---------------------- PATHWAYS HELPERS --------------------------

func MakeResourceManagerContainer(pw *pathwaysjob.PathwaysJob, rmJobName string) (*corev1.Container, error) {
	truth := true

	rmContainerSpec := corev1.Container{
		Name:            "pathways-rm",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{Privileged: &truth},
		Args: []string{
			"--server_port=29001",
			fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
			"--node_type=resource_manager",
			fmt.Sprintf("--instance_count=%d", int32(pw.Spec.Workers[0].NumSlices)),
			"--instance_type=tpuv4:2x2x1", // Remember to change
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

func MakeProxyContainer(pw *pathwaysjob.PathwaysJob, rmJobName string) (*corev1.Container, error) {
	// Some predefined variables
	truth := true

	proxyContainerSpec := corev1.Container{
		Name:            "pathways-proxy",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/proxy_server:latest",
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

// Constructs JobSet's replicated job for the Pathways worker
func MakeWorkerJob(ctx context.Context, pw *pathwaysjob.PathwaysJob, rmJobName string) (jobsetv1alpha2.ReplicatedJob, error) {
	// Some predefined variables
	truth := true
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	logx := ctrl.LoggerFrom(ctx)
	logx.Info("************************* PathwaysJob MakeWorkerJob ", "Number of jobs", pw.Spec.Workers[0].NumSlices)

	workerJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "worker",
		Replicas: int32(pw.Spec.Workers[0].NumSlices),
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: ptr.To(int32(4)),
				Completions:  ptr.To(int32(1)), // number of workers remember to change
				Parallelism:  ptr.To(int32(1)), // number of workers  remember to change
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "pathways-worker",
								Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
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
							"cloud.google.com/gke-tpu-accelerator": pw.Spec.Workers[0].Type,
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

func GetUserContainerList(pw *pathwaysjob.PathwaysJob) ([]corev1.Container, error) {
	containerList := pw.Spec.Controller.UserPodTemplate.Spec.Containers
	return containerList, nil
}

func MakeLeaderJobForColocatedDeployment(ctx context.Context, pw *pathwaysjob.PathwaysJob, rmJobName string) ([]jobsetv1alpha2.ReplicatedJob, error) {
	// Some predefined variables
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	RMContainerSpec, _ := MakeResourceManagerContainer(pw, rmJobName)
	ProxyContainerSpec, _ := MakeProxyContainer(pw, rmJobName)
	affinitySpec, _ := MakePodAffinityRules(pw)
	containerList, _ := GetUserContainerList(pw)
	containerList = append(containerList, *RMContainerSpec, *ProxyContainerSpec)

	// log3 := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	// log3.Info("PathwaysJob:...MakeLeaderJob", "Length of container list is", len(containerList))

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
							"cloud.google.com/gke-tpu-accelerator": pw.Spec.Workers[0].Type,
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

func MakeJobsForDefaultDeployment(ctx context.Context, pw *pathwaysjob.PathwaysJob, rmJobName string) ([]jobsetv1alpha2.ReplicatedJob, error) {
	// Some predefined variables
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	RMContainerSpec, _ := MakeResourceManagerContainer(pw, rmJobName)
	ProxyContainerSpec, _ := MakeProxyContainer(pw, rmJobName)

	// log3 := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	// log3.Info("PathwaysJob:...MakeLeaderJob", "Length of container list is", len(containerList))

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

	userJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "user-job",
		Replicas: 1,
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: ptr.To(int32(0)),
				Completions:  ptr.To(int32(1)),
				Parallelism:  ptr.To(int32(1)),
				Template: corev1.PodTemplateSpec{
					Spec: pw.Spec.Controller.UserPodTemplate.Spec,
					// 	corev1.PodSpec{
					// 	HostNetwork: true,                              // For performance == McJAX
					// 	DNSPolicy:   corev1.DNSClusterFirstWithHostNet, // For performance == McJAX
					// 	Tolerations: []corev1.Toleration{
					// 		{
					// 			Key:      "google.com/tpu",
					// 			Operator: "Exists",
					// 			Effect:   "NoSchedule",
					// 		},
					// 	},
					// 	Volumes: []corev1.Volume{
					// 		{
					// 			Name: "shared-tmp",
					// 			VolumeSource: corev1.VolumeSource{
					// 				HostPath: &corev1.HostPathVolumeSource{
					// 					Path: "/tmp",
					// 					Type: &volumeSourceType,
					// 				},
					// 			},
					// 		},
					// 	}, // end Volumes
					// 	Containers: []corev1.Container{*RMContainerSpec}, // end leader []containers
					// }, // end PodSpec
				},
			},
		},
	} // end replicated Job

	return []jobsetv1alpha2.ReplicatedJob{rmJob, proxyJob, userJob}, nil
}

// ---------------------- PATHWAYS STATUS HELPERS --------------------------

// func calculatePathwaysComponentStatus(ctx context.Context, rjs *jobsetv1alpha2.ReplicatedJobStatus, totalJobs int32) (string, error) {
// 	// From the replicated Job Status struct, determine if this component is in one of
// 	// Pending - one of more jobs ready but not active.
// 	// Running - all jobs active.
// 	// Suspended - one or more jobs suspended.
// 	// Completed - all jobs succeeded.
// 	// Failed - one or more jobs failed.
// 	var currentStatus string
// 	if rjs.Failed > 0 {
// 		currentStatus = "Failed"
// 	} else if rjs.Succeeded > 0 {
// 		currentStatus = "Suspended"
// 	} else if rjs.Ready > 0 {
// 		currentStatus = "Pending"
// 	} else if rjs.Active == totalJobs {
// 		currentStatus = "Running"
// 	} else if rjs.Succeeded == totalJobs {
// 		currentStatus = "Completed"
// 	}

// 	return currentStatus, nil
// }

// func updatePathwaysControllerStatus(ctx context.Context, pw *pathwaysjob.PathwaysJob) error {
// 	// for colocate mode -
// 	// find leader replicated job
// 	// call calculatePathwaysComponentStatus
// 	// update status

// 	// for deafult + headless mode -
// 	// find rm and proxy replicated jobs
// 	// call calculatePathwaysComponentStatus
// 	// update status, combining both statuses

// 	// for deafult + container mode -
// 	// find rm and proxy replicated jobs
// 	// call calculatePathwaysComponentStatus
// 	// update status, combining three statuses

// }

// func updateWorkerStatus(ctx context.Context, js *jobsetv1alpha2.JobSet) error {
// 	// find worker replicated job, find parallelisms in worker replicated Job for the job count ,
// 	// compare with ReplicatedJobStatus
// 	// call updateWorkerSliceStatus
// 	// update status
// 	log2 := ctrl.LoggerFrom(ctx)
// 	workerReplicatedJobStatus := findReplicatedJobStatusByName(js, "worker")
// 	log2.Info("PathwaysJob: in updateWorkerStatus", "Name ", workerReplicatedJobStatus.Name, "Ready ", workerReplicatedJobStatus.Ready, "Succeeded ", workerReplicatedJobStatus.Succeeded, "Failed ", workerReplicatedJobStatus.Failed, "Active ", workerReplicatedJobStatus.Active, "Suspended ", workerReplicatedJobStatus.Suspended)
// 	return nil
// }

// func updateWorkerSliceStatus(ctx context.Context) error {
// 	// find worker job, find parallelisms in worker replicated Job for the job count ,
// 	// call calculatePathwaysComponentStatus
// 	// update status
// 	// JobSetSpec -> ReplicatedJobs -> Template -> JobSpec, JobStatus

// }

// func findReplicatedJobStatusByName(js *jobsetv1alpha2.JobSet, replicatedJobName string) *jobsetv1alpha2.ReplicatedJobStatus {
// 	for _, rjob := range js.Status.ReplicatedJobsStatus {
// 		if rjob.Name == replicatedJobName {
// 			return &rjob
// 		}
// 	}
// 	return nil // Replicated job not found
// }
