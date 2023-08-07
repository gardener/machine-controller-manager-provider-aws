package errors

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	. "github.com/onsi/gomega"
	"testing"
)

type input struct {
	inputError   error
	expectedCode codes.Code
}

func TestGetMCMErrorCode(t *testing.T) {
	table := []input{
		{inputError: awserr.New(InstanceLimitExceeded, "instance-limit-exceeded", errors.New("instance limit exceeded")), expectedCode: codes.QuotaExhausted},
		{inputError: awserr.New(InsufficientCapacity, "insufficient-capacity", errors.New("insufficient capacity")), expectedCode: codes.ResourceExhausted},
		{inputError: awserr.New(TagLimitExceeded, "tag-limit-exceeded", errors.New("tag limit exceeded")), expectedCode: codes.InvalidArgument},
		{inputError: awserr.New("UnknownError", "unknown-error", errors.New("unknown error")), expectedCode: codes.Internal},
	}
	g := NewWithT(t)
	for _, entry := range table {
		g.Expect(GetMCMErrorCode(entry.inputError)).To(Equal(entry.expectedCode))
	}
}
