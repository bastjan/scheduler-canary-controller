//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulerCanary) DeepCopyInto(out *SchedulerCanary) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulerCanary.
func (in *SchedulerCanary) DeepCopy() *SchedulerCanary {
	if in == nil {
		return nil
	}
	out := new(SchedulerCanary)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SchedulerCanary) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulerCanaryList) DeepCopyInto(out *SchedulerCanaryList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SchedulerCanary, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulerCanaryList.
func (in *SchedulerCanaryList) DeepCopy() *SchedulerCanaryList {
	if in == nil {
		return nil
	}
	out := new(SchedulerCanaryList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SchedulerCanaryList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulerCanarySpec) DeepCopyInto(out *SchedulerCanarySpec) {
	*out = *in
	out.MaxPodCompletionTimeout = in.MaxPodCompletionTimeout
	out.Interval = in.Interval
	in.PodTemplate.DeepCopyInto(&out.PodTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulerCanarySpec.
func (in *SchedulerCanarySpec) DeepCopy() *SchedulerCanarySpec {
	if in == nil {
		return nil
	}
	out := new(SchedulerCanarySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulerCanaryStatus) DeepCopyInto(out *SchedulerCanaryStatus) {
	*out = *in
	in.LastCanaryCreatedAt.DeepCopyInto(&out.LastCanaryCreatedAt)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulerCanaryStatus.
func (in *SchedulerCanaryStatus) DeepCopy() *SchedulerCanaryStatus {
	if in == nil {
		return nil
	}
	out := new(SchedulerCanaryStatus)
	in.DeepCopyInto(out)
	return out
}
