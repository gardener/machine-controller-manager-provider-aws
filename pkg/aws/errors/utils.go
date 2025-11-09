// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"errors"

	"github.com/aws/smithy-go"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
)

// GetMCMErrorCodeForCreateMachine takes the error returned from the EC2API during the CreateMachine call and returns the corresponding MCM error code.
func GetMCMErrorCodeForCreateMachine(err error) codes.Code {
	var awsErr smithy.APIError
	if errors.As(err, &awsErr) {
		switch awsErr.ErrorCode() {
		case InsufficientCapacity, InsufficientAddressCapacity, InsufficientInstanceCapacity, InsufficientVolumeCapacity, InstanceLimitExceeded, VcpuLimitExceeded, VolumeLimitExceeded, MaxIOPSLimitExceeded, RouteLimitExceeded, Unsupported:
			return codes.ResourceExhausted
		}
	}
	return codes.Internal
}

// GetMCMErrorCodeForTerminateInstances takes the error returned from the EC2API during the terminateInstance call and returns the corresponding MCM error code.
func GetMCMErrorCodeForTerminateInstances(err error) codes.Code {
	var awsErr smithy.APIError
	if errors.As(err, &awsErr) {
		switch awsErr.ErrorCode() {
		case string(InstanceIDNotFound):
			return codes.NotFound
		}
	}
	return codes.Internal
}

// IsInstanceIDNotFound checks if the provider returned an InstanceIDNotFound error
func IsInstanceIDNotFound(err error) bool {
	var awsErr smithy.APIError
	if errors.As(err, &awsErr) {
		return awsErr.ErrorCode() == string(InstanceIDNotFound)
	}
	return false
}
