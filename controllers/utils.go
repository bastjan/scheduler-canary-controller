package controllers

import (
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1beta1 "github.com/appuio/scheduler-canary-controller/api/v1beta1"
)

func buildPodFromTemplate(template *corev1.PodTemplateSpec, instance *monitoringv1beta1.SchedulerCanary) (*corev1.Pod, error) {
	labels := make(map[string]string)
	maps.Copy(labels, template.Labels)
	labels[instanceLabel] = instance.Name

	annotations := make(map[string]string)
	maps.Copy(annotations, template.Annotations)
	annotations[timeoutAnnotation] = instance.Spec.MaxPodCompletionTimeout.Duration.String()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    instance.GetNamespace(),
			Labels:       labels,
			Annotations:  annotations,
			GenerateName: suffixLimit(instance.GetName(), "-"),
			Finalizers:   slices.Clone(template.Finalizers),
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
