package validation

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMachineControllerManagerProviderAWSAPIValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machine Controller Manager Provider AWS API Validation")
}
