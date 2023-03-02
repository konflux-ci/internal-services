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
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var _ = Describe("Metrics InternalRequest", Ordered, func() {
	BeforeAll(func() {

		// We need to unregister in advance otherwise it breaks with 'AlreadyRegisteredError'
		metrics.Registry.Unregister(InternalRequestAttemptConcurrentTotal)
		metrics.Registry.Unregister(InternalRequestAttemptDurationSeconds)
	})

	var (
		attemptDurationSecondsHeader = inputHeader{
			Name: "internal_request_attempt_duration_seconds",
			Help: "Time from the moment the InternalRequest starts being processed until it completes",
		}
		attemptTotalHeader = inputHeader{
			Name: "internal_request_attempt_total",
			Help: "Total number of InternalRequests processed by the operator",
		}
	)

	const (
		defaultNamespace             = "default"
		validInternalRequestReason   = "valid_internalrequest_reason"
		invalidInternalRequestReason = "invalid_internalrequest_reason"
		strategy                     = "nostrategy"
	)

	Context("When RegisterCompletedInternalRequest is called", func() {
		BeforeAll(func() {
			InternalRequestAttemptDurationSeconds = prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "internal_request_attempt_duration_seconds",
					Help:    "Time from the moment the InternalRequest starts being processed until it completes",
					Buckets: []float64{60, 600, 1800, 3600},
				},
				[]string{"request", "namespace", "reason", "succeeded"},
			)

			InternalRequestAttemptConcurrentTotal = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: "internal_request_attempt_concurrent_requests",
					Help: "Total number of concurrent InternalRequest attempts",
				},
			)
			metrics.Registry.MustRegister(InternalRequestAttemptDurationSeconds, InternalRequestAttemptConcurrentTotal)
		})

		AfterAll(func() {
			metrics.Registry.Unregister(InternalRequestAttemptDurationSeconds)
			metrics.Registry.Unregister(InternalRequestAttemptConcurrentTotal)
		})

		// Input seconds for duration of operations less or equal to the following buckets of 60, 600, 1800 and 3600 seconds
		inputSeconds := []float64{30, 500, 1500, 3000}
		elapsedSeconds := 0.0
		labels := fmt.Sprintf(`namespace="%s", reason="%s", request="%s", succeeded="true",`,
			defaultNamespace, validInternalRequestReason, "iib")

		It("increments 'InternalRequestAttemptConcurrentTotal' so we can decrement it to a non-negative number in the next test", func() {
			creationTime := metav1.Time{}
			for _, seconds := range inputSeconds {
				startTime := metav1.NewTime(creationTime.Add(time.Second * time.Duration(seconds)))
				RegisterNewInternalRequest(creationTime, &startTime)
			}
			Expect(testutil.ToFloat64(InternalRequestAttemptConcurrentTotal)).To(Equal(float64(len(inputSeconds))))
		})

		It("increments 'InternalRequestAttemptTotal' and decrements 'InternalRequestAttemptConcurrentTotal'", func() {
			completionTime := metav1.Time{}
			for _, seconds := range inputSeconds {
				completionTime := metav1.NewTime(completionTime.Add(time.Second * time.Duration(seconds)))
				elapsedSeconds += seconds
				RegisterCompletedInternalRequest("iib", defaultNamespace, validInternalRequestReason, &metav1.Time{}, &completionTime, true)
			}
			readerData := createCounterReader(attemptTotalHeader, labels, true, len(inputSeconds))
			Expect(testutil.ToFloat64(InternalRequestAttemptConcurrentTotal)).To(Equal(0.0))
			Expect(testutil.CollectAndCompare(InternalRequestAttemptTotal, strings.NewReader(readerData))).To(Succeed())
		})

		It("registers a new observation for 'InternalRequestAttemptDurationSeconds' with the elapsed time from the moment the InternalRequest attempt started (InternalRequest marked as 'Running').", func() {
			timeBuckets := []string{"60", "600", "1800", "3600"}
			// For each time bucket how many InternalRequests completed below 4 seconds
			data := []int{1, 2, 3, 4}
			readerData := createHistogramReader(attemptDurationSecondsHeader, timeBuckets, data, labels, elapsedSeconds, len(inputSeconds))
			Expect(testutil.CollectAndCompare(InternalRequestAttemptDurationSeconds, strings.NewReader(readerData))).To(Succeed())
		})
	})

})
