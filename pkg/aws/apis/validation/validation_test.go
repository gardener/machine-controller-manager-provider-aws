// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	awsapi "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
)

var _ = Describe("Validation", func() {

	providerSecret := &corev1.Secret{
		Data: map[string][]byte{
			"providerAccessKeyId":     []byte("dummy-id"),
			"providerSecretAccessKey": []byte("dummy-secret"),
			"userData":                []byte("dummy-user-data"),
		},
	}

	Describe("#ValidateAWSProviderSpec", func() {
		type setup struct {
			apply func(spec *awsapi.AWSProviderSpec)
		}
		type action struct {
			spec   *awsapi.AWSProviderSpec
			secret *corev1.Secret
		}
		type expect struct {
			errToHaveOccurred bool
			errList           field.ErrorList
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				if data.setup.apply != nil {
					data.setup.apply(data.action.spec)
				}
				validationErr := ValidateAWSProviderSpec(data.action.spec, data.action.secret, field.NewPath("providerSpec"))

				if data.expect.errToHaveOccurred {
					Expect(validationErr).NotTo(Equal(field.ErrorList{}))
					Expect(validationErr).To(Equal(data.expect.errList))
				} else {
					Expect(validationErr).To(Equal(field.ErrorList{}))
				}

			},
			Entry("Simple validation of AWS machine class", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AWS machine class with io1 type block device", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "io1",
									Iops:       1000,
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AWS machine class with gp3 type block device", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp3",
									Iops:       3500,
									Throughput: aws.Int64(200),
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AMI field missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.ami",
							BadValue: "",
							Detail:   "AMI is required",
						},
					},
				},
			}),
			Entry("Region field missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.region",
							BadValue: "",
							Detail:   "Region is required",
						},
					},
				},
			}),
			Entry("MachineType field missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:  "eu-west-1",
						KeyName: pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.machineType",
							BadValue: "",
							Detail:   "MachineType is required",
						},
					},
				},
			}),
			Entry("both IAM.Name and IAM.ARN fields missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM:         awsapi.AWSIAMProfileSpec{},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueInvalid",
							Field:    "providerSpec.iam",
							BadValue: awsapi.AWSIAMProfileSpec{},
							Detail:   "either IAM Name or ARN must be set",
						},
					},
				},
			}),
			Entry("both IAM.Name and IAM.ARN fields set", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "foo",
							ARN:  "bar",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:  "FieldValueInvalid",
							Field: "providerSpec.iam",
							BadValue: awsapi.AWSIAMProfileSpec{
								Name: "foo",
								ARN:  "bar",
							},
							Detail: "either IAM Name or ARN must be set",
						},
					},
				},
			}),
			Entry("Cluster tag missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/role/test": "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.tags[]",
							BadValue: "",
							Detail:   "Tag required of the form kubernetes.io/cluster/****",
						},
					},
				},
			}),
			Entry("Role tag missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.tags[]",
							BadValue: "",
							Detail:   "Tag required of the form kubernetes.io/role/****",
						},
					},
				},
			}),
			Entry("Multiple block devices specified", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								DeviceName: "/dev/sda",
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
							{
								DeviceName: "/root",
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("Multiple root block devices specified", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								DeviceName: "/root",
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
							{
								DeviceName: "/root",
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices",
							BadValue: "",
							Detail:   "Only one device can be specified as root",
						},
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices",
							BadValue: "",
							Detail:   "Device name '/root' duplicated 2 times, DeviceName must be unique",
						},
					},
				},
			}),
			Entry("Multiple block devices specified, one with invalid data", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								DeviceName: "/root",
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
							{
								VirtualName: "kubelet-dir",
								DeviceName:  "/dev/sda",
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: -50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices[1].ebs.volumeSize",
							BadValue: "",
							Detail:   "Please mention a valid EBS volume size",
						},
					},
				},
			}),
			Entry("Invalid block device size specified", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: -10,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices[0].ebs.volumeSize",
							BadValue: "",
							Detail:   "Please mention a valid EBS volume size",
						},
					},
				},
			}),
			Entry("EBS volume type is missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices[0].ebs.volumeType",
							BadValue: "",
							Detail:   fmt.Sprintf("Please mention a valid EBS volume type: %v", awsapi.ValidVolumeTypes),
						},
					},
				},
			}),
			Entry("EBS volume type is of not mentioned type", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp4",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices[0].ebs.volumeType",
							BadValue: "",
							Detail:   fmt.Sprintf("Please mention a valid EBS volume type: %v", awsapi.ValidVolumeTypes),
						},
					},
				},
			}),
			Entry("EBS volume of type io1 is missing iops field", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "io1",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices[0].ebs.iops",
							BadValue: "",
							Detail:   "Please mention a valid EBS volume iops",
						},
					},
				},
			}),
			Entry("EBS volume of type gp3 is missing iops and throughout field", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp3",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("Invalid EBS volume iops", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									Iops:       -100,
									VolumeType: "gp3",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.blockDevices[0].ebs.iops",
							BadValue: "",
							Detail:   "Please mention a valid EBS volume iops",
						},
					},
				},
			}),
			Entry("Invalid EBS volume throughput", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									Iops:       100,
									Throughput: aws.Int64(-200),
									VolumeType: "gp3",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueInvalid",
							Field:    "providerSpec.blockDevices[0].ebs.throughput",
							BadValue: int64(-200),
							Detail:   "Throughput should be a positive value",
						},
					},
				},
			}),
			Entry("Network Interfaces are missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.networkInterfaces[]",
							BadValue: "",
							Detail:   "Mention at least one NetworkInterface",
						},
					},
				},
			}),
			Entry("SubnetID is missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.networkInterfaces.subnetID",
							BadValue: "",
							Detail:   "SubnetID is required",
						},
					},
				},
			}),
			Entry("SecurityGroupIDs are missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.networkInterfaces.securityGroupIDs",
							BadValue: "",
							Detail:   "Mention at least one securityGroupID",
						},
					},
				},
			}),
			Entry("ProviderAccessKeyID is missing", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerSecretAccessKey": []byte("dummy-secret"),
							"userData":                []byte("dummy-user-data"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "secretRef.AWSAccessKeyID",
							BadValue: "",
							Detail:   "Mention atleast providerAccessKeyId or accessKeyID",
						},
					},
				},
			}),
			Entry("Mention atleast AWSSecretAccessKey or AWSAlternativeSecretAccessKey", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId": []byte("dummy-id"),
							"userData":            []byte("dummy-user-data"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "secretRef.AWSSecretAccessKey",
							BadValue: "",
							Detail:   "Mention atleast providerSecretAccessKey or secretAccessKey",
						},
					},
				},
			}),
			Entry("secretAccessKey is mentioned", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId": []byte("dummy-id"),
							"userData":            []byte("dummy-user-data"),
							"secretAccessKey":     []byte("dummy-user-data"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("Secret UserData is required field", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "secretRef.userData",
							BadValue: "",
							Detail:   "Mention userData",
						},
					},
				},
			}),
			Entry("Security group ID left blank for network interface", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.networkInterfaces.securityGroupIDs",
							BadValue: "",
							Detail:   "securityGroupIDs cannot be blank for networkInterface:0 securityGroupID:0",
						},
					},
				},
			}),
			Entry("AWS machine class with instanceMetadata with valid instanceMetadata.httpPutResponseHopLimit", &data{
				setup: setup{
					apply: func(spec *awsapi.AWSProviderSpec) {
						spec.InstanceMetadataOptions = &awsapi.InstanceMetadataOptions{
							HTTPPutResponseHopLimit: pointer.Int64(32),
						}
					},
				},
				action: action{
					spec:   validAWSProviderSpec(),
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AWS machine class with invalid instanceMetadata.httpPutResponseHopLimit.", &data{
				setup: setup{
					apply: func(spec *awsapi.AWSProviderSpec) {
						spec.InstanceMetadataOptions = &awsapi.InstanceMetadataOptions{
							HTTPPutResponseHopLimit: pointer.Int64(72),
						}
					},
				},
				action: action{
					spec:   validAWSProviderSpec(),
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueInvalid",
							Field:    "providerSpec.instanceMetadata.httpPutResponseHopLimit",
							BadValue: int64(72),
							Detail:   "Only values between 0 and 64, both included, are accepted",
						},
					},
				},
			}),
			Entry("AWS machine class with instanceMetadata with valid instanceMetadata.httpEndpoint", &data{
				setup: setup{
					apply: func(spec *awsapi.AWSProviderSpec) {
						spec.InstanceMetadataOptions = &awsapi.InstanceMetadataOptions{
							HTTPEndpoint: pointer.String(awsapi.HTTPEndpointDisabled),
						}
					},
				},
				action: action{
					spec:   validAWSProviderSpec(),
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AWS machine class with invalid instanceMetadata.httpEndpoint", &data{
				setup: setup{
					apply: func(spec *awsapi.AWSProviderSpec) {
						spec.InstanceMetadataOptions = &awsapi.InstanceMetadataOptions{
							HTTPEndpoint: pointer.String("foobar"),
						}
					},
				},
				action: action{
					spec:   validAWSProviderSpec(),
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueInvalid",
							Field:    "providerSpec.instanceMetadata.httpEndpoint",
							BadValue: "foobar",
							Detail:   "Accepted values: [disabled enabled]",
						},
					},
				},
			}),
			Entry("AWS machine class with instanceMetadata with valid instanceMetadata.httpTokens", &data{
				setup: setup{
					apply: func(spec *awsapi.AWSProviderSpec) {
						spec.InstanceMetadataOptions = &awsapi.InstanceMetadataOptions{
							HTTPTokens: pointer.String(awsapi.HTTPTokensRequired),
						}
					},
				},
				action: action{
					spec:   validAWSProviderSpec(),
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AWS machine class with invalid instanceMetadata.httpTokens", &data{
				setup: setup{
					apply: func(spec *awsapi.AWSProviderSpec) {
						spec.InstanceMetadataOptions = &awsapi.InstanceMetadataOptions{
							HTTPTokens: pointer.String("foobar"),
						}
					},
				},
				action: action{
					spec:   validAWSProviderSpec(),
					secret: providerSecret,
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueInvalid",
							Field:    "providerSpec.instanceMetadata.httpTokens",
							BadValue: "foobar",
							Detail:   "Accepted values: [required optional]",
						},
					},
				},
			}),
			Entry("CapacityReservationTargetSpec invalid spec configuring both preference, target id and arn", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						CapacityReservationTarget: &awsapi.AWSCapacityReservationTargetSpec{
							CapacityReservationPreference:       pointer.String("open"),
							CapacityReservationID:               pointer.String("capacity-reservation-id-abcd1234"),
							CapacityReservationResourceGroupArn: pointer.String("arn:01234:/my-resource-group"),
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"userData":                []byte("dummy-user-data"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.capacityReservation",
							BadValue: "",
							Detail:   "CapacityReservationPreference cannot be set when also providing a CapacityReservationID or CapacityReservationResourceGroupArn",
						},
					},
				},
			}),
			Entry("CapacityReservationTargetSpec invalid spec configuring both preference and target id", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						CapacityReservationTarget: &awsapi.AWSCapacityReservationTargetSpec{
							CapacityReservationPreference: pointer.String("open"),
							CapacityReservationID:         pointer.String("capacity-reservation-id-abcd1234"),
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"userData":                []byte("dummy-user-data"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.capacityReservation",
							BadValue: "",
							Detail:   "CapacityReservationPreference cannot be set when also providing a CapacityReservationID or CapacityReservationResourceGroupArn",
						},
					},
				},
			}),
			Entry("CapacityReservationTargetSpec invalid spec configuring both preference and arn", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						CapacityReservationTarget: &awsapi.AWSCapacityReservationTargetSpec{
							CapacityReservationPreference:       pointer.String("open"),
							CapacityReservationResourceGroupArn: pointer.String("arn:01234:/my-resource-group"),
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"userData":                []byte("dummy-user-data"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.capacityReservation",
							BadValue: "",
							Detail:   "CapacityReservationPreference cannot be set when also providing a CapacityReservationID or CapacityReservationResourceGroupArn",
						},
					},
				},
			}),
			Entry("CapacityReservationTargetSpec invalid spec configuring both id and arn", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						CapacityReservationTarget: &awsapi.AWSCapacityReservationTargetSpec{
							CapacityReservationID:               pointer.String("capacity-reservation-id-abcd1234"),
							CapacityReservationResourceGroupArn: pointer.String("arn:01234:/my-resource-group"),
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"userData":                []byte("dummy-user-data"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: field.ErrorList{
						{
							Type:     "FieldValueRequired",
							Field:    "providerSpec.capacityReservation",
							BadValue: "",
							Detail:   "CapacityReservationResourceGroupArn or CapacityReservationId are optional but only one should be used",
						},
					},
				},
			}),
			Entry("CapacityReservationTargetSpec valid spec configuring only CapacityReservationID", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						CapacityReservationTarget: &awsapi.AWSCapacityReservationTargetSpec{
							CapacityReservationID: pointer.String("capacity-reservation-id-abcd1234"),
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"userData":                []byte("dummy-user-data"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("CapacityReservationTargetSpec valid spec configuring only CapacityReservationResourceGroupArn", &data{
				setup: setup{},
				action: action{
					spec: &awsapi.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
							{
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						CapacityReservationTarget: &awsapi.AWSCapacityReservationTargetSpec{
							CapacityReservationResourceGroupArn: pointer.String("arn:01234:/my-resource-group"),
						},
						IAM: awsapi.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     pointer.String("test-ssh-publickey"),
						NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
							{
								SecurityGroupIDs: []string{
									"sg-00002132323",
								},
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secret: &corev1.Secret{
						Data: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"userData":                []byte("dummy-user-data"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
		)
	})

	Describe("#ValidateSecret", func() {
		It("should successfully validate the secret", func() {
			errList := ValidateSecret(&corev1.Secret{
				Data: map[string][]byte{
					"roleARN":                   []byte("arn"),
					"workloadIdentityTokenFile": []byte("file"),
					"userData":                  []byte("data"),
				},
			}, field.NewPath(""))
			Expect(errList).To(BeEmpty())
		})

		It("should fail to validate the secret", func() {
			errList := ValidateSecret(&corev1.Secret{
				Data: map[string][]byte{
					"workloadIdentityTokenFile": []byte(""),
					"roleARN":                   []byte(""),
				},
			}, field.NewPath(""))
			Expect(errList).To(
				ConsistOf(
					PointTo(
						MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeRequired),
							"Field": Equal("[].workloadIdentityTokenFile"),
						}),
					),
					PointTo(
						MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeRequired),
							"Field": Equal("[].roleARN"),
						}),
					),
					PointTo(
						MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeRequired),
							"Field": Equal("[].userData"),
						}),
					),
				),
			)
		})
	})
})

func validAWSProviderSpec() *awsapi.AWSProviderSpec {
	return &awsapi.AWSProviderSpec{
		AMI: "ami-123456789",
		BlockDevices: []awsapi.AWSBlockDeviceMappingSpec{
			{
				Ebs: awsapi.AWSEbsBlockDeviceSpec{
					VolumeSize: 50,
					VolumeType: "gp2",
				},
			},
		},
		CapacityReservationTarget: &awsapi.AWSCapacityReservationTargetSpec{
			CapacityReservationPreference: pointer.String("open"),
		},
		IAM: awsapi.AWSIAMProfileSpec{
			Name: "test-iam",
		},
		Region:      "eu-west-1",
		MachineType: "m4.large",
		KeyName:     pointer.String("test-ssh-publickey"),
		NetworkInterfaces: []awsapi.AWSNetworkInterfaceSpec{
			{
				SecurityGroupIDs: []string{
					"sg-00002132323",
				},
				SubnetID: "subnet-123456",
			},
		},
		Tags: map[string]string{
			"kubernetes.io/cluster/shoot--test": "1",
			"kubernetes.io/role/test":           "1",
		},
	}
}
