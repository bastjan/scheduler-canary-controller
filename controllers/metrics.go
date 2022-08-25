package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	metricTimedOutLabel  = "timed_out"
	metricCompletedLabel = "completed"
)

var podTimeUnscheduled = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "scheduler_canary_pod_unscheduled_seconds",
	Help: "Time spent in pending state",
}, []string{"namespace", "name", "reason"})

var podTimeUntilAcknowledged = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "scheduler_canary_pod_until_acknowledged_seconds",
	Help: "Time spent in an unacknowledged state",
}, []string{"namespace", "name", "reason"})

var podTimeUntilWaiting = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "scheduler_canary_pod_until_waiting_seconds",
	Help: "Time spent before pulling images mounting volumes",
}, []string{"namespace", "name", "reason"})

var podTimeCompleted = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "scheduler_canary_pods_completed_seconds",
	Help: "Time spent from creation to completion or from creation to the the specified .maxPodCompletionTimeout timeout",
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
