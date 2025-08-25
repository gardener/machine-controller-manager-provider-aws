// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	v1 "k8s.io/api/core/v1"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"

	providerDriver "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/cpi"
)

var _ aws.Config

func newConfig(ctx context.Context, machineClass *v1alpha1.MachineClass, secret *v1.Secret) *aws.Config {
	var (
		providerSpec   *api.AWSProviderSpec
		clientProvider cpi.ClientProvider
	)

	err := json.Unmarshal([]byte(machineClass.ProviderSpec.Raw), &providerSpec)
	if err != nil {
		providerSpec = nil
		log.Printf("Error occured while performing unmarshal %s", err.Error())
	}
	config, err := clientProvider.NewConfig(ctx, secret, providerSpec.Region)
	if err != nil {
		log.Printf("Error occured while creating new session %s", err)
	}
	return config
}

func getMachines(ctx context.Context, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	var machines []string
	var clientProvider cpi.ClientProvider
	driverprovider := providerDriver.NewAWSDriver(&clientProvider)
	machineList, err := driverprovider.ListMachines(ctx, &driver.ListMachinesRequest{
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
func getOrphanedInstances(ctx context.Context, tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	cfg := newConfig(ctx, machineClass, &v1.Secret{Data: secretData})
	svc := ec2.NewFromConfig(*cfg)
	var instancesID []string
	input := &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String(tagName),
				Values: []string{tagValue},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	}

	result, err := svc.DescribeInstances(ctx, input)
	if err != nil {
		return instancesID, err
	}
	if len(result.Reservations) != 0 {
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				instancesID = append(instancesID, *instance.InstanceId)
			}
		}
	}
	return instancesID, nil
}

// TerminateInstance terminates the specified EC2 instance.
func TerminateInstance(ctx context.Context, cfg *aws.Config, instanceID string) error {
	svc := ec2.NewFromConfig(*cfg)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
		DryRun:      aws.Bool(false),
	}

	_, err := svc.TerminateInstances(ctx, input)
	if err != nil {
		fmt.Printf("can't terminate the instance %s,%s\n", instanceID, err.Error())
		return err
	}

	fmt.Printf("Deleted an orphan VM %s,", instanceID)

	return nil
}

// getOrphanedDisks returns a list of orphan disks that couldn't get deleted
func getOrphanedDisks(ctx context.Context, tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	cfg := newConfig(ctx, machineClass, &v1.Secret{Data: secretData})
	svc := ec2.NewFromConfig(*cfg)
	var availVolID []string
	input := &ec2.DescribeVolumesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("status"),
				Values: []string{"available"},
			},
			{
				Name:   aws.String(tagName),
				Values: []string{tagValue},
			},
		},
	}

	result, err := svc.DescribeVolumes(ctx, input)
	if err != nil {
		return availVolID, err
	}

	for _, volume := range result.Volumes {
		availVolID = append(availVolID, *volume.VolumeId)
	}

	return availVolID, nil
}

// DeleteVolume deletes the specified volume
func DeleteVolume(ctx context.Context, cfg *aws.Config, VolumeID string) error {
	svc := ec2.NewFromConfig(*cfg)
	input := &ec2.DeleteVolumeInput{
		VolumeId: aws.String(VolumeID),
	}

	_, err := svc.DeleteVolume(ctx, input)
	if err != nil {
		fmt.Printf("can't delete volume .%s\n", err.Error())
		return err
	}

	fmt.Printf("Deleted an orphan disk %s,", VolumeID)

	return nil
}

// getOrphanedNICs returns a list of orphaned NICs which are present
func getOrphanedNICs(ctx context.Context, tagName string, tagValue string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) ([]string, error) {
	var orphanNICs []string
	cfg := newConfig(ctx, machineClass, &v1.Secret{Data: secretData})
	svc := ec2.NewFromConfig(*cfg)
	inputNIC := &ec2.DescribeNetworkInterfacesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String(tagName),
				Values: []string{tagValue},
			},
			{
				Name:   aws.String("status"),
				Values: []string{"available"},
			},
		},
	}
	resultNetworkInterface, err := svc.DescribeNetworkInterfaces(ctx, inputNIC)
	if err != nil {
		return orphanNICs, err
	}
	for _, nic := range resultNetworkInterface.NetworkInterfaces {
		orphanNICs = append(orphanNICs, *nic.NetworkInterfaceId)
	}
	return orphanNICs, nil

}

// DeleteNetworkInterface deletes the specified volume
func DeleteNetworkInterface(ctx context.Context, cfg *aws.Config, networkInterfaceID string) error {
	svc := ec2.NewFromConfig(*cfg)
	input := &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(networkInterfaceID),
	}

	_, err := svc.DeleteNetworkInterface(ctx, input)
	if err != nil {
		fmt.Printf("can't delete Network Interface .%s\n", err.Error())
		return err
	}

	fmt.Printf("Deleted an orphan NIC %s,", networkInterfaceID)

	return nil
}

func cleanOrphanResources(ctx context.Context, orphanVms []string, orphanVolumes []string, orphanNICs []string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) (delErrOrphanVms []string, delErrOrphanVolumes []string, delErrOrphanNICs []string) {
	cfg := newConfig(ctx, machineClass, &v1.Secret{Data: secretData})
	for _, instanceID := range orphanVms {
		if err := TerminateInstance(ctx, cfg, instanceID); err != nil {
			delErrOrphanVms = append(delErrOrphanVms, instanceID)
		}
	}

	for _, volumeID := range orphanVolumes {
		if err := DeleteVolume(ctx, cfg, volumeID); err != nil {
			delErrOrphanVolumes = append(delErrOrphanVolumes, volumeID)
		}
	}

	for _, networkInterfaceID := range orphanNICs {
		if err := DeleteNetworkInterface(ctx, cfg, networkInterfaceID); err != nil {
			delErrOrphanNICs = append(delErrOrphanNICs, networkInterfaceID)
		}
	}

	return
}
