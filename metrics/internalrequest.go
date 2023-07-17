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
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	InternalRequestConcurrentTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "internal_request_concurrent_requests",
			Help: "Total number of concurrent InternalRequest attempts",
		},
		[]string{},
	)

	InternalRequestDurationSeconds = prometheus.NewHistogramVec(
		internalRequestDurationSecondsOpts,
		internalRequestDurationSecondsLabels,
	)
	internalRequestDurationSecondsLabels = []string{
		"namespace",
		"reason",
		"request",
	}
	internalRequestDurationSecondsOpts = prometheus.HistogramOpts{
		Name:    "internal_request_duration_seconds",
		Help:    "Time from the moment the InternalRequest starts being processed until it completes",
		Buckets: []float64{10, 20, 40, 60, 150, 300, 450, 900, 1800, 3600},
	}

	InternalRequestTotal = prometheus.NewCounterVec(
		internalRequestTotalOpts,
		internalRequestTotalLabels,
	)
	internalRequestTotalLabels = []string{
		"namespace",
		"reason",
		"request",
	}
	internalRequestTotalOpts = prometheus.CounterOpts{
		Name: "internal_request_total",
		Help: "Total number of InternalRequests processed by the operator",
	}
)

// RegisterCompletedInternalRequest registers an InternalRequest execution as complete, adding a new
// observation for the InternalRequest duration and decreasing the number of concurrent executions.
// If either the startTime or the completionTime parameters are nil, no action will be taken.
func RegisterCompletedInternalRequest(startTime, completionTime *metav1.Time, namespace, reason, request string) {
	if startTime == nil || completionTime == nil {
		return
	}

	labels := prometheus.Labels{
		"namespace": namespace,
		"reason":    reason,
		"request":   request,
	}
	InternalRequestConcurrentTotal.WithLabelValues().Dec()
	InternalRequestDurationSeconds.With(labels).Observe(completionTime.Sub(startTime.Time).Seconds())
	InternalRequestTotal.With(labels).Inc()
}

// RegisterNewInternalRequest register a new InternalRequest, increasing the number of concurrent requests.
func RegisterNewInternalRequest() {
	InternalRequestConcurrentTotal.WithLabelValues().Inc()
}

func init() {
	metrics.Registry.MustRegister(
		InternalRequestConcurrentTotal,
		InternalRequestDurationSeconds,
		InternalRequestTotal,
	)
}
