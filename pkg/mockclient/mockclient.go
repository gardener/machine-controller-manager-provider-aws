package mockclient

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
)

// MockDriverSPIImpl is the mock implementation of DriverSPI interface that makes dummy calls
type MockDriverSPIImpl struct {
	FakeInstances []ec2.Instance
}

// NewSession starts a new AWS session
func (ms *MockDriverSPIImpl) NewSession(Secrets api.Secrets, region string) (*awssession.Session, error) {
	if region == "fail-at-region" {
		return nil, fmt.Errorf("Region doesn't exist while trying to create session")
	}
	return &awssession.Session{}, nil
}

// NewEC2API Returns a EC2API object
func (ms *MockDriverSPIImpl) NewEC2API(session *session.Session) ec2iface.EC2API {
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

	if *input.ImageIds[0] == "fail-query-at-DescribeImages" {
		return nil, fmt.Errorf("Couldn't find image with given ID")
	}

	rootDeviceName := "test-root-disk"

	return &ec2.DescribeImagesOutput{
		Images: []*ec2.Image{
			&ec2.Image{
				RootDeviceName: &rootDeviceName,
			},
		},
	}, nil
}

// RunInstances implements a mock run instance method
// The name of the newly created instances depends on the number of instances in cache starts from 0
func (ms *MockEC2Client) RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error) {

	if *input.ImageId == "fail-query-at-RunInstances" {
		return nil, fmt.Errorf("Couldn't run instance with given ID")
	}

	instanceID := fmt.Sprintf("i-0123456789-%d", len(*ms.FakeInstances))
	privateDNSName := fmt.Sprintf("ip-%d", len(*ms.FakeInstances))

	newInstance := ec2.Instance{
		InstanceId:     &instanceID,
		PrivateDnsName: &privateDNSName,
		State: &ec2.InstanceState{
			Code: aws.Int64(16),
			Name: aws.String("running"),
		},
		Tags: input.TagSpecifications[0].Tags,
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

	if len(input.InstanceIds) > 0 {
		if *input.InstanceIds[0] == "return-empty-list-at-DescribeInstances" {
			return &ec2.DescribeInstancesOutput{
				Reservations: []*ec2.Reservation{
					&ec2.Reservation{
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

		if *input.Filters[0].Values[0] == "kubernetes.io/cluster/return-error-at-DescribeInstances" {
			return nil, fmt.Errorf("Cloud provider returned error")
		}

		// Target all instances
		for _, instance := range *ms.FakeInstances {
			instanceToCopy := instance
			instanceList = append(instanceList, &instanceToCopy)
		}
	}

	return &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			&ec2.Reservation{
				Instances: instanceList,
			},
		},
	}, nil
}

// TerminateInstances implements a mock terminate instance method
func (ms *MockEC2Client) TerminateInstances(input *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {

	if *input.InstanceIds[0] == "i-terminate-error" {
		return nil, fmt.Errorf("Termination of instance errored out")
	} else if *input.InstanceIds[0] == "i-instance-doesnt-exist" {
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
			&ec2.InstanceStateChange{
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

	if *input.InstanceIds[0] == "i-stop-error" {
		return nil, fmt.Errorf("Stopping of instance errored out")
	} else if *input.InstanceIds[0] == "i-instance-doesnt-exist" {
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
			&ec2.InstanceStateChange{
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
