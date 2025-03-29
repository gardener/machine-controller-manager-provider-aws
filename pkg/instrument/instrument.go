// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package instrument

import (
	"errors"
	"strconv"
	"time"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
)

const prometheusProviderLabelValue = "aws"

// RecordAwsAPIMetric records a prometheus metric for AWS API calls.
// * If there is an error then it will increment the APIFailedRequestCount counter vec metric.
// * If the AWS API call is successful then it will record 2 metrics:
//   - It will increment APIRequestCount counter vec metric.
//   - It will compute the time taken for API call completion and record it.
//
// NOTE: If this function is called via `defer` then please keep in mind that parameters passed to defer are evaluated at the time of definition.
// So if you have an error that is computed later in the function then ensure that you use named return parameters.
func RecordAwsAPIMetric(err error, awsServiceName string, invocationTime time.Time) {
	if err != nil {
		metrics.APIFailedRequestCount.
			WithLabelValues(prometheusProviderLabelValue, awsServiceName).
			Inc()
		return
	}

	// No error, record metrics for successful API call.
	metrics.APIRequestCount.
		WithLabelValues(
			prometheusProviderLabelValue,
			awsServiceName,
		).Inc()

	// compute the time taken to complete the AWS service call and record it as a metric
	elapsed := time.Since(invocationTime)
	metrics.APIRequestDuration.WithLabelValues(
		prometheusProviderLabelValue,
		awsServiceName,
	).Observe(elapsed.Seconds())
}

// RecordDriverAPIMetric records a prometheus metric capturing the total duration of a successful execution for
// any driver method (e.g. CreateMachine, DeleteMachine etc.). In case an error is returned then a failed counter
// metric is recorded.
func RecordDriverAPIMetric(err error, operation string, invocationTime time.Time) {
	if err != nil {
		var (
			statusErr *status.Status
			labels    = []string{prometheusProviderLabelValue, operation}
		)
		if errors.As(err, &statusErr) {
			labels = append(labels, strconv.Itoa(int(statusErr.Code())))
		} else {
			labels = append(labels, strconv.Itoa(int(codes.Internal)))
		}
		metrics.DriverFailedAPIRequests.
			WithLabelValues(labels...).
			Inc()
		return
	}
	// compute the time taken to complete the AWS service call and record it as a metric
	elapsed := time.Since(invocationTime)
	metrics.DriverAPIRequestDuration.WithLabelValues(
		prometheusProviderLabelValue,
		operation,
	).Observe(elapsed.Seconds())
}

// AwsAPIMetricRecorderFn returns a function that can be used to record a prometheus metric for AWS API calls.
// NOTE: a pointer to an error (which itself is a fat interface pointer) is necessary to enable the callers of this function to enclose this call into a `defer` statement.
func AwsAPIMetricRecorderFn(awsServiceName string, err *error) func() {
	invocationTime := time.Now()
	return func() {
		RecordAwsAPIMetric(*err, awsServiceName, invocationTime)
	}
}

// DriverAPIMetricRecorderFn returns a function that can be used to record a prometheus metric for driver API calls.
// NOTE: a pointer to an error (which itself is a fat interface pointer) is necessary to enable the callers of this function to enclose this call into a `defer` statement.
func DriverAPIMetricRecorderFn(operation string, err *error) func() {
	invocationTime := time.Now()
	return func() {
		RecordDriverAPIMetric(*err, operation, invocationTime)
	}
}
