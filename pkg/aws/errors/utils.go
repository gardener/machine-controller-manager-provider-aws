package errors

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
)

// GetMCMErrorCodeForCreateMachine takes the error returned from the EC2API during the CreateMachine call and returns the corresponding MCM error code.
func GetMCMErrorCodeForCreateMachine(err error) codes.Code {
	awsErr := err.(awserr.Error)
	return mapErrorCodeForCreateMachine(awsErr.Code())
}

func mapErrorCodeForCreateMachine(errCode string) codes.Code {
	switch errCode {
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
