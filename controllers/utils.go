package controllers

import (
	"fmt"
	"time"

	"github.com/appuio/scheduler-canary-controller/podstate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const StateTrackingAnnotation = "scheduler-canary-controller.appuio.io/state-tracking"

type TrackedStates map[podstate.PodState]time.Time

func buildPodFromTemplate(template *corev1.PodTemplateSpec, parentObject runtime.Object, suffix string) (*corev1.Pod, error) {
	accessor, err := meta.Accessor(parentObject)
	if err != nil {
		return nil, fmt.Errorf("parentObject does not have ObjectMeta, %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   accessor.GetNamespace(),
			Labels:      copyStringMap(template.Labels),
			Annotations: copyStringMap(template.Annotations),
			Name:        suffixLimit(accessor.GetName(), suffix),
			Finalizers:  copyStringSlice(template.Finalizers),
		},
	}
	pod.Spec = *template.Spec.DeepCopy()
	return pod, nil
}

func suffixLimit(str, suffix string) string {
	maxStrLen := 63 - len(suffix)
	if len(str) > maxStrLen {
		return str[:maxStrLen] + suffix
	}
	return str + suffix
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyStringSlice(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
