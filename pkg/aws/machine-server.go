/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was copied and modified from the kubernetes-csi/drivers project
https://github.com/kubernetes-csi/drivers/blob/master/pkg/csi-common/nodeserver-default.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package aws

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gardener/machine-spec/lib/go/cmi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
)

// NOTE
//
// The basic working of the controller will work with just implementing the CreateMachine() & DeleteMachine() methods.
// You can first implement these two methods and check the working of the controller.
// Once this works you can implement the rest of the methods.
// Implementation of few methods like - ShutDownMachine() are optional, however we highly recommend implementing it as well.

// CreateMachine handles a machine creation request
//
// REQUEST PARAMETERS (cmi.CreateMachineRequest)
// Name                 string              Contains the identification name/tag used to link the machine object with VM on cloud provider
// ProviderSpec         bytes(blob)         Template/Configuration of the machine to be created is given by at the provider
// Secrets              map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.CreateMachineResponse)
// ProviderID            string              Unique identification of the VM at the cloud provider. This could be the same/different from req.MachineName.
//                                          ProviderID typically matches with the node.Spec.ProviderID on the node object.
//                                          Eg: gce://project-name/region/vm-ProviderID
// NodeName             string              Returns the name of the node-object that the VM register's with Kubernetes.
//                                          This could be different from req.MachineName as well
//
// OPTIONAL IMPLEMENTATION LOGIC
// It is optionally expected by the safety controller to use an identification mechanisms to map the VM Created by a providerSpec.
// These could be done using tag(s)/resource-groups etc.
// This logic is used by safety controller to delete orphan VMs which are not backed by any machine CRD
//
func (ms *MachinePlugin) CreateMachine(ctx context.Context, req *cmi.CreateMachineRequest) (*cmi.CreateMachineResponse, error) {
	// Log messages to track request
	glog.V(2).Infof("Machine creation request has been recieved for %q", req.MachineName)

	providerSpec, secrets, err := decodeProviderSpecAndSecret(req.ProviderSpec, req.Secrets)
	if err != nil {
		return nil, err
	}

	svc, err := ms.createSVC(secrets, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	UserDataEnc := base64.StdEncoding.EncodeToString([]byte(secrets.UserData))

	var imageIds []*string
	imageID := aws.String(providerSpec.AMI)
	imageIds = append(imageIds, imageID)

	describeImagesRequest := ec2.DescribeImagesInput{
		ImageIds: imageIds,
	}
	output, err := svc.DescribeImages(&describeImagesRequest)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var blkDeviceMappings []*ec2.BlockDeviceMapping
	deviceName := output.Images[0].RootDeviceName
	volumeSize := providerSpec.BlockDevices[0].Ebs.VolumeSize
	volumeType := providerSpec.BlockDevices[0].Ebs.VolumeType
	blkDeviceMapping := ec2.BlockDeviceMapping{
		DeviceName: deviceName,
		Ebs: &ec2.EbsBlockDevice{
			VolumeSize: &volumeSize,
			VolumeType: &volumeType,
		},
	}
	if volumeType == "io1" {
		blkDeviceMapping.Ebs.Iops = &providerSpec.BlockDevices[0].Ebs.Iops
	}
	blkDeviceMappings = append(blkDeviceMappings, &blkDeviceMapping)

	// Add tags to the created machine
	tagList := []*ec2.Tag{}
	for idx, element := range providerSpec.Tags {
		if idx == "Name" {
			// Name tag cannot be set, as its used to identify backing machine object
			glog.Warning("Name tag cannot be set on AWS instance, as its used to identify backing machine object")
			continue
		}
		newTag := ec2.Tag{
			Key:   aws.String(idx),
			Value: aws.String(element),
		}
		tagList = append(tagList, &newTag)
	}
	nameTag := ec2.Tag{
		Key:   aws.String("Name"),
		Value: aws.String(req.MachineName),
	}
	tagList = append(tagList, &nameTag)

	tagInstance := &ec2.TagSpecification{
		ResourceType: aws.String("instance"),
		Tags:         tagList,
	}

	// Specify the details of the machine that you want to create.
	inputConfig := ec2.RunInstancesInput{
		// An Amazon Linux AMI ID for t2.micro machines in the us-west-2 region
		ImageId:      aws.String(providerSpec.AMI),
		InstanceType: aws.String(providerSpec.MachineType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		UserData:     &UserDataEnc,
		KeyName:      aws.String(providerSpec.KeyName),
		SubnetId:     aws.String(providerSpec.NetworkInterfaces[0].SubnetID),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: &(providerSpec.IAM.Name),
		},
		SecurityGroupIds:    []*string{aws.String(providerSpec.NetworkInterfaces[0].SecurityGroupIDs[0])},
		BlockDeviceMappings: blkDeviceMappings,
		TagSpecifications:   []*ec2.TagSpecification{tagInstance},
	}

	runResult, err := svc.RunInstances(&inputConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &cmi.CreateMachineResponse{
		ProviderID:     encodeProviderID(providerSpec.Region, *runResult.Instances[0].InstanceId),
		NodeName:       *runResult.Instances[0].PrivateDnsName,
		LastKnownState: []byte("Created" + *runResult.Instances[0].InstanceId),
	}

	glog.V(2).Infof("VM with Provider-ID: %q created for Machine: %q", response.ProviderID, req.MachineName)
	return response, nil
}

// DeleteMachine handles a machine deletion request
//
// REQUEST PARAMETERS (cmi.DeleteMachineRequest)
// ProviderID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
func (ms *MachinePlugin) DeleteMachine(ctx context.Context, req *cmi.DeleteMachineRequest) (*cmi.DeleteMachineResponse, error) {
	// Log messages to track delete request
	glog.V(2).Infof("Machine deletion request has been recieved for %q", req.MachineName)
	defer glog.V(2).Infof("Machine deletion request has been processed for %q", req.MachineName)

	providerSpec, secrets, err := decodeProviderSpecAndSecret(req.ProviderSpec, req.Secrets)
	if err != nil {
		return nil, err
	}

	instances, err := ms.getInstancesFromMachineName(req.MachineName, providerSpec, secrets)
	if err != nil {
		return nil, err
	}

	svc, err := ms.createSVC(secrets, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, instance := range instances {
		input := &ec2.TerminateInstancesInput{
			InstanceIds: []*string{
				aws.String(*instance.InstanceId),
			},
			DryRun: aws.Bool(false),
		}
		_, err = svc.TerminateInstances(input)
		awsErr, ok := err.(awserr.Error)
		if ok &&
			(awsErr.Code() == ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdNotFound) {
			glog.V(2).Infof("VM %q for Machine %q does not exist", *instance.InstanceId, req.MachineName)
		} else if err != nil {
			glog.V(2).Infof("VM %q for Machine %q couldn't be terminated: %s",
				*instance.InstanceId,
				req.MachineName,
				err.Error(),
			)
			return nil, status.Error(codes.Internal, err.Error())
		}

		glog.V(2).Infof("VM %q for Machine %q was terminated succesfully", *instance.InstanceId, req.MachineName)
	}

	return &cmi.DeleteMachineResponse{}, nil
}

// GetMachineStatus handles a machine details fetching request
//
// REQUEST PARAMETERS (cmi.GetMachineStatusRequest)
// ProviderID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.GetMachineStatusResponse)
// Exists           bool                Returns a boolean value which is set to true when it exists on the cloud provider
// Status           enum                Contains the status of the machine on the cloud provider mapped to the enum values - {Unknown, Stopped, Running}
//
func (ms *MachinePlugin) GetMachineStatus(ctx context.Context, req *cmi.GetMachineStatusRequest) (*cmi.GetMachineStatusResponse, error) {
	// Log messages to track start and end of request
	glog.V(2).Infof("Get request has been recieved for %q", req.MachineName)

	providerSpec, secrets, err := decodeProviderSpecAndSecret(req.ProviderSpec, req.Secrets)
	if err != nil {
		return nil, err
	}

	instances, err := ms.getInstancesFromMachineName(req.MachineName, providerSpec, secrets)
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

	response := &cmi.GetMachineStatusResponse{
		NodeName:   *requiredInstance.PrivateDnsName,
		ProviderID: encodeProviderID(providerSpec.Region, *requiredInstance.InstanceId),
	}

	glog.V(2).Infof("Machine get request has been processed successfully for %q", req.MachineName)
	return response, nil
}

// ShutDownMachine handles a machine shutdown/power-off/stop request
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (cmi.ShutDownMachineRequest)
// ProviderID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
func (ms *MachinePlugin) ShutDownMachine(ctx context.Context, req *cmi.ShutDownMachineRequest) (*cmi.ShutDownMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("ShutDown machine request has been recieved for %q", req.MachineName)
	defer glog.V(2).Infof("Machine shutdown request has been processed successfully for %q", req.MachineName)

	providerSpec, secrets, err := decodeProviderSpecAndSecret(req.ProviderSpec, req.Secrets)
	if err != nil {
		return nil, err
	}

	instances, err := ms.getInstancesFromMachineName(req.MachineName, providerSpec, secrets)
	if err != nil {
		return nil, err
	}

	svc, err := ms.createSVC(secrets, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, instance := range instances {
		input := &ec2.StopInstancesInput{
			InstanceIds: []*string{
				aws.String(*instance.InstanceId),
			},
			DryRun: aws.Bool(true),
		}
		_, err = svc.StopInstances(input)
		awsErr, ok := err.(awserr.Error)
		if ok && awsErr.Code() == "DryRunOperation" {
			input.DryRun = aws.Bool(false)
			_, err := svc.StopInstances(input)
			if err != nil {
				glog.V(2).Infof("VM %q for Machine %q couldn't be stopped: %s",
					*instance.InstanceId,
					req.MachineName,
					err.Error(),
				)
				return nil, status.Error(codes.Internal, err.Error())
			}

			glog.V(2).Infof("VM %q for Machine %q was shutdown", *instance.InstanceId, req.MachineName)

		} else if ok &&
			(awsErr.Code() == ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdMalformed ||
				awsErr.Code() == ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdNotFound) {

			glog.V(2).Infof("VM %q for Machine %q does not exist", *instance.InstanceId, req.MachineName)
			return &cmi.ShutDownMachineResponse{}, nil
		}
	}

	return &cmi.ShutDownMachineResponse{}, nil
}

// ListMachines lists all the machines possibilly created by a providerSpec
// Identifying machines created by a given providerSpec depends on the OPTIONAL IMPLEMENTATION LOGIC
// you have used to identify machines created by a providerSpec. It could be tags/resource-groups etc
//
// REQUEST PARAMETERS (cmi.ListMachinesRequest)
// ProviderSpec     bytes(blob)         Template/Configuration of the machine that wouldn've been created by this ProviderSpec (Machine Class)
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.ListMachinesResponse)
// MachineList      map<string,string>  A map containing the keys as the ProviderID and value as the MachineName
//                                      for all machine's who where possibilly created by this ProviderSpec
//
func (ms *MachinePlugin) ListMachines(ctx context.Context, req *cmi.ListMachinesRequest) (*cmi.ListMachinesResponse, error) {
	// Log messages to track start and end of request
	glog.V(2).Infof("List machines request has been recieved for %q", req.ProviderSpec)

	providerSpec, secrets, err := decodeProviderSpecAndSecret(req.ProviderSpec, req.Secrets)
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

	svc, err := ms.createSVC(secrets, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	input := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag-key"),
				Values: []*string{
					&clusterName,
				},
			},
			&ec2.Filter{
				Name: aws.String("tag-key"),
				Values: []*string{
					&nodeRole,
				},
			},
			&ec2.Filter{
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
		glog.Errorf("AWS plugin is returning error while describe instances request is sent: %s", err)
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
			listOfVMs[encodeProviderID(providerSpec.Region, *instance.InstanceId)] = machineName
		}
	}

	glog.V(2).Infof("List machines request has been processed successfully")
	// Core logic ends here.
	Resp := &cmi.ListMachinesResponse{
		MachineList: listOfVMs,
	}
	return Resp, nil
}

// GetVolumeIDs returns a list of Volume IDs for all PV Specs for whom an AWS volume was found
//
// REQUEST PARAMETERS (cmi.GetVolumeIDsRequest)
// PVSpecList       bytes(blob)         PVSpecsList is a list PV specs for whom volume-IDs are required. Plugin should parse this raw data into pre-defined list of PVSpecs.
//
// RESPONSE PARAMETERS (cmi.GetVolumeIDsResponse)
// VolumeIDs       repeated string      VolumeIDs is a repeated list of VolumeIDs.
//
func (ms *MachinePlugin) GetVolumeIDs(ctx context.Context, req *cmi.GetVolumeIDsRequest) (*cmi.GetVolumeIDsResponse, error) {
	var (
		volumeIDs   []string
		volumeSpecs []*corev1.PersistentVolumeSpec
	)

	// Log messages to track start and end of request
	glog.V(2).Infof("GetVolumeIDs request has been recieved for %q", req.PVSpecList)

	err := json.Unmarshal(req.PVSpecList, &volumeSpecs)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for i := range volumeSpecs {
		spec := volumeSpecs[i]
		if spec.AWSElasticBlockStore == nil {
			// Not an aws volume
			continue
		}
		volumeID := spec.AWSElasticBlockStore.VolumeID
		volumeIDs = append(volumeIDs, volumeID)
	}

	glog.V(2).Infof("GetVolumeIDs machines request has been processed successfully. \nList: %v", volumeIDs)

	Resp := &cmi.GetVolumeIDsResponse{
		VolumeIDs: volumeIDs,
	}
	return Resp, nil
}
