// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	validation "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis/validation"
	awserror "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/errors"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

const (
	// AWSMachineClassKind for AWSMachineClass
	AWSMachineClassKind = "AWSMachineClass"

	// MachineClassKind for MachineClass
	MachineClassKind = "MachineClass"
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
		err = fmt.Errorf("Error while validating ProviderSpec %v", validationErr.ToAggregate().Error())
		klog.V(2).Infof("Validation of AWSMachineClass failed %s", err)

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return providerSpec, nil
}

// disableSrcAndDestCheck disbales the SrcAndDestCheck for NAT instances
func disableSrcAndDestCheck(svc ec2iface.EC2API, instanceID *string) error {

	srcAndDstCheckEnabled := &ec2.ModifyInstanceAttributeInput{
		InstanceId: instanceID,
		SourceDestCheck: &ec2.AttributeBooleanValue{
			Value: pointer.BoolPtr(false),
		},
	}

	_, err := svc.ModifyInstanceAttribute(srcAndDstCheckEnabled)
	if err != nil {
		return err
	}
	klog.V(2).Infof("Successfully disabled Source/Destination check on instance %s.", *instanceID)
	return nil
}

func getMachineInstancesByTagsAndStatus(svc ec2iface.EC2API, machineName string, providerSpecTags map[string]string) ([]*ec2.Instance, error) {
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
	input := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(machineName),
				},
			},
			{
				Name: aws.String("tag-key"),
				Values: []*string{
					&clusterName,
				},
			},
			{
				Name: aws.String("tag-key"),
				Values: []*string{
					&nodeRole,
				},
			},
			{
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
		klog.Errorf("AWS plugin is returning error while describe instances request is sent: %s", err)
		return nil, status.Error(codes.Internal, err.Error())
	} else {
		var instances []*ec2.Instance
		for _, reservation := range runResult.Reservations {
			instances = append(instances, reservation.Instances...)
		}
		return instances, nil
	}
}

// getMatchingInstancesForMachine extracts AWS Instance object for a given machine
func (d *Driver) getMatchingInstancesForMachine(machine *v1alpha1.Machine, providerSpec *api.AWSProviderSpec, secret *corev1.Secret) (instances []*ec2.Instance, err error) {
	svc, err := d.createSVC(secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	instances, err = getMachineInstancesByTagsAndStatus(svc, machine.Name, providerSpec.Tags)
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
		runResult, err := getInstanceByID(svc, instanceID)
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

func getInstanceByID(svc ec2iface.EC2API, instanceID string) (*ec2.DescribeInstancesOutput, error) {
	input := ec2.DescribeInstancesInput{
		InstanceIds: []*string{&instanceID},
	}
	instances, err := svc.DescribeInstances(&input)
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

func confirmInstanceByID(svc ec2iface.EC2API, instanceID string) (bool, error) {
	_, err := getInstanceByID(svc, instanceID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *Driver) generateBlockDevices(blockDevices []api.AWSBlockDeviceMappingSpec, rootDeviceName *string) ([]*ec2.BlockDeviceMapping, error) {
	// If not blockDevices are passed, return an error.
	if len(blockDevices) == 0 {
		return nil, fmt.Errorf("No block devices passed")
	}

	var blkDeviceMappings []*ec2.BlockDeviceMapping
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

		blkDeviceMapping := ec2.BlockDeviceMapping{
			DeviceName: aws.String(deviceName),
			Ebs: &ec2.EbsBlockDevice{
				Encrypted:  aws.Bool(encrypted),
				VolumeSize: aws.Int64(volumeSize),
				VolumeType: aws.String(volumeType),
			},
		}

		if deleteOnTermination != nil {
			blkDeviceMapping.Ebs.DeleteOnTermination = deleteOnTermination
		} else {
			// If deletionOnTermination is not set, default it to true
			blkDeviceMapping.Ebs.DeleteOnTermination = aws.Bool(true)
		}

		if disk.Ebs.Iops > 0 {
			blkDeviceMapping.Ebs.Iops = aws.Int64(disk.Ebs.Iops)
		}

		// adding throughput
		if disk.Ebs.Throughput != nil {
			blkDeviceMapping.Ebs.Throughput = disk.Ebs.Throughput
		}

		if snapshotID != nil {
			blkDeviceMapping.Ebs.SnapshotId = snapshotID
		}
		blkDeviceMappings = append(blkDeviceMappings, &blkDeviceMapping)
	}

	return blkDeviceMappings, nil
}

func (d *Driver) generateTags(tags map[string]string, resourceType string, machineName string) (*ec2.TagSpecification, error) {

	// Add tags to the created machine
	tagList := []*ec2.Tag{}
	for idx, element := range tags {
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
		Value: aws.String(machineName),
	}
	tagList = append(tagList, &nameTag)

	tagInstance := &ec2.TagSpecification{
		ResourceType: aws.String(resourceType),
		Tags:         tagList,
	}
	return tagInstance, nil
}

func terminateInstance(req *driver.DeleteMachineRequest, svc ec2iface.EC2API, machineID string) error {
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(machineID),
		},
		DryRun: aws.Bool(false),
	}

	_, err := svc.TerminateInstances(input)
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
