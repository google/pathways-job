package utils

import (
	"fmt"

	pathwaysapi "pathways-api/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
