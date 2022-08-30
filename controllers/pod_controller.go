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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/appuio/scheduler-canary-controller/controllers/podstate"
)

const (
	instanceLabel     = "scheduler-canary-controller.appuio.io/instance"
	timeoutAnnotation = "scheduler-canary-controller.appuio.io/timeout"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles Pods with the `instanceLabel` set.
// It tracks state changes in an annotation on the pod and calculates the timing metrics on pod completion or timeout.
// Completed pods, pods with no timeout annotation, and timed out pods are deleted.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	pod := &corev1.Pod{}
	err := r.Client.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		// Object is in the process of being deleted.
		return ctrl.Result{}, nil
	}

	instance := pod.GetLabels()[instanceLabel]

	timeout, err := timeoutFromPod(pod)
	if err != nil {
		l.Error(err, "Pod has no valid timeout annotation, deleting immediately")
	}

	timedOut := podHasReachedTimeout(*pod, timeout)
	state := podstate.State(*pod)
	l.Info("Pod is in state", "state", state, "timedOut", timedOut)
	if timedOut || state == podstate.PodCompleted {
		if err := recordSchedulingMetrics(l, instance, pod, timedOut); err != nil {
			l.Error(err, "Failed to record metrics")
		}
		return ctrl.Result{}, r.Client.Delete(ctx, pod)
	}

	if err := trackState(pod, state); err != nil {
		return ctrl.Result{}, err
	}

	deadline := pod.CreationTimestamp.Add(timeout)
	return ctrl.Result{Requeue: true, RequeueAfter: time.Until(deadline)}, r.strategicMergePatch(ctx, pod, stateTrackingPatchFromPod(*pod))
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	filter, err := predicate.LabelSelectorPredicate(v1.LabelSelector{
		MatchExpressions: []v1.LabelSelectorRequirement{
			{
				Key:      instanceLabel,
				Operator: v1.LabelSelectorOpExists,
			},
		},
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}, builder.WithPredicates(filter)).
		Complete(r)
}

func (r *PodReconciler) strategicMergePatch(ctx context.Context, obj client.Object, patch map[string]any) error {
	jp, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}
	return r.Client.Patch(ctx, obj, client.RawPatch(types.StrategicMergePatchType, jp))
}

func recordSchedulingMetrics(l logr.Logger, instance string, pod *corev1.Pod, timedOut bool) error {
	tr, err := getTrackedStates(pod)
	if err != nil {
		return fmt.Errorf("failed to get state tracking timestamps: %w", err)
	}

	createdTime, ok := tr[podstate.PodCreated]
	if !ok {
		return fmt.Errorf("pod created time not found")
	}

	metricLabels := prometheus.Labels{"namespace": pod.Namespace, "name": instance}

	cm := podTimeCompleted.MustCurryWith(metricLabels)
	if timedOut {
		cm.WithLabelValues(metricTimedOutLabel).Observe(time.Since(createdTime).Seconds())
	} else {
		cm.WithLabelValues(metricCompletedLabel).Observe(time.Since(createdTime).Seconds())
	}

	acknowledgedTime, hasAcknowledgedTime := tr[podstate.PodAcknowledged]
	scheduledTime, hasScheduledTime := tr[podstate.PodScheduled]
	waitingTime, hasWaitingTime := tr[podstate.PodWaiting]

	usm := podTimeUnscheduled.MustCurryWith(metricLabels)
	if hasScheduledTime {
		usm.WithLabelValues(metricCompletedLabel).Observe(scheduledTime.Sub(createdTime).Seconds())
	} else if hasAcknowledgedTime {
		usm.WithLabelValues(metricCompletedLabel).Observe(acknowledgedTime.Sub(createdTime).Seconds())
	} else if hasWaitingTime {
		usm.WithLabelValues(metricCompletedLabel).Observe(waitingTime.Sub(createdTime).Seconds())
	} else if timedOut {
		usm.WithLabelValues(metricTimedOutLabel).Observe(time.Since(createdTime).Seconds())
	}

	uam := podTimeUntilAcknowledged.MustCurryWith(metricLabels)
	if hasAcknowledgedTime {
		uam.WithLabelValues(metricCompletedLabel).Observe(acknowledgedTime.Sub(createdTime).Seconds())
	} else if hasWaitingTime {
		uam.WithLabelValues(metricCompletedLabel).Observe(waitingTime.Sub(createdTime).Seconds())
	} else if timedOut {
		uam.WithLabelValues(metricTimedOutLabel).Observe(time.Since(createdTime).Seconds())
	}

	ptuw := podTimeUntilWaiting.MustCurryWith(metricLabels)
	if hasWaitingTime {
		ptuw.WithLabelValues(metricCompletedLabel).Observe(waitingTime.Sub(createdTime).Seconds())
	} else if timedOut {
		ptuw.WithLabelValues(metricTimedOutLabel).Observe(time.Since(createdTime).Seconds())
	} else {
		return fmt.Errorf("pod has no waiting time")
	}

	return nil
}

func podHasReachedTimeout(pod corev1.Pod, timeout time.Duration) bool {
	if pod.CreationTimestamp.IsZero() {
		return false
	}
	return pod.CreationTimestamp.Add(timeout).Before(time.Now())
}

func timeoutFromPod(pod *corev1.Pod) (time.Duration, error) {
	timeout := pod.GetAnnotations()[timeoutAnnotation]
	if timeout == "" {
		return 0, fmt.Errorf("timeout annotation not found or empty")
	}
	return time.ParseDuration(timeout)
}
