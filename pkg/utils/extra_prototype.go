// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

// ----------------RM AND PROXY SPEC----------------

// {
// 	Name:            "pathways-rm",
// 	Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/server:latest",
// 	ImagePullPolicy: "Always",
// 	SecurityContext: &corev1.SecurityContext{Privileged: &truth},
// 	Args: []string{
// 		"--alsologtostderr",
// 		"--pathways_server_port=38677",
// 		"--pathways_server_provides_devices=false",
// 		"--pathways_device_type=NONE",
// 		"--pathways_persistent_compilation_cache=false",
// 		"--pathways_compilation_mode=compile_at_worker",
// 		fmt.Sprintf("--pathways_tmp_dir_pattern=%s", pw.Spec.PathwaysDir),
// 		"--pathways_expected_instances=tpuv4:2x2x2",
// 	},
// 	Env: []corev1.EnvVar{
// 		{Name: "REPLICATED_JOB_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/replicatedjob-name']"}}},
// 		{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
// 		{Name: "HOST_ADDRESS", Value: fmt.Sprintf("%s-%s-0-0.%s", pwWorkloadName, "leader", pwWorkloadName)},
// 		{Name: "TPU_SKIP_MDS_QUERY", Value: "true"},
// 	},
// 	Ports: []corev1.ContainerPort{{ContainerPort: 38677}, {ContainerPort: 38678}},
// }, // end pathways-rm

// {
// 	Name:            "pathways-proxy",
// 	Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/proxy_server:latest",
// 	ImagePullPolicy: "Always",
// 	SecurityContext: &corev1.SecurityContext{Privileged: &truth},
// 	Args: []string{
// 		"--alsologtostderr",
// 		"--v=0",
// 		fmt.Sprintf("--pathways_ifrt_proxy_server_resource_manager=%s-%s-0-0.%s:38677", pwWorkloadName, "leader", pwWorkloadName),
// 		"--pathways_ifrt_proxy_server_port=38681",
// 		fmt.Sprintf("--pathways_tmp_dir_pattern=%s", pw.Spec.PathwaysDir),
// 		"--pathways_plaque_network=gcp",
// 	},
// 	Ports: []corev1.ContainerPort{{ContainerPort: 38681}, {ContainerPort: 38682}},
// }, // end pathways-proxy

// NodeSelector: map[string]string{
// 	"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice",
// 	"cloud.google.com/gke-tpu-topology":    "2x2x2"},
// NodeSelector: map[string]string{"cloud.google.com/gke-tpu-accelerator": "tpu-v5-lite-podslice", "cloud.google.com/gke-tpu-topology": "4x4"},

// ----------------jetstream and tester containers----------------
// {
// 	Name:            "jetstream",
// 	Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/maxtext_jax_stable:latest", // revert to stable
// 	ImagePullPolicy: "Always",
// 	SecurityContext: &corev1.SecurityContext{Privileged: &truth},
// 	Env: []corev1.EnvVar{
// 		{Name: "XCLOUD_ENVIRONMENT", Value: "GCP"},
// 		{Name: "JAX_PLATFORMS", Value: "proxy"},
// 		{Name: "JAX_BACKEND_TARGET", Value: fmt.Sprintf("grpc://%s-%s-0-0.%s:38681", pw.GetName(), "leader", pw.GetName())},
// 		{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
// 	},
// 	Ports:   []corev1.ContainerPort{{ContainerPort: 9000}},
// 	Command: []string{"bash", "-c", "echo Start ; (JAX_TRACEBACK_FILTERING=off python3 MaxText/maxengine_server.py MaxText/configs/inference_jetstream.yml tokenizer_path=assets/tokenizer.llama2 load_parameters_path=gs://runner-maxtext-logs/2024-05-07-23-34/unscanned_chkpt/checkpoints/0/items max_prefill_predict_length=1024 max_target_length=2048 async_checkpointing=false model_name='llama2-70b' steps=1 ici_fsdp_parallelism=1 ici_autoregressive_parallelism=-1 ici_tensor_parallelism=1 scan_layers=false weight_dtype=bfloat16 per_device_batch_size=2); echo End; sleep infinity;"},
// }, // end jetstream

// {
// 	Name:            "tester",
// 	Image:           "us-docker.pkg.dev/cloud-tpu-v2-images-dev/pathways/maxtext_jax_stable:latest", // revert to stable
// 	ImagePullPolicy: "Always",
// 	SecurityContext: &corev1.SecurityContext{Privileged: &truth},
// 	Command:         []string{"bash", "-c", "echo Start ;for i in {1..5}; do echo Sending request $i; python3 JetStream/jetstream/tools/requester.py --tokenizer assets/tokenizer.llama2 --max_tokens=16 --server=0.0.0.0 --text=\"why earth is round\"; EXIT_CODE=$?; echo Completed request; echo EXIT_CODE=$EXIT_CODE; if [[ $EXIT_CODE -ne 0 ]]; then break; fi; done; echo Last EXIT_CODE=$EXIT_CODE; echo End; sleep infinity;"},
// }, // end tester

//----------------LIST----------------

// List JobSets using client

// jsList, err := jobSetClient.JobsetV1alpha2().JobSets("default").List(ctx, metav1.ListOptions{})

// 	if err != nil {
// 		log.Info("Roshani, can list JobSets: ")
// 		for _, js := range jsList.Items {
// 			if js.ObjectMeta.Name == pw.Spec.WorkloadName {
// 				log.Info("Roshani, found JobSet: ", "JobSet name", pw.Spec.WorkloadName)
// 				return ctrl.Result{}, nil
// 				// Nothing to reconcile here.
// 			}
// 		}
// 	} else {
// 		log.Info("Roshani, error listing JobSets: ", "error ", err)
// 		return ctrl.Result{}, err
// 	}

//
// // JobSet list
// var jsList *jobsetv1alpha2.JobSetList
// jsList, err := jobSetClient.JobsetV1alpha2().JobSets("default").List(ctx, metav1.ListOptions{})
// if err != nil {
// 	log.Info("Roshani, can't list JobSets: ", "error ", err)
// 	return ctrl.Result{}, err
// } else {
// 	log.Info("Roshani, can list JobSets")
// 	for _, job := range jsList.Items {
// 		for _, condition := range job.Status.Conditions {
// 			log.Info("Roshani Jobset condtion", job.ObjectMeta.Name, condition.Type)
// 		}
// 		if job.ObjectMeta.Name == pw.GetName() &&
// 			(job.Status.Conditions[0].Type == string(jobsetv1alpha2.JobSetStartupPolicyCompleted) ||
// 				job.Status.Conditions[0].Type == string(jobsetv1alpha2.JobSetStartupPolicyInProgress)) {
// 			log.Info("Roshani, found JobSet ", "JobSet name", pw.GetName())
// 			log.Info("Roshani, nothing to reconcile here")
// 			return ctrl.Result{}, nil
// 			// Nothing to reconcile here.
// 		}
// 	}
// }

// Currently leading to race conditions ---.
// var pwList pathwaysjob.PathwaysJobList
// if err := r.List(ctx, &pwList, &client.ListOptions{}); err != nil {
// 	log.Error(err, "Roshani, failed to list Pathways")
// 	return ctrl.Result{}, err
// } else {
// 	log.Info("Roshani, successfully listed Pathways")
// 	for _, job := range pwList.Items {
// 		log.Info("ROSHANI", "Job name ", job.Spec.WorkloadName, "Pathways workload name ", pw.GetName())
// 		if job.Spec.WorkloadName == pw.GetName() {
// 			log.Info("Roshani, found Pathways, not creating workload: ", "JobSet name", pw.GetName())
// 			return ctrl.Result{}, nil
// 			// Nothing to reconcile here.
// 		}
// 	}
// }

// --------LIST childJobSets --------------
// func (r *PathwaysJobReconciler) listChildJobSets(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) ([]jobsetv1alpha2.JobSet, error) {
// 	log3 := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
// 	// ctx = ctrl.LoggerInto(ctx, log3)
// 	log3.Info("PathwaysJob: in listChildJobSets", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())

// 	var jsList *jobsetv1alpha2.JobSetList
// 	jsList, err := jobSetClient.JobsetV1alpha2().JobSets(pw.GetObjectMeta().GetNamespace()).List(ctx, metav1.ListOptions{})

// 	if err != nil {
// 		log3.Info("PathwaysJob: can't list JobSets: ", "error ", err)
// 		return nil, err
// 	}
// 	return jsList.Items, nil
// }

// report status

// // 3. Update the cluster - create update and delete other resources
// log.Info("PathwaysJob: creating JobSet \n")
// if err := r.createJobSet(ctx, pw, jobSetClient); err != nil {
// 	log.Error(err, "PathwaysJob: failed to create JobSet \n")
// 	return ctrl.Result{}, err
// }

// childJobSets, err := r.listChildJobSets(ctx, pw, jobSetClient)
// if err != nil {
// 	log.Error(err, "PathwaysJob: failed to list JobSets \n")
// 	return ctrl.Result{}, err
// }

// // 2.1.1 List childJobSets
// for _, jobset := range childJobSets {
// 	if jobset.GetName() == pw.GetName() {
// 		log.Info("PathwaysJob: JobSet exists, not creating \n\n\n")
// 		for _, c := range jobset.Status.Conditions {
// 			log.Info("PathwaysJob: Condition is ", "Type", c.Type)
// 		}
// 	}
// }

// function to listChildJobSets, based on https://github.com/kubernetes-sigs/jobset/blob/main/client-go/clientset/versioned/typed/jobset/v1alpha2/jobset.go#L44

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
