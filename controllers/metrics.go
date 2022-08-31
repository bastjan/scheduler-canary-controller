package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	metricTimedOutLabel  = "timed_out"
	metricCompletedLabel = "completed"
)

var (
	bucketsSchedulingTiming = []float64{0.1, 0.2, 0.3, 0.5, 1, 5, 10}
	bucketsPodCompletion    = []float64{1, 5, 10, 30}
)

var podTimeUnscheduled = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "scheduler_canary_pod_unscheduled_seconds",
	Help:    "Time spent in pending state",
	Buckets: bucketsSchedulingTiming,
}, []string{"namespace", "name", "reason"})

var podTimeUntilAcknowledged = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "scheduler_canary_pod_until_acknowledged_seconds",
	Help:    "Time spent in an unacknowledged state",
	Buckets: bucketsSchedulingTiming,
}, []string{"namespace", "name", "reason"})

var podTimeUntilWaiting = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "scheduler_canary_pod_until_waiting_seconds",
	Help:    "Time spent before pulling images, mounting volumes, and creating the pod sandbox",
	Buckets: bucketsSchedulingTiming,
}, []string{"namespace", "name", "reason"})

var podTimeCompleted = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "scheduler_canary_pod_until_completed_seconds",
	Help:    "Time spent from creation to completion or from creation to the the specified .maxPodCompletionTimeout timeout",
	Buckets: bucketsPodCompletion,
}, []string{"namespace", "name", "reason"})

func init() {
	metrics.Registry.MustRegister(
		podTimeUnscheduled,
		podTimeUntilAcknowledged,
		podTimeUntilWaiting,

		podTimeCompleted,
	)
}

// initMetrics ensures the metrics are present in the output as soon as the instance is created.
func initMetrics(namespace, name string) {
	for _, l := range [...]string{metricCompletedLabel, metricTimedOutLabel} {
		podTimeUnscheduled.WithLabelValues(namespace, name, l)
		podTimeUntilAcknowledged.WithLabelValues(namespace, name, l)
		podTimeUntilWaiting.WithLabelValues(namespace, name, l)

		podTimeCompleted.WithLabelValues(namespace, name, l)
	}
}
