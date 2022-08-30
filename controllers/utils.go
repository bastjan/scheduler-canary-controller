package controllers

import (
	monitoringv1beta1 "github.com/appuio/scheduler-canary-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildPodFromTemplate(template *corev1.PodTemplateSpec, instance *monitoringv1beta1.SchedulerCanary) (*corev1.Pod, error) {
	labels := copyStringMap(template.Labels)
	labels[instanceLabel] = instance.Name

	annotations := copyStringMap(template.Annotations)
	annotations[timeoutAnnotation] = instance.Spec.MaxPodCompletionTimeout.Duration.String()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    instance.GetNamespace(),
			Labels:       labels,
			Annotations:  annotations,
			GenerateName: suffixLimit(instance.GetName(), "-"),
			Finalizers:   copyStringSlice(template.Finalizers),
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
