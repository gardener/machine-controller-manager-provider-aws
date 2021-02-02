package validation

import (
	"fmt"

	awsapi "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName: "test-ssh-publickey",
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
			Entry("IAM.Name field missing", &data{
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
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
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
							Field:    "providerSpec.iam.name",
							BadValue: "",
							Detail:   "IAM Name is required",
						},
					},
				},
			}),
			Entry("KeyName field missing", &data{
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
							Field:    "providerSpec.keyName",
							BadValue: "",
							Detail:   "KeyName is required",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
						KeyName:     "test-ssh-publickey",
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
		)
	})
})
