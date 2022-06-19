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

// InitializeResourcesTracker initializes the type ResourcesTrackerImpl variable and tries
// to delete the orphan resources present before the actual IT runs.
// create a cleanup function to delete the list of orphan resources.
// 1. get list of orphan resources.
// 2. Mark them for deletion and call cleanup.
// 3. Print the orphan resources which got error in deletion.
func (r *ResourcesTrackerImpl) InitializeResourcesTracker(machineClass *v1alpha1.MachineClass, secretData map[string][]byte, clusterName string) error {

	r.MachineClass = machineClass
	r.SecretData = secretData
	r.ClusterName = clusterName

	initialVMs, initialVolumes, initialMachines, initialNICs, err := r.probeResources()
	if err != nil {
		fmt.Printf("Error in initial probe of orphaned resources: %s", err.Error())
		return err
	}

	delErrOrphanVMs, delErrOrphanVolumes, delErrOrphanNICs := cleanOrphanResources(initialVMs, initialVolumes, initialNICs, r.MachineClass, r.SecretData)
	if delErrOrphanVMs != nil || delErrOrphanVolumes != nil || initialMachines != nil || delErrOrphanNICs != nil {
		err := fmt.Errorf("error in cleaning the following orphan resources. Clean them up before proceeding with the test.\nvirtual machines: %v\ndisks: %v\nmcm machines: %v\nnics: %v", delErrOrphanVMs, delErrOrphanVolumes, initialMachines, delErrOrphanNICs)
		return err
	}

	return nil
}

// probeResources will look for resources currently available and returns them
func (r *ResourcesTrackerImpl) probeResources() ([]string, []string, []string, []string, error) {
	// Check for VM instances with matching tags/labels
	// Describe volumes attached to VM instance & delete the volumes
	// Finally delete the VM instance

	integrationTestTag := "tag:kubernetes.io/role/integration-test"
	integrationTestTagValue := "1"

	orphanVMs, err := getOrphanedInstances(integrationTestTag, integrationTestTagValue, r.MachineClass, r.SecretData)
	if err != nil {
		return orphanVMs, nil, nil, nil, err
	}

	// Check for available volumes in cloud provider with tag/label [Status:available]
	orphanVols, err := getOrphanedDisks(integrationTestTag, integrationTestTagValue, r.MachineClass, r.SecretData)
	if err != nil {
		return orphanVMs, orphanVols, nil, nil, err
	}

	availMachines, err := getMachines(r.MachineClass, r.SecretData)
	if err != nil {
		return orphanVMs, orphanVols, availMachines, nil, err
	}

	orphanNICs, err := getOrphanedNICs(integrationTestTag, integrationTestTagValue, r.MachineClass, r.SecretData)

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

	if afterTestExecutionVMs != nil || afterTestExecutionAvailDisks != nil || afterTestExecutionAvailmachines != nil || afterTestExecutionNICs != nil {
		fmt.Printf("The following resources are orphans ... trying to delete them \n")
		fmt.Printf("Virtual Machines: %v\nVolumes: %v\nNICs: %v\nMCM Machines %v\n ", afterTestExecutionVMs, afterTestExecutionAvailDisks, afterTestExecutionNICs, afterTestExecutionAvailmachines)
		return true
	}

	return false
}
