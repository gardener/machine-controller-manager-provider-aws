package aws

import (
	"errors"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"strconv"
	"time"
)

const prometheusProviderLabelValue = "aws"

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
		}
		metrics.DriverFailedAPIRequests.
			WithLabelValues(labels...).
			Inc()
		return
	}
	// compute the time taken to complete the AZ service call and record it as a metric
	elapsed := time.Since(invocationTime)
	metrics.DriverAPIRequestDuration.WithLabelValues(
		prometheusProviderLabelValue,
		operation,
	).Observe(elapsed.Seconds())
}
