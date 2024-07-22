package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1beta1 "github.com/appuio/scheduler-canary-controller/api/v1beta1"
)

var _ = Describe("SchedulerCanary controller", func() {
	Context("Creating a SchedulerCanary Manifest", func() {
		var creationTime time.Time
		BeforeEach(func() {
			ctx := context.Background()

			schedulerCanary := &monitoringv1beta1.SchedulerCanary{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-canary",
					Namespace: "default",
				},
				Spec: monitoringv1beta1.SchedulerCanarySpec{
					Interval:                metav1.Duration{Duration: time.Minute},
					MaxPodCompletionTimeout: metav1.Duration{Duration: time.Minute},
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "scheduler-canary",
									Image: "busybox",
								},
							},
						},
					},
				},
			}
			creationTime = time.Now()
			Expect(k8sClient.Create(ctx, schedulerCanary)).Should(Succeed())
		})

		It("should create a pod", func() {
			ctx := context.Background()

			Eventually(func() (int, error) {
				pods := &corev1.PodList{}

				err := k8sClient.List(ctx, pods, client.InNamespace("default"), client.MatchingLabels(map[string]string{
					instanceLabel: "my-canary",
				}))

				return len(pods.Items), err
			}, "10s", "250ms").Should(BeNumerically(">=", 1))
		})

		It("should update last created status", func() {
			ctx := context.Background()
			Eventually(func() (time.Time, error) {
				schedulerCanary := &monitoringv1beta1.SchedulerCanary{}

				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "my-canary"}, schedulerCanary)

				return schedulerCanary.Status.LastCanaryCreatedAt.Time, err
			}, "10s", "250ms").Should(BeTemporally(">=", creationTime.Truncate(time.Second)))
		})

	})

	AfterEach(func() {
		ctx := context.Background()

		k8sClient.DeleteAllOf(ctx, &monitoringv1beta1.SchedulerCanary{}, client.InNamespace("default"))
		k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("default"))
	})
})
