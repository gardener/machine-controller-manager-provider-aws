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

package mockclient

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	corev1 "k8s.io/api/core/v1"
)

const (
	// FailAtRegion string to fail call due to invalid region
	FailAtRegion string = "fail-at-region"
	// FailQueryAtDescribeImages string to fail call at Describeimages call
	FailQueryAtDescribeImages string = "fail-query-at-DescribeImages"
	// FailQueryAtRunInstances string to fail call at RunInstances call
	FailQueryAtRunInstances string = "aws:///eu-west-1/i-fail-query-at-RunInstances"
	// FailQueryAtTerminateInstances string to fail call at TerminateInstances call
	FailQueryAtTerminateInstances string = "fail-query-at-TerminateInstances"
	// InstanceTerminateError string returns instance terminated error
	InstanceTerminateError string = "i-instance-terminate-error"
	// InstanceDoesntExistError string returns instance doesn't exist error
	InstanceDoesntExistError string = "i-instance-doesnt-exist"
	// InstanceStopError string returns error mentioning instance has been stopped
	InstanceStopError string = "i-instance-stop-error"
	// ReturnEmptyListAtDescribeInstances string returns empty list at DescribeInstances call
	ReturnEmptyListAtDescribeInstances string = "return-empty-list-at-DescribeInstances"
	// ReturnErrorAtDescribeInstances string returns error at DescribeInstances call
	ReturnErrorAtDescribeInstances string = "return-error-at-DescribeInstances"
	// SetInstanceID string sets the instance ID provided at keyname
	SetInstanceID string = "set-instance-id"
)

// MockPluginSPIImpl is the mock implementation of PluginSPI interface that makes dummy calls
type MockPluginSPIImpl struct {
	FakeInstances []ec2.Instance
}

// NewSession starts a new AWS session
func (ms *MockPluginSPIImpl) NewSession(secret *corev1.Secret, region string) (*awssession.Session, error) {
	if region == FailAtRegion {
		return nil, fmt.Errorf("Region doesn't exist while trying to create session")
	}
	return &awssession.Session{}, nil
}

// NewEC2API Returns a EC2API object
func (ms *MockPluginSPIImpl) NewEC2API(session *session.Session) ec2iface.EC2API {
	return &MockEC2Client{
		FakeInstances: &ms.FakeInstances,
	}
}

// MockEC2Client is the mock implementation of an EC2Client
type MockEC2Client struct {
	ec2iface.EC2API
	FakeInstances *[]ec2.Instance
}

// DescribeImages implements a mock describe image method
func (ms *MockEC2Client) DescribeImages(input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {

	if *input.ImageIds[0] == FailQueryAtDescribeImages {
		return nil, fmt.Errorf("Couldn't find image with given ID")
	}

	rootDeviceName := "test-root-disk"

	return &ec2.DescribeImagesOutput{
		Images: []*ec2.Image{
			{
				RootDeviceName: &rootDeviceName,
			},
		},
	}, nil
}

// RunInstances implements a mock run instance method
// The name of the newly created instances depends on the number of instances in cache starts from 0
func (ms *MockEC2Client) RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error) {

	if *input.ImageId == FailQueryAtRunInstances {
		return nil, fmt.Errorf("Couldn't run instance with given ID")
	}

	instanceID := fmt.Sprintf("i-0123456789-%d", len(*ms.FakeInstances))
	privateDNSName := fmt.Sprintf("ip-%d", len(*ms.FakeInstances))

	if strings.Contains(*input.ImageId, SetInstanceID) {
		instanceID = *input.KeyName
	}

	newInstance := ec2.Instance{
		InstanceId:     &instanceID,
		PrivateDnsName: &privateDNSName,
		State: &ec2.InstanceState{
			Code: aws.Int64(16),
			Name: aws.String("running"),
		},
		Tags: deepCopyTagList(input.TagSpecifications[0].Tags),
	}
	*ms.FakeInstances = append(*ms.FakeInstances, newInstance)

	return &ec2.Reservation{
		Instances: []*ec2.Instance{
			&newInstance,
		},
	}, nil
}

// DescribeInstances implements a mock run instance method
func (ms *MockEC2Client) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	found := false
	instanceList := make([]*ec2.Instance, 0)

	for _, filter := range input.Filters {
		if *filter.Values[0] == "kubernetes.io/cluster/"+ReturnErrorAtDescribeInstances {
			return nil, fmt.Errorf("Cloud provider returned error")
		}
	}

	if len(input.InstanceIds) > 0 {
		if *input.InstanceIds[0] == ReturnEmptyListAtDescribeInstances {
			return &ec2.DescribeInstancesOutput{
				Reservations: []*ec2.Reservation{
					{
						Instances: instanceList,
					},
				},
			}, nil
		}

		// Target Specific instances
		for _, instanceID := range input.InstanceIds {
			for _, instance := range *ms.FakeInstances {
				if *instance.InstanceId == *instanceID {
					found = true
					instanceToCopy := instance
					instanceList = append(instanceList, &instanceToCopy)
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("Couldn't find any instance matching requirement")
		}
	} else {

		// Target all instances
		for _, instance := range *ms.FakeInstances {
			instanceToCopy := instance
			instanceList = append(instanceList, &instanceToCopy)
		}
	}

	return &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{
				Instances: instanceList,
			},
		},
	}, nil
}

// TerminateInstances implements a mock terminate instance method
func (ms *MockEC2Client) TerminateInstances(input *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {

	if *input.InstanceIds[0] == FailQueryAtTerminateInstances {
		return nil, awserr.New(
			ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdMalformed, "",
			fmt.Errorf("Termination of instance errorred out"),
		)
	} else if *input.InstanceIds[0] == InstanceDoesntExistError {
		// If instance with instance ID doesn't exist
		return nil, awserr.New(
			ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdNotFound, "",
			fmt.Errorf("Instance with instance-ID doesn't exist"),
		)
	}

	var desiredInstance ec2.Instance
	found := false
	newInstanceList := make([]ec2.Instance, 0)

	for _, instanceID := range input.InstanceIds {
		for _, instance := range *ms.FakeInstances {
			if *instance.InstanceId == *instanceID {
				// Do not append InstanceID, there by removing it
				found = true
				desiredInstance = instance
			} else {
				newInstanceList = append(newInstanceList, instance)
			}
		}
	}
	ms.FakeInstances = &newInstanceList

	if !found {
		return nil, fmt.Errorf("Couldn't find instance with given instance-ID %s", *input.InstanceIds[0])
	}

	return &ec2.TerminateInstancesOutput{
		TerminatingInstances: []*ec2.InstanceStateChange{
			{
				PreviousState: desiredInstance.State,
				InstanceId:    input.InstanceIds[0],
				CurrentState: &ec2.InstanceState{
					Code: aws.Int64(32),
					Name: aws.String("shutting-down"),
				},
			},
		},
	}, nil
}

// StopInstances implements a mock stop instance method
func (ms *MockEC2Client) StopInstances(input *ec2.StopInstancesInput) (*ec2.StopInstancesOutput, error) {

	if *input.InstanceIds[0] == InstanceStopError {
		return nil, fmt.Errorf("Stopping of instance errored out")
	} else if *input.InstanceIds[0] == InstanceDoesntExistError {
		// If instance with instance ID doesn't exist
		return nil, awserr.New(
			ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdMalformed,
			"Instance with instance-ID doesn't exist",
			fmt.Errorf("Instance with instance-ID doesn't exist"),
		)
	} else if *input.DryRun {
		// If it is a dry run
		return nil, awserr.New(
			"DryRunOperation",
			"This is a dryRun call",
			fmt.Errorf("This is a dry run call"),
		)
	}

	var desiredInstance ec2.Instance
	found := false

	for _, instanceID := range input.InstanceIds {
		for _, instance := range *ms.FakeInstances {
			if *instance.InstanceId == *instanceID {
				// Do not append InstanceID, there by removing it
				found = true
				desiredInstance = instance
			} else {
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("Couldn't find any instance matching requirement")
	}

	return &ec2.StopInstancesOutput{
		StoppingInstances: []*ec2.InstanceStateChange{
			{
				PreviousState: desiredInstance.State,
				InstanceId:    input.InstanceIds[0],
				CurrentState: &ec2.InstanceState{
					Code: aws.Int64(64),
					Name: aws.String("stopping"),
				},
			},
		},
	}, nil
}

// deepCopyTagList copies inTags list to outTags
func deepCopyTagList(inTags []*ec2.Tag) []*ec2.Tag {
	var outTags []*ec2.Tag

	for _, tagPtr := range inTags {
		tag := &ec2.Tag{}
		if tagPtr.Key != nil {
			key := *tagPtr.Key
			tag.Key = &key
		}
		if tagPtr.Value != nil {
			value := *tagPtr.Value
			tag.Value = &value
		}
		outTags = append(outTags, tag)
	}

	return outTags
}
