package aws

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMachineControllerManagerProviderAWSSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machine Controller Manager Provider AWS Suite")
}
