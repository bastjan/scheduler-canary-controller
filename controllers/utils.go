package controllers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func buildPodFromTemplate(template *corev1.PodTemplateSpec, parentObject runtime.Object) (*corev1.Pod, error) {
	accessor, err := meta.Accessor(parentObject)
	if err != nil {
		return nil, fmt.Errorf("parentObject does not have ObjectMeta, %v", err)
	}
	prefix := fmt.Sprintf("%s-", accessor.GetName())

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    accessor.GetNamespace(),
			Labels:       copyStringMap(template.Labels),
			Annotations:  copyStringMap(template.Annotations),
			GenerateName: prefix,
			Finalizers:   copyStringSlice(template.Finalizers),
		},
	}
	pod.Spec = *template.Spec.DeepCopy()
	return pod, nil
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
