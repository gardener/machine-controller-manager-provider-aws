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

func TestGetMCMErrorCodeForCreateMachine(t *testing.T) {
	table := []input{
		{inputError: awserr.New(InsufficientCapacity, "insufficient-capacity", errors.New("insufficient capacity")), expectedCode: codes.ResourceExhausted},
		{inputError: awserr.New("UnknownError", "unknown-error", errors.New("unknown error")), expectedCode: codes.Internal},
	}
	g := NewWithT(t)
	for _, entry := range table {
		g.Expect(GetMCMErrorCodeForCreateMachine(entry.inputError)).To(Equal(entry.expectedCode))
	}
}

func TestGetMCMErrorCodeForTerminateInstances(t *testing.T) {
	table := []input{
		{inputError: awserr.New(InstanceIDNotFound, "instance-id-not-found", errors.New("instance id not found")), expectedCode: codes.NotFound},
		{inputError: awserr.New("UnknownError", "unknown-error", errors.New("unknown error")), expectedCode: codes.Internal},
	}
	g := NewWithT(t)
	for _, entry := range table {
		g.Expect(GetMCMErrorCodeForTerminateInstances(entry.inputError)).To(Equal(entry.expectedCode))
	}
}
