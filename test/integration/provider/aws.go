package provider

/**
	Orphaned Resources
	- VMs:
		Describe instances with specified tag name:<cluster-name>
		Report/Print out instances found
		Describe volumes attached to the instance (using instance id)
		Report/Print out volumes found
		Delete attached volumes found
		Terminate instances found
	- Disks:
		Describe volumes with tag status:available
		Report/Print out volumes found
		Delete identified volumes
**/

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	v1 "k8s.io/api/core/v1"

	providerDriver "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/spi"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
)

var _ aws.Config

func newSession(machineClass *v1alpha1.MachineClass, secret *v1.Secret) *session.Session {
	var (
		providerSpec *api.AWSProviderSpec
		sPI          spi.PluginSPIImpl
	)

	err := json.Unmarshal([]byte(machineClass.ProviderSpec.Raw), &providerSpec)
	if err != nil {
		providerSpec = nil
		log.Printf("Error occured while performing unmarshal %s", err.Error())
	}
	sess, err := sPI.NewSession(secret, providerSpec.Region)
	if err != nil {
		log.Printf("Error occured while creating new session %s", err)
	}
	return sess
}

func DescribeMachines(machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	var machines []string
	var sPI spi.PluginSPIImpl
	driverprovider := providerDriver.NewAWSDriver(&sPI)
	machineList, err := driverprovider.ListMachines(context.TODO(), &driver.ListMachinesRequest{
		MachineClass: machineClass,
		Secret:       &v1.Secret{Data: secretData},
	})
	if err != nil {
		return nil, err
	} else if len(machineList.MachineList) != 0 {
		fmt.Printf("\nAvailable Machines: ")
		for _, machine := range machineList.MachineList {
			machines = append(machines, machine)
		}
	}
	return machines, nil
}

// DescribeInstancesWithTag describes the instance with the specified tag
func DescribeInstancesWithTag(tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	sess := newSession(machineClass, &v1.Secret{Data: secretData})
	svc := ec2.New(sess)
	var instancesID []string
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String(tagName),
				Values: []*string{
					aws.String(tagValue),
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
		},
	}

	result, err := svc.DescribeInstances(input)
	checkAWSError(err)
	if len(result.Reservations) != 0 {
		fmt.Printf("\nAvailable Instances: ")
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				instancesID = append(instancesID, *instance.InstanceId)

				// describe volumes attached to instance & delete them
				//DescribeVolumesAttached(*instance.InstanceId)

				// terminate the instance
				//TerminateInstance(*instance.InstanceId)

			}
		}
	}
	return instancesID, nil
}

// TerminateInstance terminates the specified EC2 instance.
func TerminateInstance(instanceID string) error {
	ses, _ := session.NewSession()
	svc := ec2.New(ses)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	result, err := svc.TerminateInstances(input)
	checkAWSError(err)

	fmt.Println(result)
	return nil
}

// DescribeAvailableVolumes describes volumes with the specified tag
func DescribeAvailableVolumes(tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	sess := newSession(machineClass, &v1.Secret{Data: secretData})
	svc := ec2.New(sess)
	var availVolID []string
	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("status"),
				Values: []*string{
					aws.String("available"),
				},
			},
			{
				Name: aws.String(tagName),
				Values: []*string{
					aws.String(tagValue),
				},
			},
		},
	}

	result, err := svc.DescribeVolumes(input)
	checkAWSError(err)

	for _, volume := range result.Volumes {
		fmt.Printf("%s", *volume.VolumeId)
		availVolID = append(availVolID, *volume.VolumeId)

		// delete the volume
		DeleteVolume(*volume.VolumeId)
	}

	return availVolID, nil
}

// DescribeVolumesAttached describes volumes that are attached to a specific instance
func DescribeVolumesAttached(InstanceID string) ([]string, error) {
	ses, _ := session.NewSession()
	svc := ec2.New(ses)
	var volumesID []string
	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("attachment.instance-id"),
				Values: []*string{
					aws.String(InstanceID),
				},
			},
			{
				Name: aws.String("attachment.delete-on-termination"),
				Values: []*string{
					aws.String("true"),
				},
			},
		},
	}

	result, err := svc.DescribeVolumes(input)
	checkAWSError(err)

	for _, volume := range result.Volumes {
		volumesID = append(volumesID, *volume.VolumeId)
		// delete the volume
		//DeleteVolume(*volume.VolumeId)
	}

	return volumesID, nil
}

// DeleteVolume deletes the specified volume
func DeleteVolume(VolumeID string) error {
	// TO-DO: deletes an available volume with the specified volume ID
	// If the command succeeds, no output is returned.
	ses, _ := session.NewSession()
	svc := ec2.New(ses)
	input := &ec2.DeleteVolumeInput{
		VolumeId: aws.String(VolumeID),
	}

	result, err := svc.DeleteVolume(input)
	checkAWSError(err)
	fmt.Println(result)
	return nil
}

// AdditionalResourcesCheck describes VPCs and network interfaces
func AdditionalResourcesCheck(tagName string, tagValue string) error {
	// TO-DO: Checks for Network interfaces and VPCs
	// If the command succeeds, no output is returned.
	ses, _ := session.NewSession()
	svc := ec2.New(ses)
	inputVPC := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String(tagName),
				Values: []*string{
					aws.String(tagValue),
				},
			},
		},
	}
	resultVPC, err := svc.DescribeVpcs(inputVPC)
	checkAWSError(err)

	for _, vpc := range resultVPC.Vpcs {
		fmt.Println(*vpc.VpcId)

		inputNI := &ec2.DescribeNetworkInterfacesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						aws.String(*vpc.VpcId),
					},
				},
			},
		}

		resultDescribeNetworkInterface, err := svc.DescribeNetworkInterfaces(inputNI)
		checkAWSError(err)

		fmt.Println(resultDescribeNetworkInterface)

		for _, networkinterface := range resultDescribeNetworkInterface.NetworkInterfaces {
			fmt.Println(*networkinterface.Attachment.AttachmentId)
			input := &ec2.DetachNetworkInterfaceInput{
				AttachmentId: aws.String(*networkinterface.Attachment.AttachmentId),
			}

			resultDetachNetworkInterface, err := svc.DetachNetworkInterface(input)
			checkAWSError(err)

			fmt.Println(resultDetachNetworkInterface)
		}

		for _, networkinterfaceid := range resultDescribeNetworkInterface.NetworkInterfaces {
			fmt.Println(*networkinterfaceid.NetworkInterfaceId)
			input := &ec2.DeleteNetworkInterfaceInput{
				NetworkInterfaceId: aws.String(*networkinterfaceid.NetworkInterfaceId),
			}

			resultDeleteNetworkInterface, err := svc.DeleteNetworkInterface(input)
			checkAWSError(err)

			fmt.Println(resultDeleteNetworkInterface)
		}
	}

	fmt.Println(resultVPC)
	return nil
}

func checkAWSError(err error) error {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return err.(awserr.Error)
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			return err
		}
	}
	return nil
}
