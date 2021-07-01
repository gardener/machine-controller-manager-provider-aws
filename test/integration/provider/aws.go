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

func getMachines(machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
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
		for _, machine := range machineList.MachineList {
			machines = append(machines, machine)
		}
	}
	return machines, nil
}

// getOrphanesInstances returns list of Orphan resources that couldn't be deleted
func getOrphanedInstances(tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
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
	if err != nil {
		return instancesID, err
	}

	if len(result.Reservations) != 0 {
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				// terminate the instance
				if err = TerminateInstance(sess, *instance.InstanceId); err != nil {
					instancesID = append(instancesID, *instance.InstanceId)
				}
			}
		}
	}
	return instancesID, nil
}

// TerminateInstance terminates the specified EC2 instance.
func TerminateInstance(ses *session.Session, instanceID string) error {
	svc := ec2.New(ses)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
		DryRun: aws.Bool(false),
	}

	_, err := svc.TerminateInstances(input)
	if err != nil {
		fmt.Printf("can't terminate the instance %s,%s\n", instanceID, err.Error())
		return err
	}

	fmt.Printf("Deleted an orphan VM %s,", instanceID)

	return nil
}

// getOrphanedDisks returns a list of orphan disks that couldn't get deleted
func getOrphanedDisks(tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
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
	if err != nil {
		return availVolID, err
	}

	for _, volume := range result.Volumes {
		// delete the volume
		if err = DeleteVolume(sess, *volume.VolumeId); err != nil {
			availVolID = append(availVolID, *volume.VolumeId)
		}
	}

	return availVolID, nil
}

// DeleteVolume deletes the specified volume
func DeleteVolume(ses *session.Session, VolumeID string) error {
	svc := ec2.New(ses)
	input := &ec2.DeleteVolumeInput{
		VolumeId: aws.String(VolumeID),
	}

	_, err := svc.DeleteVolume(input)
	if err != nil {
		fmt.Printf("can't delete volume .%s\n", err.Error())
		return err
	}

	fmt.Printf("Deleted an orphan disk %s,", VolumeID)

	return nil
}

//getOrphanedNICs returns a list of orphaned NICs which are present
func getOrphanedNICs(tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	var orphanNICs []string
	sess := newSession(machineClass, &v1.Secret{Data: secretData})
	svc := ec2.New(sess)
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
	if err != nil {
		return orphanNICs, err
	}

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
				{
					Name: aws.String("status"),
					Values: []*string{
						aws.String("available"),
					},
				},
			},
		}

		resultNetworkInterface, err := svc.DescribeNetworkInterfaces(inputNI)
		if err != nil {
			return orphanNICs, err
		}

		for _, nic := range resultNetworkInterface.NetworkInterfaces {
			orphanNICs = append(orphanNICs, *nic.NetworkInterfaceId)
		}

	}

	return orphanNICs, nil
}
