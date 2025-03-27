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
	"slices"
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
	InstanceType       string
	NumVMs             int32
	GKEAcceleratorType string
)

const (
	PathwaysHeadJobName             = "pathways-head"
	DefaultPathwaysRMAndWorkerImage = "us-docker.pkg.dev/cloud-tpu-v2-images/pathways/server"
	DefaultPathwaysProxyImage       = "us-docker.pkg.dev/cloud-tpu-v2-images/pathways/proxy_server"
)

// Map to convert machine type to TPU version
var MachineTypeToTPUVersionMap = map[string]string{
	//v6e
	"ct6e-standard-4t": "tpuv6e",
	"ct6e-standard-8t": "tpuv6e1t",
	//v5p
	"ct5p-hightpu-4t": "tpuv5",
	//v5e
	"ct5lp-hightpu-4t": "tpuv5e",
	"ct5lp-hightpu-8t": "tpuv5e1t",
	//v4
	"ct4p-hightpu-4t": "tpuv4",
}

// Map to convert machine type to GKE Accelerator Type
var MachineTypeToGKEAcceleratorTypeMap = map[string]string{
	//v6e
	"ct6e-standard-4t": "tpu-v6e-slice",
	"ct6e-standard-8t": "tpu-v6e-slice",
	//v5p
	"ct5p-hightpu-4t": "tpu-v5p-slice",
	//v5e
	"ct5lp-hightpu-4t": "tpu-v5-lite-podslice",
	"ct5lp-hightpu-8t": "tpu-v5-lite-podslice",
	//v4
	"ct4p-hightpu-4t": "tpu-v4-podslice",
}

// Allowed topologies for each machine type
var ValidTpuTopologiesMap = map[string][]string{
	//v6e
	"ct6e-standard-4t": {"1x1", "2x2", "2x4", "4x4", "4x8", "8x8", "8x16", "16x16"},
	"ct6e-standard-8t": {"2x4"},
	//v5p
	"ct5p-hightpu-4t": {
		"2x2x1", "2x2x2", "2x2x4", "2x4x4", "4x4x4", "4x4x8", "4x4x12", "4x8x8",
		"4x4x20", "4x8x12", "4x4x28", "8x8x8", "4x12x12", "4x8x20", "4x4x44",
		"8x8x12", "4x4x52", "4x8x28", "4x12x20", "8x8x16", "4x4x68", "8x12x12",
		"4x4x76", "8x8x20", "4x12x28", "4x8x44", "4x4x92", "8x12x16", "4x20x20",
		"4x8x52", "12x12x12", "8x8x28", "4x4x116", "8x12x20", "4x4x124", "8x16x16",
		"4x12x44", "4x8x68", "4x20x28", "12x12x16", "4x4x148", "4x8x76", "4x12x52",
		"8x16x20", "4x4x164", "8x12x28", "4x4x172", "8x8x44", "12x12x20", "4x8x92",
		"4x4x188", "12x16x16", "4x28x28", "8x20x20", "4x12x68", "8x8x52", "4x4x212",
		"12x12x24", "4x20x44", "8x16x28", "4x12x76", "4x8x116", "4x4x236", "12x16x20",
		"4x4x244", "4x8x124", "12x12x28", "16x16x16", "4x20x52", "8x12x44", "8x8x68",
		"4x12x92", "8x20x28", "12x16x24", "4x8x148", "12x20x20", "8x8x76", "4x28x44",
		"8x12x52", "16x16x20", "12x12x36", "4x8x164", "12x16x28", "4x20x68", "4x8x172",
		"4x12x116", "8x16x44", "12x20x24", "4x28x52", "8x8x92", "4x12x124", "4x8x188",
		"4x20x76", "16x16x24", "12x24x24", "16x20x28",
	},
	//v5e
	"ct5lp-hightpu-4t": {
		"2x2", "4x4", "4x8", "8x8", "8x16", "16x16",
	},
	"ct5lp-hightpu-8t": {"2x4"},
	//v4
	"ct4p-hightpu-4t": {
		"2x2x1", "2x2x2", "2x2x4", "2x4x4", "4x4x4", "4x4x8", "4x8x8", "4x4x16", "8x8x8",
		"8x8x12", "8x8x16", "4x16x16", "8x16x16",
	},
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

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
	jobSetClient := jobsetclient.NewForConfigOrDie(kubeconfig)

	// 3. Figure out if PathwaysJob is already present.
	childJobSet, err := r.getChildJobSet(ctx, pw, jobSetClient)
	if err != nil {
		log.Info("PathwaysJob: can't find JobSet, may create one!")
	} else if childJobSet != nil {
		log.Info("PathwaysJob: JobSet exists, not creating")
		// 2.2 Find out JobSet's status and update the PathwaysJobStatus accordingly.
		r.setPathwaysJobStatusBasedOnJobSetStatus(ctx, pw, childJobSet)
		if pw.Status.TerminalState != "" {
			log.Info("PathwaysJob: DONE.")
		}
		return ctrl.Result{}, nil
	}

	// 3. Create a child JobSet in PathwaysJob.
	log.Info("PathwaysJob: creating JobSet \n")
	if err := r.createJobSet(ctx, pw, jobSetClient); err != nil {
		pw.Status.TerminalState = string(pathwaysjob.PathwaysJobFailed)
		r.Status().Update(ctx, pw)
		log.Error(err, "PathwaysJob: failed to create JobSet")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PathwaysJobReconciler) getChildJobSet(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) (*jobsetv1alpha2.JobSet, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	ctx = ctrl.LoggerInto(ctx, log)
	log.Info("PathwaysJob: in getChildJobSet", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())

	var js *jobsetv1alpha2.JobSet
	js, err := jobSetClient.JobsetV1alpha2().JobSets(pw.GetObjectMeta().GetNamespace()).Get(ctx, pw.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return js, nil
}

// Create the JobSet for 'colocate_head_with_workers' or 'default' deployment modes.
func (r *PathwaysJobReconciler) createJobSet(ctx context.Context, pw *pathwaysjob.PathwaysJob, jobSetClient *jobsetclient.Clientset) error {
	log := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("PathwaysJob: in createJobSet", "Name ", pw.GetName(), "Namespace ", pw.GetNamespace())
	var job jobsetv1alpha2.ReplicatedJob

	err := calculateTPUInfo(ctx, pw)
	if err != nil {
		log.Info("PathwaysJob: in createJobSet calculateTPUInfo ", " Error: ", err)
		return err
	} else {
		log.Info("PathwaysJob: in createJobSet calculateTPUInfo ", "InstanceType", InstanceType, "NumVMs", NumVMs)
	}
	// Pathways Spec + JobSet for training or batch inference ------
	if pw.Spec.Controller.DeploymentMode == pathwaysjob.ColocateHeadWithWorkers {
		job, _ = MakePathwaysHeadJobForColocateHeadWithWorkersDeployment(ctx, pw)
	} else {
		job, _ = MakePathwaysHeadJobForDefaultDeployment(ctx, pw)
	}

	workerJob, _ := MakeWorkerJob(ctx, pw)
	successPolicy := MakeSuccessPolicy(pw)

	mainJobSetConfig := jobsetv1alpha2.JobSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pw.GetName(),
			Namespace: pw.GetNamespace(),
		},
		Spec: jobsetv1alpha2.JobSetSpec{
			StartupPolicy: &jobsetv1alpha2.StartupPolicy{
				StartupPolicyOrder: jobsetv1alpha2.InOrder,
			}, // create jobs in the order specified in JobSet.
			FailurePolicy: &jobsetv1alpha2.FailurePolicy{
				MaxRestarts: pw.Spec.MaxRestarts,
			},
			SuccessPolicy:  successPolicy,
			ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{job, workerJob},
		},
	}

	// Set Pathways controller as the owner of the JobSet for garbage collection.
	if err := ctrl.SetControllerReference(pw, &mainJobSetConfig, r.Scheme); err != nil {
		log.Info("PathwaysJob: failed to set Pathways as owner of JobSet.", "error ", err)
	} else {
		log.Info("PathwaysJob: successfully set Pathways as owner of JobSet.")
	}

	js, err := jobSetClient.JobsetV1alpha2().JobSets(pw.GetObjectMeta().GetNamespace()).Create(ctx, &mainJobSetConfig, metav1.CreateOptions{})

	if err != nil {
		return err
	} else {
		log.Info("PathwaysJob: successfully created JobSet: ", "JobSet name", js.Name)
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

// ---------------------- PATHWAYSJOB HELPERS --------------------------

// Find the status of the underlying JobSet for 'colocated_head_with_workers' or 'default' deployment modes.
// Update the conditions within PathwaysJobStatus and persist the changes to the PathwaysJobStatus.
func (r *PathwaysJobReconciler) setPathwaysJobStatusBasedOnJobSetStatus(ctx context.Context, pw *pathwaysjob.PathwaysJob, js *jobsetv1alpha2.JobSet) {
	log := ctrl.LoggerFrom(ctx).WithValues("pathwaysjob", klog.KObj(pw))
	ctx = ctrl.LoggerInto(ctx, log)

	if len(js.Status.Conditions) == 0 {
		log.Info("PathwaysJob setPathwaysJobStatusBasedOnJobSetStatus: No conditions found in JobSet yet.")
		return
	}
	var pathwaysCondition *metav1.Condition
	var persistNeeded bool

	if len(js.Status.Conditions) > 0 {
		c := js.Status.Conditions[len(js.Status.Conditions)-1] // last condition of JobSet represents the current state of JobSet
		// If PathwaysJob is in terminal state or JobSet condition is not valid, no further action is needed.
		if pw.Status.TerminalState != "" {
			log.Info("PathwaysJob setPathwaysJobStatusBasedOnJobSetStatus: in TERMINAL state")
		}
		if c.Status != metav1.ConditionTrue {
			log.Info("PathwaysJob setPathwaysJobStatusBasedOnJobSetStatus: invalid JobSet condition")
			return
		}

		if c.Type == string(jobsetv1alpha2.JobSetStartupPolicyInProgress) && c.Status == metav1.ConditionTrue {
			// If JobSet is in JobSetStartupPolicyInProgress state, PathwaysJob will be in PathwaysJobPending state
			pathwaysCondition = makePendingCondition(c)
		}
		if c.Type == string(jobsetv1alpha2.JobSetStartupPolicyCompleted) && c.Status == metav1.ConditionTrue {
			// If JobSet is in JobSetStartupPolicyCompleted state, PathwaysJob will be in PathwaysJobRunning state
			pathwaysCondition = makeRunningCondition(c)
		} else if c.Type == string(jobsetv1alpha2.JobSetSuspended) && c.Status == metav1.ConditionTrue && pw.Status.TerminalState == "" {
			// If JobSet is in JobSetSuspended state, PathwaysJob will be in PathwaysJobSuspended state
			pathwaysCondition = makeSuspendCondition(c)
		} else if c.Type == string(jobsetv1alpha2.JobSetCompleted) && c.Status == metav1.ConditionTrue {
			// If JobSet is in JobSetCompleted state, PathwaysJob will be in PathwaysJobCompleted state
			pathwaysCondition = makeCompletedCondition(c)
			pw.Status.TerminalState = string(pathwaysjob.PathwaysJobCompleted)
		} else if c.Type == string(jobsetv1alpha2.JobSetFailed) && c.Status == metav1.ConditionTrue {
			// If JobSet is in JobSetFailed state, PathwaysJob will be in PathwaysJobFailed state
			pathwaysCondition = makeFailedCondition(c)
			pw.Status.TerminalState = string(pathwaysjob.PathwaysJobFailed)
		}

		// Persist PathwaysJob status update
		persistNeeded = checkandUpdateConditionsIfNeeded(pw, pathwaysCondition)
		if persistNeeded {
			if err := r.Status().Update(ctx, pw); err != nil {
				log.Error(err, "updating pathwaysjob status in setPathwaysJobStatusBasedOnJobSetStatus")
			}
			log.Info("PathwaysJob setPathwaysJobStatusBasedOnJobSetStatus: Status updated successfully!", "Condition ", pw.Status.Conditions[len(pw.Status.Conditions)-1].Type, "Status", pw.Status.Conditions[len(pw.Status.Conditions)-1].Status, "Reason ", pw.Status.Conditions[len(pw.Status.Conditions)-1].Reason, "Message", pw.Status.Conditions[len(pw.Status.Conditions)-1].Message)
		}
	}
}

// Check current PathwaysJob condition with new PathwaysJob condition and update if needed.
func checkandUpdateConditionsIfNeeded(pw *pathwaysjob.PathwaysJob, newCondition *metav1.Condition) bool {
	// currentCondition is pw.Status.Conditions[len(pw.Status.Conditions)-1]
	// No condition is tracked in PathwaysJob currently and the new condition is valid,
	// append the new condition.
	if len(pw.Status.Conditions) == 0 && newCondition.Status == metav1.ConditionTrue {
		pw.Status.Conditions = append(pw.Status.Conditions, *newCondition)
		return true
	}
	// If new condition is valid and is not the same as the current condition type,
	// then invalidate the old condition and append the new condition.
	if string(pw.Status.Conditions[len(pw.Status.Conditions)-1].Type) != string(newCondition.Type) && newCondition.Status == metav1.ConditionTrue {
		pw.Status.Conditions[len(pw.Status.Conditions)-1].Status = metav1.ConditionFalse
		pw.Status.Conditions = append(pw.Status.Conditions, *newCondition)
		return true
	}
	// In all other cases, no update is required.
	return false
}

// Construct the Pending condition for PathwaysJob
func makePendingCondition(jsCondition metav1.Condition) *metav1.Condition {
	pendingCondition := metav1.Condition{
		Type:               string(pathwaysjob.PathwaysJobPending),
		Status:             metav1.ConditionTrue,
		Reason:             jsCondition.Reason,
		Message:            "pathwaysJob pending : jobSet is getting created",
		LastTransitionTime: jsCondition.LastTransitionTime,
	}
	return &pendingCondition
}

// Construct the Running condition for PathwaysJob
func makeRunningCondition(jsCondition metav1.Condition) *metav1.Condition {
	runningCondition := metav1.Condition{
		Type:               string(pathwaysjob.PathwaysJobRunning),
		Status:             metav1.ConditionTrue,
		Reason:             jsCondition.Reason,
		Message:            "pathwaysJob running: jobSet is running",
		LastTransitionTime: jsCondition.LastTransitionTime,
	}
	return &runningCondition
}

// Construct the Suspended condition for PathwaysJob
func makeSuspendCondition(jsCondition metav1.Condition) *metav1.Condition {
	suspendCondition := metav1.Condition{
		Type:               string(pathwaysjob.PathwaysJobSuspended),
		Status:             jsCondition.Status,
		Reason:             jsCondition.Reason,
		Message:            "pathwaysJob suspended: " + jsCondition.Message,
		LastTransitionTime: jsCondition.LastTransitionTime,
	}
	return &suspendCondition
}

// Construct the Completed condition for PathwaysJob
func makeCompletedCondition(jsCondition metav1.Condition) *metav1.Condition {
	completedCondition := metav1.Condition{
		Type:               string(pathwaysjob.PathwaysJobCompleted),
		Status:             jsCondition.Status,
		Reason:             jsCondition.Reason,
		Message:            "pathwaysJob completed: " + jsCondition.Message,
		LastTransitionTime: jsCondition.LastTransitionTime,
	}
	return &completedCondition
}

// Construct the Failed condition for PathwaysJob
func makeFailedCondition(jsCondition metav1.Condition) *metav1.Condition {
	failedCondition := metav1.Condition{
		Type:               string(pathwaysjob.PathwaysJobFailed),
		Status:             jsCondition.Status,
		Reason:             jsCondition.Reason,
		Message:            "pathwaysJob failed: " + jsCondition.Message,
		LastTransitionTime: jsCondition.LastTransitionTime,
	}
	return &failedCondition
}

// Find TPU version from the worker's type (- used to determine Pathways instance_type)
func constructTPUVersionFromMachineType(machineType pathwaysjob.MachineType) string {
	// Worker types are already validated in the YAML.
	return MachineTypeToTPUVersionMap[string(machineType)]
}

// Find GKE accelerator type from the machine type (- used for nodeSelector)
func constructGKEAcceleratorTypeFromMachineType(machineType pathwaysjob.MachineType) string {
	// Worker types are already validated in the YAML.
	return MachineTypeToGKEAcceleratorTypeMap[string(machineType)]
}

// Validate that topology provided is valid for the provided worker type.
func validateTPUTopologyWithWorkerType(ctx context.Context, machineType pathwaysjob.MachineType, topology string) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	if slices.Contains(ValidTpuTopologiesMap[string(machineType)], topology) {
		return topology, nil
	} else {
		log.Info("Invalid topology!!! ", "Machine type ", string(machineType), " cannot have topology ", topology)
		return "", fmt.Errorf("invalid TPU topology for worker type")
	}
}

// Calculate the number of VMs based on the Machine Type and Topology (- used in completions/parallelisms)
func calculateVMsFromMachineTypeAndTopology(machineType pathwaysjob.MachineType, topology string) int32 {
	parts := strings.Split(topology, "x") // Examples - 2x2x4 or 4x4
	// Calculate the number of chips based on the Topology.
	// The topology must have already been validated with the worker type.
	chips := 1
	for _, part := range parts {
		num, _ := strconv.Atoi(part)
		chips *= num
	}
	vms := 1
	chipsperVM := 4
	if (machineType == pathwaysjob.Ct6e_standard_8t) || (machineType == pathwaysjob.Ct5lp_hightpu_8t) {
		chipsperVM = 8
	}
	if chips >= chipsperVM {
		vms = chips / chipsperVM
	}
	return int32(vms)
}

// Calculate all TPU related information
func calculateTPUInfo(ctx context.Context, pw *pathwaysjob.PathwaysJob) error {
	// setting public variables
	tpuVersion := constructTPUVersionFromMachineType(pw.Spec.Workers[0].Type)
	tpuTopology, err := validateTPUTopologyWithWorkerType(ctx, pw.Spec.Workers[0].Type, pw.Spec.Workers[0].Topology)
	if err != nil {
		return err
	}
	InstanceType = tpuVersion + ":" + tpuTopology
	GKEAcceleratorType = constructGKEAcceleratorTypeFromMachineType(pw.Spec.Workers[0].Type)
	NumVMs = calculateVMsFromMachineTypeAndTopology(pw.Spec.Workers[0].Type, pw.Spec.Workers[0].Topology)
	return nil
}

// Construct image tag based on Pathways version
func makeImageTagUsingPathwaysVersion(pw *pathwaysjob.PathwaysJob) string {
	var tag string
	if pw.Spec.PathwaysVersion != "" {
		tag = string(pw.Spec.PathwaysVersion)
	} else {
		tag = "latest"
	}
	return tag
}

// Construct success policy based on deployment mode and user workload spec.
func MakeSuccessPolicy(pw *pathwaysjob.PathwaysJob) *jobsetv1alpha2.SuccessPolicy {
	userJobName := "pathways-head"
	if isUserPodProvided(pw) {
		return &jobsetv1alpha2.SuccessPolicy{Operator: jobsetv1alpha2.OperatorAll, TargetReplicatedJobs: []string{userJobName}}
	} else {
		return nil
	}
}

// Pick the Component Image based on whether custom image or Pathways version are provided.
func MakeComponentImage(pw *pathwaysjob.PathwaysJob, customComponent pathwaysjob.PathwaysComponentType) string {
	if pw.Spec.CustomComponents != nil {
		for _, component := range pw.Spec.CustomComponents {
			if component.ComponentType == customComponent && component.Image != "" {
				return component.Image
			}
		}
	}

	// No custom components are provided or no custom image is provided for this component, return default images.
	if customComponent == pathwaysjob.PathwaysProxy {
		return fmt.Sprintf("%s:%s", DefaultPathwaysProxyImage, makeImageTagUsingPathwaysVersion(pw))
	}
	return fmt.Sprintf("%s:%s", DefaultPathwaysRMAndWorkerImage, makeImageTagUsingPathwaysVersion(pw))
}

// Append custom component flags
func AppendCustomComponentFlags(pw *pathwaysjob.PathwaysJob, customComponent pathwaysjob.PathwaysComponentType, args []string) []string {
	if pw.Spec.CustomComponents != nil {
		for _, component := range pw.Spec.CustomComponents {
			if component.ComponentType == customComponent && component.CustomFlags != nil {
				return append(args, component.CustomFlags...)
			}
		}
	}

	// No custom components are provided or no custom flags are provided for this component, return default args.
	return args
}

// Constructs the Pathways resource manager container spec for the underlying JobSet
func MakeResourceManagerContainer(pw *pathwaysjob.PathwaysJob, isInitContainer bool) (*corev1.Container, error) {

	args := []string{
		"--server_port=29001",
		fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
		"--node_type=resource_manager",
		fmt.Sprintf("--instance_count=%d", int32(pw.Spec.Workers[0].NumSlices)),
		fmt.Sprintf("--instance_type=%s", InstanceType),
	}

	if pw.Spec.Controller.EnableMetricsCollection {
		args = append(args, "--enable_metrics_collection=true")
	}

	// Append all the custom pathways server flags to the existing flags.
	args = AppendCustomComponentFlags(pw, pathwaysjob.PathwaysServer, args)

	rmContainerSpec := corev1.Container{
		Name:            "pathways-rm",
		Image:           MakeComponentImage(pw, pathwaysjob.PathwaysServer),
		ImagePullPolicy: "Always",
		Args:            args,
		Env: []corev1.EnvVar{
			{Name: "REPLICATED_JOB_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/replicatedjob-name']"}}},
			{Name: "JOBSET_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['jobset.sigs.k8s.io/jobset-name']"}}},
			// {Name: "HOST_ADDRESS", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.labels['jobset.sigs.k8s.io/coordinator']"}}},
			{Name: "HOST_ADDRESS", Value: fmt.Sprintf("%s-%s-0-0.%s", pw.GetName(), PathwaysHeadJobName, pw.GetName())},
			{Name: "TPU_SKIP_MDS_QUERY", Value: "true"},
		},
		Ports: []corev1.ContainerPort{{ContainerPort: 29001}, {ContainerPort: 29002}},
		// Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(8000000000, resource.DecimalSI)}},
	}

	// Init containers can have restartPolicy but regular containers cannot have restartPolicy.
	var restartPolicy corev1.ContainerRestartPolicy
	if isInitContainer {
		restartPolicy = corev1.ContainerRestartPolicyAlways
		rmContainerSpec.RestartPolicy = &restartPolicy
	}
	return &rmContainerSpec, nil
}

// Constructs the Pathways proxy container spec for the underlying JobSet
func MakeProxyContainer(pw *pathwaysjob.PathwaysJob, isInitContainer bool) (*corev1.Container, error) {

	args := []string{
		"--server_port=29000",
		fmt.Sprintf("--resource_manager_address=%s-%s-0-0.%s:29001", pw.GetName(), PathwaysHeadJobName, pw.GetName()),
		fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
	}

	if pw.Spec.Controller.EnableMetricsCollection {
		args = append(args, "--enable_metrics_collection=true")
	}
	// Append all the custom pathways proxy server flags to the existing flags.
	args = AppendCustomComponentFlags(pw, pathwaysjob.PathwaysProxy, args)

	if pw.Spec.Controller.ElasticSlices > 0 {
		args = append(args, fmt.Sprintf("--num_elastic_slices=%d", int32(pw.Spec.Controller.ElasticSlices)))
	}

	proxyContainerSpec := corev1.Container{
		Name:            "pathways-proxy",
		Image:           MakeComponentImage(pw, pathwaysjob.PathwaysProxy),
		ImagePullPolicy: "Always",
		Args:            args,
		Ports:           []corev1.ContainerPort{{ContainerPort: 29000}},
		// Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": *resource.NewQuantity(4, resource.DecimalSI), "memory": *resource.NewQuantity(100000000000, resource.DecimalSI)}}, //100GiB
	}
	// Init containers can have restartPolicy but regular containers cannot have restartPolicy.
	var restartPolicy corev1.ContainerRestartPolicy
	if isInitContainer {
		restartPolicy = corev1.ContainerRestartPolicyAlways
		proxyContainerSpec.RestartPolicy = &restartPolicy
	}
	return &proxyContainerSpec, nil
}

// Construct the initContainers to enable colocated python on Pathways workers.
func MakeColocateHeadWithWorkersPythonInitContainers(pw *pathwaysjob.PathwaysJob) ([]corev1.Container, error) {
	restartPolicy := corev1.ContainerRestartPolicyAlways

	if pw.Spec.CustomComponents != nil {
		for _, component := range pw.Spec.CustomComponents {
			if component.ComponentType == pathwaysjob.PathwaysColocatedPythonSidecar && component.Image != "" {
				colocatedPythonContainer := corev1.Container{
					Name:            "colocated-python-sidecar",
					Image:           component.Image,
					ImagePullPolicy: "Always",
					Env: []corev1.EnvVar{
						{Name: "GRPC_SERVER_ADDRESS", Value: "'0.0.0.0:50051'"},
					},
					Ports: []corev1.ContainerPort{{ContainerPort: 50051}},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "shared-tmp",
							MountPath: "/tmp",
						},
					},
					RestartPolicy: &restartPolicy,
				}
				return []corev1.Container{colocatedPythonContainer}, nil
			}
		}
	}
	return nil, nil
}

// Constructs Pathways worker replicated job for both 'colocated_head_with_workers' and 'default' deployment modes.
func MakeWorkerJob(ctx context.Context, pw *pathwaysjob.PathwaysJob) (jobsetv1alpha2.ReplicatedJob, error) {
	volumeSourceType := corev1.HostPathDirectoryOrCreate
	initContainers, _ := MakeColocateHeadWithWorkersPythonInitContainers(pw)
	backOffLimit := 0

	objectMeta := metav1.ObjectMeta{
		Annotations: map[string]string{
			"alpha.jobset.sigs.k8s.io/exclusive-topology": "cloud.google.com/gke-nodepool",
		},
	}

	args := []string{
		"--server_port=29005",
		fmt.Sprintf("--resource_manager_address=%s-%s-0-0.%s:29001", pw.GetName(), PathwaysHeadJobName, pw.GetName()),
		fmt.Sprintf("--gcs_scratch_location=%s", pw.Spec.PathwaysDir),
	}

	if pw.Spec.Controller.EnableMetricsCollection {
		args = append(args, "--enable_metrics_collection=true")
	}
	// Append all the custom pathways worker flags to the existing flags.
	args = AppendCustomComponentFlags(pw, pathwaysjob.PathwaysWorker, args)

	if pw.Spec.Controller.ElasticSlices > 0 && pw.Spec.Controller.MaxSliceRestarts > 0 {
		backOffLimit = ptr.To(int32(NumVMs * pw.Spec.Controller.MaxSliceRestarts))
	}

	workerJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "worker",
		Replicas: int32(pw.Spec.Workers[0].NumSlices),
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: backOffLimit,
				Completions:  ptr.To(int32(NumVMs)), // number of workers remember to change
				Parallelism:  ptr.To(int32(NumVMs)), // number of workers  remember to change
				Template: corev1.PodTemplateSpec{
					ObjectMeta: objectMeta,
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "pathways-worker",
								Image:           MakeComponentImage(pw, pathwaysjob.PathwaysWorker),
								ImagePullPolicy: "Always",
								Args:            args,
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
						InitContainers: initContainers,
						NodeSelector: map[string]string{
							"cloud.google.com/gke-tpu-accelerator": GKEAcceleratorType,
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

// Affinity rules to allow the leader pod to coexist with worker pod in the 'colocated_head_with_workers' mode.
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

// Checks whether user pod is provided or the workload is in headless mode.
func isUserPodProvided(pw *pathwaysjob.PathwaysJob) bool {
	return pw.Spec.Controller.UserPodTemplate != nil
}

// Get the init containers (any sidecars) from the User's pod spec.
// This is used to inject the containers into the leader pod in the 'colocate_head_with_workers' deployment mode and
// pathways-head pod in the 'default' deployment mode.
func GetContainerListFromUserPodSpec(pw *pathwaysjob.PathwaysJob) ([]corev1.Container, error) {
	// When workload is to be run in headless mode, no user pod will be provided.
	if isUserPodProvided(pw) {
		return pw.Spec.Controller.UserPodTemplate.Spec.Containers, nil
	}
	return nil, fmt.Errorf("no user pod provided, probably headless mode")
}

// Pods containing the following Toleration are allowed to be scheduled on TPUs.
// This toleration is needed only for the colocate_head_with_workers mode.
func MakeTolerationToAllowSchedulingOnTPU(pw *pathwaysjob.PathwaysJob) []corev1.Toleration {
	if pw.Spec.Controller.DeploymentMode == pathwaysjob.ColocateHeadWithWorkers {
		return []corev1.Toleration{
			{
				Key:      "google.com/tpu",
				Operator: "Exists",
				Effect:   "NoSchedule",
			},
		}
	} else {
		return []corev1.Toleration{}
	}
}

func MakePathwaysHeadPodSpec(pw *pathwaysjob.PathwaysJob) *corev1.PodSpec {
	var pathwaysHeadPodSpec *corev1.PodSpec
	if isUserPodProvided(pw) {
		// Inject Pathways RM and proxy into the user provided pod spec
		// in the form of initContainers. The user container is the main container,
		// whose success or failure will be tracked.
		// Ensure DNSPolicy and HostNetwork are set as needed.
		RMContainerSpec, _ := MakeResourceManagerContainer(pw, true)
		ProxyContainerSpec, _ := MakeProxyContainer(pw, true)
		initContainerList := []corev1.Container{*RMContainerSpec, *ProxyContainerSpec}
		pathwaysHeadPodSpec = pw.Spec.Controller.UserPodTemplate.Spec.DeepCopy()
		pathwaysHeadPodSpec.HostNetwork = true
		pathwaysHeadPodSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		pathwaysHeadPodSpec.InitContainers = initContainerList
	} else {
		// In Headless mode, RM and proxy are the the main containers.
		// Ensure DNSPolicy and HostNetwork are set as needed.
		RMContainerSpec, _ := MakeResourceManagerContainer(pw, false)
		ProxyContainerSpec, _ := MakeProxyContainer(pw, false)
		containerList := []corev1.Container{*RMContainerSpec, *ProxyContainerSpec}
		pathwaysHeadPodSpec = &corev1.PodSpec{
			HostNetwork: true,                              // For performance == McJAX
			DNSPolicy:   corev1.DNSClusterFirstWithHostNet, // For performance == McJAX
			Containers:  containerList,
		} // end PodSpec
	}
	return pathwaysHeadPodSpec
}

func MakePathwaysHeadReplicatedJob(pathwaysHeadPodSpec corev1.PodSpec) jobsetv1alpha2.ReplicatedJob {
	pathwaysHeadJob := jobsetv1alpha2.ReplicatedJob{
		Name:     "pathways-head",
		Replicas: 1,
		Template: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				BackoffLimit: ptr.To(int32(0)),
				Completions:  ptr.To(int32(1)),
				Parallelism:  ptr.To(int32(1)),
				Template: corev1.PodTemplateSpec{
					Spec: pathwaysHeadPodSpec,
				},
			},
		},
	} // end replicated Job
	return pathwaysHeadJob
}

// Construct pathways-head replicated job containing Pathways RM, Pathways Proxy and the user job containers for the 'colocate' deployment mode.
// In the colocate_head_with_workers mode, the Pathways head pod is placed on TPU nodes, beside a worker pod.
func MakePathwaysHeadJobForColocateHeadWithWorkersDeployment(ctx context.Context, pw *pathwaysjob.PathwaysJob) (jobsetv1alpha2.ReplicatedJob, error) {
	podSpec := *MakePathwaysHeadPodSpec(pw)
	// Add affinity and tolerations to allow the Pathways head pod to be scheduled on TPUs.
	affinitySpec, _ := MakePodAffinityRules(pw)
	tolerations := MakeTolerationToAllowSchedulingOnTPU(pw)
	podSpec.Affinity = affinitySpec
	podSpec.Tolerations = tolerations

	return MakePathwaysHeadReplicatedJob(podSpec), nil
}

// Construct pathways-head replicated job containing Pathways RM, Pathways Proxy and the user job containers for the 'default' deployment mode.
// In the default mode, the Pathways head pod is placed on CPU nodes.
func MakePathwaysHeadJobForDefaultDeployment(ctx context.Context, pw *pathwaysjob.PathwaysJob) (jobsetv1alpha2.ReplicatedJob, error) {
	podSpec := *MakePathwaysHeadPodSpec(pw)
	return MakePathwaysHeadReplicatedJob(podSpec), nil
}
