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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SchedulerCanarySpec defines the desired state of SchedulerCanary
type SchedulerCanarySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of SchedulerCanary. Edit schedulercanary_types.go to remove/update
	Foo string `json:"foo,omitempty"`

	// PodTemplate is the pod template to use for the canary pods.
	PodTemplate corev1.PodTemplateSpec `json:"podTemplate,omitempty"`
}

// SchedulerCanaryStatus defines the observed state of SchedulerCanary
type SchedulerCanaryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PodName is the name of the canary pod.
	PodName string `json:"podName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="POD",type="string",JSONPath=`.status.podName`

// SchedulerCanary is the Schema for the schedulercanaries API
type SchedulerCanary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulerCanarySpec   `json:"spec,omitempty"`
	Status SchedulerCanaryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SchedulerCanaryList contains a list of SchedulerCanary
type SchedulerCanaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SchedulerCanary `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SchedulerCanary{}, &SchedulerCanaryList{})
}
