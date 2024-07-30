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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pathwaysapi "pathways-api/api/v1"
)

// PathwaysAPIReconciler reconciles a PathwaysAPI object
type PathwaysAPIReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pathways-api.pathways.domain,resources=pathwaysapis/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PathwaysAPI object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *PathwaysAPIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	pw := &pathwaysapi.PathwaysAPI{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, pw); err != nil {
		// log.Error(err, "unable to fetch Pathways ")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log := ctrl.LoggerFrom(ctx).WithValues("pathwaysapi", klog.KObj(pw))
	pwMessage := pw.Spec.TextMessage
	tpuType := pw.Spec.TpuType
	numSlices := pw.Spec.NumSlices
	workloadMode := pw.Spec.WorkloadMode

	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("ROSHANI CONTROLLER WORKING...", "TextMessage", pwMessage, "TpuType", tpuType, "NumSlices", numSlices, "WorkloadMode", workloadMode)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PathwaysAPIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pathwaysapi.PathwaysAPI{}).
		Complete(r)
}
