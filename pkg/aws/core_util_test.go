// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/interfaces"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/mockclient"
)

var (
	testMachine = "test-machine"
)

var _ = Describe("CoreUtils", func() {

	Context("#generateTags", func() {

		It("should convert multiples tags successfully", func() {
			awsDriver := &Driver{}
			tags := map[string]string{
				"tag-1": "value-tag-1",
				"tag-2": "value-tag-2",
				"tag-3": "value-tag-3",
			}

			tagsGenerated, err := awsDriver.generateTags(tags, resourceTypeInstance, testMachine)
			expectedTags := ec2types.TagSpecification{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("tag-1"),
						Value: aws.String("value-tag-1"),
					},
					{
						Key:   aws.String("tag-2"),
						Value: aws.String("value-tag-2"),
					},
					{
						Key:   aws.String("tag-3"),
						Value: aws.String("value-tag-3"),
					},
					{
						Key:   aws.String("Name"),
						Value: aws.String(testMachine),
					},
				},
			}

			Expect(tagsGenerated.ResourceType).To(Equal(expectedTags.ResourceType))
			Expect(tagsGenerated.Tags).To(ConsistOf(expectedTags.Tags))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should convert zero tags successfully", func() {
			awsDriver := &Driver{}
			tags := map[string]string{}

			tagsGenerated, err := awsDriver.generateTags(tags, resourceTypeInstance, testMachine)
			expectedTags := ec2types.TagSpecification{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(testMachine),
					},
				},
			}

			Expect(tagsGenerated).To(Equal(expectedTags))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("#generateBlockDevices", func() {

		It("should convert multiples blockDevices successfully", func() {
			awsDriver := &Driver{}
			disks := []api.AWSBlockDeviceMappingSpec{
				{
					DeviceName: "/root",
					Ebs: api.AWSEbsBlockDeviceSpec{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           false,
						VolumeSize:          32,
						VolumeType:          "gp2",
					},
				},
				{
					DeviceName: "/dev/xvdg",
					Ebs: api.AWSEbsBlockDeviceSpec{
						DeleteOnTermination: aws.Bool(false),
						Encrypted:           true,
						Iops:                100,
						VolumeSize:          64,
						VolumeType:          "io1",
					},
				},
				{
					DeviceName: "/dev/xvdg2",
					Ebs: api.AWSEbsBlockDeviceSpec{
						Encrypted:  true,
						Iops:       100,
						VolumeSize: 64,
						VolumeType: "io1",
					},
				},
				{
					DeviceName: "/dev/xvdg",
					Ebs: api.AWSEbsBlockDeviceSpec{
						DeleteOnTermination: aws.Bool(false),
						Encrypted:           true,
						Iops:                1000,
						VolumeSize:          10,
						VolumeType:          "gp3",
						Throughput:          aws.Int32(200),
					},
				},
				{
					DeviceName: "/dev/xvdg",
					Ebs: api.AWSEbsBlockDeviceSpec{
						DeleteOnTermination: aws.Bool(false),
						Encrypted:           true,
						VolumeSize:          10,
						VolumeType:          "gp3",
					},
				},
			}

			rootDevice := aws.String("/dev/sda")
			disksGenerated, err := awsDriver.generateBlockDevices(disks, rootDevice)
			expectedDisks := []ec2types.BlockDeviceMapping{
				{
					DeviceName: aws.String("/dev/sda"),
					Ebs: &ec2types.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           aws.Bool(false),
						VolumeSize:          aws.Int32(32),
						Iops:                nil,
						VolumeType:          ec2types.VolumeTypeGp2,
					},
				},
				{
					DeviceName: aws.String("/dev/xvdg"),
					Ebs: &ec2types.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(false),
						Encrypted:           aws.Bool(true),
						VolumeSize:          aws.Int32(64),
						Iops:                aws.Int32(100),
						VolumeType:          ec2types.VolumeTypeIo1,
					},
				},
				{
					DeviceName: aws.String("/dev/xvdg2"),
					Ebs: &ec2types.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           aws.Bool(true),
						VolumeSize:          aws.Int32(64),
						Iops:                aws.Int32(100),
						VolumeType:          ec2types.VolumeTypeIo1,
					},
				},
				{
					DeviceName: aws.String("/dev/xvdg"),
					Ebs: &ec2types.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(false),
						Encrypted:           aws.Bool(true),
						VolumeSize:          aws.Int32(10),
						Iops:                aws.Int32(1000),
						Throughput:          aws.Int32(200),
						VolumeType:          ec2types.VolumeTypeGp3,
					},
				},
				{
					DeviceName: aws.String("/dev/xvdg"),
					Ebs: &ec2types.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(false),
						Encrypted:           aws.Bool(true),
						VolumeSize:          aws.Int32(10),
						Iops:                nil,
						VolumeType:          ec2types.VolumeTypeGp3,
					},
				},
			}

			Expect(disksGenerated).To(Equal(expectedDisks))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should convert single blockDevices without deviceName successfully", func() {
			awsDriver := &Driver{}
			disks := []api.AWSBlockDeviceMappingSpec{
				{
					Ebs: api.AWSEbsBlockDeviceSpec{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           false,
						VolumeSize:          32,
						VolumeType:          "gp2",
					},
				},
			}

			rootDevice := aws.String("/dev/sda")
			disksGenerated, err := awsDriver.generateBlockDevices(disks, rootDevice)
			expectedDisks := []ec2types.BlockDeviceMapping{
				{
					DeviceName: aws.String("/dev/sda"),
					Ebs: &ec2types.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           aws.Bool(false),
						VolumeSize:          aws.Int32(32),
						Iops:                nil,
						VolumeType:          ec2types.VolumeTypeGp2,
					},
				},
			}

			Expect(disksGenerated).To(Equal(expectedDisks))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Convert zero blockDevices should have errored", func() {
			awsDriver := &Driver{}
			disks := []api.AWSBlockDeviceMappingSpec{}

			rootDevice := aws.String("/dev/sda")
			disksGenerated, err := awsDriver.generateBlockDevices(disks, rootDevice)
			var expectedDisks []ec2types.BlockDeviceMapping

			Expect(disksGenerated).To(Equal(expectedDisks))
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(fmt.Errorf("no block devices passed")))
		})

		It("should not encrypt blockDevices by default", func() {
			awsDriver := &Driver{}
			disks := []api.AWSBlockDeviceMappingSpec{
				{
					Ebs: api.AWSEbsBlockDeviceSpec{
						VolumeSize: 32,
						VolumeType: "gp2",
					},
				},
			}

			rootDevice := aws.String("/dev/sda")
			disksGenerated, err := awsDriver.generateBlockDevices(disks, rootDevice)
			expectedDisks := []ec2types.BlockDeviceMapping{
				{
					DeviceName: aws.String("/dev/sda"),
					Ebs: &ec2types.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           aws.Bool(false),
						VolumeSize:          aws.Int32(32),
						Iops:                nil,
						VolumeType:          ec2types.VolumeTypeGp2,
					},
				},
			}

			Expect(disksGenerated).To(Equal(expectedDisks))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("#getMachineInstancesByTagsAndStatus", func() {
		var (
			ctx                context.Context
			machineName        string
			providerSpecTags   map[string]string
			mockClientProvider *mockclient.MockClientProvider
			mockClient         interfaces.Ec2Client
		)

		BeforeEach(func() {
			ctx = context.Background()
			machineName = "test-machine-0"
			providerSpecTags = map[string]string{
				"kubernetes.io/cluster/shoot--test": "1",
				"kubernetes.io/role/node":           "1",
			}
			mockClientProvider = &mockclient.MockClientProvider{
				FakeInstances: make([]ec2types.Instance, 0),
			}
			mockClient = mockClientProvider.NewEC2Client(nil)
		})

		// createTestInstanceWithDefaultTags creates a test instance with default tags (Name + providerSpecTags)
		createTestInstanceWithDefaultTags := func(instanceID string, state ec2types.InstanceStateName) ec2types.Instance {
			tags := map[string]string{
				"Name": machineName,
			}
			for k, v := range providerSpecTags {
				tags[k] = v
			}
			var instanceTags []ec2types.Tag
			for key, value := range tags {
				instanceTags = append(instanceTags, ec2types.Tag{
					Key:   aws.String(key),
					Value: aws.String(value),
				})
			}
			return ec2types.Instance{
				InstanceId: aws.String(instanceID),
				State: &ec2types.InstanceState{
					Name: state,
				},
				Tags: instanceTags,
			}
		}

		It("should return a single instance with matching tags", func() {
			instance := createTestInstanceWithDefaultTags("i-test-instance-1", ec2types.InstanceStateNameRunning)
			mockClientProvider.FakeInstances = append(mockClientProvider.FakeInstances, instance)

			instances, err := getMachineInstancesByTagsAndStatus(ctx, mockClient, machineName, providerSpecTags)

			Expect(err).ToNot(HaveOccurred())
			Expect(instances).To(HaveLen(1))
			Expect(*instances[0].InstanceId).To(Equal("i-test-instance-1"))
			Expect(instances[0].State.Name).To(Equal(ec2types.InstanceStateNameRunning))
		})

		It("should return multiple instances with matching tags", func() {
			for i := range 3 {
				instance := createTestInstanceWithDefaultTags(fmt.Sprintf("i-test-instance-%d", i), ec2types.InstanceStateNameRunning)
				mockClientProvider.FakeInstances = append(mockClientProvider.FakeInstances, instance)
			}

			instances, err := getMachineInstancesByTagsAndStatus(ctx, mockClient, machineName, providerSpecTags)

			Expect(err).ToNot(HaveOccurred())
			Expect(instances).To(HaveLen(3))
		})

		It("should return instances across multiple pages", func() {
			mockClientProvider.PageSize = 2

			for i := range 5 {
				instance := createTestInstanceWithDefaultTags(fmt.Sprintf("i-test-instance-%d", i), ec2types.InstanceStateNameRunning)
				mockClientProvider.FakeInstances = append(mockClientProvider.FakeInstances, instance)
			}

			mockClient = mockClientProvider.NewEC2Client(nil)

			instances, err := getMachineInstancesByTagsAndStatus(ctx, mockClient, machineName, providerSpecTags)

			Expect(err).ToNot(HaveOccurred())
			Expect(instances).To(HaveLen(5))
		})

		It("should return empty list when no instances exist", func() {
			// Don't create any instances
			instances, err := getMachineInstancesByTagsAndStatus(ctx, mockClient, machineName, providerSpecTags)

			Expect(err).ToNot(HaveOccurred())
			Expect(instances).To(HaveLen(0))
		})

		It("should handle error from DescribeInstances API", func() {
			// trigger error in mock
			errorTags := map[string]string{
				"kubernetes.io/cluster/" + mockclient.ReturnErrorAtDescribeInstances: "1",
				"kubernetes.io/role/node": "1",
			}

			instances, err := getMachineInstancesByTagsAndStatus(ctx, mockClient, machineName, errorTags)

			Expect(err).To(HaveOccurred())
			Expect(instances).To(BeNil())
		})

		It("should stop pagination on duplicate token", func() {
			// Render the mock to return duplicate token to the 3rd page
			mockClientProvider.TriggerDuplicateToken = 2
			mockClientProvider.PageSize = 1
			mockClient = mockClientProvider.NewEC2Client(nil)

			instance := createTestInstanceWithDefaultTags("i-test-instance-1", ec2types.InstanceStateNameRunning)
			instance2 := createTestInstanceWithDefaultTags("i-test-instance-2", ec2types.InstanceStateNameRunning)
			instance3 := createTestInstanceWithDefaultTags("i-test-instance-3", ec2types.InstanceStateNameRunning)
			instance4 := createTestInstanceWithDefaultTags("i-test-instance-4", ec2types.InstanceStateNameRunning)
			mockClientProvider.FakeInstances = append(mockClientProvider.FakeInstances, instance, instance2, instance3, instance4)

			instances, err := getMachineInstancesByTagsAndStatus(ctx, mockClient, machineName, providerSpecTags)

			Expect(err).ToNot(HaveOccurred())
			// The paginator stops when it detects a duplicate token at the third page, so only two instances are returned
			Expect(len(instances)).To(Equal(2))
			Expect(*instances[0].InstanceId).To(Equal("i-test-instance-1"))
			Expect(*instances[1].InstanceId).To(Equal("i-test-instance-2"))
		})
	})
})
