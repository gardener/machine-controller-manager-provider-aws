// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package instrument

import (
	"errors"
	"strconv"
	"testing"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var (
	testErr          = errors.New("test-error")
	defaultErrorCode = strconv.Itoa(int(codes.Internal))
	testStatusErr    = status.New(codes.InvalidArgument, "test-status-error")
)

const serviceName = "test-service"

func TestAPIMetricRecorderFn(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{"assert that function captures failed API request count when the error is not nil", testErr},
		{"assert that function captures successful API request count when the error is nil", nil},
	}
	g := NewWithT(t)
	reg := prometheus.NewRegistry()
	g.Expect(reg.Register(metrics.APIRequestCount)).To(Succeed())
	g.Expect(reg.Register(metrics.APIFailedRequestCount)).To(Succeed())
	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			defer metrics.APIRequestCount.Reset()
			defer metrics.APIFailedRequestCount.Reset()
			defer metrics.APIRequestDuration.Reset()
			_ = deferredMetricsRecorderInvoker(tc.err != nil, false, AwsAPIMetricRecorderFn)
			if tc.err != nil {
				g.Expect(testutil.CollectAndCount(metrics.APIRequestCount)).To(Equal(0))
				g.Expect(testutil.CollectAndCount(metrics.APIFailedRequestCount)).To(Equal(1))
				g.Expect(testutil.ToFloat64(metrics.APIFailedRequestCount.WithLabelValues(prometheusProviderLabelValue, serviceName))).To(Equal(float64(1)))
			} else {
				g.Expect(testutil.CollectAndCount(metrics.APIRequestCount)).To(Equal(1))
				g.Expect(testutil.CollectAndCount(metrics.APIFailedRequestCount)).To(Equal(0))
				g.Expect(testutil.ToFloat64(metrics.APIRequestCount.WithLabelValues(prometheusProviderLabelValue, serviceName))).To(Equal(float64(1)))
			}
		})
	}
}

func TestDriverAPIMetricRecorderFn(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{"assert that function captures failed driver API request with default error code for internal error when there is an error", testErr},
		{"assert that function captures failed driver API request with error code from status.Status on error", testStatusErr},
		{"assert that function captures successful driver API request count when the error is nil", nil},
	}
	g := NewWithT(t)
	reg := prometheus.NewRegistry()
	g.Expect(reg.Register(metrics.DriverFailedAPIRequests)).To(Succeed())
	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			defer metrics.DriverFailedAPIRequests.Reset()
			_ = deferredMetricsRecorderInvoker(tc.err != nil, isStatusErr(tc.err), DriverAPIMetricRecorderFn)
			if tc.err != nil {
				expectedErrCode := getExpectedErrorCode(tc.err)
				g.Expect(testutil.CollectAndCount(metrics.DriverFailedAPIRequests)).To(Equal(1))
				g.Expect(testutil.ToFloat64(metrics.DriverFailedAPIRequests.WithLabelValues(prometheusProviderLabelValue, serviceName, expectedErrCode))).To(Equal(float64(1)))
			} else {
				g.Expect(testutil.CollectAndCount(metrics.DriverFailedAPIRequests)).To(Equal(0))
			}
		})
	}
}

func isStatusErr(err error) bool {
	if err == nil {
		return false
	}
	var statusErr *status.Status
	return errors.As(err, &statusErr)
}

func getExpectedErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var statusErr *status.Status
	if errors.As(err, &statusErr) {
		return strconv.Itoa(int(statusErr.Code()))
	} else {
		return defaultErrorCode
	}
}

type recorderFn func(string, *error) func()

func deferredMetricsRecorderInvoker(shouldReturnErr bool, isStatusErr bool, fn recorderFn) (err error) {
	defer fn(serviceName, &err)()
	if shouldReturnErr {
		if isStatusErr {
			err = testStatusErr
		} else {
			err = testErr
		}
	}
	return
}
