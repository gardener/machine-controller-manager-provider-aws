// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package aws contains the cloud provider specific implementations to manage machines
package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/machine-controller-manager-provider-aws/pkg/instrument"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"

	awserror "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/errors"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/spi"
)

const (
	createMachineOperationLabel    = "create_machine"
	deleteMachineOperationLabel    = "delete_machine"
	listMachinesOperationLabel     = "list_machine"
	getMachineStatusOperationLabel = "get_machine_status"
	getVolumeIDsOperationLabel     = "get_volume_ids"
)

// Driver is the driver struct for holding AWS machine information
type Driver struct {
	SPI spi.SessionProviderInterface
}

const (
	// ProviderAWS string const to identify AWS provider
	ProviderAWS                  = "AWS"
	resourceTypeInstance         = "instance"
	resourceTypeVolume           = "volume"
	resourceTypeNetworkInterface = "network-interface"
	// awsEBSDriverName is the name of the CSI driver for EBS
	awsEBSDriverName = "ebs.csi.aws.com"
	awsPlacement     = "machine.sapcloud.io/awsPlacement"
)

var maxElapsedTimeInBackoff = 5 * time.Minute

// NewAWSDriver returns an empty AWSDriver object
func NewAWSDriver(spi spi.SessionProviderInterface) driver.Driver {
	return &Driver{
		SPI: spi,
	}
}

// CreateMachine handles a machine creation request
func (d *Driver) CreateMachine(_ context.Context, req *driver.CreateMachineRequest) (resp *driver.CreateMachineResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(createMachineOperationLabel, &err)()

	var (
		exists       bool
		userData     []byte
		machine      = req.Machine
		secret       = req.Secret
		machineClass = req.MachineClass
	)

	// Check if the MachineClass is for the supported cloud provider
	if req.MachineClass.Provider != ProviderAWS {
		err = fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAWS)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Log messages to track request
	klog.V(3).Infof("Machine creation request has been recieved for %q", req.Machine.Name)

	providerSpec, err := decodeProviderSpecAndSecret(machineClass, secret)
	if err != nil {
		return nil, err
	}

	svc, err := d.createSVC(secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(awserror.GetMCMErrorCodeForCreateMachine(err), err.Error())
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
		return nil, status.Error(awserror.GetMCMErrorCodeForCreateMachine(err), err.Error())
	} else if len(output.Images) < 1 {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Image %s not found", *imageID))
	}

	blkDeviceMappings, err := d.generateBlockDevices(providerSpec.BlockDevices, output.Images[0].RootDeviceName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tagInstance, err := d.generateTags(providerSpec.Tags, resourceTypeInstance, req.Machine.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	tagVolume, err := d.generateTags(providerSpec.Tags, resourceTypeVolume, req.Machine.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	tagNetworkInterface, err := d.generateTags(providerSpec.Tags, resourceTypeNetworkInterface, req.Machine.Name)
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

		if netIf.Ipv6AddressCount != nil {
			spec.Ipv6AddressCount = netIf.Ipv6AddressCount
		}

		networkInterfaceSpecs = append(networkInterfaceSpecs, spec)
	}

	// Specify the details of the machine that you want to create.
	iam := &ec2.IamInstanceProfileSpecification{}
	if len(providerSpec.IAM.Name) > 0 {
		iam.Name = &providerSpec.IAM.Name
	} else if len(providerSpec.IAM.ARN) > 0 {
		iam.Arn = &providerSpec.IAM.ARN
	}

	var metadataOptions *ec2.InstanceMetadataOptionsRequest
	if providerSpec.InstanceMetadataOptions != nil {
		metadataOptions = &ec2.InstanceMetadataOptionsRequest{
			HttpEndpoint:            providerSpec.InstanceMetadataOptions.HTTPEndpoint,
			HttpPutResponseHopLimit: providerSpec.InstanceMetadataOptions.HTTPPutResponseHopLimit,
			HttpTokens:              providerSpec.InstanceMetadataOptions.HTTPTokens,
		}
	}

	inputConfig := ec2.RunInstancesInput{
		BlockDeviceMappings: blkDeviceMappings,
		ImageId:             aws.String(providerSpec.AMI),
		InstanceType:        aws.String(providerSpec.MachineType),
		MinCount:            aws.Int64(1),
		MaxCount:            aws.Int64(1),
		UserData:            &UserDataEnc,
		IamInstanceProfile:  iam,
		NetworkInterfaces:   networkInterfaceSpecs,
		TagSpecifications:   []*ec2.TagSpecification{tagInstance, tagVolume, tagNetworkInterface},
		MetadataOptions:     metadataOptions,
	}

	if providerSpec.KeyName != nil && len(*providerSpec.KeyName) > 0 {
		inputConfig.KeyName = aws.String(*providerSpec.KeyName)
	}

	if cpuOptions := providerSpec.CPUOptions; cpuOptions != nil {
		inputConfig.CpuOptions = &ec2.CpuOptionsRequest{
			CoreCount:      cpuOptions.CoreCount,
			ThreadsPerCore: cpuOptions.ThreadsPerCore,
		}
	}

	// Set the AWS Capacity Reservation target. Using an 'open' preference means that if the reservation is not found, then
	// instances are launched with regular on-demand capacity.
	if providerSpec.CapacityReservationTarget != nil {
		inputConfig.CapacityReservationSpecification = &ec2.CapacityReservationSpecification{
			CapacityReservationPreference: providerSpec.CapacityReservationTarget.CapacityReservationPreference,
			CapacityReservationTarget: &ec2.CapacityReservationTarget{
				CapacityReservationId:               providerSpec.CapacityReservationTarget.CapacityReservationID,
				CapacityReservationResourceGroupArn: providerSpec.CapacityReservationTarget.CapacityReservationResourceGroupArn,
			},
		}
	}

	placement, err := getPlacementObj(req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	} else if placement != nil {
		inputConfig.Placement = placement
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
		return nil, status.Error(awserror.GetMCMErrorCodeForCreateMachine(err), err.Error())
	}

	response := &driver.CreateMachineResponse{
		ProviderID: encodeInstanceID(providerSpec.Region, *runResult.Instances[0].InstanceId),
		NodeName:   *runResult.Instances[0].PrivateDnsName,
	}

	klog.V(2).Infof("Waiting for VM with Provider-ID %q, for machine %q to be visible to all AWS endpoints", response.ProviderID, machine.Name)

	operation := func() error {
		_, err = confirmInstanceByID(svc, *runResult.Instances[0].InstanceId)
		return err
	}

	if err = retryWithExponentialBackOff(operation, maxElapsedTimeInBackoff); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("creation of VM %q failed, timed out waiting for eventual consistency. Multiple VMs backing machine obj might spawn, they will be orphan collected", response.ProviderID))
	}

	klog.V(2).Infof("VM with Provider-ID %q, for machine %q should be visible to all AWS endpoints now", response.ProviderID, machine.Name)
	klog.V(3).Infof("VM with Provider-ID: %q created for Machine: %q", response.ProviderID, machine.Name)
	return response, nil
}

// InitializeMachine should handle post-creation, one-time VM instance initialization operations. (Ex: Like setting up special network config, etc)
// The AWS Provider leverages this method to perform disabling of source destination checks for NAT instances.
// See [driver.Driver.InitializeMachine] for further information
func (d *Driver) InitializeMachine(_ context.Context, request *driver.InitializeMachineRequest) (*driver.InitializeMachineResponse, error) {
	providerSpec, err := decodeProviderSpecAndSecret(request.MachineClass, request.Secret)
	if err != nil {
		return nil, err
	}
	svc, err := d.createSVC(request.Secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Uninitialized, err.Error())
	}
	instances, err := d.getMatchingInstancesForMachine(request.Machine, svc, providerSpec.Tags)
	if err != nil {
		if isNotFoundError(err) {
			klog.Errorf("Could not get matching instance for uninitialized machine %q from provider: %s", request.Machine.Name, err)
			return nil, status.Error(codes.Uninitialized, err.Error())
		}
		return nil, err
	}
	targetInstance := instances[0]
	providerID := encodeInstanceID(providerSpec.Region, *targetInstance.InstanceId)
	// if SrcAnDstCheckEnabled is false then disable the SrcAndDestCheck on running NAT instance
	if providerSpec.SrcAndDstChecksEnabled != nil && !*providerSpec.SrcAndDstChecksEnabled && *targetInstance.SourceDestCheck {
		klog.V(3).Infof("Disabling SourceDestCheck on VM %q associated with machine %s", providerID, request.Machine.Name)
		err = disableSrcAndDestCheck(svc, targetInstance.InstanceId)
		if err != nil {
			return nil, status.Error(codes.Uninitialized, err.Error())
		}
	}
	for i, netIf := range providerSpec.NetworkInterfaces {
		for _, instanceNetIf := range targetInstance.NetworkInterfaces {
			if netIf.Ipv6PrefixCount != nil && *instanceNetIf.Attachment.DeviceIndex == int64(i) {
				input := &ec2.AssignIpv6AddressesInput{
					NetworkInterfaceId: instanceNetIf.NetworkInterfaceId,
					Ipv6PrefixCount:    netIf.Ipv6PrefixCount,
				}
				klog.V(3).Infof("On VM %q associated with machine %s, assigning ipv6PrefixCount: %d to networkInterface %q",
					providerID, request.Machine.Name, *netIf.Ipv6PrefixCount, *instanceNetIf.NetworkInterfaceId)
				_, err = svc.AssignIpv6Addresses(input)
				if err != nil {
					return nil, status.Error(codes.Uninitialized, err.Error())
				}
			}
		}
	}
	return &driver.InitializeMachineResponse{
		ProviderID: providerID,
		NodeName:   *targetInstance.PrivateDnsName,
	}, nil
}

// returns Placement Object required in ec2.RunInstancesInput
func getPlacementObj(req *driver.CreateMachineRequest) (placementobj *ec2.Placement, err error) {
	placementobj = &ec2.Placement{}

	requestAnnotations := req.Machine.Spec.NodeTemplateSpec.ObjectMeta.Annotations

	if placementAnnotation, ok := requestAnnotations[awsPlacement]; ok && placementAnnotation != "" {
		placementAnnotationsRaw := []byte(placementAnnotation)
		if err = json.Unmarshal(placementAnnotationsRaw, placementobj); err != nil {
			return nil, err
		}
	}

	if *placementobj == (ec2.Placement{}) {
		return nil, nil
	}
	return placementobj, nil
}

// DeleteMachine handles a machine deletion request
func (d *Driver) DeleteMachine(_ context.Context, req *driver.DeleteMachineRequest) (resp *driver.DeleteMachineResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(deleteMachineOperationLabel, &err)()

	var (
		instances  []*ec2.Instance
		instanceID string
		secret     = req.Secret
	)

	// Check if the MachineClass is for the supported cloud provider
	if req.MachineClass.Provider != ProviderAWS {
		err = fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAWS)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Log messages to track delete request
	klog.V(3).Infof("Machine deletion request has been received for %q", req.Machine.Name)
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

		_, instanceID, err = decodeRegionAndInstanceID(req.Machine.Spec.ProviderID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		err = terminateInstance(req, svc, instanceID)
		if err != nil {
			return nil, err
		}
		klog.V(3).Infof("VM %q for Machine %q was terminated successfully", req.Machine.Spec.ProviderID, req.Machine.Name)

	} else {
		// ProviderID doesn't exist, hence check for any existing machine and then delete if exists
		instances, err = getMachineInstancesByTagsAndStatus(svc, req.Machine.Name, providerSpec.Tags)
		if err != nil {
			if isNotFoundError(err) {
				klog.V(3).Infof("No matching VM found. Termination successful for machine object %q", req.Machine.Name)
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
func (d *Driver) GetMachineStatus(_ context.Context, req *driver.GetMachineStatusRequest) (resp *driver.GetMachineStatusResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(getMachineStatusOperationLabel, &err)()

	var (
		secret       = req.Secret
		machineClass = req.MachineClass
	)

	// Check if the MachineClass is for the supported cloud provider
	if req.MachineClass.Provider != ProviderAWS {
		err = fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAWS)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Log messages to track start and end of request
	klog.V(3).Infof("Get request has been recieved for %q", req.Machine.Name)
	providerSpec, err := decodeProviderSpecAndSecret(machineClass, secret)
	if err != nil {
		return nil, err
	}

	svc, err := d.createSVC(secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	instances, err := d.getMatchingInstancesForMachine(req.Machine, svc, providerSpec.Tags)
	if err != nil {
		return nil, err
	} else if len(instances) > 1 {
		instanceIDs := make([]string, 0, len(instances))
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

	// if SrcAnDstCheckEnabled is false then check attribute on instance and return Uninitialized error if not matching.
	if providerSpec.SrcAndDstChecksEnabled != nil && !*providerSpec.SrcAndDstChecksEnabled {
		if *requiredInstance.SourceDestCheck {
			msg := fmt.Sprintf("VM %q associated with machine %q has SourceDestCheck=%t despite providerSpec.SrcAndDstChecksEnabled=%t",
				*requiredInstance.InstanceId, req.Machine.Name, *requiredInstance.SourceDestCheck, *providerSpec.SrcAndDstChecksEnabled)
			klog.Warning(msg)
			return response, status.Error(codes.Uninitialized, msg)
		}
	}

	klog.V(3).Infof("Machine get request has been processed successfully for %q", req.Machine.Name)
	return response, nil
}

// ListMachines lists all the machines possibly created by a machineClass
func (d *Driver) ListMachines(_ context.Context, req *driver.ListMachinesRequest) (resp *driver.ListMachinesResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(listMachinesOperationLabel, &err)()

	var (
		machineClass = req.MachineClass
		secret       = req.Secret
	)

	// Check if the MachineClass is for the supported cloud provider
	if req.MachineClass.Provider != ProviderAWS {
		err = fmt.Errorf("requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, ProviderAWS)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

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
	resp = &driver.ListMachinesResponse{
		MachineList: listOfVMs,
	}
	return resp, nil
}

// GetVolumeIDs returns a list of Volume IDs for all PV Specs for whom a provider volume was found
func (d *Driver) GetVolumeIDs(_ context.Context, req *driver.GetVolumeIDsRequest) (resp *driver.GetVolumeIDsResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(getVolumeIDsOperationLabel, &err)()

	var (
		volumeID  string
		volumeIDs []string
	)

	// Log messages to track start and end of request
	klog.V(3).Infof("GetVolumeIDs request has been received for %q", req.PVSpecs)

	for _, spec := range req.PVSpecs {

		if spec.AWSElasticBlockStore != nil {
			volumeID, err = kubernetesVolumeIDToEBSVolumeID(spec.AWSElasticBlockStore.VolumeID)
			if err != nil {
				klog.Errorf("Failed to translate Kubernetes volume ID '%s' to EBS volume ID: %v", spec.AWSElasticBlockStore.VolumeID, err)
				continue
			}

			volumeIDs = append(volumeIDs, volumeID)
		} else if spec.CSI != nil && spec.CSI.Driver == awsEBSDriverName && spec.CSI.VolumeHandle != "" {
			volumeID = spec.CSI.VolumeHandle
			volumeIDs = append(volumeIDs, volumeID)
		}
	}

	klog.V(3).Infof("GetVolumeIDs machines request has been processed successfully. \nList: %v", volumeIDs)

	resp = &driver.GetVolumeIDsResponse{
		VolumeIDs: volumeIDs,
	}
	return resp, nil
}
