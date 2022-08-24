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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	monitoringv1beta1 "github.com/appuio/scheduler-canary-controller/api/v1beta1"
	"github.com/appuio/scheduler-canary-controller/controllers/podstate"
)

var podTimeUnscheduled = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "scheduler_canary_pod_unscheduled_seconds",
	Help: "Time spent in pending state",
}, []string{"namespace", "name"})

var podTimeUntilAcknowledged = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "scheduler_canary_pod_until_acknowledged_seconds",
	Help: "Time spent in an unacknowledged state",
}, []string{"namespace", "name"})

var podTimeUntilWaiting = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "scheduler_canary_pod_until_waiting_seconds",
	Help: "Time spent before pulling images mounting volumes",
}, []string{"namespace", "name"})

var podsTimeouted = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "scheduler_canary_pods_timeouted",
	Help: "Pods that reached the specified .maxPodCompletionTimeout timeout",
}, []string{"namespace", "name"})

func init() {
	metrics.Registry.MustRegister(
		podTimeUnscheduled,
		podTimeUntilAcknowledged,
		podTimeUntilWaiting,

		podsTimeouted,
	)
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

	// TODO delete labels on instance deletion?
	initMetrics(instance.Namespace, instance.Name)

	podPresent, pod, err := r.findPod(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if podPresent {
		return withSafetyRequeue(r.checkCanaryPod(ctx, instance, pod))
	}

	return r.createCanaryPod(ctx, instance)
}

// hasPod checks if the pod is present in the cluster.
func (r *SchedulerCanaryReconciler) findPod(ctx context.Context, instance *monitoringv1beta1.SchedulerCanary) (bool, *corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: podName(instance, "-0"), Namespace: instance.Namespace}, pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, pod, nil
}

func (r *SchedulerCanaryReconciler) checkCanaryPod(ctx context.Context, instance *monitoringv1beta1.SchedulerCanary, pod *corev1.Pod) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("pod", pod.Name)
	l.Info("Checking canary pod")

	if !pod.DeletionTimestamp.IsZero() {
		// Pod is in the process of being deleted.
		l.Info("Pod is deleting")
		return ctrl.Result{}, nil
	}

	state := podstate.State(*pod)
	l.Info("Pod is in state", "state", state)
	if state == podstate.PodCompleted {
		err := calculateTimes(l, instance, pod)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.Client.Delete(ctx, pod)
	}
	if podHasReachedTimeout(*pod, instance.Spec.MaxPodCompletionTimeoutWithDefault()) {
		l.Info("Pod has reached timeout, deleting")
		podsTimeouted.WithLabelValues(instance.Namespace, instance.Name).Inc()
		return ctrl.Result{}, r.Client.Delete(ctx, pod)
	}

	err := trackState(pod, state)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.strategicMergePatch(ctx, pod, stateTrackingPatchFromPod(*pod))
}

func (r *SchedulerCanaryReconciler) createCanaryPod(ctx context.Context, instance *monitoringv1beta1.SchedulerCanary) (reconcile.Result, error) {
	l := log.FromContext(ctx)

	pod, err := buildPodFromTemplate(&instance.Spec.PodTemplate, instance, "-0")
	if err != nil {
		return ctrl.Result{}, err
	}

	err = controllerutil.SetControllerReference(instance, pod, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = trackState(pod, podstate.PodCreated)
	if err != nil {
		return ctrl.Result{}, err
	}

	l.Info("Creating pod")

	if err := r.Client.Create(ctx, pod); err != nil {
		return ctrl.Result{}, err
	}

	l.Info("Pod created", "pod", pod.Name)
	l.Info("Reconciled")

	return ctrl.Result{}, nil
}

func (r *SchedulerCanaryReconciler) strategicMergePatch(ctx context.Context, obj client.Object, patch map[string]any) error {
	jp, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}
	return r.Client.Patch(ctx, obj, client.RawPatch(types.StrategicMergePatchType, jp))
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerCanaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1beta1.SchedulerCanary{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func podName(instance *monitoringv1beta1.SchedulerCanary, suffix string) string {
	return suffixLimit(instance.Name, suffix)
}

func calculateTimes(l logr.Logger, instance *monitoringv1beta1.SchedulerCanary, pod *corev1.Pod) error {
	tr, err := getTrackedStates(pod)
	if err != nil {
		return err
	}

	createdTime, ok := tr[podstate.PodCreated]
	if !ok {
		l.Info("WARNING: Pod created time not found, skipping calculation")
		return nil
	}

	acknowledgedTime, hasAcknowledgedTime := tr[podstate.PodAcknowledged]
	scheduledTime, hasScheduledTime := tr[podstate.PodScheduled]
	waitingTime, hasWaitingTime := tr[podstate.PodWaiting]

	usm := podTimeUnscheduled.WithLabelValues(instance.Namespace, instance.Name)
	if hasScheduledTime {
		usm.Observe(scheduledTime.Sub(createdTime).Seconds())
	} else if hasAcknowledgedTime {
		usm.Observe(acknowledgedTime.Sub(createdTime).Seconds())
	} else if hasWaitingTime {
		usm.Observe(waitingTime.Sub(createdTime).Seconds())
	}

	uam := podTimeUntilAcknowledged.WithLabelValues(instance.Namespace, instance.Name)
	if hasAcknowledgedTime {
		uam.Observe(acknowledgedTime.Sub(createdTime).Seconds())
	} else if hasWaitingTime {
		uam.Observe(waitingTime.Sub(createdTime).Seconds())
	}

	if hasWaitingTime {
		podTimeUntilWaiting.
			WithLabelValues(instance.Namespace, instance.Name).
			Observe(waitingTime.Sub(createdTime).Seconds())
	} else {
		l.Info("WARNING: No pod waiting time found, skipping calculation")
	}

	return nil
}

func podHasReachedTimeout(pod corev1.Pod, timeout time.Duration) bool {
	if pod.CreationTimestamp.IsZero() {
		return false
	}
	return pod.CreationTimestamp.Add(timeout).Before(time.Now())
}

// withSafetyRequeue ensures the request is requeued after one minute.
func withSafetyRequeue(r ctrl.Result, err error) (ctrl.Result, error) {
	if err != nil || !r.Requeue || r.RequeueAfter == 0 {
		return r, err
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// initMetrics ensures the metrics are present in the output as soon as the instance is created.
func initMetrics(namespace, name string) {
	podTimeUnscheduled.WithLabelValues(namespace, name)
	podTimeUntilAcknowledged.WithLabelValues(namespace, name)
	podTimeUntilWaiting.WithLabelValues(namespace, name)

	podsTimeouted.WithLabelValues(namespace, name)
}
