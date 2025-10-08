// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"testing"

	"github.com/aws/smithy-go"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	. "github.com/onsi/gomega"
)

type input struct {
	inputError   error
	expectedCode codes.Code
}

func TestGetMCMErrorCodeForCreateMachine(t *testing.T) {
	table := []input{
		{inputError: &smithy.GenericAPIError{Code: "InsufficientCapacity"}, expectedCode: codes.ResourceExhausted},
		{inputError: &smithy.GenericAPIError{Code: "unknown error"}, expectedCode: codes.Internal},
	}
	g := NewWithT(t)
	for _, entry := range table {
		g.Expect(GetMCMErrorCodeForCreateMachine(entry.inputError)).To(Equal(entry.expectedCode))
	}
}

func TestGetMCMErrorCodeForTerminateInstances(t *testing.T) {
	table := []input{
		{inputError: &smithy.GenericAPIError{Code: "InvalidInstanceID.NotFound"}, expectedCode: codes.NotFound},
		{inputError: &smithy.GenericAPIError{Code: "unknown error"}, expectedCode: codes.Internal},
	}
	g := NewWithT(t)
	for _, entry := range table {
		g.Expect(GetMCMErrorCodeForTerminateInstances(entry.inputError)).To(Equal(entry.expectedCode))
	}
}
