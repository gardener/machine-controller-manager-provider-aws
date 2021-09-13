package provider

import (
	"fmt"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
)

//ResourcesTrackerImpl type keeps a note of resources which are initialized in MCM IT suite and are used in provider IT
type ResourcesTrackerImpl struct {
	MachineClass *v1alpha1.MachineClass
	SecretData   map[string][]byte
	ClusterName  string
}

//InitializeResourcesTracker initializes the type ResourcesTrackerImpl variable and tries
//to delete the orphan resources present before the actual IT runs.
func (r *ResourcesTrackerImpl) InitializeResourcesTracker(machineClass *v1alpha1.MachineClass, secretData map[string][]byte, clusterName string) error {

	r.MachineClass = machineClass
	r.SecretData = secretData
	r.ClusterName = clusterName

	initialVMs, initialVolumes, initialMachines, initialNICs, err := r.probeResources()
	if err != nil {
		fmt.Printf("Error in initial probe of orphaned resources: %s", err.Error())
		return err
	}

	if initialVMs != nil || initialVolumes != nil || initialMachines != nil || initialNICs != nil {
		err := fmt.Errorf("orphan resources are available. Clean them up before proceeding with the test.\nvirtual machines: %v\ndisks: %v\nmcm machines: %v\nnics: %v", initialVMs, initialVolumes, initialMachines, initialNICs)
		return err
	}
	return nil
}

// probeResources will look for resources currently available and returns them
func (r *ResourcesTrackerImpl) probeResources() ([]string, []string, []string, []string, error) {
	// Check for VM instances with matching tags/labels
	// Describe volumes attached to VM instance & delete the volumes
	// Finally delete the VM instance

	clusterTag := "tag:kubernetes.io/cluster/" + r.ClusterName
	clusterTagValue := "1"

	integrationtestTag := "tag:kubernetes.io/role/integration-test"
	integrationTestTagValue := "1"

	orphanVMs, err := getOrphanedInstances(integrationtestTag, integrationTestTagValue, r.MachineClass, r.SecretData)
	if err != nil {
		return orphanVMs, nil, nil, nil, err
	}

	// Check for available volumes in cloud provider with tag/label [Status:available]
	orphanVols, err := getOrphanedDisks(integrationtestTag, integrationTestTagValue, r.MachineClass, r.SecretData)
	if err != nil {
		return orphanVMs, orphanVols, nil, nil, err
	}

	availMachines, err := getMachines(r.MachineClass, r.SecretData)
	if err != nil {
		return orphanVMs, orphanVols, availMachines, nil, err
	}

	orphanNICs, err := getOrphanedNICs(clusterTag, clusterTagValue, r.MachineClass, r.SecretData)

	return orphanVMs, orphanVols, availMachines, orphanNICs, err

}

// IsOrphanedResourcesAvailable checks whether there are any orphaned resources left.
//If yes, then prints them and returns true. If not, then returns false
func (r *ResourcesTrackerImpl) IsOrphanedResourcesAvailable() bool {
	afterTestExecutionVMs, afterTestExecutionAvailDisks, afterTestExecutionAvailmachines, afterTestExecutionNICs, err := r.probeResources()
	if err != nil {
		fmt.Printf("Error probing orphaned resources: %s", err.Error())
		return true
	}

	if afterTestExecutionVMs != nil || afterTestExecutionAvailDisks != nil || afterTestExecutionAvailmachines != nil {
		fmt.Printf("attempting to delete orphan resrouces... the following resources are orphaned\n")
		fmt.Printf("Virtual Machines: %v\nVolumes: %v\nMCM Machines: %v\n", afterTestExecutionVMs, afterTestExecutionAvailDisks, afterTestExecutionAvailmachines)
		return true
	}

	if afterTestExecutionNICs != nil {
		fmt.Printf("Manually delete the orphan NICs after the test!\n")
		fmt.Printf("NICs: %v\n", afterTestExecutionNICs)
		return true
	}

	return false

}
