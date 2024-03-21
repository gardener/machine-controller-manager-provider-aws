// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
)

// GetMCMErrorCodeForCreateMachine takes the error returned from the EC2API during the CreateMachine call and returns the corresponding MCM error code.
func GetMCMErrorCodeForCreateMachine(err error) codes.Code {
	awsErr := err.(awserr.Error)
	switch awsErr.Code() {
	case InsufficientCapacity, InsufficientAddressCapacity, InsufficientInstanceCapacity, InsufficientVolumeCapacity, InstanceLimitExceeded, VcpuLimitExceeded, VolumeLimitExceeded, MaxIOPSLimitExceeded, RouteLimitExceeded:
		return codes.ResourceExhausted
	default:
		return codes.Internal
	}
}

// GetMCMErrorCodeForTerminateInstances takes the error returned from the EC2API during the terminateInstance call and returns the corresponding MCM error code.
func GetMCMErrorCodeForTerminateInstances(err error) codes.Code {
	awsErr := err.(awserr.Error)
	switch awsErr.Code() {
	case InstanceIDNotFound:
		return codes.NotFound
	default:
		return codes.Internal
	}
}
