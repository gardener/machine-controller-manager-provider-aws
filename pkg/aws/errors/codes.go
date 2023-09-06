package errors

import "github.com/aws/aws-sdk-go/service/ec2"

// See https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html# for more information on the various error codes
// returned by the amazon EC2 API
const (

	// InstanceLimitExceeded is returned when you've reached the limit on the number of instances you can run concurrently.
	// This error can occur if you are launching an instance or if you are creating a Capacity Reservation.
	// Capacity Reservations count towards your On-Demand Instance limits.
	// If your request fails due to limit constraints, increase your On-Demand Instance limit for the required instance type and try again.
	InstanceLimitExceeded = "InstanceLimitExceeded"

	// VcpuLimitExceeded is returned when you've reached the limit on the number of vCPUs (virtual processing units)
	// assigned to the running instances in your account. You are limited to running one or more On-Demand instances in an AWS account,
	// and Amazon EC2 measures usage towards each limit based on the total number of vCPUs that are assigned to the running
	// On-Demand instances in your AWS account. If your request fails due to limit constraints, increase your On-Demand instance limits and try again.
	VcpuLimitExceeded = "VcpuLimitExceeded"

	// MaxIOPSLimitExceeded is returned when you've reached the limit on your IOPS usage for that AWS Region.
	// For more information, see Amazon EBS quotas (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-resource-quotas.html)
	MaxIOPSLimitExceeded = "MaxIOPSLimitExceeded"

	// VolumeLimitExceeded is returned when you've reached the limit on your Amazon EBS volume storage.
	// For more information, see Amazon EBS quotas (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-resource-quotas.html).
	VolumeLimitExceeded = "VolumeLimitExceeded"

	// InsufficientAddressCapacity is retured when AWS does not have enough available addresses to satisfy your minimum request.
	// Reduce the number of addresses you are requesting or wait for additional capacity to become available.
	InsufficientAddressCapacity = "InsufficientAddressCapacity"

	// InsufficientCapacity is returned when there is not enough capacity to fulfill your import instance request.
	// You can wait for additional capacity to become available.
	InsufficientCapacity = "InsufficientCapacity"

	// InsufficientInstanceCapacity is returned when there is not enough capacity to fulfill your request.
	// This error can occur if you launch a new instance, restart a stopped instance, create a new Capacity Reservation, or modify an existing Capacity Reservation.
	// Reduce the number of instances in your request, or wait for additional capacity to become available.
	// You can also try launching an instance by selecting different instance types (which you can resize at a later stage).
	// The returned message might also give specific guidance about how to solve the problem.
	InsufficientInstanceCapacity = "InsufficientInstanceCapacity"

	// InsufficientVolumeCapacity is returned when there is not enough capacity to fulfill your EBS volume provision request.
	// You can try to provision a different volume type, EBS volume in a different availability zone, or you can wait for additional capacity to become available.
	InsufficientVolumeCapacity = "InsufficientVolumeCapacity"

	// RouteLimitExceeded is returned when you've reached the limit on the number of routes that you can add to a route table.
	RouteLimitExceeded = "RouteLimitExceeded"

	// InstanceIdNotFound is returned when the specified instance does not exist. This error might occur because the ID of a recently created instance has not propagated through the system.
	// For more information, see Eventual consistency(https://docs.aws.amazon.com/AWSEC2/latest/APIReference/query-api-troubleshooting.html#eventual-consistency).
	InstanceIdNotFound = ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdNotFound
)
