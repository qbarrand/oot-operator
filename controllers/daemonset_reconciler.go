package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type DaemonSetReconciler struct {
	client client.Client
}

func NewDaemonSetReconciler(client client.Client) *DaemonSetReconciler {
	return &DaemonSetReconciler{client: client}
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ds := appsv1.DaemonSet{}

	logger := log.FromContext(ctx)

	if err := r.client.Get(ctx, req.NamespacedName, &ds); err != nil {
		return ctrl.Result{}, fmt.Errorf("could not get DaemonSet %s: %v", req.String(), err)
	}

	if metav1.Now().After(ds.CreationTimestamp.Time.Add(1*time.Minute)) && ds.Status.DesiredNumberScheduled == 0 {
		logger.Info("After one minute, there is no node to schedule the DaemonSet on; deleting")
		return ctrl.Result{}, r.client.Delete(ctx, &ds)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.
		NewControllerManagedBy(mgr).
		Named("DaemonSetReconciler").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(
			predicate.NewPredicateFuncs(ModuleLabelNotEmptyFilter),
		).
		Complete(r)
}

func ModuleLabelNotEmptyFilter(obj client.Object) bool {
	return obj.GetLabels()[ModuleNameLabel] != ""
}
