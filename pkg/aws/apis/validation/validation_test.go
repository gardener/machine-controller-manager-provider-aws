package validation

import (
	"fmt"

	awsapi "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
			errList           []error
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				validationErr := ValidateAWSProviderSpec(data.action.spec, data.action.secret)

				if data.expect.errToHaveOccurred {
					Expect(validationErr).NotTo(Equal(nil))
					Expect(validationErr).To(Equal(data.expect.errList))
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
					errList: []error{
						fmt.Errorf("AMI is required field"),
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
					errList: []error{
						fmt.Errorf("Region is required field"),
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
					errList: []error{
						fmt.Errorf("MachineType is required field"),
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
					errList: []error{
						fmt.Errorf("IAM Name is required field"),
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
					errList: []error{
						fmt.Errorf("KeyName is required field"),
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
					errList: []error{
						fmt.Errorf("Tag is required of the form kubernetes.io/cluster/****"),
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
					errList: []error{
						fmt.Errorf("Tag is required of the form kubernetes.io/role/****"),
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
								Ebs: awsapi.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
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
					errList: []error{
						fmt.Errorf("Can only specify one (root) block device"),
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
					errList: []error{
						fmt.Errorf("Please mention a valid ebs volume size"),
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
					errList: []error{
						fmt.Errorf("Please mention a valid ebs volume type"),
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
					errList: []error{
						fmt.Errorf("Please mention a valid ebs volume iops"),
					},
				},
			}),
			Entry("NICs are missing", &data{
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
					errList: []error{
						fmt.Errorf("Mention at least one NetworkInterface"),
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
					errList: []error{
						fmt.Errorf("SubnetID is required"),
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
					errList: []error{
						fmt.Errorf("Mention at least one securityGroupID"),
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
					errList: []error{
						fmt.Errorf("Secret providerAccessKeyId is required field"),
					},
				},
			}),
			Entry("Secret ProviderSecretAccessKey is required field", &data{
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
					errList: []error{
						fmt.Errorf("Secret providerSecretAccessKey is required field"),
					},
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
					errList: []error{
						fmt.Errorf("Secret userData is required field"),
					},
				},
			}),
			Entry("Security group ID left blank for NIC", &data{
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
					errList: []error{
						fmt.Errorf("securityGroupIDs cannot be blank for networkInterface:0 securityGroupID:0"),
					},
				},
			}),
		)
	})
})
