package podstate

import corev1 "k8s.io/api/core/v1"

type PodState string

const (
	PodCreated      PodState = "created"
	PodScheduled    PodState = "scheduled"
	PodAcknowledged PodState = "acknowledged"
	PodWaiting      PodState = "waiting"
	PodRunning      PodState = "running"
	PodCompleted    PodState = "completed"
	PodFailed       PodState = "failed"
)

func State(pod corev1.Pod) PodState {
	if isFailed(pod) {
		return PodFailed
	}

	if isCompleted(pod) {
		return PodCompleted
	}

	if isRunning(pod) {
		return PodRunning
	}

	if isWaiting(pod) {
		return PodWaiting
	}

	if isAcknowledged(pod) {
		return PodAcknowledged
	}

	if isScheduled(pod) {
		return PodScheduled
	}

	return PodCreated
}

func isScheduled(pod corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isAcknowledged(pod corev1.Pod) bool {
	return pod.Status.StartTime != nil
}

func isWaiting(pod corev1.Pod) bool {
	return len(pod.Status.ContainerStatuses) > 0
}

func isRunning(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning
}

func isCompleted(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
}

func isFailed(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodFailed
}
