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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	cmicommon "github.com/gardener/machine-controller-manager-provider-aws/pkg/cmi-common"
)

type machineServer struct {
	*cmicommon.DefaultMachineServer
}

func (ms *machineServer) CreateMachine(ctx context.Context, req *cmi.CreateMachineRequest) (*cmi.CreateMachineResponse, error) {
	fmt.Println("TestLogs: Machine Created ", req.Name)

	var ProviderSpec api.AWSProviderSpec
	err := json.Unmarshal(req.ProviderSpec, &ProviderSpec)
	if err != nil {
		glog.Errorf("Could not parse ProviderSpec into AWSProviderSpec", err)
	}

	ProviderAccessKeyId, KeyIDExists := req.Secrets["providerAccessKeyId"]
	ProviderAccessKey, KeyExists := req.Secrets["providerSecretAccessKey"]
	UserData, UserDataExists := req.Secrets["userData"]
	if !KeyIDExists || !KeyExists || !UserDataExists {
		glog.Errorf("Invalidate Secret Map")
		return nil, fmt.Errorf("Invalidate Secret Map")
	}

	//TODO: Make validation better to make sure if all the fields under secret are covered.
	var Secrets api.Secrets
	Secrets.ProviderAccessKeyId = string(ProviderAccessKeyId)
	Secrets.ProviderSecretAccessKey = string(ProviderAccessKey)
	Secrets.UserData = string(UserData)

	Resp := &cmi.CreateMachineResponse{}

	// Core logic for MachineCreation - CP specific

	//Validate the Spec and Secrets
	ValidationErr := validateAWSProviderSpec(&ProviderSpec, &Secrets)
	if ValidationErr != nil {
		Resp = &cmi.CreateMachineResponse{
			Name:  req.Name,
			Error: ValidationErrToString(ValidationErr),
		}
		fmt.Println("Error while validating ProviderSpec", ValidationErr)
		return Resp, fmt.Errorf(Resp.Error)
	}

	svc := createSVC(ProviderSpec, Secrets)
	UserDataEnc := base64.StdEncoding.EncodeToString(UserData)

	var imageIds []*string
	imageID := aws.String(ProviderSpec.AMI)
	imageIds = append(imageIds, imageID)

	describeImagesRequest := ec2.DescribeImagesInput{
		ImageIds: imageIds,
	}
	output, err := svc.DescribeImages(&describeImagesRequest)
	if err != nil {
		Resp.Error = err.Error()
		return Resp, err
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
		Resp.Error = err.Error()
		return Resp, err
	}

	// End of Core Logic MachineCreation - CP Specific
	// TODO Move from fmt to glog - generic-way at all places.
	fmt.Println("TestLogs: Printing ProviderSpec : ", ProviderSpec)
	Resp = &cmi.CreateMachineResponse{
		Name:      req.Name,
		MachineID: encodeMachineID(ProviderSpec.Region, *runResult.Instances[0].InstanceId),
		NodeName:  *runResult.Instances[0].PrivateDnsName,
		Error:     "",
	}
	return Resp, nil
}

func (ms *machineServer) DeleteMachine(ctx context.Context, req *cmi.DeleteMachineRequest) (*cmi.DeleteMachineResponse, error) {
	fmt.Println("TestLogs: Machine Deleted ...", req.MachineID)

	var ProviderSpec api.AWSProviderSpec
	err := json.Unmarshal(req.ProviderSpec, &ProviderSpec)
	if err != nil {
		glog.Errorf("Could not parse ProviderSpec into AWSProviderSpec", err)
	}

	//Validate if map contains necessary values.
	ProviderAccessKeyId, KeyIDExists := req.Secrets["providerAccessKeyId"]
	ProviderAccessKey, KeyExists := req.Secrets["providerSecretAccessKey"]
	UserData, UserDataExists := req.Secrets["userData"]
	if !KeyIDExists || !KeyExists || !UserDataExists {
		glog.Errorf("Invalid Secret Map")
		return &cmi.DeleteMachineResponse{}, nil
	}

	var Secrets api.Secrets
	Secrets.ProviderAccessKeyId = string(ProviderAccessKeyId)
	Secrets.ProviderSecretAccessKey = string(ProviderAccessKey)
	Secrets.UserData = string(UserData)

	ListMachinesRequest := &cmi.ListMachinesRequest{
		ProviderSpec: req.ProviderSpec,
		Secrets:      req.Secrets,
		MachineID:    req.MachineID,
	}

	Resp := &cmi.DeleteMachineResponse{}

	// Core Logic for Machine-deletion - CP Specific

	//Validate the Spec and Secrets
	ValidationErr := validateAWSProviderSpec(&ProviderSpec, &Secrets)
	if ValidationErr != nil {
		Resp = &cmi.DeleteMachineResponse{
			Error: ValidationErrToString(ValidationErr),
		}
		fmt.Println("Error while validating ProviderSpec", ValidationErr)
		return Resp, fmt.Errorf(Resp.Error)
	}

	result, err := ms.ListMachines(ctx, ListMachinesRequest)
	if err != nil {
		Resp.Error = err.Error()
		return Resp, err
	} else if len(result.MachineList) == 0 {
		// No running instance exists with the given machine-ID
		glog.V(2).Infof("No VM matching the machine-ID found on the provider %q", req.MachineID)
		return Resp, nil
	}

	machineID := decodeMachineID(req.MachineID)

	svc := createSVC(ProviderSpec, Secrets)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(machineID),
		},
		DryRun: aws.Bool(true),
	}
	_, err = svc.TerminateInstances(input)
	awsErr, ok := err.(awserr.Error)
	if ok && awsErr.Code() == "DryRunOperation" {
		input.DryRun = aws.Bool(false)
		output, err := svc.TerminateInstances(input)
		if err != nil {
			glog.Errorf("Could not terminate machine: %s", err.Error())
			return Resp, err
		}

		vmState := output.TerminatingInstances[0]
		//glog.Info(vmState.PreviousState, vmState.CurrentState)

		if *vmState.CurrentState.Name == "shutting-down" ||
			*vmState.CurrentState.Name == "terminated" {
			return Resp, nil
		}

		err = errors.New("Machine already terminated")
	}

	glog.Errorf("Could not terminate machine: %s", err.Error())

	// End of core logic for Machine-deletion - CP Specific
	if err != nil {
		Resp = &cmi.DeleteMachineResponse{
			Error: err.Error(),
		}
	} else {
		_, error := fmt.Printf("Something went wrong while deleting the machine:", req.MachineID)
		Resp = &cmi.DeleteMachineResponse{
			Error: error.Error(),
		}
	}

	return Resp, nil
}

// ListMachines returns a VM matching the machineID
// If machineID is an empty string then it returns all matching instances in terms of
// map[string]string
func (ms *machineServer) ListMachines(ctx context.Context, req *cmi.ListMachinesRequest) (*cmi.ListMachinesResponse, error) {
	fmt.Println("TestLogs: List of Machine - Request-parameter MachineID:", req.MachineID)

	var ProviderSpec api.AWSProviderSpec
	err := json.Unmarshal(req.ProviderSpec, &ProviderSpec)
	if err != nil {
		glog.Errorf("Could not parse ProviderSpec into AWSProviderSpec", err)
	}

	ProviderAccessKeyId, KeyIDExists := req.Secrets["providerAccessKeyId"]
	ProviderAccessKey, KeyExists := req.Secrets["providerSecretAccessKey"]
	UserData, UserDataExists := req.Secrets["userData"]
	if !KeyIDExists || !KeyExists || !UserDataExists {
		glog.Errorf("Invalidate Secret Map")
		return nil, fmt.Errorf("Invalidate Secret Map")
	}

	//TODO: Make validation better to make sure if all the fields under secret are covered.
	var Secrets api.Secrets
	Secrets.ProviderAccessKeyId = string(ProviderAccessKeyId)
	Secrets.ProviderSecretAccessKey = string(ProviderAccessKey)
	Secrets.UserData = string(UserData)

	listOfVMs := make(map[string]string)
	Resp := &cmi.ListMachinesResponse{
		MachineList: listOfVMs,
		Error:       "",
	}

	// Core Logic for Listing the Machines - Provider Specific

	//Validate the Spec and Secrets
	ValidationErr := validateAWSProviderSpec(&ProviderSpec, &Secrets)
	if ValidationErr != nil {
		Resp = &cmi.ListMachinesResponse{
			Error: ValidationErrToString(ValidationErr),
		}
		fmt.Println("Error while validating ProviderSpec", ValidationErr)
		return Resp, fmt.Errorf(Resp.Error)
	}

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
		return Resp, nil
	}

	svc := createSVC(ProviderSpec, Secrets)
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

	// When targeting particular VM
	if req.MachineID != "" {
		machineID := decodeMachineID(req.MachineID)
		instanceFilter := &ec2.Filter{
			Name: aws.String("instance-id"),
			Values: []*string{
				&machineID,
			},
		}
		input.Filters = append(input.Filters, instanceFilter)
	}

	runResult, err := svc.DescribeInstances(&input)
	if err != nil {
		glog.Errorf("AWS driver is returning error while describe instances request is sent: %s", err)
		return Resp, err
	}

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
	Resp = &cmi.ListMachinesResponse{
		MachineList: listOfVMs,
		Error:       "",
	}
	return Resp, nil
}

// Helper function to create SVC
func createSVC(ProviderSpec api.AWSProviderSpec, Secrets api.Secrets) *ec2.EC2 {

	accessKeyID := strings.TrimSpace(Secrets.ProviderAccessKeyId)
	secretAccessKey := strings.TrimSpace(Secrets.ProviderSecretAccessKey)

	if accessKeyID != "" && secretAccessKey != "" {
		return ec2.New(session.New(&aws.Config{
			Region: aws.String(ProviderSpec.Region),
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			}),
		}))
	}

	return ec2.New(session.New(&aws.Config{
		Region: aws.String(ProviderSpec.Region),
	}))
}

func encodeMachineID(region, machineID string) string {
	return fmt.Sprintf("aws:///%s/%s", region, machineID)
}

func decodeMachineID(id string) string {
	splitProviderID := strings.Split(id, "/")
	return splitProviderID[len(splitProviderID)-1]
}
