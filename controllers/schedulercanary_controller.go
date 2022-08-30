/*
Copyright 2022.

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

package controllers

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	monitoringv1beta1 "github.com/appuio/scheduler-canary-controller/api/v1beta1"
	"github.com/appuio/scheduler-canary-controller/controllers/podstate"
)

// SchedulerCanaryReconciler reconciles a SchedulerCanary object
type SchedulerCanaryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=monitoring.appuio.io,resources=schedulercanaries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.appuio.io,resources=schedulercanaries/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.appuio.io,resources=schedulercanaries/finalizers,verbs=update

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SchedulerCanary object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *SchedulerCanaryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling")

	instance := &monitoringv1beta1.SchedulerCanary{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Object is in the process of being deleted.
		return ctrl.Result{}, nil
	}

	// TODO delete labels on instance deletion?
	initMetrics(instance.Namespace, instance.Name)

	nextReconcile := instance.Status.LastCanaryCreatedAt.Add(instance.Spec.IntervalWithDefault())
	if !nextReconcile.Before(time.Now()) {
		rqi := time.Until(nextReconcile)
		l.Info("Next reconcile in", "duration", rqi)
		return ctrl.Result{Requeue: true, RequeueAfter: rqi}, nil
	}

	if err := r.createCanaryPod(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	instance.Status.LastCanaryCreatedAt = metav1.Now()
	err = r.Client.Status().Update(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: instance.Spec.IntervalWithDefault()}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerCanaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1beta1.SchedulerCanary{}).
		Complete(r)
}

func (r *SchedulerCanaryReconciler) createCanaryPod(ctx context.Context, instance *monitoringv1beta1.SchedulerCanary) error {
	l := log.FromContext(ctx)

	pod, err := buildPodFromTemplate(&instance.Spec.PodTemplate, instance)
	if err != nil {
		return err
	}

	err = controllerutil.SetControllerReference(instance, pod, r.Scheme)
	if err != nil {
		return err
	}

	err = trackState(pod, podstate.PodCreated)
	if err != nil {
		return err
	}

	l.Info("Creating pod")
	if err := r.Client.Create(ctx, pod); err != nil {
		return err
	}

	l.Info("Pod created", "pod", pod.Name)
	l.Info("Reconciled")
	return nil
}
