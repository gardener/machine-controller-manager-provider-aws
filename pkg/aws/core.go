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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	awserror "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/errors"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/cpi"
)

const (
	createMachineOperationLabel     = "create_machine"
	initializeMachineOperationLabel = "initialize_machine"
	deleteMachineOperationLabel     = "delete_machine"
	listMachinesOperationLabel      = "list_machine"
	getMachineStatusOperationLabel  = "get_machine_status"
	getVolumeIDsOperationLabel      = "get_volume_ids"
)

// Driver is the driver struct for holding AWS machine information
type Driver struct {
	CPI cpi.ClientProviderInterface
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
func NewAWSDriver(cpi cpi.ClientProviderInterface) driver.Driver {
	return &Driver{
		CPI: cpi,
	}
}

// CreateMachine handles a machine creation request
func (d *Driver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (resp *driver.CreateMachineResponse, err error) {
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

	client, err := d.createClient(ctx, secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(awserror.GetMCMErrorCodeForCreateMachine(err), err.Error())
	}

	if userData, exists = secret.Data["userData"]; !exists {
		return nil, status.Error(codes.Internal, "userData doesn't exist")
	}
	UserDataEnc := base64.StdEncoding.EncodeToString([]byte(userData))

	var imageIds []string
	imageID := providerSpec.AMI
	imageIds = append(imageIds, imageID)

	describeImagesRequest := &ec2.DescribeImagesInput{
		ImageIds: imageIds,
	}
	output, err := client.DescribeImages(ctx, describeImagesRequest)
	if err != nil {
		return nil, status.Error(awserror.GetMCMErrorCodeForCreateMachine(err), err.Error())
	} else if len(output.Images) < 1 {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Image %s not found", imageID))
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

	var networkInterfaceSpecs []ec2types.InstanceNetworkInterfaceSpecification

	for i, netIf := range providerSpec.NetworkInterfaces {
		spec := ec2types.InstanceNetworkInterfaceSpecification{
			Groups:                   netIf.SecurityGroupIDs,
			DeviceIndex:              aws.Int32(int32(i)), // #nosec: G115 -- index will not exceed int32 limits
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
			spec.PrimaryIpv6 = aws.Bool(true)
		}

		networkInterfaceSpecs = append(networkInterfaceSpecs, spec)
	}

	// Specify the details of the machine that you want to create.
	iam := &ec2types.IamInstanceProfileSpecification{}
	if len(providerSpec.IAM.Name) > 0 {
		iam.Name = &providerSpec.IAM.Name
	} else if len(providerSpec.IAM.ARN) > 0 {
		iam.Arn = &providerSpec.IAM.ARN
	}

	var metadataOptions *ec2types.InstanceMetadataOptionsRequest
	if providerSpec.InstanceMetadataOptions != nil {
		metadataOptions = &ec2types.InstanceMetadataOptionsRequest{
			HttpEndpoint:            ec2types.InstanceMetadataEndpointState(providerSpec.InstanceMetadataOptions.HTTPEndpoint),
			HttpPutResponseHopLimit: providerSpec.InstanceMetadataOptions.HTTPPutResponseHopLimit,
			HttpTokens:              ec2types.HttpTokensState(providerSpec.InstanceMetadataOptions.HTTPTokens),
		}
	}

	inputConfig := &ec2.RunInstancesInput{
		BlockDeviceMappings: blkDeviceMappings,
		ImageId:             aws.String(providerSpec.AMI),
		InstanceType:        ec2types.InstanceType(providerSpec.MachineType),
		MinCount:            aws.Int32(1),
		MaxCount:            aws.Int32(1),
		UserData:            &UserDataEnc,
		IamInstanceProfile:  iam,
		NetworkInterfaces:   networkInterfaceSpecs,
		TagSpecifications:   []ec2types.TagSpecification{tagInstance, tagVolume, tagNetworkInterface},
		MetadataOptions:     metadataOptions,
	}

	if providerSpec.KeyName != nil && len(*providerSpec.KeyName) > 0 {
		inputConfig.KeyName = aws.String(*providerSpec.KeyName)
	}

	if cpuOptions := providerSpec.CPUOptions; cpuOptions != nil {
		inputConfig.CpuOptions = &ec2types.CpuOptionsRequest{
			CoreCount:      cpuOptions.CoreCount,
			ThreadsPerCore: cpuOptions.ThreadsPerCore,
		}
	}

	// Set the AWS Capacity Reservation target. Using an 'open' preference means that if the reservation is not found, then
	// instances are launched with regular on-demand capacity.
	if providerSpec.CapacityReservationTarget != nil {
		inputConfig.CapacityReservationSpecification = &ec2types.CapacityReservationSpecification{
			CapacityReservationPreference: ec2types.CapacityReservationPreference(providerSpec.CapacityReservationTarget.CapacityReservationPreference),
			CapacityReservationTarget: &ec2types.CapacityReservationTarget{
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
		inputConfig.InstanceMarketOptions = &ec2types.InstanceMarketOptionsRequest{
			MarketType: ec2types.MarketTypeSpot,
			SpotOptions: &ec2types.SpotMarketOptions{
				SpotInstanceType: ec2types.SpotInstanceTypeOneTime,
			},
		}

		if *providerSpec.SpotPrice != "" {
			inputConfig.InstanceMarketOptions.SpotOptions.MaxPrice = providerSpec.SpotPrice
		}
	}

	runResult, err := client.RunInstances(ctx, inputConfig)
	if err != nil {
		return nil, status.Error(awserror.GetMCMErrorCodeForCreateMachine(err), err.Error())
	}
	var instanceID, providerID, nodeName string

	for _, instance := range runResult.Instances {
		if instance.InstanceId != nil {
			instanceID = *instance.InstanceId
			providerID = encodeInstanceID(providerSpec.Region, instanceID)
			if instance.PrivateDnsName != nil {
				nodeName = *instance.PrivateDnsName
			}
			break
		}
	}

	if instanceID == "" {
		return nil, status.Error(codes.Internal, fmt.Sprintf("creation of VM failed for machine %q - no non-empty instanceID found in runResult", machine.Name))
	}

	klog.V(2).Infof("Waiting for VM with Provider-ID %q, for machine %q to be visible to all AWS endpoints", providerID, machine.Name)
	if nodeName == "" {
		klog.Warningf("VM with Provider-ID %q, for machine %q does not yet have a nodeName (instance.PrivateDnsName)", providerID, machine.Name)
	}

	operation := func() (*ec2types.Instance, error) {
		instancesOutput, err := getInstanceByID(ctx, client, instanceID)
		if err != nil {
			return nil, err
		}
		for _, reservation := range instancesOutput.Reservations {
			for _, instance := range reservation.Instances {
				return &instance, nil
			}
		}
		return nil, status.Error(codes.NotFound, fmt.Sprintf("instance with instanceID %s not found", instanceID))
	}

	instance, err := retryWithExponentialBackOff(operation, maxElapsedTimeInBackoff)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("creation of VM %q failed, timed out waiting for eventual consistency. Multiple VMs backing machine obj might spawn, they will be orphan collected", providerID))
	}

	if instance.PrivateDnsName != nil {
		nodeName = *instance.PrivateDnsName
	}

	if nodeName == "" {
		msg := fmt.Sprintf("VM with Provider-ID %q, for machine %q does not yet have a nodeName (instance.PrivateDnsName)", providerID, machine.Name)
		klog.Error(msg)
		return nil, status.Error(codes.Internal, msg)
	}

	response := &driver.CreateMachineResponse{
		ProviderID: providerID,
		NodeName:   nodeName,
	}

	klog.V(2).Infof("VM with Provider-ID %q, for machine %q, nodeName: %q should be visible to all AWS endpoints now", response.ProviderID, machine.Name, nodeName)
	klog.V(3).Infof("VM with Provider-ID: %q created for Machine: %q", response.ProviderID, machine.Name)
	return response, nil
}

// InitializeMachine should handle post-creation, one-time VM instance initialization operations. (Ex: Like setting up special network config, etc)
// The AWS Provider leverages this method to perform disabling of source destination checks for NAT instances.
// See [driver.Driver.InitializeMachine] for further information
func (d *Driver) InitializeMachine(ctx context.Context, request *driver.InitializeMachineRequest) (resp *driver.InitializeMachineResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(initializeMachineOperationLabel, &err)()

	providerSpec, err := decodeProviderSpecAndSecret(request.MachineClass, request.Secret)
	if err != nil {
		return nil, err
	}
	client, err := d.createClient(ctx, request.Secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Uninitialized, err.Error())
	}
	instances, err := d.getMatchingInstancesForMachine(ctx, request.Machine, client, providerSpec.Tags)
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
	if providerSpec.SrcAndDstChecksEnabled != nil && !*providerSpec.SrcAndDstChecksEnabled && ptr.Deref(targetInstance.SourceDestCheck, true) {
		klog.V(3).Infof("Disabling SourceDestCheck on VM %q associated with machine %s", providerID, request.Machine.Name)
		err = disableSrcAndDestCheck(ctx, client, targetInstance.InstanceId)
		if err != nil {
			return nil, status.Error(codes.Uninitialized, err.Error())
		}
	}
	for i, netIf := range providerSpec.NetworkInterfaces {
		for _, instanceNetIf := range targetInstance.NetworkInterfaces {
			// #nosec: G115 -- index will not exceed int32 limits
			if netIf.Ipv6PrefixCount != nil && *instanceNetIf.Attachment.DeviceIndex == int32(i) {
				input := &ec2.AssignIpv6AddressesInput{
					NetworkInterfaceId: instanceNetIf.NetworkInterfaceId,
					Ipv6PrefixCount:    netIf.Ipv6PrefixCount,
				}
				klog.V(3).Infof("On VM %q associated with machine %s, assigning ipv6PrefixCount: %d to networkInterface %q",
					providerID, request.Machine.Name, *netIf.Ipv6PrefixCount, *instanceNetIf.NetworkInterfaceId)
				_, err = client.AssignIpv6Addresses(ctx, input)
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
func getPlacementObj(req *driver.CreateMachineRequest) (placementobj *ec2types.Placement, err error) {
	placementobj = &ec2types.Placement{}

	requestAnnotations := req.Machine.Spec.NodeTemplateSpec.ObjectMeta.Annotations

	if placementAnnotation, ok := requestAnnotations[awsPlacement]; ok && placementAnnotation != "" {
		placementAnnotationsRaw := []byte(placementAnnotation)
		if err = json.Unmarshal(placementAnnotationsRaw, placementobj); err != nil {
			return nil, err
		}
	}

	if *placementobj == (ec2types.Placement{}) {
		return nil, nil
	}
	return placementobj, nil
}

// DeleteMachine handles a machine deletion request
func (d *Driver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (resp *driver.DeleteMachineResponse, err error) {
	defer instrument.DriverAPIMetricRecorderFn(deleteMachineOperationLabel, &err)()

	var (
		instances  []ec2types.Instance
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

	client, err := d.createClient(ctx, req.Secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if req.Machine.Spec.ProviderID != "" {
		// ProviderID exists for machine object, hence terminate the correponding VM

		_, instanceID, err = decodeRegionAndInstanceID(req.Machine.Spec.ProviderID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		err = terminateInstance(ctx, req, client, instanceID)
		if err != nil {
			return nil, err
		}
		klog.V(3).Infof("VM %q for Machine %q was terminated successfully", req.Machine.Spec.ProviderID, req.Machine.Name)

	} else {
		// ProviderID doesn't exist, hence check for any existing machine and then delete if exists
		instances, err = getMachineInstancesByTagsAndStatus(ctx, client, req.Machine.Name, providerSpec.Tags)
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
			err = terminateInstance(ctx, req, client, *instance.InstanceId)
			if err != nil {
				return nil, err
			}
			klog.V(3).Infof("VM %q for Machine %q was terminated succesfully", *instance.InstanceId, req.Machine.Name)
		}
	}

	return &driver.DeleteMachineResponse{}, nil
}

// GetMachineStatus handles a machine get status request
func (d *Driver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (resp *driver.GetMachineStatusResponse, err error) {
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

	client, err := d.createClient(ctx, secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	instances, err := d.getMatchingInstancesForMachine(ctx, req.Machine, client, providerSpec.Tags)
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
		if ptr.Deref(requiredInstance.SourceDestCheck, true) {
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
func (d *Driver) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (resp *driver.ListMachinesResponse, err error) {
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

	client, err := d.createClient(ctx, secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	input := &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: []string{clusterName},
			},
			{
				Name:   aws.String("tag-key"),
				Values: []string{nodeRole},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []string{
					string(ec2types.InstanceStateNamePending),
					string(ec2types.InstanceStateNameRunning),
					string(ec2types.InstanceStateNameStopping),
					string(ec2types.InstanceStateNameStopped),
				},
			},
		},
	}

	listOfVMs := make(map[string]string)
	var nextToken *string
	pageCount := 0

	for {
		input.NextToken = nextToken

		runResult, err := client.DescribeInstances(ctx, input)
		if err != nil {
			klog.Errorf("AWS plugin is returning error while describe instances request is sent: %s (NextToken: %s)", err, ptr.Deref(nextToken, "<nil>"))
			return nil, status.Error(codes.Internal, err.Error())
		}
		pageCount++

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

		if runResult.NextToken == nil {
			break
		}
		nextToken = runResult.NextToken
		klog.V(4).Infof("Fetching next page (page %d) of ListMachines, with NextToken: %s", pageCount+1, *nextToken)
	}

	klog.V(3).Infof("List machines request has been processed successfully, retrieved %d pages, %d VMs for machineClass %q", pageCount, len(listOfVMs), machineClass.Name)
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
