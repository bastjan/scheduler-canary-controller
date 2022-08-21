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

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	monitoringv1beta1 "github.com/appuio/scheduler-canary-controller/api/v1beta1"
)

var podTimeInPending = prometheus.NewSummary(prometheus.SummaryOpts{
	Name: "scheduler_canary_pod_time_in_pending_seconds",
	Help: "Time spent in pending state",
})

func init() {
	metrics.Registry.MustRegister(podTimeInPending)
}

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

	if instance.Status.PodName != "" {
		l.Info("Instance has existing pod", "pod", instance.Status.PodName)
		return r.checkCanaryPod(ctx, instance)
	}

	return r.createCanaryPod(ctx, instance)
}

func (r *SchedulerCanaryReconciler) checkCanaryPod(ctx context.Context, instance *monitoringv1beta1.SchedulerCanary) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("pod", instance.Status.PodName)
	l.Info("Checking canary pod")

	pod := &corev1.Pod{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: instance.Status.PodName, Namespace: instance.Namespace}, pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			l.Info("WARNING: Pod disappeared")
			instance.Status.PodName = ""
			return ctrl.Result{}, r.Client.Status().Update(ctx, instance)
		}
		return ctrl.Result{}, err
	}
	if !pod.DeletionTimestamp.IsZero() {
		// Pod is in the process of being deleted.
		l.Info("Pod is deleting")
		return ctrl.Result{}, nil
	}

	l.Info("Pod found", "phase", pod.Status.Phase)
	if pod.Status.Phase == corev1.PodSucceeded {
		l.Info("Canary pod succeeded")
		d := pod.Status.StartTime.Sub(pod.CreationTimestamp.Time)
		l.Info("Time spent in pending state", "duration", d, "start", pod.Status.StartTime.Time, "creation", pod.CreationTimestamp.Time)
		podTimeInPending.Observe(d.Seconds())

		if err := r.Client.Delete(ctx, pod); err != nil {
			return ctrl.Result{}, err
		}
		instance.Status.PodName = ""
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *SchedulerCanaryReconciler) createCanaryPod(ctx context.Context, instance *monitoringv1beta1.SchedulerCanary) (reconcile.Result, error) {
	l := log.FromContext(ctx)

	pod, err := buildPodFromTemplate(&instance.Spec.PodTemplate, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = controllerutil.SetControllerReference(instance, pod, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	l.Info("Creating pod")

	if err := r.Client.Create(ctx, pod); err != nil {
		return ctrl.Result{}, err
	}

	l.Info("Pod created", "pod", pod.Name)

	instance.Status.PodName = pod.Name
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	l.Info("Reconciled")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerCanaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1beta1.SchedulerCanary{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
