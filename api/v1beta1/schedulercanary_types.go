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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SchedulerCanarySpec defines the desired state of SchedulerCanary
type SchedulerCanarySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// MaxPodCompletionTimeout is the maximum amount of time to wait for a pod to complete.
	// After the timeout expires, the pod will be deleted, the canary will be marked as failed, and a new canary will be created.
	// The default is 15 minutes.
	MaxPodCompletionTimeout time.Duration `json:"maxPodCompletionTimeout,omitempty"`

	// PodTemplate is the pod template to use for the canary pods.
	PodTemplate corev1.PodTemplateSpec `json:"podTemplate,omitempty"`
}

func (s SchedulerCanarySpec) MaxPodCompletionTimeoutWithDefault() time.Duration {
	if s.MaxPodCompletionTimeout == 0 {
		return 15 * time.Minute
	}
	return s.MaxPodCompletionTimeout
}

// SchedulerCanaryStatus defines the observed state of SchedulerCanary
type SchedulerCanaryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
