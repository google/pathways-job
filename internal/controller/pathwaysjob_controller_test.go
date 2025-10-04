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
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"

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
						Annotations: map[string]string{
							"test-annotation": "test-value",
						},
					},
					Spec: pathwaysjobv1.PathwaysJobSpec{
						Controller: &pathwaysjobv1.ControllerSpec{
							DeploymentMode: pathwaysjobv1.Default,
						},
						Workers: []pathwaysjobv1.WorkerSpec{
							{
								Type:      pathwaysjobv1.Ct5lp_hightpu_4t,
								Topology:  "2x2",
								NumSlices: 1,
							},
						},
						PathwaysDir: "gs://test-bucket/pathways",
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
		It("should successfully reconcile the resource and create a JobSet", func() {
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

			By("Checking if JobSet was created")
			jobSet := &jobsetv1alpha2.JobSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, jobSet)
			}, "10s", "1s").Should(Succeed())

			Expect(jobSet.Name).To(Equal(resourceName))
			Expect(len(jobSet.Spec.ReplicatedJobs)).To(Equal(2))
			Expect(jobSet.Spec.ReplicatedJobs[0].Name).To(Equal(PathwaysHeadJobName))
			Expect(jobSet.Spec.ReplicatedJobs[1].Name).To(Equal("worker"))

			By("Checking if annotations were propagated to JobSet")
			Expect(jobSet.Annotations).To(HaveKeyWithValue("test-annotation", "test-value"))
			headJobTemplate := jobSet.Spec.ReplicatedJobs[0].Template
			Expect(headJobTemplate.Annotations).To(HaveKeyWithValue("test-annotation", "test-value"))
		})
	})
})

func TestMakeWorkerJobNodeSelector(t *testing.T) {
	cases := []struct {
		desc string
		pw   *pathwaysjobv1.PathwaysJob
		want map[string]string
	}{
		{
			desc: "no user specified node selector",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     "ct4p-hightpu-4t",
							Topology: "2x2x2",
						},
					},
				},
			},
			want: map[string]string{
				"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice",
				"cloud.google.com/gke-tpu-topology":    "2x2x2",
			},
		},
		{
			desc: "reservation",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     "ct4p-hightpu-4t",
							Topology: "2x2x2",
							NodeSelector: map[string]string{
								"cloud.google.com/reservation-name": "reservation-name",
							},
						},
					},
				},
			},
			want: map[string]string{
				"cloud.google.com/reservation-name":    "reservation-name",
				"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice",
				"cloud.google.com/gke-tpu-topology":    "2x2x2",
			},
		},
		{
			desc: "user specify other labels",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     "ct4p-hightpu-4t",
							Topology: "2x2x2",
							NodeSelector: map[string]string{
								"key": "value",
							},
						},
					},
				},
			},
			want: map[string]string{
				"key":                                  "value",
				"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice",
				"cloud.google.com/gke-tpu-topology":    "2x2x2",
			},
		},
		{
			desc: "override accelerator and topology",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							Type:     "ct4p-hightpu-4t",
							Topology: "2x2x2",
							NodeSelector: map[string]string{
								"cloud.google.com/gke-tpu-accelerator": "tpu-v6e-slice",
								"cloud.google.com/gke-tpu-topology":    "2x2",
							},
						},
					},
				},
			},
			want: map[string]string{
				"cloud.google.com/gke-tpu-accelerator": "tpu-v4-podslice",
				"cloud.google.com/gke-tpu-topology":    "2x2x2",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			calculateTPUInfo(context.TODO(), tc.pw)
			got := makeWorkerJobNodeSelector(tc.pw)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("makeWorkerJobNodeSelector() = %v, want %v", got, tc.want)
			}
		})
	}
}
