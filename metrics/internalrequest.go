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

package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	InternalRequestAttemptConcurrentTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "internal_request_attempt_concurrent_requests",
			Help: "Total number of concurrent InternalRequest attempts",
		},
	)

	InternalRequestAttemptDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "internal_request_attempt_duration_seconds",
			Help:    "Time from the moment the InternalRequest starts being processed until it completes",
			Buckets: []float64{10, 20, 40, 60, 150, 300, 450, 900, 1800, 3600},
		},
		[]string{"request", "namespace", "reason", "succeeded"},
	)

	InternalRequestAttemptTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "internal_request_attempt_total",
			Help: "Total number of InternalRequests processed by the operator",
		},
		[]string{"request", "namespace", "reason", "succeeded"},
	)
)

// RegisterCompletedInternalRequest decrements the 'internal_request_attempt_concurrent_total' metric, increments `internal_request_attempt_total`
// and registers a new observation for 'internal_request_attempt_duration_seconds' with the elapsed time from the moment the
// InternalRequest attempt started (InternalRequest marked as 'Running').
func RegisterCompletedInternalRequest(request, namespace, reason string, startTime, completionTime *metav1.Time, succeeded bool) {
	labels := prometheus.Labels{
		"request":   request,
		"namespace": namespace,
		"reason":    reason,
		"succeeded": strconv.FormatBool(succeeded),
	}
	InternalRequestAttemptConcurrentTotal.Dec()
	InternalRequestAttemptDurationSeconds.With(labels).Observe(completionTime.Sub(startTime.Time).Seconds())
	InternalRequestAttemptTotal.With(labels).Inc()
}

// RegisterNewInternalRequest increments the number of the 'internal_request_attempt_concurrent_total' metric which represents the number of concurrent running InternalRequests.
func RegisterNewInternalRequest(creationTime metav1.Time, startTime *metav1.Time) {
	InternalRequestAttemptConcurrentTotal.Inc()
}

func init() {
	metrics.Registry.MustRegister(
		InternalRequestAttemptConcurrentTotal,
		InternalRequestAttemptDurationSeconds,
		InternalRequestAttemptTotal,
	)
}
