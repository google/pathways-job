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

	pathwaysapi "pathways-api/api/v1"
)

// PathwaysAPIReconciler reconciles a PathwaysAPI object
type PathwaysAPIReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PathwaysAPI object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile

// +kubebuilder:rbac:groups="",resources=events,verbs=create;watch;update;patch
// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis/finalizers,verbs=update
// +kubebuilder:rbac:groups=jobset.x-k8s.io,resources=jobsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jobset.x-k8s.io,resources=jobsets/status,verbs=get;update;patch
func (r *PathwaysAPIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := &pathwaysapi.PathwaysAPI{}
	log := ctrl.LoggerFrom(ctx).WithValues("pathwaysapi", klog.KObj(pw))
	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("ROSHANI CONTROLLER WORKING...", "WorkloadName ", pw.Spec.WorkloadName, " NumSlices ", pw.Spec.NumSlices, "WorkerNodeSelector", pw.Spec.PathwaysWorkerNodeSelector)

	// 1. Fetch the Pathways object
	// if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, pw); err != nil {
	if err := r.Get(ctx, req.NamespacedName, pw); err != nil {
		log.Info("Unable to fetch Pathways ")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Process the object

	kubeconfig := ctrl.GetConfigOrDie()
	log.Info("Roshani, config established...")

	jobSetClient := jobsetclient.NewForConfigOrDie(kubeconfig)
	log.Info("Roshani, client built for JobSet...")

	// 2.1 Figure out if PathwaysJob is already present and in "Suspended / Completed / Failed states",
	// if it is the case, there is nothing to do.

	// JobSet list
	var jsList *jobsetv1alpha2.JobSetList

	jsList, err := jobSetClient.JobsetV1alpha2().JobSets("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Info("Roshani, can't list JobSets: ", "error ", err)
		return ctrl.Result{}, err
	} else {
		log.Info("Roshani, can list JobSets")
		for _, job := range jsList.Items {
			for _, condition := range job.Status.Conditions {
				log.Info("Roshani Jobset condtion", job.ObjectMeta.Name, condition.Type)
			}
			if job.ObjectMeta.Name == pw.Spec.WorkloadName &&
				(job.Status.Conditions[0].Type == string(jobsetv1alpha2.JobSetStartupPolicyCompleted) ||
					job.Status.Conditions[0].Type == string(jobsetv1alpha2.JobSetStartupPolicyInProgress)) {
				log.Info("Roshani, found JobSet ", "JobSet name", pw.Spec.WorkloadName)
				log.Info("Roshani, nothing to reconcile here")
				return ctrl.Result{}, nil
				// Nothing to reconcile here.
			}
		}
	}

	// Currently leading to race conditions ---.
	// var pwList pathwaysapi.PathwaysAPIList
	// if err := r.List(ctx, &pwList, &client.ListOptions{}); err != nil {
	// 	log.Error(err, "Roshani, failed to list Pathways")
	// 	return ctrl.Result{}, err
	// } else {
	// 	log.Info("Roshani, successfully listed Pathways")
	// 	for _, job := range pwList.Items {
	// 		log.Info("ROSHANI", "Job name ", job.Spec.WorkloadName, "Pathways workload name ", pw.Spec.WorkloadName)
	// 		if job.Spec.WorkloadName == pw.Spec.WorkloadName {
	// 			log.Info("Roshani, found Pathways, not creating workload: ", "JobSet name", pw.Spec.WorkloadName)
	// 			return ctrl.Result{}, nil
	// 			// Nothing to reconcile here.
	// 		}
	// 	}
	// }

	// 3. Update the cluster - create update and delete other resources
	if err := r.createJobSet(ctx, pw, jobSetClient); err != nil {
		log.Error(err, "Roshani, failed to create JobSet \n")
		return ctrl.Result{}, err
	} else {
		log.Info("Roshani, successfully created JobSet \n")
	}

	//4. Update the object's status using Conditions

	//5. Return a result
	return ctrl.Result{}, nil
}

// function to listChildJobSets, based on https://github.com/kubernetes-sigs/jobset/blob/main/client-go/clientset/versioned/typed/jobset/v1alpha2/jobset.go#L44

//
// function to updatePathwaysJob Status ~~ updateJobSetStatus. Pathways status is same as JobSet Status. This function will mainly update Conditions and Message.
// similar to https://github.com/kubernetes-sigs/jobset/blob/main/pkg/controllers/jobset_controller.go#L248
// JobSet conditions - https://github.com/kubernetes-sigs/jobset/blob/main/pkg/controllers/jobset_controller.go#L822

// function to suspendJobSet

// function to resumeJobSet

// function to deleteJobSet, based on https://github.com/kubernetes-sigs/jobset/blob/main/client-go/clientset/versioned/typed/jobset/v1alpha2/jobset.go#L41

// function isJobSetFinished reuse jobSetFinished

// funtion pathwaysJobFinished  (?)

// function setCondition and updateCondition

// function setPathwaysJobCompletedCondition

// function setPathwaysJobFailedCondition

// function setPathwaysJobSuspendedCondition

// function setPathwaysJobResumedCondition

func (r *PathwaysAPIReconciler) createJobSet(ctx context.Context, pw *pathwaysapi.PathwaysAPI, jobSetClient *jobsetclient.Clientset) error {
	// log := ctrl.LoggerFrom(ctx)
	log2 := ctrl.LoggerFrom(ctx).WithValues("pathwaysapi", klog.KObj(pw))
	ctx = ctrl.LoggerInto(ctx, log2)

	log2.Info("ROSHANI in createJobSet")

	// Some predefined variables
	truth := true
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	RMContainerSpec, _ := MakeResourceManagerContainer(pw)
	ProxyContainerSpec, _ := MakeProxyContainer(pw)
	affinitySpec, _ := MakePodAffinityRules(pw)

	//		// Pathways Spec + JobSet for batch inference ------

	mainJobSetConfig := jobsetv1alpha2.JobSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: pw.Spec.WorkloadName,
		},
		Spec: jobsetv1alpha2.JobSetSpec{
			FailurePolicy: &jobsetv1alpha2.FailurePolicy{
				MaxRestarts: 4,
			},
			ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
				{
					Name:     "leader",
					Replicas: 1,
					Template: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							BackoffLimit: ptr.To(int32(0)),
							Completions:  ptr.To(int32(1)),
							Parallelism:  ptr.To(int32(1)),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Affinity:     affinitySpec,
									NodeSelector: pw.Spec.PathwaysControllerNodeSelector,
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
									Containers: []corev1.Container{
										*RMContainerSpec,
										*ProxyContainerSpec,
										{
											Name:            "jetstream",
											Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/maxtext_jax_stable:latest", // revert to stable
											ImagePullPolicy: "Always",
											SecurityContext: &corev1.SecurityContext{Privileged: &truth},
											Env: []corev1.EnvVar{
												{Name: "XCLOUD_ENVIRONMENT", Value: "GCP"},
												{Name: "JAX_PLATFORMS", Value: "proxy"},
												{Name: "JAX_BACKEND_TARGET", Value: fmt.Sprintf("grpc://%s-%s-0-0.%s:38681", pw.Spec.WorkloadName, "leader", pw.Spec.WorkloadName)},
												{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
											},
											Ports:   []corev1.ContainerPort{{ContainerPort: 9000}},
											Command: []string{"bash", "-c", "echo Start ; (JAX_TRACEBACK_FILTERING=off python3 MaxText/maxengine_server.py MaxText/configs/inference_jetstream.yml tokenizer_path=assets/tokenizer.llama2 load_parameters_path=gs://runner-maxtext-logs/2024-05-07-23-34/unscanned_chkpt/checkpoints/0/items max_prefill_predict_length=1024 max_target_length=2048 async_checkpointing=false model_name='llama2-70b' steps=1 ici_fsdp_parallelism=1 ici_autoregressive_parallelism=-1 ici_tensor_parallelism=1 scan_layers=false weight_dtype=bfloat16 per_device_batch_size=2); echo End; sleep infinity;"},
										}, // end jetstream

										{
											Name:            "tester",
											Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/maxtext_jax_stable:latest", // revert to stable
											ImagePullPolicy: "Always",
											SecurityContext: &corev1.SecurityContext{Privileged: &truth},
											Command:         []string{"bash", "-c", "echo Start ;for i in {1..5}; do echo Sending request $i; python3 JetStream/jetstream/tools/requester.py --tokenizer assets/tokenizer.llama2 --max_tokens=16 --server=0.0.0.0 --text=\"why earth is round\"; EXIT_CODE=$?; echo Completed request; echo EXIT_CODE=$EXIT_CODE; if [[ $EXIT_CODE -ne 0 ]]; then break; fi; done; echo Last EXIT_CODE=$EXIT_CODE; echo End; sleep infinity;"},
										}, // end tester
									}, // end leader []containers
								}, // end PodSpec
							},
						},
					},
				}, // end replicated Job
				{
					Name:     "worker",
					Replicas: int32(pw.Spec.NumSlices),
					Template: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							BackoffLimit: ptr.To(int32(0)),
							Completions:  ptr.To(int32(2)),
							Parallelism:  ptr.To(int32(2)),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:            "pathways-worker",
											Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
											ImagePullPolicy: "Always",
											SecurityContext: &corev1.SecurityContext{Privileged: &truth},
											Args: []string{
												"--alsologtostderr",
												"--pathways_server_port=38679",
												fmt.Sprintf("--pathways_resource_manager=%s-%s-0-0.%s:38677", pw.Spec.WorkloadName, "leader", pw.Spec.WorkloadName),
												"--pathways_persistent_compilation_cache=false",
												"--pathways_compilation_mode=compile_at_worker",
												"--xla_tpu_enable_data_parallel_all_reduce_opt=true",
												"--xla_tpu_data_parallel_opt_different_sized_ops=true",
												"--xla_tpu_enable_async_collective_fusion=true",
												"--xla_tpu_enable_async_collective_fusion_fuse_all_gather=true",
												"--xla_tpu_enable_async_collective_fusion_multiple_steps=true",
												"--xla_tpu_overlap_compute_collective_tc=true",
												"--xla_enable_async_all_gather=true",
												fmt.Sprintf("--pathways_tmp_dir_pattern=%s", pw.Spec.PathwaysDir),
											},
											Env: []corev1.EnvVar{
												{Name: "TPU_MIN_LOG_LEVEL", Value: "0"},
												{Name: "TF_CPP_MIN_LOG_LEVEL", Value: "0"},
												{Name: "XCLOUD_ENVIRONMENT", Value: "GCP"},
											},
											Ports: []corev1.ContainerPort{{ContainerPort: 38679}, {ContainerPort: 38680}, {ContainerPort: 8471}, {ContainerPort: 8080}},
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "shared-tmp",
													MountPath: "/tmp",
												},
											},
											Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"google.com/tpu": *resource.NewQuantity(4, resource.DecimalSI)}},
										}, // end Pathways worker container
									},
									NodeSelector: pw.Spec.PathwaysWorkerNodeSelector,
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
								},
							},
						},
					},
				}, // end worker replicated job
			},
		},
	}
	// var lock sync.Mutex

	// Set Pathways controller as the owner of the JobSet for garbage collection.
	// if err := ctrl.SetControllerReference(pw, &mainJobSetConfig, r.Scheme); err != nil {
	// 	// lock.Lock()
	// 	// defer lock.Unlock()
	// 	log2.Info("Roshani, failed to set Pathways as owner of JobSet.", "error ", err) // - clearly not working
	// 	// return err
	// } else {
	// 	log2.Info("Roshani, successfully set Pathways as owner of JobSet.")
	// }

	js, err := jobSetClient.JobsetV1alpha2().JobSets("default").Create(ctx, &mainJobSetConfig, metav1.CreateOptions{})

	if err != nil {
		log2.Info("Roshani, failed to create JobSet: ", "JobSet name", js.Name)
		return err
	} else {
		log2.Info("Roshani, successfully created JobSet: ", "JobSet name", js.Name)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PathwaysAPIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pathwaysapi.PathwaysAPI{}).
		// Owns(&jobsetv1alpha2.JobSet{}). // For JobSet
		Complete(r)
}

// helpers

func MakeResourceManagerContainer(pw *pathwaysapi.PathwaysAPI) (*corev1.Container, error) {
	truth := true
	rmContainerSpec := corev1.Container{
		Name:            "pathways-rm",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{Privileged: &truth},
		Args: []string{
			"--alsologtostderr",
			"--pathways_server_port=38677",
			"--pathways_server_provides_devices=false",
			"--pathways_device_type=NONE",
			"--pathways_persistent_compilation_cache=false",
			"--pathways_compilation_mode=compile_at_worker",
			fmt.Sprintf("--pathways_tmp_dir_pattern=%s", pw.Spec.PathwaysDir),
			"--pathways_expected_instances=tpuv4:2x2x2", // CHANGE
		},
		Env: []corev1.EnvVar{
			{Name: "REPLICATED_JOB_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/replicatedjob-name']"}}},
			{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
			{Name: "HOST_ADDRESS", Value: fmt.Sprintf("%s-%s-0-0.%s", pw.Spec.WorkloadName, "leader", pw.Spec.WorkloadName)},
			{Name: "TPU_SKIP_MDS_QUERY", Value: "true"},
		},
		Ports:     []corev1.ContainerPort{{ContainerPort: 38677}, {ContainerPort: 38678}},
		Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(8000000000, resource.DecimalSI)}},
	}
	return &rmContainerSpec, nil
}

func MakeProxyContainer(pw *pathwaysapi.PathwaysAPI) (*corev1.Container, error) {
	truth := true
	proxyContainerSpec := corev1.Container{
		Name:            "pathways-proxy",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/proxy_server:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{Privileged: &truth},
		Args: []string{
			"--alsologtostderr",
			"--v=0",
			fmt.Sprintf("--pathways_ifrt_proxy_server_resource_manager=%s-%s-0-0.%s:38677", pw.Spec.WorkloadName, "leader", pw.Spec.WorkloadName),
			"--pathways_ifrt_proxy_server_port=38681",
			fmt.Sprintf("--pathways_tmp_dir_pattern=%s", pw.Spec.PathwaysDir),
			"--pathways_plaque_network=gcp",
		},
		Ports:     []corev1.ContainerPort{{ContainerPort: 38681}, {ContainerPort: 38682}},
		Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(10000000000, resource.DecimalSI)}},
	}
	return &proxyContainerSpec, nil
}

func MakePodAffinityRules(pw *pathwaysapi.PathwaysAPI) (*corev1.Affinity, error) {
	affinity := corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "jobset.sigs.k8s.io/jobset-name",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{pw.Spec.WorkloadName},
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
								Values:   []string{pw.Spec.WorkloadName},
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
