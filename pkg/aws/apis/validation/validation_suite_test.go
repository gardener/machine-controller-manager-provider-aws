// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMachineControllerManagerProviderAWSAPIValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machine Controller Manager Provider AWS API Validation")
}
