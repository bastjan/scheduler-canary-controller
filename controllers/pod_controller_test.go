package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/component-base/metrics/testutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Pod controller", func() {

	Context("When a Pod with SchedulerCanary instance label is created", func() {
		const (
			namespace    = "default"
			podName      = "my-canary-pod"
			instanceName = "my-canary"
		)

		BeforeEach(func() {
			ctx := context.Background()

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
					Labels: map[string]string{
						instanceLabel: instanceName,
					},
					Annotations: map[string]string{
						timeoutAnnotation:       "1m",
						StateTrackingAnnotation: fmt.Sprintf(`{"created":"%s"}`, time.Now().Format(time.RFC3339Nano)),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "scheduler-canary",
							Image: "busybox",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
		})

		It("measures state change time on completion and deletes the pod", func() {
			ctx := context.Background()

			By("setting the pod to waiting")
			{
				var pod corev1.Pod
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod)).Should(Succeed())
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "scheduler-canary"}}
				Expect(k8sClient.Status().Update(ctx, &pod)).Should(Succeed())
			}

			By("waiting for the tracking annotation to be updated")
			Eventually(func() string {
				var pod corev1.Pod
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod)).Should(Succeed())
				return pod.Annotations[StateTrackingAnnotation]
			}).Should(ContainSubstring(`"waiting"`))

			By("setting the pod status to completed")
			{
				var pod corev1.Pod
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod)).Should(Succeed())
				pod.Status.Phase = corev1.PodSucceeded
				Expect(k8sClient.Status().Update(ctx, &pod)).Should(Succeed())
			}

			metricLabels := map[string]string{"namespace": namespace, "name": instanceName, "reason": metricCompletedLabel}
			for desc, m := range map[string]*prometheus.HistogramVec{
				"podTimeUnscheduled":       podTimeUnscheduled,
				"podTimeUntilAcknowledged": podTimeUntilAcknowledged,
				"podTimeUntilWaiting":      podTimeUntilWaiting,
				"podTimeCompleted":         podTimeCompleted,
			} {
				By(fmt.Sprintf("checking %s metric", desc))
				Eventually(func() int {
					c, err := testutil.GetHistogramMetricCount(m.With(metricLabels))
					Expect(err).ShouldNot(HaveOccurred())
					return int(c)
				}).Should(Equal(1))
			}

			By("checking if the pod is deleted")
			Eventually(func() (bool, error) {
				return checkPodNotFound(ctx, k8sClient, namespace, podName)
			}).Should(BeTrue())
		})

		It("tracks timed out pods and deletes them", func() {
			ctx := context.Background()

			By("shortening the pod timeout")
			{
				var pod corev1.Pod
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod)).Should(Succeed())
				pod.Annotations[timeoutAnnotation] = "1ms"
				Expect(k8sClient.Update(ctx, &pod)).Should(Succeed())
			}

			metricLabels := map[string]string{"namespace": namespace, "name": instanceName, "reason": metricTimedOutLabel}
			for desc, m := range map[string]*prometheus.HistogramVec{
				"podTimeUnscheduled":       podTimeUnscheduled,
				"podTimeUntilAcknowledged": podTimeUntilAcknowledged,
				"podTimeUntilWaiting":      podTimeUntilWaiting,
				"podTimeCompleted":         podTimeCompleted,
			} {
				By(fmt.Sprintf("checking %s metric", desc))
				Eventually(func() int {
					c, err := testutil.GetHistogramMetricCount(m.With(metricLabels))
					Expect(err).ShouldNot(HaveOccurred())
					return int(c)
				}).Should(Equal(1))
			}

			By("checking if the pod is deleted")
			Eventually(func() (bool, error) {
				return checkPodNotFound(ctx, k8sClient, namespace, podName)
			}).Should(BeTrue())
		})
	})

	AfterEach(func() {
		ctx := context.Background()

		podTimeCompleted.Reset()
		k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("default"))
	})

})

func checkPodNotFound(ctx context.Context, k8sClient client.Client, namespace, name string) (bool, error) {
	var pod corev1.Pod
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &pod)
	if err != nil && errors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}
