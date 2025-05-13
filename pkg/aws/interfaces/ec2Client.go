package interfaces

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// Ec2Client is the interface for clients providing the EC2 service
type Ec2Client interface {
	ModifyInstanceAttribute(context.Context, *ec2.ModifyInstanceAttributeInput, ...func(*ec2.Options)) (*ec2.ModifyInstanceAttributeOutput, error)
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	TerminateInstances(context.Context, *ec2.TerminateInstancesInput, ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	DescribeImages(context.Context, *ec2.DescribeImagesInput, ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
	RunInstances(context.Context, *ec2.RunInstancesInput, ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	AssignIpv6Addresses(context.Context, *ec2.AssignIpv6AddressesInput, ...func(*ec2.Options)) (*ec2.AssignIpv6AddressesOutput, error)
}
