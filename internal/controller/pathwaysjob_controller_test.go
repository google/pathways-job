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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pathwaysjobv1 "pathways-job/api/v1"
)

var _ = Describe("PathwaysJob Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		pathwaysjob := &pathwaysjobv1.PathwaysJob{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PathwaysJob")
			err := k8sClient.Get(ctx, typeNamespacedName, pathwaysjob)
			if err != nil && errors.IsNotFound(err) {
				resource := &pathwaysjobv1.PathwaysJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &pathwaysjobv1.PathwaysJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance PathwaysJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PathwaysJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

// Test machine type and topology
var _ = Describe("PathwaysJob Helpers", func() {
	Context("When calculating TPU info", func() {
		It("should successfully calculate TPU info", func() {
			pw := &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     pathwaysjobv1.Ct6e_standard_4t,
							Topology: "2x2",
						},
					},
				},
			}
			err := calculateTPUInfo(context.Background(), pw)
			Expect(err).NotTo(HaveOccurred())
			Expect(InstanceType).To(Equal("tpuv6e:2x2"))
			Expect(GKEAcceleratorType).To(Equal("tpu-v6e-slice"))
			Expect(NumVMs).To(Equal(int32(1)))
		})
		It("should successfully calculate TPU info for v5p", func() {
			pw := &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     pathwaysjobv1.Ct5p_hightpu_4t,
							Topology: "2x2x2",
						},
					},
				},
			}
			err := calculateTPUInfo(context.Background(), pw)
			Expect(err).NotTo(HaveOccurred())
			Expect(InstanceType).To(Equal("tpuv5:2x2x2"))
			Expect(GKEAcceleratorType).To(Equal("tpu-v5p-slice"))
			Expect(NumVMs).To(Equal(int32(2)))
		})
		It("should successfully calculate TPU info for v5e", func() {
			pw := &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     pathwaysjobv1.Ct5lp_hightpu_4t,
							Topology: "4x4",
						},
					},
				},
			}
			err := calculateTPUInfo(context.Background(), pw)
			Expect(err).NotTo(HaveOccurred())
			Expect(InstanceType).To(Equal("tpuv5e:4x4"))
			Expect(GKEAcceleratorType).To(Equal("tpu-v5-lite-podslice"))
			Expect(NumVMs).To(Equal(int32(4)))
		})
		It("should successfully calculate TPU info for v6e 8t", func() {
			pw := &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     pathwaysjobv1.Ct6e_standard_8t,
							Topology: "2x4",
						},
					},
				},
			}
			err := calculateTPUInfo(context.Background(), pw)
			Expect(err).NotTo(HaveOccurred())
			Expect(InstanceType).To(Equal("tpuv6e1t:2x4"))
			Expect(GKEAcceleratorType).To(Equal("tpu-v6e-slice"))
			Expect(NumVMs).To(Equal(int32(1)))
		})
	})
})

// Test Pathways make worker job
var _ = Describe("PathwaysJob MakeWorkerJob", func() {
	Context("When making worker job", func() {
		It("should successfully make worker job for v5p", func() {
			pw := &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:      pathwaysjobv1.Ct5p_hightpu_4t,
							Topology:  "2x2x2",
							NumSlices: 2,
						},
					},
				},
			}
			calculateTPUInfo(context.Background(), pw)
			workerJob, err := MakeWorkerJob(context.Background(), pw)
			Expect(err).NotTo(HaveOccurred())
			Expect(workerJob.Replicas).To(Equal(int32(2)))
			Expect(*workerJob.Template.Spec.Completions).To(Equal(int32(2)))
			Expect(*workerJob.Template.Spec.Parallelism).To(Equal(int32(2)))
			Expect(workerJob.Template.Spec.Template.Spec.NodeSelector["cloud.google.com/gke-tpu-accelerator"]).To(Equal("tpu-v5p-slice"))
			Expect(workerJob.Template.Spec.Template.Spec.NodeSelector["cloud.google.com/gke-tpu-topology"]).To(Equal("2x2x2"))
		})
	})
})

// Test Pathways proxy container
var _ = Describe("PathwaysJob MakeProxyContainer", func() {
	Context("When making proxy container", func() {
		It("should successfully make proxy container", func() {
			pw := &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					PathwaysDir: "gs://test-bucket/tmp",
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:      pathwaysjobv1.Ct5p_hightpu_4t,
							Topology:  "2x2x2",
							NumSlices: 2,
						},
					},
				},
			}
			calculateTPUInfo(context.Background(), pw)
			proxyContainer, err := MakeProxyContainer(pw, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(proxyContainer.Name).To(Equal("pathways-proxy"))
			Expect(proxyContainer.Args).To(ContainElement("--gcs_scratch_location=gs://test-bucket/tmp"))
		})
	})
})

// Test Pathways RM container
var _ = Describe("PathwaysJob MakeRMContainer", func() {
	Context("When making RM container", func() {
		It("should successfully make RM container", func() {
			pw := &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					PathwaysDir: "gs://test-bucket/tmp",
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:      pathwaysjobv1.Ct5p_hightpu_4t,
							Topology:  "2x2x2",
							NumSlices: 2,
						},
					},
				},
			}
			calculateTPUInfo(context.Background(), pw)
			rmContainer, err := MakeResourceManagerContainer(pw, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(rmContainer.Name).To(Equal("pathways-rm"))
			Expect(rmContainer.Args).To(ContainElement("--gcs_scratch_location=gs://test-bucket/tmp"))
			Expect(rmContainer.Args).To(ContainElement("--instance_type=tpuv5:2x2x2"))
		})
	})
})

// Test Pathways job conditions
var _ = Describe("PathwaysJob Conditions", func() {
	Context("When checking conditions", func() {
		It("should successfully make pending condition", func() {
			jsCondition := metav1.Condition{
				Type:               string(jobsetv1alpha2.JobSetStartupPolicyInProgress),
				Status:             metav1.ConditionTrue,
				Reason:             "JobSetStartupPolicyInProgress",
				Message:            "JobSetStartupPolicyInProgress",
				LastTransitionTime: metav1.Now(),
			}
			pendingCondition := makePendingCondition(jsCondition)
			Expect(pendingCondition.Type).To(Equal(string(pathwaysjobv1.PathwaysJobPending)))
			Expect(pendingCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(pendingCondition.Reason).To(Equal("JobSetStartupPolicyInProgress"))
		})
		It("should successfully make running condition", func() {
			jsCondition := metav1.Condition{
				Type:               string(jobsetv1alpha2.JobSetStartupPolicyCompleted),
				Status:             metav1.ConditionTrue,
				Reason:             "JobSetStartupPolicyCompleted",
				Message:            "JobSetStartupPolicyCompleted",
				LastTransitionTime: metav1.Now(),
			}
			runningCondition := makeRunningCondition(jsCondition)
			Expect(runningCondition.Type).To(Equal(string(pathwaysjobv1.PathwaysJobRunning)))
			Expect(runningCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(runningCondition.Reason).To(Equal("JobSetStartupPolicyCompleted"))
		})
		It("should successfully make suspend condition", func() {
			jsCondition := metav1.Condition{
				Type:               string(jobsetv1alpha2.JobSetSuspended),
				Status:             metav1.ConditionTrue,
				Reason:             "JobSetSuspended",
				Message:            "JobSetSuspended",
				LastTransitionTime: metav1.Now(),
			}
			suspendCondition := makeSuspendCondition(jsCondition)
			Expect(suspendCondition.Type).To(Equal(string(pathwaysjobv1.PathwaysJobSuspended)))
			Expect(suspendCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(suspendCondition.Reason).To(Equal("JobSetSuspended"))
		})
		It("should successfully make completed condition", func() {
			jsCondition := metav1.Condition{
				Type:               string(jobsetv1alpha2.JobSetCompleted),
				Status:             metav1.ConditionTrue,
				Reason:             "JobSetCompleted",
				Message:            "JobSetCompleted",
				LastTransitionTime: metav1.Now(),
			}
			completedCondition := makeCompletedCondition(jsCondition)
			Expect(completedCondition.Type).To(Equal(string(pathwaysjobv1.PathwaysJobCompleted)))
			Expect(completedCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(completedCondition.Reason).To(Equal("JobSetCompleted"))
		})
		It("should successfully make failed condition", func() {
			jsCondition := metav1.Condition{
				Type:               string(jobsetv1alpha2.JobSetFailed),
				Status:             metav1.ConditionTrue,
				Reason:             "JobSetFailed",
				Message:            "JobSetFailed",
				LastTransitionTime: metav1.Now(),
			}
			failedCondition := makeFailedCondition(jsCondition)
			Expect(failedCondition.Type).To(Equal(string(pathwaysjobv1.PathwaysJobFailed)))
			Expect(failedCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(failedCondition.Reason).To(Equal("JobSetFailed"))
		})
	})
})
