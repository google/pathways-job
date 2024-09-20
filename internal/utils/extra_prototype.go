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
