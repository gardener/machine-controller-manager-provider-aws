// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
