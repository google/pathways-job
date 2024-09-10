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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	jobsetclient "sigs.k8s.io/jobset/client-go/clientset/versioned"
	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
	lwsclient "sigs.k8s.io/lws/client-go/clientset/versioned"

	// jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	// jobsetclient "sigs.k8s.io/jobset/client-go/clientset/versioned"
	// leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
	// lwsclient "sigs.k8s.io/lws/client-go/clientset/versioned"

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

// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis/finalizers,verbs=update
// +kubebuilder:rbac:groups=leaderworkerset.x-k8s.io,resources=leaderworkersets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=leaderworkerset.x-k8s.io,resources=leaderworkersets/status,verbs=get;update;patch
func (r *PathwaysAPIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	pw := &pathwaysapi.PathwaysAPI{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, pw); err != nil {
		// log.Error(err, "unable to fetch Pathways ")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log := ctrl.LoggerFrom(ctx).WithValues("pathwaysapi", klog.KObj(pw))
	pwMessage := pw.Spec.WorkloadName
	tpuType := pw.Spec.TpuType
	numSlices := pw.Spec.NumSlices
	workloadMode := pw.Spec.WorkloadMode
	workloadType := pw.Spec.WorkloadType

	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("ROSHANI CONTROLLER WORKING...", "TextMessage", pwMessage, "TpuType", tpuType, "NumSlices", numSlices, "WorkloadMode", workloadMode)

	kubeconfig := ctrl.GetConfigOrDie()
	log.Info("Roshani, config established...")

	truth := true
	size := int32(5) // total number of workers (across all slices) + 1
	replicas := int32(1)
	fmt.Printf("Replicas: %d , Size: %d \n", replicas, size)

	client := lwsclient.NewForConfigOrDie(kubeconfig)
	log.Info("Roshani, client built...")
	if workloadType == "inference" {
		lws, err := client.LeaderworkersetV1().LeaderWorkerSets("default").Create(ctx, &leaderworkersetv1.LeaderWorkerSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        pwMessage,
				Annotations: map[string]string{"leaderworkerset.sigs.k8s.io/exclusive-topology": "cloud.google.com/gke-nodepool"},
			},
			Spec: leaderworkersetv1.LeaderWorkerSetSpec{
				Replicas:      ptr.To(replicas),
				StartupPolicy: "LeaderCreated", // this seems to be a mandatory field now
				LeaderWorkerTemplate: leaderworkersetv1.LeaderWorkerTemplate{
					Size: ptr.To(size),
					LeaderTemplate: &corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:   workloadType,
							Labels: map[string]string{"xpk.google.com/workload": "pathways-headless"},
						},
						Spec: corev1.PodSpec{
							// NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v5-lite-podslice", "cloud.google.com/gke-tpu-topology": "4x4"},
							// NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice", "cloud.google.com/gke-tpu-topology": "2x2x1"},
							NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice", "cloud.google.com/gke-tpu-topology": "2x2x2"},
							Tolerations: []corev1.Toleration{
								{
									Key:      "google.com/tpu",
									Operator: "Exists",
									Effect:   "NoSchedule",
								},
							},
							Containers: []corev1.Container{
								{
									Name:            "pathways-proxy",
									Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/proxy_server:latest",
									ImagePullPolicy: "Always",
									SecurityContext: &corev1.SecurityContext{Privileged: &truth},
									Args:            []string{"--alsologtostderr", "--v=0", "--pathways_ifrt_proxy_server_resource_manager=$(LWS_LEADER_ADDRESS):38677", "--pathways_ifrt_proxy_server_port=38681", "--pathways_tmp_dir_pattern=gs://cloud-pathways-staging/tmp", "--pathways_plaque_network=gcp"},
									Ports:           []corev1.ContainerPort{{ContainerPort: 38681}, {ContainerPort: 38682}},
									// Resources: []corev1.ResourceRequirements{
									// 	{Limits: corev1.ResourceList{{cpu: "24", memory: 100G,},},
									// 	},
									// },
								},
								{
									Name:            "pathways-rm",
									Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
									ImagePullPolicy: "Always",
									SecurityContext: &corev1.SecurityContext{Privileged: &truth},
									Env:             []corev1.EnvVar{{Name: "HOST_ADDRESS", Value: "$(LWS_LEADER_ADDRESS)"}, {Name: "TPU_SKIP_MDS_QUERY", Value: "true"}},
									Args: []string{"--pathways_server_port=38677",
										"--pathways_server_provides_devices=false",
										"--pathways_device_type=NONE",
										"--pathways_persistent_compilation_cache=false",
										"--pathways_compilation_mode=compile_at_worker",
										"--pathways_tmp_dir_pattern=gs://cloud-pathways-staging/tmp",
										"--pathways_resource_manager_expected_num_worker_jobs=2"},
									Ports: []corev1.ContainerPort{{ContainerPort: 38677}, {ContainerPort: 38678}},
									// Resources: []corev1.ResourceRequirements{Limits: {map[corev1.ResourceName]resource.Quantity{cpu: "24", memory: 100G,},},},
								},
							},
						},
					},
					WorkerTemplate: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:   workloadType,
							Labels: map[string]string{"xpk.google.com/workload": "pathways-headless"},
						},
						Spec: corev1.PodSpec{
							// NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v5-lite-podslice", "cloud.google.com/gke-tpu-topology": "4x4"},
							// NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice", "cloud.google.com/gke-tpu-topology": "2x2x1"},
							Tolerations: []corev1.Toleration{
								{
									Key:      "google.com/tpu",
									Operator: "Exists",
									Effect:   "NoSchedule",
								},
							},
							NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice", "cloud.google.com/gke-tpu-topology": "2x2x2"},
							Containers: []corev1.Container{
								{
									Name:            "worker",
									Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
									ImagePullPolicy: "Always",
									SecurityContext: &corev1.SecurityContext{Privileged: &truth},
									Args: []string{"--alsologtostderr", "--pathways_server_port=38679", "--pathways_resource_manager=$(LWS_LEADER_ADDRESS):38677", "--pathways_persistent_compilation_cache=false", "--pathways_compilation_mode=compile_at_worker",
										"--xla_tpu_enable_data_parallel_all_reduce_opt=true", "--xla_tpu_data_parallel_opt_different_sized_ops=true", "--xla_tpu_enable_async_collective_fusion=true", "--xla_tpu_enable_async_collective_fusion_fuse_all_gather=true",
										"--xla_tpu_enable_async_collective_fusion_multiple_steps=true", "--xla_tpu_overlap_compute_collective_tc=true", "--xla_enable_async_all_gather=true", "--pathways_tmp_dir_pattern=gs://cloud-pathways-staging/tmp"},
									Ports: []corev1.ContainerPort{{ContainerPort: 38679}, {ContainerPort: 38680}, {ContainerPort: 8471}, {ContainerPort: 8080}},
									// Resources: []corev1.ResourceRequirements{Limits: {map[corev1.ResourceName]resource.Quantity{cpu: "24", memory: 100G,},},},
								},
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})

		// Pathways Spec + LWS ------

		// lws, err := client.LeaderworkersetV1().LeaderWorkerSets("default").Create(ctx, &leaderworkersetv1.LeaderWorkerSet{
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name: pwMessage,
		// 	},
		// 	Spec: leaderworkersetv1.LeaderWorkerSetSpec{
		// 		Replicas: ptr.To(numSlices),
		// 		LeaderWorkerTemplate: leaderworkersetv1.LeaderWorkerTemplate{
		// 			LeaderTemplate: &corev1.PodTemplateSpec{
		// 				ObjectMeta: metav1.ObjectMeta{
		// 					Name: workloadType,
		// 				},
		// 				Spec: corev1.PodSpec{
		// 					Containers: []corev1.Container{
		// 						{
		// 							Name:    "bash-container",
		// 							Image:   "bash:latest",
		// 							Command: []string{"/bin/sh"},
		// 							Args:    []string{"-c", "while true; do echo hello; sleep 10; done"},
		// 						},
		// 					},
		// 					// RestartPolicy: "Never",
		// 				},
		// 			},
		// 			WorkerTemplate: corev1.PodTemplateSpec{
		// 				ObjectMeta: metav1.ObjectMeta{
		// 					Name: "workers",
		// 				},
		// 				Spec: corev1.PodSpec{
		// 					Containers: []corev1.Container{
		// 						{
		// 							Name:    "bash-container",
		// 							Image:   "bash:latest",
		// 							Command: []string{"/bin/sh"},
		// 							Args:    []string{"-c", "while true; do echo hello; sleep 10; done"},
		// 						},
		// 					},
		// 					// RestartPolicy: "Never",
		// 				},
		// 			},
		// 			Size: ptr.To(int32(2)),
		// 		},
		// 		StartupPolicy: "LeaderReady",
		// 	},
		// }, metav1.CreateOptions{})

		if err != nil {
			panic(err)
		}
		log.Info("Roshani, created LeaderWorkerSet...")
		fmt.Printf("successfully created LeaderWorkerSet: %s\n", lws.Name)
	} else if workloadType == "training" {
		//		// Pathways Spec + JobSet ------
		client := jobsetclient.NewForConfigOrDie(kubeconfig)

		js, err := client.JobsetV1alpha2().JobSets("default").Create(ctx, &jobsetv1alpha2.JobSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: pwMessage,
			},
			Spec: jobsetv1alpha2.JobSetSpec{
				ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
					{
						Name:     "worker",
						Replicas: 2,
						Template: batchv1.JobTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{"alpha.jobset.sigs.k8s.io/exclusive-topology": "cloud.google.com/gke-nodepool"},
							},
							Spec: batchv1.JobSpec{
								BackoffLimit: ptr.To(int32(0)),
								Completions:  ptr.To(int32(1)),
								Parallelism:  ptr.To(int32(1)),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										TerminationGracePeriodSeconds: ptr.To(int64(30)),
										Containers: []corev1.Container{
											{
												Name:            "pathways-worker",
												Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
												ImagePullPolicy: "Always",
												SecurityContext: &corev1.SecurityContext{Privileged: &truth},
												Args: []string{
													"--alsologtostderr",
													"--pathways_server_port=38677",
													fmt.Sprintf("--pathways_resource_manager=%s-rm-0-0.%s:38677", pwMessage, pwMessage),
													"--pathways_persistent_compilation_cache=false",
													"--pathways_compilation_mode=compile_at_worker",
													"--xla_tpu_enable_data_parallel_all_reduce_opt=true",
													"--xla_tpu_data_parallel_opt_different_sized_ops=true",
													"--xla_tpu_enable_async_collective_fusion=true",
													"--xla_tpu_enable_async_collective_fusion_fuse_all_gather=true",
													"--xla_tpu_enable_async_collective_fusion_multiple_steps=true",
													"--xla_tpu_overlap_compute_collective_tc=true",
													"--xla_enable_async_all_gather=true",
													"--pathways_tmp_dir_pattern=gs://cloud-pathways-staging/tmp",
												},
												Ports: []corev1.ContainerPort{{ContainerPort: 38677}, {ContainerPort: 8471}, {ContainerPort: 8080}},
												VolumeMounts: []corev1.VolumeMount{
													{
														Name:      "shared-tmp",
														MountPath: "/tmp",
													},
												},
												// Resources: corev1.ResourceRequirements{
												// 	Limits: {map[corev1.ResourceName]Res{"google.com/tpu", 4},
												// },
												// Resources: corev1.ResourceRequirements{Limits: {map[corev1.ResourceName]{cpu: "24", memory: 100G,},},},
											},
										},
										NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v5-lite-podslice", "cloud.google.com/gke-tpu-topology": "4x4"},
										// Volumes: []corev1.Volume{
										// {Name: "shared-tmp", VolumeSource: corev1.VolumeSource{HostPath: *corev1.HostPathVolumeSource{Path: "/tmp", Type: *corev1.HostPathType("DirectoryOrCreate")}}},
										// },
									},
								},
							},
						},
					},
					{
						Name:     "rm",
						Replicas: 1,
						Template: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								BackoffLimit: ptr.To(int32(0)),
								Completions:  ptr.To(int32(1)),
								Parallelism:  ptr.To(int32(1)),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										// TerminationGracePeriodSeconds: ptr.To(int64(30)),
										Containers: []corev1.Container{
											{
												Name:            "pathways-rm",
												Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
												ImagePullPolicy: "Always",
												SecurityContext: &corev1.SecurityContext{Privileged: &truth},
												Args: []string{
													"--alsologtostderr",
													"--pathways_server_port=38677",
													"--pathways_server_provides_devices=false",
													"--pathways_persistent_compilation_cache=false",
													"--pathways_compilation_mode=compile_at_worker",
													"--pathways_tmp_dir_pattern=gs://cloud-pathways-staging/tmp",
													"--pathways_expected_instances=tpuv4:2x2x2,tpuv4:2x2x2",
												},
												Env: []corev1.EnvVar{
													{Name: "REPLICATED_JOB_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/replicatedjob-name']"}}},
													{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
													{Name: "HOST_ADDRESS", Value: fmt.Sprintf("%s-%s-0-0.%s", pwMessage, "rm", pwMessage)},
													{Name: "TPU_SKIP_MDS_QUERY", Value: "true"},
												},
												Ports: []corev1.ContainerPort{{ContainerPort: 38677}},
												VolumeMounts: []corev1.VolumeMount{
													{
														Name:      "shared-tmp",
														MountPath: "/tmp",
													},
												},
												// Resources: corev1.ResourceRequirements{
												// 	Limits: {map[corev1.ResourceName]{"google.com/tpu", 4},
												// },
												// Resources: corev1.ResourceRequirements{Limits: {map[corev1.ResourceName]{cpu: "24", memory: 100G,},},},
											},
										},
										NodeSelector: map[string]string{"cloud.google.com/gke-nodepool": "cpu-rm-np"},
										// Volumes: []corev1.Volume{
										// {Name: "shared-tmp", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp", Type: &corev1.HostPathType("DirectoryOrCreate")}}},
										// },
									},
								},
							},
						},
					},
					{
						Name:     "proxy",
						Replicas: 1,
						Template: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								BackoffLimit: ptr.To(int32(0)),
								Completions:  ptr.To(int32(1)),
								Parallelism:  ptr.To(int32(1)),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										// TerminationGracePeriodSeconds: ptr.To(int64(30)),
										Containers: []corev1.Container{
											{
												Name:            "pathways-proxy",
												Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/proxy_server:latest",
												ImagePullPolicy: "Always",
												SecurityContext: &corev1.SecurityContext{Privileged: &truth},
												Args: []string{
													"--alsologtostderr",
													"--v=0",
													fmt.Sprintf("--pathways_ifrt_proxy_server_resource_manager=%s-%s-0-0.%s:38677", pwMessage, "rm", pwMessage),
													"--pathways_ifrt_server_port=38676",
													"--pathways_tmp_dir_pattern=gs://cloud-pathways-staging/tmp",
													"--pathways_plaque_network=gcp",
												},
												Ports: []corev1.ContainerPort{{ContainerPort: 38676}},
												VolumeMounts: []corev1.VolumeMount{
													{
														Name:      "shared-tmp",
														MountPath: "/tmp",
													},
												},
												// Resources: corev1.ResourceRequirements{
												// 	Limits: corev1.ResourceList{"google.com/tpu": resource.Quantity{i: 4}},
												// },
												// Resources: corev1.ResourceRequirements{Limits: {map[corev1.ResourceName]{cpu: "24", memory: 100G,},},},
											},
										},
										NodeSelector: map[string]string{"cloud.google.com/gke-nodepool": "cpu-proxy-np"},
										// Volumes: []corev1.Volume{
										// 	{Name: "shared-tmp", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp", Type: &corev1.HostPathDirectoryOrCreate}}},
										// },
									},
								},
							},
						},
					},
					{
						Name:     "main",
						Replicas: 1,
						Template: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								BackoffLimit: ptr.To(int32(0)),
								Completions:  ptr.To(int32(1)),
								Parallelism:  ptr.To(int32(1)),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										// TerminationGracePeriodSeconds: ptr.To(int64(30)),
										Containers: []corev1.Container{
											{
												Name:            "maxtext",
												Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/maxtext_jax_stable:latest",
												ImagePullPolicy: "Always",
												SecurityContext: &corev1.SecurityContext{Privileged: &truth},
												Env: []corev1.EnvVar{
													{Name: "XCLOUD_ENVIRONMENT", Value: "GCP"},
													{Name: "JAX_PLATFORMS", Value: "proxy"},
													{Name: "JAX_BACKEND_TARGET", Value: fmt.Sprintf("grpc://%s-%s-0-0.%s:38676", pwMessage, "proxy", pwMessage)},
													{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
												},
												VolumeMounts: []corev1.VolumeMount{
													{
														Name:      "shared-tmp",
														MountPath: "/tmp",
													},
												},
												// Resources: corev1.ResourceRequirements{
												// 	Limits: corev1.ResourceList{"google.com/tpu": resource.Quantity{i: 4}},
												// },
												// Resources: corev1.ResourceRequirements{Limits: {map[corev1.ResourceName]{cpu: "24", memory: 100G,},},},
											},
										},
										NodeSelector: map[string]string{"cloud.google.com/gke-nodepool": "cpu-user-np"},
										// Volumes: []corev1.Volume{
										// 	{Name: "shared-tmp", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp", Type: &corev1.HostPathDirectoryOrCreate}}},
										// },
									},
								},
							},
						},
					},
				},
				SuccessPolicy: &jobsetv1alpha2.SuccessPolicy{
					Operator:             "All",
					TargetReplicatedJobs: []string{"main"},
				},
				FailurePolicy: &jobsetv1alpha2.FailurePolicy{
					MaxRestarts: 0,
				},
			},
		}, metav1.CreateOptions{})

		// js, err := client.JobsetV1alpha2().JobSets("default").Create(ctx, &jobsetv1alpha2.JobSet{
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name: pwMessage,
		// 	},
		// 	Spec: jobsetv1alpha2.JobSetSpec{
		// 		ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
		// 			{
		// 				Name: "rjob",
		// 				Template: batchv1.JobTemplateSpec{
		// 					ObjectMeta: metav1.ObjectMeta{
		// 						Name: "job",
		// 					},
		// 					Spec: batchv1.JobSpec{
		// 						Parallelism:  ptr.To(numSlices),
		// 						Completions:  ptr.To(numSlices),
		// 						BackoffLimit: ptr.To(int32(0)),
		// 						Template: corev1.PodTemplateSpec{
		// 							Spec: corev1.PodSpec{
		// 								Containers: []corev1.Container{
		// 									{
		// 										Name:    "bash-container",
		// 										Image:   "bash:latest",
		// 										Command: []string{"echo"},
		// 										Args:    []string{"Hello"},
		// 									},
		// 								},
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// }, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		log.Info("Roshani, created JobSet...")
		fmt.Printf("successfully created JobSet: %s\n", js.Name)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PathwaysAPIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pathwaysapi.PathwaysAPI{}).
		// For JobSet and LWS
		Complete(r)
}
