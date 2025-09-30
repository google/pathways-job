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

func TestValidateCapacityNodeSelector(t *testing.T) {
	cases := []struct {
		desc    string
		pw      *pathwaysjobv1.PathwaysJob
		wantErr bool
	}{
		{
			desc: "empty capacity node selector",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{},
					},
				},
			},
		},
		{
			desc: "no colon",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							CapacityNodeSelector: "no colon",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "more than 1 colon",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							CapacityNodeSelector: `cloud.google.com/reservation-name:: reservation-name`,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "one colon",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							CapacityNodeSelector: `cloud.google.com/reservation-name: reservation-name`,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := validateCapacityNodeSelector(tc.pw)
			if got == nil && tc.wantErr {
				t.Error("validateCapacityNodeSelector() = nil, want error")
			}
			if got != nil && !tc.wantErr {
				t.Errorf("validateCapacityNodeSelector() = %v, want nil", got)
			}
		})
	}
}

func TestMakeCapacityNodeSelector(t *testing.T) {
	cases := []struct {
		desc string
		pw   *pathwaysjobv1.PathwaysJob
		want bool
	}{
		{
			desc: "flex start",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							CapacityNodeSelector: `cloud.google.com/gke-queued: "true"`,
						},
					},
				},
			},
			want: true,
		},
		{
			desc: "spot",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							CapacityNodeSelector: `cloud.google.com/gke-spot: "true"`,
						},
					},
				},
			},
			want: true,
		},
		{
			desc: "reservation",
			pw: &pathwaysjobv1.PathwaysJob{
				Spec: pathwaysjobv1.PathwaysJobSpec{
					Workers: []pathwaysjobv1.WorkerSpec{
						{
							CapacityNodeSelector: `cloud.google.com/reservation-name: reservation-name`,
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, got := makeCapacityNodeSelector(tc.pw)
			if got != tc.want {
				t.Errorf("makeCapacityNodeSelector() = %v, want %v", got, tc.want)
			}
		})
	}
}
