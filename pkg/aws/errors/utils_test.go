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
