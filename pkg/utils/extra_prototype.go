// Copyright 2024 Google LLC
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

// RM AND PROXY SPEC-

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
// var pwList pathwaysapi.PathwaysAPIList
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
