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

	"github.com/appuio/scheduler-canary-controller/controllers/podstate"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
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

	// ensure we check timeouts very earlier so pods still get deleted even if we mess up later
	timeout, err := timeoutFromPod(pod)
	if err != nil {
		l.Error(err, "Pod has no timeout annotation, deleting immediately")
	}
	if podHasReachedTimeout(*pod, timeout) {
		l.Info("Pod has reached timeout, deleting")
		podsTimeouted.WithLabelValues(pod.Namespace, instance).Inc()
		return ctrl.Result{}, r.Client.Delete(ctx, pod)
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

func calculateTimes(l logr.Logger, instance string, pod *corev1.Pod) error {
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

	usm := podTimeUnscheduled.WithLabelValues(pod.Namespace, instance)
	if hasScheduledTime {
		usm.Observe(scheduledTime.Sub(createdTime).Seconds())
	} else if hasAcknowledgedTime {
		usm.Observe(acknowledgedTime.Sub(createdTime).Seconds())
	} else if hasWaitingTime {
		usm.Observe(waitingTime.Sub(createdTime).Seconds())
	}

	uam := podTimeUntilAcknowledged.WithLabelValues(pod.Namespace, instance)
	if hasAcknowledgedTime {
		uam.Observe(acknowledgedTime.Sub(createdTime).Seconds())
	} else if hasWaitingTime {
		uam.Observe(waitingTime.Sub(createdTime).Seconds())
	}

	if hasWaitingTime {
		podTimeUntilWaiting.
			WithLabelValues(pod.Namespace, instance).
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

func timeoutFromPod(pod *corev1.Pod) (time.Duration, error) {
	timeout := pod.GetAnnotations()[timeoutAnnotation]
	if timeout == "" {
		return 0, fmt.Errorf("timeout annotation not found or empty")
	}
	return time.ParseDuration(timeout)
}
