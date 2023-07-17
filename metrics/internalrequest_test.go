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
	"github.com/redhat-appstudio/operator-toolkit/test"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

var _ = Describe("Metrics InternalRequest", Ordered, func() {

	var (
		initializeMetrics func()
	)

	When("RegisterCompletedInternalRequest is called", func() {
		var completionTime, startTime *metav1.Time

		BeforeEach(func() {
			initializeMetrics()

			completionTime = &metav1.Time{}
			startTime = &metav1.Time{Time: completionTime.Add(-60 * time.Second)}
		})

		It("does nothing if the start time is nil", func() {
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(0)))
			RegisterCompletedInternalRequest(nil, completionTime, "", "", "")
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(0)))
		})

		It("does nothing if the completion time is nil", func() {
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(0)))
			RegisterCompletedInternalRequest(startTime, nil, "", "", "")
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(0)))
		})

		It("decrements InternalRequestConcurrentTotal", func() {
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(0)))
			RegisterCompletedInternalRequest(startTime, completionTime, "", "", "")
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(-1)))
		})

		It("adds an observation to InternalRequestDurationSeconds", func() {
			RegisterCompletedInternalRequest(startTime, completionTime,
				internalRequestDurationSecondsLabels[0],
				internalRequestDurationSecondsLabels[1],
				internalRequestDurationSecondsLabels[2],
			)
			Expect(testutil.CollectAndCompare(InternalRequestDurationSeconds,
				test.NewHistogramReader(
					internalRequestDurationSecondsOpts,
					internalRequestDurationSecondsLabels,
					startTime, completionTime,
				))).To(Succeed())
		})

		It("increments InternalRequestTotal", func() {
			RegisterCompletedInternalRequest(startTime, completionTime,
				internalRequestTotalLabels[0],
				internalRequestTotalLabels[1],
				internalRequestTotalLabels[2],
			)
			Expect(testutil.CollectAndCompare(InternalRequestTotal,
				test.NewCounterReader(
					internalRequestTotalOpts,
					internalRequestTotalLabels,
				))).To(Succeed())
		})
	})

	When("RegisterNewRelease is called", func() {
		BeforeEach(func() {
			initializeMetrics()
		})

		It("increments ReleaseConcurrentTotal", func() {
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(0)))
			RegisterNewInternalRequest()
			Expect(testutil.ToFloat64(InternalRequestConcurrentTotal.WithLabelValues())).To(Equal(float64(1)))
		})
	})

	initializeMetrics = func() {
		InternalRequestTotal.Reset()
		InternalRequestConcurrentTotal.Reset()
		InternalRequestDurationSeconds.Reset()
	}

})
