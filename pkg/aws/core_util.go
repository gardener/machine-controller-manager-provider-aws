// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/interfaces"

	"github.com/aws/aws-sdk-go-v2/aws"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	validation "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis/validation"
	awserror "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/errors"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/instrument"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

// labels used for recording prometheus metrics
const (
	instanceDisableSourceDestCheckServiceLabel = "instance_disable_source_dest_check"
	instanceGetByTagsAndStatusServiceLabel     = "instance_get_by_tag_and_status"
	instanceGetByMachineServiceLabel           = "instance_get_by_machine"
	instanceGetByIDServiceLabel                = "instance_get_by_id"
	instanceTerminateServiceLabel              = "instance_terminate"
)

// decodeProviderSpecAndSecret converts request parameters to api.ProviderSpec & api.Secrets
func decodeProviderSpecAndSecret(machineClass *v1alpha1.MachineClass, secret *corev1.Secret) (*api.AWSProviderSpec, error) {
	var (
		providerSpec *api.AWSProviderSpec
	)

	// Extract providerSpec
	if machineClass == nil {
		return nil, status.Error(codes.InvalidArgument, "MachineClass ProviderSpec is nil")
	}

	err := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Validate the Spec and Secrets
	validationErr := validation.ValidateAWSProviderSpec(providerSpec, secret, field.NewPath("providerSpec"))
	if validationErr.ToAggregate() != nil && len(validationErr.ToAggregate().Errors()) > 0 {
		err = fmt.Errorf("error while validating ProviderSpec %v", validationErr.ToAggregate().Error())
		klog.V(2).Infof("Validation of AWSMachineClass failed %s", err)

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return providerSpec, nil
}

// disableSrcAndDestCheck disbales the SrcAndDestCheck for NAT instances
func disableSrcAndDestCheck(ctx context.Context, svc interfaces.Ec2Client, instanceID *string) (err error) {
	defer instrument.AwsAPIMetricRecorderFn(instanceDisableSourceDestCheckServiceLabel, &err)()
	srcAndDstCheckEnabled := &ec2.ModifyInstanceAttributeInput{
		InstanceId: instanceID,
		SourceDestCheck: &ec2types.AttributeBooleanValue{
			Value: ptr.To(false),
		},
	}

	_, err = svc.ModifyInstanceAttribute(ctx, srcAndDstCheckEnabled)
	if err != nil {
		return err
	}
	klog.V(2).Infof("Successfully disabled Source/Destination check on instance %s.", *instanceID)
	return nil
}

func getMachineInstancesByTagsAndStatus(ctx context.Context, svc interfaces.Ec2Client, machineName string, providerSpecTags map[string]string) (instances []ec2types.Instance, err error) {
	defer instrument.AwsAPIMetricRecorderFn(instanceGetByTagsAndStatusServiceLabel, &err)()
	var (
		clusterName string
		nodeRole    string
	)
	for key := range providerSpecTags {
		if strings.Contains(key, "kubernetes.io/cluster/") {
			clusterName = key
		} else if strings.Contains(key, "kubernetes.io/role/") {
			nodeRole = key
		}
	}
	input := &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{machineName},
			},
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

	var nextToken *string
	pageCount := 0
	for {
		input.NextToken = nextToken

		runResult, err := svc.DescribeInstances(ctx, input)
		if err != nil {
			klog.Errorf("AWS plugin encountered an error while sending DescribeInstances request: %s (NextToken: %s)", err, ptr.Deref(nextToken, "<nil>"))
			return nil, status.Error(codes.Internal, err.Error())
		}
		pageCount++

		for _, reservation := range runResult.Reservations {
			instances = append(instances, reservation.Instances...)
		}

		// Exit if there no are more results
		if runResult.NextToken == nil || *runResult.NextToken == "" {
			break
		}
		klog.V(3).Infof("Fetching next page (page %d) of ListMachines, with NextToken: %s", pageCount+1, *runResult.NextToken)
		nextToken = runResult.NextToken
	}
	klog.V(3).Infof("Found %d instances for machine %s using tags/status in %d pages", len(instances), machineName, pageCount)
	return instances, nil
}

// getMatchingInstancesForMachine extracts AWS Instance object for a given machine
func (d *Driver) getMatchingInstancesForMachine(ctx context.Context, machine *v1alpha1.Machine, svc interfaces.Ec2Client, providerSpecTags map[string]string) (instances []ec2types.Instance, err error) {
	defer instrument.AwsAPIMetricRecorderFn(instanceGetByMachineServiceLabel, &err)()
	instances, err = getMachineInstancesByTagsAndStatus(ctx, svc, machine.Name, providerSpecTags)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		//if getMachineInstancesByTagsAndStatus does not return any instances, try fetching matching instances using ProviderID
		klog.V(3).Infof("No VM instances found for machine %s using tags/status. Now looking for VM using providerID", machine.Name)
		if machine.Spec.ProviderID == "" {
			return nil, status.Error(codes.NotFound, "No ProviderID found on the machine")
		}
		_, instanceID, err := decodeRegionAndInstanceID(machine.Spec.ProviderID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		runResult, err := getInstanceByID(ctx, svc, instanceID)
		if err != nil {
			return nil, err
		}
		for _, reservation := range runResult.Reservations {
			instances = append(instances, reservation.Instances...)
		}
		if len(instances) == 0 {
			errMessage := "AWS plugin is returning no VM instances backing this machine object"
			return nil, status.Error(codes.NotFound, errMessage)
		}
	}
	return instances, nil
}

func getInstanceByID(ctx context.Context, svc interfaces.Ec2Client, instanceID string) (instances *ec2.DescribeInstancesOutput, err error) {
	defer instrument.AwsAPIMetricRecorderFn(instanceGetByIDServiceLabel, &err)()
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}
	instances, err = svc.DescribeInstances(ctx, input)
	if err != nil {
		if awserror.IsInstanceIDNotFound(err) {
			errMessage := "AWS plugin is returning no VM instances backing this machine object"
			return nil, status.Error(codes.NotFound, errMessage)
		}
		klog.Errorf("AWS plugin is returning error while describe instances request is sent: %s", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return instances, err
}

func (d *Driver) generateBlockDevices(blockDevices []api.AWSBlockDeviceMappingSpec, rootDeviceName *string) ([]ec2types.BlockDeviceMapping, error) {
	// If not blockDevices are passed, return an error.
	if len(blockDevices) == 0 {
		return nil, fmt.Errorf("no block devices passed")
	}

	var blkDeviceMappings []ec2types.BlockDeviceMapping
	// if blockDevices is empty, AWS will automatically create a root partition
	for _, disk := range blockDevices {

		deviceName := disk.DeviceName
		if disk.DeviceName == "/root" || len(blockDevices) == 1 {
			deviceName = *rootDeviceName
		}
		deleteOnTermination := disk.Ebs.DeleteOnTermination
		volumeSize := disk.Ebs.VolumeSize
		volumeType := disk.Ebs.VolumeType
		encrypted := disk.Ebs.Encrypted
		snapshotID := disk.Ebs.SnapshotID

		blkDeviceMapping := ec2types.BlockDeviceMapping{
			DeviceName: aws.String(deviceName),
			Ebs: &ec2types.EbsBlockDevice{
				Encrypted:  aws.Bool(encrypted),
				VolumeSize: aws.Int32(volumeSize),
				VolumeType: ec2types.VolumeType(volumeType),
			},
		}

		if deleteOnTermination != nil {
			blkDeviceMapping.Ebs.DeleteOnTermination = deleteOnTermination
		} else {
			// If deletionOnTermination is not set, default it to true
			blkDeviceMapping.Ebs.DeleteOnTermination = aws.Bool(true)
		}

		if disk.Ebs.Iops > 0 {
			blkDeviceMapping.Ebs.Iops = aws.Int32(disk.Ebs.Iops)
		}

		// adding throughput
		if disk.Ebs.Throughput != nil {
			blkDeviceMapping.Ebs.Throughput = disk.Ebs.Throughput
		}

		if snapshotID != nil {
			blkDeviceMapping.Ebs.SnapshotId = snapshotID
		}
		blkDeviceMappings = append(blkDeviceMappings, blkDeviceMapping)
	}

	return blkDeviceMappings, nil
}

func (d *Driver) generateTags(tags map[string]string, resourceType string, machineName string) (ec2types.TagSpecification, error) {

	// Add tags to the created machine
	var tagList []ec2types.Tag
	for idx, element := range tags {
		if idx == "Name" {
			// Name tag cannot be set, as its used to identify backing machine object
			continue
		}
		newTag := ec2types.Tag{
			Key:   aws.String(idx),
			Value: aws.String(element),
		}
		tagList = append(tagList, newTag)
	}
	nameTag := ec2types.Tag{
		Key:   aws.String("Name"),
		Value: aws.String(machineName),
	}
	tagList = append(tagList, nameTag)

	tagInstance := ec2types.TagSpecification{
		ResourceType: ec2types.ResourceType(resourceType),
		Tags:         tagList,
	}
	return tagInstance, nil
}

func terminateInstance(ctx context.Context, req *driver.DeleteMachineRequest, svc interfaces.Ec2Client, machineID string) (err error) {
	defer instrument.AwsAPIMetricRecorderFn(instanceTerminateServiceLabel, &err)()
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{machineID},
		DryRun:      aws.Bool(false),
	}

	_, err = svc.TerminateInstances(ctx, input)
	if err != nil {
		// if error code is NotFound, then assume VM is terminated.
		// In case of eventual consistency, the VM might be present and still we get a NotFound error.
		// Such cases will be handled by the orphan collection logic.
		errcode := awserror.GetMCMErrorCodeForTerminateInstances(err)
		if errcode == codes.NotFound {
			klog.V(2).Infof("no backing VM for %s machine found while trying to terminate instance. Orphan collection will remove the VM if it is due to eventual consistency", req.Machine.Name)
			return nil
		}
		klog.Errorf("VM %q for Machine %q couldn't be terminated: %s",
			req.Machine.Spec.ProviderID,
			req.Machine.Name,
			err.Error(),
		)
		return status.Error(errcode, err.Error())
	}

	return nil
}
