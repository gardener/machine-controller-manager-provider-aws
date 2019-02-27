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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/sampleprovider/machineserver.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package aws

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gardener/machine-spec/lib/go/cmi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
)

// CreateMachine handles a machine creation request
// REQUIRED METHOD
//
// REQUEST PARAMETERS (cmi.CreateMachineRequest)
// Name                 string              Contains the identification name/tag used to link the machine object with VM on cloud provider
// ProviderSpec         bytes(blob)         Template/Configuration of the machine to be created is given by at the provider
// Secrets              map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.CreateMachineResponse)
// MachineID            string              Unique identification of the VM at the cloud provider. This could be the same/different from req.Name.
//                                          MachineID typically matches with the node.Spec.ProviderID on the node object.
//                                          Eg: gce://project-name/region/vm-machineID
// NodeName             string              Returns the name of the node-object that the VM register's with Kubernetes.
//                                          This could be different from req.Name as well
//
func (ms *MachineServer) CreateMachine(ctx context.Context, req *cmi.CreateMachineRequest) (*cmi.CreateMachineResponse, error) {
	fmt.Println("TestLogs: Machine Created ", req.Name)

	var ProviderSpec api.AWSProviderSpec
	err := json.Unmarshal(req.ProviderSpec, &ProviderSpec)
	if err != nil {
		glog.Errorf("Could not parse ProviderSpec into AWSProviderSpec, Error: %s", err)
	}

	ProviderAccessKeyID, KeyIDExists := req.Secrets["providerAccessKeyId"]
	ProviderAccessKey, KeyExists := req.Secrets["providerSecretAccessKey"]
	UserData, UserDataExists := req.Secrets["userData"]
	if !KeyIDExists || !KeyExists || !UserDataExists {
		glog.Errorf("Invalidate Secret Map")
		return nil, fmt.Errorf("Invalidate Secret Map")
	}

	//TODO: Make validation better to make sure if all the fields under secret are covered.
	var Secrets api.Secrets
	Secrets.ProviderAccessKeyID = string(ProviderAccessKeyID)
	Secrets.ProviderSecretAccessKey = string(ProviderAccessKey)
	Secrets.UserData = string(UserData)

	Resp := &cmi.CreateMachineResponse{}

	// Core logic for MachineCreation - CP specific

	//Validate the Spec and Secrets
	/*
		alidationErr := validateAWSProviderSpec(&ProviderSpec, &Secrets)
		if validationerr.ToAggregate() != nil && len(validationerr.ToAggregate().Errors()) > 0 {
			glog.Errorf("Validation of Machine failed %s", validationerr.ToAggregate().Error())
			return nil
		}*/

	svc := createSVC(Secrets, ProviderSpec.Region)
	UserDataEnc := base64.StdEncoding.EncodeToString(UserData)

	var imageIds []*string
	imageID := aws.String(ProviderSpec.AMI)
	imageIds = append(imageIds, imageID)

	describeImagesRequest := ec2.DescribeImagesInput{
		ImageIds: imageIds,
	}
	output, err := svc.DescribeImages(&describeImagesRequest)
	if err != nil {
		return nil, err
	}

	var blkDeviceMappings []*ec2.BlockDeviceMapping
	deviceName := output.Images[0].RootDeviceName
	volumeSize := ProviderSpec.BlockDevices[0].Ebs.VolumeSize
	volumeType := ProviderSpec.BlockDevices[0].Ebs.VolumeType
	blkDeviceMapping := ec2.BlockDeviceMapping{
		DeviceName: deviceName,
		Ebs: &ec2.EbsBlockDevice{
			VolumeSize: &volumeSize,
			VolumeType: &volumeType,
		},
	}
	if volumeType == "io1" {
		blkDeviceMapping.Ebs.Iops = &ProviderSpec.BlockDevices[0].Ebs.Iops
	}
	blkDeviceMappings = append(blkDeviceMappings, &blkDeviceMapping)

	// Add tags to the created machine
	tagList := []*ec2.Tag{}
	for idx, element := range ProviderSpec.Tags {
		if idx == "Name" {
			// Name tag cannot be set, as its used to identify backing machine object
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
		Value: aws.String(req.Name),
	}
	tagList = append(tagList, &nameTag)

	tagInstance := &ec2.TagSpecification{
		ResourceType: aws.String("instance"),
		Tags:         tagList,
	}

	// Specify the details of the machine that you want to create.
	inputConfig := ec2.RunInstancesInput{
		// An Amazon Linux AMI ID for t2.micro machines in the us-west-2 region
		ImageId:      aws.String(ProviderSpec.AMI),
		InstanceType: aws.String(ProviderSpec.MachineType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		UserData:     &UserDataEnc,
		KeyName:      aws.String(ProviderSpec.KeyName),
		SubnetId:     aws.String(ProviderSpec.NetworkInterfaces[0].SubnetID),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: &(ProviderSpec.IAM.Name),
		},
		SecurityGroupIds:    []*string{aws.String(ProviderSpec.NetworkInterfaces[0].SecurityGroupIDs[0])},
		BlockDeviceMappings: blkDeviceMappings,
		TagSpecifications:   []*ec2.TagSpecification{tagInstance},
	}

	runResult, err := svc.RunInstances(&inputConfig)
	if err != nil {
		return Resp, err
	}

	// End of Core Logic MachineCreation - CP Specific
	// TODO Move from fmt to glog - generic-way at all places.
	fmt.Println("TestLogs: Printing ProviderSpec : ", ProviderSpec)
	Resp = &cmi.CreateMachineResponse{
		MachineID: encodeMachineID(ProviderSpec.Region, *runResult.Instances[0].InstanceId),
		NodeName:  *runResult.Instances[0].PrivateDnsName,
	}
	return Resp, nil
}

// DeleteMachine handles a machine deletion request
// REQUIRED METHOD
//
// REQUEST PARAMETERS (cmi.DeleteMachineRequest)
// MachineID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
func (ms *MachineServer) DeleteMachine(ctx context.Context, req *cmi.DeleteMachineRequest) (*cmi.DeleteMachineResponse, error) {
	fmt.Println("TestLogs: Machine Deleted ...", req.MachineID)

	//Validate if map contains necessary values.
	ProviderAccessKeyID, KeyIDExists := req.Secrets["providerAccessKeyId"]
	ProviderAccessKey, KeyExists := req.Secrets["providerSecretAccessKey"]
	if !KeyIDExists || !KeyExists {
		glog.Errorf("Invalid Secret Map")
		return &cmi.DeleteMachineResponse{}, nil
	}

	var Secrets api.Secrets
	Secrets.ProviderAccessKeyID = string(ProviderAccessKeyID)
	Secrets.ProviderSecretAccessKey = string(ProviderAccessKey)

	region, machineID := decodeRegionAndMachineID(req.MachineID)

	svc := createSVC(Secrets, region)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(machineID),
		},
		DryRun: aws.Bool(true),
	}
	_, err := svc.TerminateInstances(input)
	awsErr, ok := err.(awserr.Error)
	if ok && awsErr.Code() == "DryRunOperation" {
		input.DryRun = aws.Bool(false)
		output, err := svc.TerminateInstances(input)
		if err != nil {
			glog.Errorf("Could not terminate machine: %s", err.Error())
			return nil, err
		}

		vmState := output.TerminatingInstances[0]
		//glog.Info(vmState.PreviousState, vmState.CurrentState)

		if *vmState.CurrentState.Name == "shutting-down" ||
			*vmState.CurrentState.Name == "terminated" {
			return nil, nil
		}

		err = errors.New("Machine already terminated")
	}

	glog.Errorf("Could not terminate machine: %s", err.Error())
	return nil, err
}

// GetMachine handles a machine details fetching request
//
// REQUEST PARAMETERS (cmi.GetMachineRequest)
// MachineID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.GetMachineResponse)
// Exists           bool                Returns a boolean value which is set to true when it exists on the cloud provider
// Status           enum                Contains the status of the machine on the cloud provider mapped to the enum values - {Unknown, Stopped, Running}
//
func (ms *MachineServer) GetMachine(ctx context.Context, req *cmi.GetMachineRequest) (*cmi.GetMachineResponse, error) {
	fmt.Println("TestLogs: Get Machine")

	ProviderAccessKeyID, KeyIDExists := req.Secrets["providerAccessKeyId"]
	ProviderAccessKey, KeyExists := req.Secrets["providerSecretAccessKey"]
	if !KeyIDExists || !KeyExists {
		glog.Errorf("Invalidate Secret Map")
		return nil, fmt.Errorf("Invalidate Secret Map")
	}

	//TODO: Make validation better to make sure if all the fields under secret are covered.
	var Secrets api.Secrets
	Secrets.ProviderAccessKeyID = string(ProviderAccessKeyID)
	Secrets.ProviderSecretAccessKey = string(ProviderAccessKey)
	region, machineID := decodeRegionAndMachineID(req.MachineID)

	svc := createSVC(Secrets, region)
	input := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("instance-id"),
				Values: []*string{
					&machineID,
				},
			},
		},
	}

	runResult, err := svc.DescribeInstances(&input)
	if err != nil {
		glog.Errorf("AWS driver is returning error while describe instances request is sent: %s", err)
		return nil, err
	}

	count := 0
	for _, reservation := range runResult.Reservations {
		for range reservation.Instances {
			count++
		}
	}

	if count > 1 {
		response := cmi.GetMachineResponse{
			Exists: true,
		}
		return &response, nil
	}

	response := cmi.GetMachineResponse{
		Exists: false,
	}
	return &response, nil
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
// MachineList      map<string,string>  A map containing the keys as the MachineID and value as the MachineName
//                                      for all machine's who where possibilly created by this ProviderSpec
//
func (ms *MachineServer) ListMachines(ctx context.Context, req *cmi.ListMachinesRequest) (*cmi.ListMachinesResponse, error) {
	fmt.Println("TestLogs: List of Machine ")

	var ProviderSpec api.AWSProviderSpec
	err := json.Unmarshal(req.ProviderSpec, &ProviderSpec)
	if err != nil {
		glog.Errorf("Could not parse ProviderSpec into AWSProviderSpec, Error: %s", err)
	}

	ProviderAccessKeyID, KeyIDExists := req.Secrets["providerAccessKeyId"]
	ProviderAccessKey, KeyExists := req.Secrets["providerSecretAccessKey"]
	UserData, UserDataExists := req.Secrets["userData"]
	if !KeyIDExists || !KeyExists || !UserDataExists {
		glog.Errorf("Invalidate Secret Map")
		return nil, fmt.Errorf("Invalidate Secret Map")
	}

	//TODO: Make validation better to make sure if all the fields under secret are covered.
	var Secrets api.Secrets
	Secrets.ProviderAccessKeyID = string(ProviderAccessKeyID)
	Secrets.ProviderSecretAccessKey = string(ProviderAccessKey)
	Secrets.UserData = string(UserData)

	// Core Logic for Listing the Machines - Provider Specific

	/*
		//Validate the Spec and Secrets
		ValidationErr := validateAWSProviderSpec(&ProviderSpec, &Secrets)
		if ValidationErr != nil {
			Resp = &cmi.ListMachinesResponse{
				Error: ValidationErrToString(ValidationErr),
			}
			fmt.Println("Error while validating ProviderSpec", ValidationErr)
			return Resp, fmt.Errorf(Resp.Error)
		}*/

	clusterName := ""
	nodeRole := ""

	for key := range ProviderSpec.Tags {
		if strings.Contains(key, "kubernetes.io/cluster/") {
			clusterName = key
		} else if strings.Contains(key, "kubernetes.io/role/") {
			nodeRole = key
		}
	}

	if clusterName == "" || nodeRole == "" {
		return nil, errors.New("Couldn't map machine class to a cluster")
	}

	svc := createSVC(Secrets, ProviderSpec.Region)
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
		glog.Errorf("AWS driver is returning error while describe instances request is sent: %s", err)
		return nil, err
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
			listOfVMs[encodeMachineID(ProviderSpec.Region, *instance.InstanceId)] = machineName
		}
	}

	// Core logic ends here.
	Resp := &cmi.ListMachinesResponse{
		MachineList: listOfVMs,
	}
	return Resp, nil
}

// ShutDownMachine handles a machine shutdown/power-off/stop request
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (cmi.ShutDownMachineRequest)
// MachineID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
func (ms *MachineServer) ShutDownMachine(ctx context.Context, req *cmi.ShutDownMachineRequest) (*cmi.ShutDownMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("ShutDown machine request has been recieved for %q", req.MachineID)
	return nil, status.Error(codes.Unimplemented, "")
}
