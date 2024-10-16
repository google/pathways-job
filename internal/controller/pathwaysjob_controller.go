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

	// 2. Process the object

	kubeconfig := ctrl.GetConfigOrDie()
	// log.Info("PathwaysJob: config established...")

	jobSetClient := jobsetclient.NewForConfigOrDie(kubeconfig)
	// log.Info("PathwaysJob: client built for JobSet...")

	// 2.1 Figure out if PathwaysJob is already present and in "Suspended / Completed / Failed states",
	// if it is the case, there is nothing to do.

	childJobSets, err := r.listChildJobSets(ctx, pw, jobSetClient)
	if err != nil {
		log.Error(err, "PathwaysJob: failed to list JobSets \n")
		return ctrl.Result{}, err
	}

	// 2.1.1 List childJobSets
	for _, jobset := range childJobSets {
		if jobset.GetName() == pw.GetName() {
			log.Info("PathwaysJob: JobSet exists, not creating \n\n\n")
			for _, c := range jobset.Status.Conditions {
				log.Info("PathwaysJob: Condition is ", "Type", c.Type)
			}
		} else {
			// 3. Update the cluster - create update and delete other resources
			log.Info("PathwaysJob: creating JobSet \n")
			if err := r.createJobSet(ctx, pw, jobSetClient); err != nil {
				log.Error(err, "PathwaysJob: failed to create JobSet \n")
				return ctrl.Result{}, err
			}
		}
	}
	// report status

	// // 3. Update the cluster - create update and delete other resources
	// log.Info("PathwaysJob: creating JobSet \n")
	// if err := r.createJobSet(ctx, pw, jobSetClient); err != nil {
	// 	log.Error(err, "PathwaysJob: failed to create JobSet \n")
	// 	return ctrl.Result{}, err
	// }

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

func (r *PathwaysJobReconciler) listChildJobSets(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) ([]jobsetv1alpha2.JobSet, error) {
	log3 := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	// ctx = ctrl.LoggerInto(ctx, log3)
	log3.Info("PathwaysJob: in listChildJobSets", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())

	var jsList *jobsetv1alpha2.JobSetList
	jsList, err := jobSetClient.JobsetV1alpha2().JobSets(pw.GetObjectMeta().GetNamespace()).List(ctx, metav1.ListOptions{})

	if err != nil {
		log3.Info("PathwaysJob: can't list JobSets: ", "error ", err)
		return nil, err
	}
	return jsList.Items, nil
}

func (r *PathwaysJobReconciler) createJobSet(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) error {
	log2 := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	// ctx = ctrl.LoggerInto(ctx, log2)

	log2.Info("PathwaysJob: in createJobSet", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())

	// Some predefined variables
	truth := true
	volumeSourceType := corev1.HostPathDirectoryOrCreate

	//		// Pathways Spec + JobSet for batch inference ------
	leaderJob, _ := MakeLeaderJob(ctx, pw)

	mainJobSetConfig := jobsetv1alpha2.JobSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pw.GetName(),
			Namespace: pw.GetNamespace(),
		},
		Spec: jobsetv1alpha2.JobSetSpec{
			FailurePolicy: &jobsetv1alpha2.FailurePolicy{
				MaxRestarts: 4,
			},
			ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
				*leaderJob,
				{
					Name:     "worker",
					Replicas: int32(pw.Spec.Workers[0].NumSlices),
					Template: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							BackoffLimit: ptr.To(int32(0)),
							Completions:  ptr.To(int32(2)), // remember to update
							Parallelism:  ptr.To(int32(2)), // remember to update
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:            "pathways-worker",
											Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
											ImagePullPolicy: "Always",
											SecurityContext: &corev1.SecurityContext{Privileged: &truth},
											Args: []string{
												"--server_port=38679",
												fmt.Sprintf("--resource_manager_address=%s-%s-0-0.%s:38677", pw.GetName(), "leader", pw.GetName()),
												fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
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
				}, // end worker replicated job
			},
		},
	}
	// var lock sync.Mutex

	// Set Pathways controller as the owner of the JobSet for garbage collection.
	if err := ctrl.SetControllerReference(pw, &mainJobSetConfig, r.Scheme); err != nil {
		log2.Info("PathwaysJob: failed to set Pathways as owner of JobSet.", "error ", err)
	} else {
		log2.Info("PathwaysJob: successfully set Pathways as owner of JobSet.")
	}

	js, err := jobSetClient.JobsetV1alpha2().JobSets(pw.GetObjectMeta().GetNamespace()).Create(ctx, &mainJobSetConfig, metav1.CreateOptions{})

	if err != nil {
		log2.Info("PathwaysJob: failed to create JobSet: ", "JobSet name", js.Name)
		return err
	} else {
		log2.Info("PathwaysJob: successfully created JobSet: ", "JobSet name", js.Name)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PathwaysJobReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// if err := jobsetv1alpha2.AddToScheme(mgr.GetScheme()); err != nil {
	// 	return err
	// }

	return ctrl.NewControllerManagedBy(mgr).
		For(&pathwaysjob.PathwaysJob{}).
		// Owns(&jobsetv1alpha2.JobSet{}). // For JobSet
		Complete(r)
}

// Some Pathways helpers

func MakeResourceManagerContainer(pw *pathwaysjob.PathwaysJob) (*corev1.Container, error) {
	truth := true
	rmContainerSpec := corev1.Container{
		Name:            "pathways-rm",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{Privileged: &truth},
		Args: []string{
			"--server_port=38677",
			fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
			"--node_type=resource_manager",
			fmt.Sprintf("--instance_count=%d", int32(pw.Spec.Workers[0].NumSlices)),
			"--instance_type=tpuv4:2x2x2", // Change
		},
		Env: []corev1.EnvVar{
			{Name: "REPLICATED_JOB_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/replicatedjob-name']"}}},
			{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
			{Name: "HOST_ADDRESS", Value: fmt.Sprintf("%s-%s-0-0.%s", pw.GetName(), "leader", pw.GetName())},
			{Name: "TPU_SKIP_MDS_QUERY", Value: "true"},
		},
		Ports: []corev1.ContainerPort{{ContainerPort: 38677}, {ContainerPort: 38678}},
		// Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(8000000000, resource.DecimalSI)}},
	}
	return &rmContainerSpec, nil
}

func MakeProxyContainer(pw *pathwaysjob.PathwaysJob) (*corev1.Container, error) {
	truth := true
	proxyContainerSpec := corev1.Container{
		Name:            "pathways-proxy",
		Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/proxy_server:latest",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{Privileged: &truth},
		Args: []string{
			"--server_port=38681",
			fmt.Sprintf("--resource_manager_address=%s-%s-0-0.%s:38677", pw.GetName(), "leader", pw.GetName()),
			fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
		},
		Ports: []corev1.ContainerPort{{ContainerPort: 38681}, {ContainerPort: 38682}},
		// Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(10000000000, resource.DecimalSI)}},
	}
	return &proxyContainerSpec, nil
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
	containerList := pw.Spec.UserPodTemplate.Spec.Containers
	return containerList, nil
}

func MakeLeaderJob(ctx context.Context, pw *pathwaysjob.PathwaysJob) (*jobsetv1alpha2.ReplicatedJob, error) {
	// truth := true
	volumeSourceType := corev1.HostPathDirectoryOrCreate
	RMContainerSpec, _ := MakeResourceManagerContainer(pw)
	ProxyContainerSpec, _ := MakeProxyContainer(pw)
	affinitySpec, _ := MakePodAffinityRules(pw)
	containerList, _ := GetUserContainerList(pw)
	containerList = append(containerList, *RMContainerSpec, *ProxyContainerSpec)

	// log3 := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	// log3.Info("PathwaysJob:...MakeLeaderJob", "Length of container list is", len(containerList))

	leaderJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "leader",
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
	return &leaderJob, nil
}
