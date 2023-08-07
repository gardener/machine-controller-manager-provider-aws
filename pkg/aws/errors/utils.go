package errors

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
)

// GetMCMErrorCode takes the error returned from the EC2API and returns the corresponding MCM error code.
func GetMCMErrorCode(err error) codes.Code {
	awsErr := err.(awserr.Error)
	return mapErrorCode(awsErr.Code())
}

func mapErrorCode(errCode string) codes.Code {
	switch errCode {
	case InsufficientCapacity, InsufficientAddressCapacity, InsufficientInstanceCapacity, InsufficientVolumeCapacity:
		return codes.ResourceExhausted
	case InstanceLimitExceeded, VcpuLimitExceeded, VolumeLimitExceeded, MaxIOPSLimitExceeded:
		return codes.QuotaExhausted
	case TagLimitExceeded, PrivateIpAddressLimitExceeded, AttachmentLimitExceeded, NetworkInterfaceLimitExceeded, VolumeIOPSLimit:
		return codes.InvalidArgument
	default:
		return codes.Internal
	}
}
