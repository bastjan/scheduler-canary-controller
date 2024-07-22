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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
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

// Reconcile reconciles SchedulerCanary manifests.
// It creates new canary pods from the given template at the given interval.
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

	initMetrics(instance.Namespace, instance.Name)

	nextReconcile := instance.Status.LastCanaryCreatedAt.Add(instance.Spec.IntervalWithDefault())
	if !nextReconcile.Before(time.Now()) {
		rqi := time.Until(nextReconcile)
		l.Info("Next reconcile in", "duration", rqi)
		return ctrl.Result{Requeue: true, RequeueAfter: rqi}, nil
	}

	if instance.Spec.ForbidParallelRuns {
		// Check if there is already a canary pod running.
		pods := &corev1.PodList{}
		if err := r.Client.List(ctx, pods, client.InNamespace(instance.Namespace), client.MatchingLabels{instanceLabel: instance.Name}); err != nil {
			return ctrl.Result{}, fmt.Errorf("forbidParallelRuns: error checking for already running pods, failed to list pods: %w", err)
		}
		if len(pods.Items) > 0 {
			podNames := make([]string, len(pods.Items))
			for i, pod := range pods.Items {
				podNames[i] = pod.Name
			}
			l.Info("ForbidParallelRuns: already running pods found, skipping pod creation", "pods", podNames)
			return ctrl.Result{}, nil
		}
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
		Owns(&corev1.Pod{}).
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
