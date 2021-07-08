/**
	Overview
		- Tests the provider specific Machine Controller
	Prerequisites
		- secret yaml file for the hyperscaler/provider passed as input
		- control cluster and target clusters kube-config passed as input (optional)
	BeforeSuite
		- Check and create control cluster and target clusters if required
		- Check and create crds ( machineclass, machines, machinesets and machinedeployment ) if required
		  using file available in kubernetes/crds directory of machine-controller-manager repo
		- Start the Machine Controller manager ( as goroutine )
		- apply secret resource for accesing the cloud provider service in the control cluster
		- Create machineclass resource from file available in kubernetes directory of provider specific repo in control cluster
	AfterSuite
		- Delete the control and target clusters // As of now we are reusing the cluster so this is not required

	Test: differentRegion Scheduling Strategy Test
        1) Create machine in region other than where the target cluster exists. (e.g machine in eu-west-1 and target cluster exists in us-east-1)
           Expected Output
			 - should fail because no cluster in same region exists)

    Test: sameRegion Scheduling Strategy Test
        1) Create machine in same region/zone as target cluster and attach it to the cluster
           Expected Output
			 - should successfully attach the machine to the target cluster (new node added)
		2) Delete machine
			Expected Output
			 - should successfully delete the machine from the target cluster (less one node)
 **/

package controller_test

import (
	"github.com/gardener/machine-controller-manager-provider-aws/test/integration/provider"
	"github.com/gardener/machine-controller-manager/pkg/test/integration/common"
	. "github.com/onsi/ginkgo"
)

var commons = common.NewIntegrationTestFramework(&provider.ResourcesTrackerImpl{})

var _ = BeforeSuite(commons.SetupBeforeSuite)

var _ = AfterSuite(commons.Cleanup)

var _ = Describe("Machine controllers test", func() {
	commons.BeforeEachCheck()
	commons.ControllerTests()
})
