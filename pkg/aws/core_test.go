/*
Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/mockclient"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	awsAccessKeyIDIsMissing               = "machine codes error: code = [Internal] message = [Error while validating ProviderSpec secretRef.AWSAccessKeyID: Required value: Mention atleast providerAccessKeyId or accessKeyID]"
	awsSecretAccessKeyIsMissing           = "machine codes error: code = [Internal] message = [Error while validating ProviderSpec secretRef.AWSSecretAccessKey: Required value: Mention atleast providerSecretAccessKey or secretAccessKey]"
	awsSecretAccessKeyNUserDataAreMissing = "machine codes error: code = [Internal] message = [Error while validating ProviderSpec [secretRef.AWSSecretAccessKey: Required value: Mention atleast providerSecretAccessKey or secretAccessKey, secretRef.userData: Required value: Mention userData]]"
	regionNAMIMissing                     = "machine codes error: code = [Internal] message = [Error while validating ProviderSpec [providerSpec.ami: Required value: AMI is required, providerSpec.region: Required value: Region is required]]"
	userDataIsMissing                     = "machine codes error: code = [Internal] message = [Error while validating ProviderSpec secretRef.userData: Required value: Mention userData]"
	cloudProviderReturnedError            = "machine codes error: code = [Internal] message = [Cloud provider returned error]"
)

var _ = Describe("MachineServer", func() {

	// Some initializations
	providerSpec := []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")
	providerSecret := &corev1.Secret{
		Data: map[string][]byte{
			"providerAccessKeyId":     []byte("dummy-id"),
			"providerSecretAccessKey": []byte("dummy-secret"),
			"userData":                []byte("dummy-user-data"),
		},
	}

	Describe("#CreateMachine", func() {
		type setup struct {
		}
		type action struct {
			machineRequest *driver.CreateMachineRequest
		}
		type expect struct {
			machineResponse   *driver.CreateMachineResponse
			errToHaveOccurred bool
			errMessage        string
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewAWSDriver(mockPluginSPIImpl)

				ctx := context.Background()
				response, err := ms.CreateMachine(ctx, data.action.machineRequest)

				//klog.Error(err)
				//klog.Error(data.expect.errMessage)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(data.expect.machineResponse.ProviderID).To(Equal(response.ProviderID))
					Expect(data.expect.machineResponse.NodeName).To(Equal(response.NodeName))
				}
			},
			Entry("Simple Machine Creation Request", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:   "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("Machine creation request with volume type io1", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"io1\",\"iops\":50}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:   "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("Machine creation request for spot instance type", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"spotPrice\":\"\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:   "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("Machine creation request for spot instance type with max price", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"spotPrice\":\"500\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:   "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("Machine creation request for capacity reservations fails if more than one type given", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"capacityReservation\":{\"capacityReservationId\":\"cr-05c28b843c05abcde\",\"capacityReservationResourceGroupArn\":\"arn:aws:resource-groups:us-west-1:123456789012:group/my-test-cr-group\"},\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage: "machine codes error: code = [Internal] message = [Error while validating ProviderSpec providerSpec.capacityReservation: Required value: Either capacityReservationResourceGroupArn or capacityReservationId needs to be specified - but not both.]",
				},
			}),
			Entry("Machine creation request for capacity reservations with capacityReservationId", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"capacityReservation\":{\"capacityReservationId\":\"cr-05c28b843c05abcde\"},\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:   "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("Machine creation request for an AWS Capacity Reservation Group with capacityReservationResourceGroupArn", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"capacityReservation\":{\"capacityReservationResourceGroupArn\":\"arn:aws:resource-groups:us-west-1:123456789012:group/my-test-cr-group\"},\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:   "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("Unmarshalling for provider spec fails", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [unexpected end of JSON input]",
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerSecretAccessKey": []byte("dummy-secret"),
								"userData":                []byte("dummy-user-data"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsAccessKeyIDIsMissing,
				},
			}),
			Entry("providerSecretAccessKey missing for provider secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerAccessKeyId": []byte("dummy-secret"),
								"userData":            []byte("dummy-user-data"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsSecretAccessKeyIsMissing,
				},
			}),
			Entry("userData missing for provider secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerAccessKeyId":     []byte("dummy-secret"),
								"providerSecretAccessKey": []byte("dummy-secret"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        userDataIsMissing,
				},
			}),
			Entry("Validation for providerSpec fails. Missing AMI & Region.", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        regionNAMIMissing,
				},
			}),
			Entry("Invalid region that doesn't exist", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"" + mockclient.FailAtRegion + "\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [Region doesn't exist while trying to create session]",
				},
			}),
			Entry("Invalid image ID that doesn't exist", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"" + mockclient.FailQueryAtDescribeImages + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [Couldn't find image with given ID]",
				},
			}),
			Entry("Name tag cannot be set on AWS instances", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\",\"Name\":\"dummy\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:   "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			Entry("RunInstance call fails", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"" + mockclient.FailQueryAtRunInstances + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [Couldn't run instance with given ID]",
				},
			}),
		)
	})

	Describe("#DeleteMachine", func() {
		type setup struct {
			createMachineRequest *driver.CreateMachineRequest
			resetProviderToEmpty bool
		}
		type action struct {
			deleteMachineRequest *driver.DeleteMachineRequest
		}
		type expect struct {
			deleteMachineResponse *driver.DeleteMachineResponse
			errToHaveOccurred     bool
			errMessage            string
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewAWSDriver(mockPluginSPIImpl)

				ctx := context.Background()

				if data.setup.createMachineRequest != nil {
					_, err := ms.CreateMachine(ctx, data.setup.createMachineRequest)
					Expect(err).ToNot(HaveOccurred())
				}

				_, err := ms.DeleteMachine(ctx, data.action.deleteMachineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("Simple Machine Delete Request", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     false,
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerSecretAccessKey": []byte("dummy-secret"),
								"userData":                []byte("dummy-user-data"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsAccessKeyIDIsMissing,
				},
			}),
			Entry("providerSecretAccessKey & userData missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerAccessKeyId": []byte("dummy-id"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsSecretAccessKeyNUserDataAreMissing,
				},
			}),
			Entry("Termination of instance that doesn't exist on provider", &data{
				setup: setup{},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     true,
					errMessage:            "machine codes error: code = [Internal] message = [Couldn't find instance with given instance-ID i-0123456789-0]",
				},
			}),
			Entry("Termination of instance that doesn't exist on provider", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"" + mockclient.SetInstanceID + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"" + mockclient.FailQueryAtTerminateInstances + "\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass([]byte("{\"ami\":\"" + mockclient.SetInstanceID + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"" + mockclient.FailQueryAtTerminateInstances + "\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [Couldn't find instance with given instance-ID i-0123456789-0]",
				},
			}),
			Entry("Termination of machine without any backing instance", &data{
				setup: setup{},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     false,
				},
			}),
			Entry("Termination of machine without any backing instance but also failure at describe instances", &data{
				setup: setup{},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/" + mockclient.ReturnErrorAtDescribeInstances + "\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     true,
					errMessage:            cloudProviderReturnedError,
				},
			}),
			Entry("Termination of machine with any backing instance but no providerID", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						// Setting machineIndex to -1 to simulate no providerID
						Machine:      newMachine(-1),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     false,
				},
			}),
		)
	})

	Describe("#GetMachine", func() {
		type setup struct {
			createMachineRequest *driver.CreateMachineRequest
		}
		type action struct {
			getMachineRequest *driver.GetMachineStatusRequest
		}
		type expect struct {
			getMachineResponse *driver.GetMachineStatusResponse
			errToHaveOccurred  bool
			errMessage         string
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewAWSDriver(mockPluginSPIImpl)
				ctx := context.Background()

				if data.setup.createMachineRequest != nil {
					_, err := ms.CreateMachine(ctx, data.setup.createMachineRequest)
					Expect(err).ToNot(HaveOccurred())
				}

				_, err := ms.GetMachineStatus(ctx, data.action.getMachineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("Simple Machine Get Request", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerSecretAccessKey": []byte("dummy-secret"),
								"userData":                []byte("dummy-user-data"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsAccessKeyIDIsMissing,
				},
			}),
			Entry("providerSecretAccessKey missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerAccessKeyId": []byte("dummy-id"),
								"userData":            []byte("dummy-user-data"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsSecretAccessKeyIsMissing,
				},
			}),
			Entry("userData missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerAccessKeyId":     []byte("dummy-id"),
								"providerSecretAccessKey": []byte("dummy-secret"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        userDataIsMissing,
				},
			}),
			Entry("Machine deletion where provider-ID is missing", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(-1),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("Get request without a create request", &data{
				setup: setup{},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					getMachineResponse: &driver.GetMachineStatusResponse{},
					errToHaveOccurred:  true,
					errMessage:         "machine codes error: code = [NotFound] message = [AWS plugin is returning no VM instances backing this machine object]",
				},
			}),
		)
	})

	Describe("#ListMachines", func() {
		type setup struct {
			createMachineRequest []*driver.CreateMachineRequest
		}
		type action struct {
			listMachineRequest *driver.ListMachinesRequest
		}
		type expect struct {
			listMachineResponse *driver.ListMachinesResponse
			errToHaveOccurred   bool
			errMessage          string
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewAWSDriver(mockPluginSPIImpl)
				ctx := context.Background()

				for _, createReq := range data.setup.createMachineRequest {
					_, err := ms.CreateMachine(ctx, createReq)
					Expect(err).ToNot(HaveOccurred())
				}

				listResponse, err := ms.ListMachines(ctx, data.action.listMachineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(len(listResponse.MachineList)).To(Equal(len(data.expect.listMachineResponse.MachineList)))
					//Expect(listResponse.MachineList).To(Equal(data.expect.listMachineResponse))
				}
			},
			Entry("Simple Machine List Request", &data{
				setup: setup{
					createMachineRequest: []*driver.CreateMachineRequest{
						{
							Machine:      newMachine(0),
							MachineClass: newMachineClass(providerSpec),
							Secret:       providerSecret,
						},
						{
							Machine:      newMachine(1),
							MachineClass: newMachineClass(providerSpec),
							Secret:       providerSecret,
						},
						{
							Machine:      newMachine(2),
							MachineClass: newMachineClass(providerSpec),
							Secret:       providerSecret,
						},
					},
				},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
					listMachineResponse: &driver.ListMachinesResponse{
						MachineList: map[string]string{
							"aws:///eu-west-1/i-0123456789-0": "machine-0",
							"aws:///eu-west-1/i-0123456789-1": "machine-1",
							"aws:///eu-west-1/i-0123456789-2": "machine-2",
						},
					},
				},
			}),
			Entry("Unexpected end of JSON input", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass([]byte("")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [unexpected end of JSON input]",
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerSecretAccessKey": []byte("dummy-secret"),
								"userData":                []byte("dummy-user-data"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsAccessKeyIDIsMissing,
				},
			}),
			Entry("providerSecretAccessKey missing for secret", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerAccessKeyId": []byte("dummy-id"),
								"userData":            []byte("dummy-user-data"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        awsSecretAccessKeyIsMissing,
				},
			}),
			Entry("userData missing for secret", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass(providerSpec),
						Secret: &corev1.Secret{
							Data: map[string][]byte{
								"providerAccessKeyId":     []byte("dummy-id"),
								"providerSecretAccessKey": []byte("dummy-secret"),
							},
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        userDataIsMissing,
				},
			}),

			Entry("Validation for providerSpec fails. Missing AMI & Region.", &data{
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass([]byte("{\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        regionNAMIMissing,
				},
			}),

			Entry("Region doesn't exist", &data{
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"" + mockclient.FailAtRegion + "\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [Region doesn't exist while trying to create session]",
				},
			}),
			Entry("Cluster details missing in machine class", &data{
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [Error while validating ProviderSpec providerSpec.tags[]: Required value: Tag required of the form kubernetes.io/cluster/****]",
				},
			}),
			Entry("Cloud provider returned error while describing instance", &data{
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/" + mockclient.ReturnErrorAtDescribeInstances + "\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        cloudProviderReturnedError,
				},
			}),
			Entry("List request without a create request", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					listMachineResponse: &driver.ListMachinesResponse{},
				},
			}),
		)
	})

	Describe("#GetVolumeIDs", func() {
		type setup struct {
		}
		type action struct {
			getListOfVolumeIDsForExistingPVsRequest *driver.GetVolumeIDsRequest
		}
		type expect struct {
			getListOfVolumeIDsForExistingPVsResponse *driver.GetVolumeIDsResponse
			errToHaveOccurred                        bool
			errMessage                               string
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewAWSDriver(mockPluginSPIImpl)
				ctx := context.Background()

				response, err := ms.GetVolumeIDs(
					ctx,
					data.action.getListOfVolumeIDsForExistingPVsRequest,
				)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(data.expect.errMessage))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(response).To(Equal(data.expect.getListOfVolumeIDsForExistingPVsResponse))
				}
			},
			Entry("GetVolumeIDs request for AWSEBS", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecs: []*corev1.PersistentVolumeSpec{
							{
								Capacity: map[corev1.ResourceName]resource.Quantity{},
								PersistentVolumeSource: corev1.PersistentVolumeSource{
									AWSElasticBlockStore: &corev1.AWSElasticBlockStoreVolumeSource{
										VolumeID: "aws://eu-east-2b/vol-xxxxyyyyzzzz11112",
									},
								},
								AccessModes:                   []corev1.PersistentVolumeAccessMode{},
								ClaimRef:                      &corev1.ObjectReference{},
								PersistentVolumeReclaimPolicy: "",
								StorageClassName:              "",
								MountOptions:                  []string{},
								NodeAffinity:                  &corev1.VolumeNodeAffinity{},
							},
						},
					},
				},
				expect: expect{
					getListOfVolumeIDsForExistingPVsResponse: &driver.GetVolumeIDsResponse{
						VolumeIDs: []string{
							"vol-xxxxyyyyzzzz11112",
						},
					},
				},
			}),
			Entry("GetVolumeIDs request for EBS.CSI", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecs: []*corev1.PersistentVolumeSpec{
							{
								Capacity: map[corev1.ResourceName]resource.Quantity{},
								PersistentVolumeSource: corev1.PersistentVolumeSource{
									CSI: &corev1.CSIPersistentVolumeSource{
										Driver:       "ebs.csi.aws.com",
										VolumeHandle: "vol-xxxxyyyyzzzz11112",
									},
								},
								AccessModes:                   []corev1.PersistentVolumeAccessMode{},
								ClaimRef:                      &corev1.ObjectReference{},
								PersistentVolumeReclaimPolicy: "",
								StorageClassName:              "",
								MountOptions:                  []string{},
								NodeAffinity:                  &corev1.VolumeNodeAffinity{},
							},
						},
					},
				},
				expect: expect{
					getListOfVolumeIDsForExistingPVsResponse: &driver.GetVolumeIDsResponse{
						VolumeIDs: []string{
							"vol-xxxxyyyyzzzz11112",
						},
					},
				},
			}),
			Entry("GetVolumeIDs request for Mock.CSI", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecs: []*corev1.PersistentVolumeSpec{
							{
								Capacity: map[corev1.ResourceName]resource.Quantity{},
								PersistentVolumeSource: corev1.PersistentVolumeSource{
									CSI: &corev1.CSIPersistentVolumeSource{
										Driver:       "mock.csi.aws.com",
										VolumeHandle: "vol-xxxxyyyyzzzz11112",
									},
								},
								AccessModes:                   []corev1.PersistentVolumeAccessMode{},
								ClaimRef:                      &corev1.ObjectReference{},
								PersistentVolumeReclaimPolicy: "",
								StorageClassName:              "",
								MountOptions:                  []string{},
								NodeAffinity:                  &corev1.VolumeNodeAffinity{},
							},
						},
					},
				},
				expect: expect{
					getListOfVolumeIDsForExistingPVsResponse: &driver.GetVolumeIDsResponse{
						VolumeIDs: nil,
					},
				},
			}),
			Entry("GetVolumeIDs request for multiple volumes", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecs: []*corev1.PersistentVolumeSpec{
							{
								PersistentVolumeSource: corev1.PersistentVolumeSource{
									CSI: &corev1.CSIPersistentVolumeSource{
										Driver:       "io.kubernetes.storage.mock",
										VolumeHandle: "vol-2345",
									},
								},
							},
							{
								PersistentVolumeSource: corev1.PersistentVolumeSource{
									CSI: &corev1.CSIPersistentVolumeSource{
										Driver:       "ebs.csi.aws.com",
										VolumeHandle: "vol-1234",
									},
								},
							},
							{
								PersistentVolumeSource: corev1.PersistentVolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/mnt/data",
									},
								},
							},
						},
					},
				},
				expect: expect{
					getListOfVolumeIDsForExistingPVsResponse: &driver.GetVolumeIDsResponse{
						VolumeIDs: []string{
							"vol-1234",
						},
					},
				},
			}),
		)
	})

	Describe("#GenerateMachineClassForMigration", func() {
		type setup struct {
		}
		type action struct {
			generateMachineClassForMigrationRequest *driver.GenerateMachineClassForMigrationRequest
		}
		type expect struct {
			machineClass *v1alpha1.MachineClass
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewAWSDriver(mockPluginSPIImpl)
				ctx := context.Background()

				_, _ = ms.GenerateMachineClassForMigration(
					ctx,
					data.action.generateMachineClassForMigrationRequest,
				)

				Expect(data.action.generateMachineClassForMigrationRequest.MachineClass).To(Equal(data.expect.machineClass))
			},
			Entry("Simple migration request with all fields set", &data{
				action: action{
					generateMachineClassForMigrationRequest: &driver.GenerateMachineClassForMigrationRequest{
						ProviderSpecificMachineClass: &v1alpha1.AWSMachineClass{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-mc",
								Labels: map[string]string{
									"key1": "value1",
									"key2": "value2",
								},
								Annotations: map[string]string{
									"key1": "value1",
									"key2": "value2",
								},
								Finalizers: []string{
									"mcm/finalizer",
								},
							},
							TypeMeta: v1.TypeMeta{},
							Spec: v1alpha1.AWSMachineClassSpec{
								AMI: "ami-123",
								BlockDevices: []v1alpha1.AWSBlockDeviceMappingSpec{
									{
										DeviceName: "bd-1",
										Ebs: v1alpha1.AWSEbsBlockDeviceSpec{
											DeleteOnTermination: aws.Bool(true),
											Encrypted:           false,
											Iops:                50,
											KmsKeyID:            aws.String("kms-123"),
											SnapshotID:          aws.String("snapid-123"),
											VolumeSize:          50,
											VolumeType:          "ebs",
										},
										NoDevice:    "bd-1",
										VirtualName: "bd-1",
									},
									{
										DeviceName: "bd-2",
										Ebs: v1alpha1.AWSEbsBlockDeviceSpec{
											DeleteOnTermination: aws.Bool(true),
											Encrypted:           false,
											Iops:                50,
											KmsKeyID:            aws.String("kms-123"),
											SnapshotID:          aws.String("snapid-123"),
											VolumeSize:          50,
											VolumeType:          "ebs",
										},
										NoDevice:    "bd-2",
										VirtualName: "bd-2",
									},
								},
								CredentialsSecretRef: &corev1.SecretReference{
									Name:      "test-credentials",
									Namespace: "test-namespace",
								},
								EbsOptimized: true,
								IAM: v1alpha1.AWSIAMProfileSpec{
									ARN:  "arn-123",
									Name: "name-123",
								},
								MachineType: "x-large",
								KeyName:     "keyname-123",
								Monitoring:  false,
								NetworkInterfaces: []v1alpha1.AWSNetworkInterfaceSpec{
									{
										AssociatePublicIPAddress: aws.Bool(false),
										DeleteOnTermination:      aws.Bool(true),
										Description:              aws.String("description-123"),
										SecurityGroupIDs: []string{
											"sg-1",
											"sg-2",
										},
										SubnetID: "test-subnet-id",
									},
								},
								Region:    "region-123",
								SpotPrice: aws.String("500"),
								SecretRef: &corev1.SecretReference{
									Name:      "test-secret",
									Namespace: "test-namespace",
								},
								Tags: map[string]string{
									"key1": "value1",
									"key2": "value2",
								},
							},
						},
						MachineClass: &v1alpha1.MachineClass{},
						ClassSpec: &v1alpha1.ClassSpec{
							//APIGroup: "",
							Kind: AWSMachineClassKind,
							Name: "test-mc",
						},
					},
				},
				expect: expect{
					machineClass: &v1alpha1.MachineClass{
						TypeMeta: v1.TypeMeta{},
						ObjectMeta: v1.ObjectMeta{
							Name: "test-mc",
							Labels: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Annotations: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
							Finalizers: []string{
								"mcm/finalizer",
							},
						},
						ProviderSpec: runtime.RawExtension{
							Raw: []byte("{\"apiVersion\":\"mcm.gardener.cloud/v1alpha1\",\"ami\":\"ami-123\",\"blockDevices\":[{\"deviceName\":\"bd-1\",\"ebs\":{\"deleteOnTermination\":true,\"iops\":50,\"kmsKeyID\":\"kms-123\",\"snapshotID\":\"snapid-123\",\"volumeSize\":50,\"volumeType\":\"ebs\"},\"noDevice\":\"bd-1\",\"virtualName\":\"bd-1\"},{\"deviceName\":\"bd-2\",\"ebs\":{\"deleteOnTermination\":true,\"iops\":50,\"kmsKeyID\":\"kms-123\",\"snapshotID\":\"snapid-123\",\"volumeSize\":50,\"volumeType\":\"ebs\"},\"noDevice\":\"bd-2\",\"virtualName\":\"bd-2\"}],\"ebsOptimized\":true,\"iam\":{\"arn\":\"arn-123\",\"name\":\"name-123\"},\"machineType\":\"x-large\",\"keyName\":\"keyname-123\",\"networkInterfaces\":[{\"associatePublicIPAddress\":false,\"deleteOnTermination\":true,\"description\":\"description-123\",\"securityGroupIDs\":[\"sg-1\",\"sg-2\"],\"subnetID\":\"test-subnet-id\"}],\"region\":\"region-123\",\"spotPrice\":\"500\",\"tags\":{\"key1\":\"value1\",\"key2\":\"value2\"}}"),
						},
						SecretRef: &corev1.SecretReference{
							Name:      "test-secret",
							Namespace: "test-namespace",
						},
						CredentialsSecretRef: &corev1.SecretReference{
							Name:      "test-credentials",
							Namespace: "test-namespace",
						},
						Provider: ProviderAWS,
					},
				},
			}),
		)
	})

})
