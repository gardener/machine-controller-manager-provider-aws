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
		- To-Do : Delete the control and target clusters // As of now we are reusing the cluster

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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gardener/machine-controller-manager-provider-aws/test/integration/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	cloudProviderSecret   = flag.String("cloud-provider-secret", "", "the path to the cloud provider secret for using to interact with the cloud provider")
	controlKubeConfigPath = flag.String("control-kubeconfig", "", "the path to the kubeconfig  of the control cluster for managing machines")
	targetKubeConfigPath  = flag.String("target-kubeconfig", "", "the path to the kubeconfig  of the target cluster where the nodes are added or removed")
	controlKubeCluster    *helpers.Cluster
	targetKubeCluster     *helpers.Cluster
	numberOfBgProcesses   int16
	mcmRepoPath           = "../../../dev/mcm"
)

var _ = Describe("Machine Resource", func() {
	BeforeSuite(func() {
		/*Check control cluster and target clusters are accessible
		- Check and create crds ( machineclass, machines, machinesets and machinedeployment ) if required
		  using file available in kubernetes/crds directory of machine-controller-manager repo
		- Start the Machine Controller manager and machine controller (provider-specific)
		- apply secret resource for accesing the cloud provider service in the control cluster
		- Create machineclass resource from file available in kubernetes directory of provider specific repo in control cluster
		*/
		By("Checking for the clusters if provided are available")
		Expect(prepareClusters()).To(BeNil())
		By("Fetching kubernetes/crds and applying them into control cluster")
		Expect(applyCrds()).To(BeNil())
		By("Starting Machine Controller Manager")
		Expect(startMachineControllerManager()).To(BeNil())
		By("Starting Machine Controller")
		Expect(startMachineController()).To(BeNil())
		By("Parsing cloud-provider-secret file and applying")
		Expect(applyCloudProviderSecret()).To(BeNil())
		By("Applying MachineClass")
		Expect(applyMachineClass()).To(BeNil())
	})
	BeforeEach(func() {
		By("Check the number of goroutines running are 2")
		Expect(numberOfBgProcesses).To(BeEquivalentTo(2))
		// Nodes are healthy
	})

	Describe("Creating one machine resource", func() {
		Context("In Control cluster", func() {
			Context("when the nodes in target cluster are listed", func() {
				It("should correctly list existing nodes +1", func() {
					// Probe nodes currently available in target cluster
					// apply machine resource yaml file
					fmt.Println("wait for 30 sec before probing for nodes")
					time.Sleep(30 * time.Second) // probe nodes again after some wait
					// check whether there is one node more
				})
			})
		})
	})

	Describe("Deleting one machine resource", func() {
		BeforeEach(func() {
			// Check there are no machine deployment and machinesets resources existing
			// Nodes are healthy in target cluster
		})
		Context("When there are machine resources available in control cluster", func() {
			// check for machine resources
			Context("When one machine is deleted randomly", func() {
				// Keep count of nodes available
				//delete machine resource
				It("should list existing nodes -1 in target cluster", func() {
					// check there are n-1 nodes
				})
			})
		})
		Context("when there are no machines available", func() {
			// delete one machine (non-existent) by random text as name of resource
			It("should list existing nodes ", func() {
				// check there are no changes to nodes
			})
		})
	})
})

func prepareClusters() error {
	/* TO-DO: prepareClusters checks for
	- the validity of controlKubeConfig and targetKubeConfig flags
	- if required then creates the cluster using cloudProviderSecret
	- It should return an error if thre is a error
	*/

	if *controlKubeConfigPath != "" {
		*controlKubeConfigPath, _ = filepath.Abs(*controlKubeConfigPath)
		// if control cluster config is available but not the target, then set control and target clusters as same
		if *targetKubeConfigPath == "" {
			*targetKubeConfigPath = *controlKubeConfigPath
			fmt.Println("Missing targetKubeConfig. control cluster will be set as target too")
		}
		*targetKubeConfigPath, _ = filepath.Abs(*targetKubeConfigPath)
		// use the current context in controlkubeconfig
		var err error
		controlKubeCluster, err = helpers.NewCluster(*controlKubeConfigPath)
		if err != nil {
			return err
		}
		targetKubeCluster, err = helpers.NewCluster(*targetKubeConfigPath)
		if err != nil {
			return err
		}

		// update clientset and check whether the cluster is accessible
		err = controlKubeCluster.FillClientSets()
		if err != nil {
			fmt.Println("Failed to check nodes in the cluster")
			return err
		}

		err = targetKubeCluster.FillClientSets()
		if err != nil {
			fmt.Println("Failed to check nodes in the cluster")
			return err
		}
	} else if *targetKubeConfigPath != "" {
		return fmt.Errorf("controlKubeconfig path is mandatory if using targetKubeConfigPath. Aborting!!!")
	} else if *cloudProviderSecret != "" {
		*cloudProviderSecret, _ = filepath.Abs(*cloudProviderSecret)
		// TO-DO: validate cloudProviderSecret yaml file and Create cluster using the secrets in it.
		// Also set controlKubeCluster and targetKubeCluster
	} else {
		return fmt.Errorf("missing mandatory flag cloudProviderSecret. Aborting!!!")
	}
	return nil
}

func applyCrds() error {
	/* TO-DO: applyCrds will
	- create the custom resources in the controlKubeConfig
	- yaml files are available in kubernetes/crds directory of machine-controller-manager repo
	- resources to be applied are machineclass, machines, machinesets and machinedeployment
	*/

	var files []string
	dst := mcmRepoPath
	src := "https://github.com/gardener/machine-controller-manager.git"
	applyCrdsDirectory := fmt.Sprintf("%s/kubernetes/crds", dst)

	helpers.CheckDst(dst)
	helpers.CloningRepo(dst, src)

	err := filepath.Walk(applyCrdsDirectory, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		fmt.Println(file)
		fi, err := os.Stat(file)
		if err != nil {
			fmt.Println("\nError file does not exist!")
			return err
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			// do directory stuff
			fmt.Printf("\n%s is a directory. Therefore nothing will happen!\n", file)
		case mode.IsRegular():
			// do file stuff
			fmt.Printf("\n%s is a file. Therefore applying yaml ...", file)
			err := controlKubeCluster.ApplyYamlFile(file)
			if err != nil {
				if strings.Contains(err.Error(), "already exists") {
					fmt.Printf("\n%s already exists, so skipping ...\n", file)
				} else {
					fmt.Printf("\nFailed to create deployment %s, in the cluster.\n", file)
					return err
				}

			}
		}
	}

	return err
}

func startMachineControllerManager() error {
	/*
		 TO-DO: startMachineControllerManager starts the machine controller manager
			 - if mcmContainerImage flag is non-empty then, start a pod in the control-cluster with specified image
			 - if mcmContainerImage is empty, runs machine controller manager locally
				 clone the required repo and then use make
	*/
	command := fmt.Sprintf("make start CONTROL_KUBECONFIG=%s TARGET_KUBECONFIG=%s", *controlKubeConfigPath, *targetKubeConfigPath)
	fmt.Println("starting MachineControllerManager with command: ", command)
	dst_path := fmt.Sprintf("%s", mcmRepoPath)
	go execCommandAsRoutine(command, dst_path)
	return nil
}

func startMachineController() error {
	/*
		 TO-DO: startMachineController starts the machine controller
			 - if mcContainerImage flag is non-empty then, start a pod in the control-cluster with specified image
			 - if mcContainerImage is empty, runs machine controller locally
	*/
	command := fmt.Sprintf("make start CONTROL_KUBECONFIG=%s TARGET_KUBECONFIG=%s", *controlKubeConfigPath, *targetKubeConfigPath)
	fmt.Println("starting MachineController with command: ", command)
	go execCommandAsRoutine(command, "../../..")
	return nil
}

func applyCloudProviderSecret() error {
	/* TO-DO: applyCloudProviderSecret
	- load the yaml file
	- check if there is a  secret alredy with the same name in the controlCluster then validate for using it to interact with the hyperscaler
	- create the secret if not existing
	*/
	return nil
}

func applyMachineClass() error {
	/* TO-DO: applyMachineClass creates machineclass using
	- the file available in kubernetes directory of provider specific repo in control cluster
	*/
	return nil
}

func execCommandAsRoutine(cmd string, dir string) {
	defer func() {
		numberOfBgProcesses = numberOfBgProcesses - 1
	}()
	numberOfBgProcesses++
	fmt.Println("Goroutine started")
	args := strings.Fields(cmd)
	command := exec.Command(args[0], args[1:]...)
	command.Dir = dir
	out, err := command.CombinedOutput()
	if err != nil {
		fmt.Println("Error is ", err)
		fmt.Printf("output is %s\n ", out)
	} else {
		fmt.Printf("output is %s\n ", out)
	}
}
