// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mockclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/errors"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/interfaces"

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
	// InconsistencyInAPIs string makes RunInstances and DescribeInstances APIs out of sync
	InconsistencyInAPIs string = "apis-are-inconsistent"
	// InsufficientCapacity string makes RunInstances return an InsufficientCapacity error code
	InsufficientCapacity = "insufficient-capacity"
)

var (
	// AWSInvalidRegionError denotes an error with an InvalidRegion error code.
	AWSInvalidRegionError = &smithy.GenericAPIError{Code: "region doesn't exist while trying to create session"}
	// AWSImageNotFoundError denotes an error with an ImageNotFound error code.
	AWSImageNotFoundError = &smithy.GenericAPIError{Code: "couldn't find image with given ID"}
	// AWSInternalErrorForRunInstances denotes an error returned by RunInstances call with Internal error code
	AWSInternalErrorForRunInstances = &smithy.GenericAPIError{Code: "couldn't run instance with given ID"}
	// AWSInsufficientCapacityError denotes an error with an InsufficientCapacity error code.
	AWSInsufficientCapacityError = &smithy.GenericAPIError{Code: errors.InsufficientCapacity}
	// AWSInternalErrorForDescribeInstances denotes an error returned by DescribeInstances call with an Internal error code
	AWSInternalErrorForDescribeInstances = &smithy.GenericAPIError{Code: "cloud provider returned error"}
	// AWSInstanceNotFoundError returns denotes an error with InvalidInstanceID.NotFound error code
	AWSInstanceNotFoundError = &smithy.GenericAPIError{Code: string(errors.InstanceIDNotFound)}
)

// MockClientProvider is the mock implementation of ClientProvider interface that makes dummy calls
type MockClientProvider struct {
	FakeInstances []ec2types.Instance
}

// NewConfig returns a new AWS Config
func (ms *MockClientProvider) NewConfig(_ context.Context, _ *corev1.Secret, region string) (*aws.Config, error) {
	if region == FailAtRegion {
		return nil, AWSInvalidRegionError
	}
	return &aws.Config{}, nil
}

// NewEC2Client Returns a new mock for the EC2 Client
func (ms *MockClientProvider) NewEC2Client(_ *aws.Config) interfaces.Ec2Client {
	return &MockEC2Client{
		FakeInstances: &ms.FakeInstances,
	}
}

// MockEC2Client is the mock implementation of an EC2Client
type MockEC2Client struct {
	interfaces.Ec2Client
	FakeInstances *[]ec2types.Instance
}

// DescribeImages implements a mock describe image method
func (ms *MockEC2Client) DescribeImages(_ context.Context, input *ec2.DescribeImagesInput, _ ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {

	if input.ImageIds[0] == FailQueryAtDescribeImages {
		return nil, AWSImageNotFoundError
	}

	rootDeviceName := "test-root-disk"

	return &ec2.DescribeImagesOutput{
		Images: []ec2types.Image{
			{
				RootDeviceName: &rootDeviceName,
			},
		},
	}, nil
}

// RunInstances implements a mock run instance method
// The name of the newly created instances depends on the number of instances in cache starts from 0
func (ms *MockEC2Client) RunInstances(_ context.Context, input *ec2.RunInstancesInput, _ ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error) {

	if *input.ImageId == FailQueryAtRunInstances {
		if *input.KeyName == InsufficientCapacity {
			return nil, AWSInsufficientCapacityError
		}
		return nil, AWSInternalErrorForRunInstances
	}

	instanceID := fmt.Sprintf("i-0123456789-%d", len(*ms.FakeInstances))
	privateDNSName := fmt.Sprintf("ip-%d", len(*ms.FakeInstances))

	placement := input.Placement
	if placement != nil {
		instanceID = fmt.Sprintf(
			"i-0123456789-%d/placement={affinity:%s,availabilityZone:%s,tenancy:%s}",
			len(*ms.FakeInstances),
			*placement.Affinity,
			*placement.AvailabilityZone,
			placement.Tenancy,
		)
	}

	if strings.Contains(*input.ImageId, SetInstanceID) {
		if *input.KeyName == InconsistencyInAPIs {
			instanceID = InstanceDoesntExistError
		} else {
			instanceID = *input.KeyName
		}
	}

	newInstance := ec2types.Instance{
		InstanceId:     &instanceID,
		PrivateDnsName: &privateDNSName,
		State: &ec2types.InstanceState{
			Code: aws.Int32(16),
			Name: ec2types.InstanceStateName("running"),
		},
		Tags: deepCopyTagList(input.TagSpecifications[0].Tags),
	}
	*ms.FakeInstances = append(*ms.FakeInstances, newInstance)

	return &ec2.RunInstancesOutput{
		Instances: []ec2types.Instance{
			newInstance,
		},
	}, nil
}

// DescribeInstances implements a mock run instance method
func (ms *MockEC2Client) DescribeInstances(_ context.Context, input *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	found := false
	instanceList := make([]ec2types.Instance, 0)

	for _, filter := range input.Filters {
		if filter.Values[0] == "kubernetes.io/cluster/"+ReturnErrorAtDescribeInstances {
			return nil, AWSInternalErrorForDescribeInstances
		}
	}

	if len(input.InstanceIds) > 0 {
		if input.InstanceIds[0] == ReturnEmptyListAtDescribeInstances {
			return &ec2.DescribeInstancesOutput{
				Reservations: []ec2types.Reservation{
					{
						Instances: instanceList,
					},
				},
			}, nil
		} else if input.InstanceIds[0] == InstanceDoesntExistError {
			return nil, &smithy.GenericAPIError{Code: string(ec2types.UnsuccessfulInstanceCreditSpecificationErrorCodeInstanceNotFound)}
		}

		// Target Specific instances
		for _, instanceID := range input.InstanceIds {
			for _, instance := range *ms.FakeInstances {
				if *instance.InstanceId == instanceID {
					found = true
					instanceToCopy := instance
					instanceList = append(instanceList, instanceToCopy)
				}
			}
		}
		if !found {
			return nil, AWSInstanceNotFoundError
		}
	} else {

		// Target all instances
		for _, instance := range *ms.FakeInstances {
			instanceToCopy := instance
			instanceList = append(instanceList, instanceToCopy)
		}
	}

	return &ec2.DescribeInstancesOutput{
		Reservations: []ec2types.Reservation{
			{
				Instances: instanceList,
			},
		},
	}, nil
}

// TerminateInstances implements a mock terminate instance method
func (ms *MockEC2Client) TerminateInstances(_ context.Context, input *ec2.TerminateInstancesInput, _ ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {

	if input.InstanceIds[0] == FailQueryAtTerminateInstances {
		return nil, &smithy.GenericAPIError{Code: string(ec2types.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceId)}
	}

	var desiredInstance ec2types.Instance
	found := false
	newInstanceList := make([]ec2types.Instance, 0)

	for _, instanceID := range input.InstanceIds {
		for _, instance := range *ms.FakeInstances {
			if *instance.InstanceId == instanceID {
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
		return nil, AWSInstanceNotFoundError
	}

	return &ec2.TerminateInstancesOutput{
		TerminatingInstances: []ec2types.InstanceStateChange{
			{
				PreviousState: desiredInstance.State,
				InstanceId:    aws.String(input.InstanceIds[0]),
				CurrentState: &ec2types.InstanceState{
					Code: aws.Int32(32),
					Name: ec2types.InstanceStateName("shutting-down"),
				},
			},
		},
	}, nil
}

// StopInstances implements a mock stop instance method
func (ms *MockEC2Client) StopInstances(_ context.Context, input *ec2.StopInstancesInput, _ ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error) {

	if input.InstanceIds[0] == InstanceStopError {
		return nil, fmt.Errorf("Stopping of instance errored out")
	} else if *input.DryRun {
		// If it is a dry run
		return nil, fmt.Errorf("This is a dry run call")
	}

	var desiredInstance ec2types.Instance
	found := false

	for _, instanceID := range input.InstanceIds {
		for _, instance := range *ms.FakeInstances {
			if *instance.InstanceId == instanceID {
				// Do not append InstanceID, there by removing it
				found = true
				desiredInstance = instance
			}
		}
	}

	if !found {
		return nil, AWSInstanceNotFoundError
	}

	return &ec2.StopInstancesOutput{
		StoppingInstances: []ec2types.InstanceStateChange{
			{
				PreviousState: desiredInstance.State,
				InstanceId:    aws.String(input.InstanceIds[0]),
				CurrentState: &ec2types.InstanceState{
					Code: aws.Int32(64),
					Name: ec2types.InstanceStateName("stopping"),
				},
			},
		},
	}, nil
}

// deepCopyTagList copies inTags list to outTags
func deepCopyTagList(inTags []ec2types.Tag) []ec2types.Tag {
	var outTags []ec2types.Tag

	for _, tagPtr := range inTags {
		tag := ec2types.Tag{}
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
