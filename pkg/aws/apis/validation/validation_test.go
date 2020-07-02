package validation

//. "github.com/onsi/ginkgo"
//. "github.com/onsi/ginkgo/extensions/table"
//. "github.com/onsi/gomega"

/*
TODO fix this
var _ = Describe("Validation", func() {

	Describe("#ValidateAWSProviderSpec", func() {
		type setup struct {
		}
		type action struct {
			spec    *api.AWSProviderSpec
			secrets *api.Secrets
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
				validationErr := ValidateAWSProviderSpec(data.action.spec, data.action.secrets)

				if data.expect.errToHaveOccurred {
					Expect(validationErr).NotTo(Equal(nil))
					Expect(validationErr).To(Equal(data.expect.errList))
				}

			},
			Entry("Simple validation of AWS machine class", &data{
				setup: setup{},
				action: action{
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AWS machine class with io1 type block device", &data{
				setup: setup{},
				action: action{
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "io1",
									Iops:       1000,
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("AMI field missing", &data{
				setup: setup{},
				action: action{
					spec: &api.AWSProviderSpec{
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:  "eu-west-1",
						KeyName: "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: -10,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "io1",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
								SubnetID: "subnet-123456",
							},
						},
						Tags: map[string]string{
							"kubernetes.io/cluster/shoot--test": "1",
							"kubernetes.io/role/test":           "1",
						},
					},
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderSecretAccessKey: "dummy-secret",
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: []error{
						fmt.Errorf("Secret ProviderAccessKeyID is required field"),
					},
				},
			}),
			Entry("Secret ProviderSecretAccessKey is required field", &data{
				setup: setup{},
				action: action{
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:            "dummy-user-data",
						ProviderAccessKeyID: "dummy-id",
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: []error{
						fmt.Errorf("Secret ProviderSecretAccessKey is required field"),
					},
				},
			}),
			Entry("Secret UserData is required field", &data{
				setup: setup{},
				action: action{
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{

						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errList: []error{
						fmt.Errorf("Secret UserData is required field"),
					},
				},
			}),
			Entry("Security group ID left blank for NIC", &data{
				setup: setup{},
				action: action{
					spec: &api.AWSProviderSpec{
						AMI: "ami-123456789",
						BlockDevices: []api.AWSBlockDeviceMappingSpec{
							api.AWSBlockDeviceMappingSpec{
								Ebs: api.AWSEbsBlockDeviceSpec{
									VolumeSize: 50,
									VolumeType: "gp2",
								},
							},
						},
						IAM: api.AWSIAMProfileSpec{
							Name: "test-iam",
						},
						Region:      "eu-west-1",
						MachineType: "m4.large",
						KeyName:     "test-ssh-publickey",
						NetworkInterfaces: []api.AWSNetworkInterfaceSpec{
							api.AWSNetworkInterfaceSpec{
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
					secrets: &api.Secrets{
						UserData:                "dummy-user-data",
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
					},
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
*/
