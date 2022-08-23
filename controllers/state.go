package controllers

import (
	"encoding/json"
	"time"

	"github.com/appuio/scheduler-canary-controller/podstate"
	corev1 "k8s.io/api/core/v1"
)

const StateTrackingAnnotation = "scheduler-canary-controller.appuio.io/state-tracking"

type TrackedStates map[podstate.PodState]time.Time

func stateTrackingPatchFromPod(pod corev1.Pod) map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				StateTrackingAnnotation: pod.Annotations[StateTrackingAnnotation],
			},
		},
	}
}

func getTrackedStates(pod *corev1.Pod) (TrackedStates, error) {
	var tr TrackedStates

	if a, exists := pod.Annotations[StateTrackingAnnotation]; exists {
		if err := json.Unmarshal([]byte(a), &tr); err != nil {
			return nil, err
		}
		return tr, nil
	}

	return make(TrackedStates), nil
}

func trackState(pod *corev1.Pod, state podstate.PodState) error {
	tr, err := getTrackedStates(pod)
	if err != nil {
		return err
	}

	if _, exists := tr[state]; !exists {
		tr[state] = time.Now()
	}

	s, err := json.Marshal(tr)
	pod.Annotations[StateTrackingAnnotation] = string(s)
	return err
}
