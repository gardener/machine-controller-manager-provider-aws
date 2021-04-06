/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package aws contains the cloud provider specific implementations to manage machines
package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/spi"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog"
)

// Driver is the driver struct for holding AWS machine information
type Driver struct {
	SPI spi.SessionProviderInterface
}

const (
	resourceTypeInstance = "instance"
	resourceTypeVolume   = "volume"
	// awsEBSDriverName is the name of the CSI driver for EBS
	awsEBSDriverName = "ebs.csi.aws.com"
)

// NewAWSDriver returns an empty AWSDriver object
func NewAWSDriver(spi spi.SessionProviderInterface) driver.Driver {
	return &Driver{
		SPI: spi,
	}
}

// CreateMachine handles a machine creation request
func (d *Driver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	var (
		exists       bool
		userData     []byte
		machine      = req.Machine
		secret       = req.Secret
		machineClass = req.MachineClass
	)

	// Log messages to track request
	klog.V(3).Infof("Machine creation request has been recieved for %q", req.Machine.Name)

	providerSpec, err := decodeProviderSpecAndSecret(machineClass, secret)
	if err != nil {
		return nil, err
	}

	svc, err := d.createSVC(secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if userData, exists = secret.Data["userData"]; !exists {
		return nil, status.Error(codes.Internal, "userData doesn't exist")
	}
	UserDataEnc := base64.StdEncoding.EncodeToString([]byte(userData))

	var imageIds []*string
	imageID := aws.String(providerSpec.AMI)
	imageIds = append(imageIds, imageID)

	describeImagesRequest := ec2.DescribeImagesInput{
		ImageIds: imageIds,
	}
	output, err := svc.DescribeImages(&describeImagesRequest)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	} else if len(output.Images) < 1 {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Image %s not found", *imageID))
	}

	blkDeviceMappings, err := d.generateBlockDevices(providerSpec.BlockDevices, output.Images[0].RootDeviceName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	tagInstance, err := d.generateTags(providerSpec.Tags, resourceTypeInstance, req.Machine.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	tagVolume, err := d.generateTags(providerSpec.Tags, resourceTypeVolume, req.Machine.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var networkInterfaceSpecs []*ec2.InstanceNetworkInterfaceSpecification
	for i, netIf := range providerSpec.NetworkInterfaces {
		spec := &ec2.InstanceNetworkInterfaceSpecification{
			Groups:                   aws.StringSlice(netIf.SecurityGroupIDs),
			DeviceIndex:              aws.Int64(int64(i)),
			AssociatePublicIpAddress: netIf.AssociatePublicIPAddress,
			DeleteOnTermination:      netIf.DeleteOnTermination,
			Description:              netIf.Description,
			SubnetId:                 aws.String(netIf.SubnetID),
		}

		if netIf.DeleteOnTermination == nil {
			spec.DeleteOnTermination = aws.Bool(true)
		}

		networkInterfaceSpecs = append(networkInterfaceSpecs, spec)
	}

	// Specify the details of the machine that you want to create.
	inputConfig := ec2.RunInstancesInput{
		BlockDeviceMappings: blkDeviceMappings,
		ImageId:             aws.String(providerSpec.AMI),
		InstanceType:        aws.String(providerSpec.MachineType),
		MinCount:            aws.Int64(1),
		MaxCount:            aws.Int64(1),
		UserData:            &UserDataEnc,
		KeyName:             aws.String(providerSpec.KeyName),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: &(providerSpec.IAM.Name),
		},
		NetworkInterfaces: networkInterfaceSpecs,
		TagSpecifications: []*ec2.TagSpecification{tagInstance, tagVolume},
	}

	// Set the AWS Capacity Reservation target. Using an 'open' preference means that if the reservation is not found, then
	// instances are launched with regular on-demand capacity.
	if providerSpec.CapacityReservationTarget != nil {
		if providerSpec.CapacityReservationTarget.CapacityReservationResourceGroupArn != nil {
			inputConfig.CapacityReservationSpecification = &ec2.CapacityReservationSpecification{
				CapacityReservationPreference: aws.String("open"),
				CapacityReservationTarget: &ec2.CapacityReservationTarget{
					CapacityReservationResourceGroupArn: providerSpec.CapacityReservationTarget.CapacityReservationResourceGroupArn,
				},
			}
		} else if providerSpec.CapacityReservationTarget.CapacityReservationId != nil {
			inputConfig.CapacityReservationSpecification = &ec2.CapacityReservationSpecification{
				CapacityReservationPreference: aws.String("open"),
				CapacityReservationTarget: &ec2.CapacityReservationTarget{
					CapacityReservationId: providerSpec.CapacityReservationTarget.CapacityReservationId,
				},
			}
		}
	}

	// Set spot price if it has been set
	if providerSpec.SpotPrice != nil {
		inputConfig.InstanceMarketOptions = &ec2.InstanceMarketOptionsRequest{
			MarketType: aws.String(ec2.MarketTypeSpot),
			SpotOptions: &ec2.SpotMarketOptions{
				SpotInstanceType: aws.String(ec2.SpotInstanceTypeOneTime),
			},
		}

		if *providerSpec.SpotPrice != "" {
			inputConfig.InstanceMarketOptions.SpotOptions.MaxPrice = providerSpec.SpotPrice
		}
	}

	runResult, err := svc.RunInstances(&inputConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &driver.CreateMachineResponse{
		ProviderID: encodeInstanceID(providerSpec.Region, *runResult.Instances[0].InstanceId),
		NodeName:   *runResult.Instances[0].PrivateDnsName,
	}

	klog.V(3).Infof("VM with Provider-ID: %q created for Machine: %q", response.ProviderID, machine.Name)
	return response, nil
}

// DeleteMachine handles a machine deletion request
func (d *Driver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	var (
		err    error
		secret = req.Secret
	)
	// Log messages to track delete request
	klog.V(3).Infof("Machine deletion request has been recieved for %q", req.Machine.Name)
	defer klog.V(3).Infof("Machine deletion request has been processed for %q", req.Machine.Name)

	providerSpec, err := decodeProviderSpecAndSecret(req.MachineClass, secret)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	svc, err := d.createSVC(req.Secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if req.Machine.Spec.ProviderID != "" {
		// ProviderID exists for machine object, hence terminate the correponding VM

		_, instanceID, err := decodeRegionAndInstanceID(req.Machine.Spec.ProviderID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = terminateInstance(req, svc, instanceID)
		if err != nil {
			return nil, err
		}
		klog.V(3).Infof("VM %q for Machine %q was terminated succesfully", req.Machine.Spec.ProviderID, req.Machine.Name)

	} else {
		// ProviderID doesn't exist, hence check for any existing machine and then delete if exists

		instances, err := d.getInstancesFromMachineName(req.Machine.Name, providerSpec, req.Secret)
		if err != nil {
			status, ok := status.FromError(err)
			if ok && status.Code() == codes.NotFound {
				klog.V(3).Infof("No matching VM found. Termination succesful for machine object %q", req.Machine.Name)
				return &driver.DeleteMachineResponse{}, nil
			}
			return nil, err
		}

		// If instance(s) exist, terminate them
		for _, instance := range instances {
			// For each instance backing machine, terminate the VMs
			err = terminateInstance(req, svc, *instance.InstanceId)
			if err != nil {
				return nil, err
			}
			klog.V(3).Infof("VM %q for Machine %q was terminated succesfully", *instance.InstanceId, req.Machine.Name)
		}
	}

	return &driver.DeleteMachineResponse{}, nil
}

// GetMachineStatus handles a machine get status request
func (d *Driver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	var (
		secret       = req.Secret
		machineClass = req.MachineClass
	)

	// Log messages to track start and end of request
	klog.V(3).Infof("Get request has been recieved for %q", req.Machine.Name)

	providerSpec, err := decodeProviderSpecAndSecret(machineClass, secret)
	if err != nil {
		return nil, err
	}

	instances, err := d.getInstancesFromMachineName(req.Machine.Name, providerSpec, secret)

	if err != nil {
		return nil, err
	} else if len(instances) > 1 {
		instanceIDs := []string{}
		for _, instance := range instances {
			instanceIDs = append(instanceIDs, *instance.InstanceId)
		}

		errMessage := fmt.Sprintf("AWS plugin is returning multiple VM instances backing this machine object. IDs for all backing VMs - %v ", instanceIDs)
		return nil, status.Error(codes.OutOfRange, errMessage)
	}

	requiredInstance := instances[0]

	response := &driver.GetMachineStatusResponse{
		NodeName:   *requiredInstance.PrivateDnsName,
		ProviderID: encodeInstanceID(providerSpec.Region, *requiredInstance.InstanceId),
	}

	klog.V(3).Infof("Machine get request has been processed successfully for %q", req.Machine.Name)
	return response, nil
}

// ListMachines lists all the machines possibilly created by a machineClass
func (d *Driver) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	var (
		machineClass = req.MachineClass
		secret       = req.Secret
	)

	// Log messages to track start and end of request
	klog.V(3).Infof("List machines request has been recieved for %q", machineClass.Name)

	providerSpec, err := decodeProviderSpecAndSecret(machineClass, secret)
	if err != nil {
		return nil, err
	}

	clusterName := ""
	nodeRole := ""

	for key := range providerSpec.Tags {
		if strings.Contains(key, "kubernetes.io/cluster/") {
			clusterName = key
		} else if strings.Contains(key, "kubernetes.io/role/") {
			nodeRole = key
		}
	}

	svc, err := d.createSVC(secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	input := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag-key"),
				Values: []*string{
					&clusterName,
				},
			},
			{
				Name: aws.String("tag-key"),
				Values: []*string{
					&nodeRole,
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("pending"),
					aws.String("running"),
					aws.String("stopping"),
					aws.String("stopped"),
				},
			},
		},
	}

	runResult, err := svc.DescribeInstances(&input)
	if err != nil {
		klog.Errorf("AWS plugin is returning error while describe instances request is sent: %s", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	listOfVMs := make(map[string]string)
	for _, reservation := range runResult.Reservations {
		for _, instance := range reservation.Instances {

			machineName := ""
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					machineName = *tag.Value
					break
				}
			}
			listOfVMs[encodeInstanceID(providerSpec.Region, *instance.InstanceId)] = machineName
		}
	}

	klog.V(3).Infof("List machines request has been processed successfully")
	// Core logic ends here.
	resp := &driver.ListMachinesResponse{
		MachineList: listOfVMs,
	}
	return resp, nil
}

// GetVolumeIDs returns a list of Volume IDs for all PV Specs for whom an provider volume was found
func (d *Driver) GetVolumeIDs(ctx context.Context, req *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	var (
		volumeIDs []string
	)

	// Log messages to track start and end of request
	klog.V(3).Infof("GetVolumeIDs request has been recieved for %q", req.PVSpecs)

	for _, spec := range req.PVSpecs {

		if spec.AWSElasticBlockStore != nil {
			volumeID, err := kubernetesVolumeIDToEBSVolumeID(spec.AWSElasticBlockStore.VolumeID)
			if err != nil {
				klog.Errorf("Failed to translate Kubernetes volume ID '%s' to EBS volume ID: %v", spec.AWSElasticBlockStore.VolumeID, err)
				continue
			}

			volumeIDs = append(volumeIDs, volumeID)
		} else if spec.CSI != nil && spec.CSI.Driver == awsEBSDriverName && spec.CSI.VolumeHandle != "" {
			volumeID := spec.CSI.VolumeHandle
			volumeIDs = append(volumeIDs, volumeID)
		}
	}

	klog.V(3).Infof("GetVolumeIDs machines request has been processed successfully. \nList: %v", volumeIDs)

	resp := &driver.GetVolumeIDsResponse{
		VolumeIDs: volumeIDs,
	}
	return resp, nil
}

// GenerateMachineClassForMigration converts providerSpecificMachineClass to (generic) MachineClass
func (d *Driver) GenerateMachineClassForMigration(ctx context.Context, req *driver.GenerateMachineClassForMigrationRequest) (*driver.GenerateMachineClassForMigrationResponse, error) {
	klog.V(1).Infof("Migrate request has been recieved for %v", req.MachineClass.Name)
	defer klog.V(1).Infof("Migrate request has been processed for %v", req.MachineClass.Name)

	awsMachineClass := req.ProviderSpecificMachineClass.(*v1alpha1.AWSMachineClass)

	// Check if incoming CR is valid CR for migration
	// In this case, the MachineClassKind to be matching
	if req.ClassSpec.Kind != AWSMachineClassKind {
		return nil, status.Error(codes.Internal, "Migration cannot be done for this machineClass kind")
	}

	return &driver.GenerateMachineClassForMigrationResponse{}, fillUpMachineClass(awsMachineClass, req.MachineClass)
}
