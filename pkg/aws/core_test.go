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

/*
TODO to adopt this
import (
	"context"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/mockclient"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
						Machine:      newMachine(),
						ProviderSpec: newMachineClass(providerSpec),
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
		)
	})

})

Entry("Machine creation request with volume type io1", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"io1\",\"iops\":50}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
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
						MachineName:  "test",
						ProviderSpec: []byte(""),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = unexpected end of JSON input",
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerSecretAccessKey": []byte("dummy-secret"),
							"userData":                []byte("dummy-user-data"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: false, \nProviderSecretAccessKey: true, \nUserData: true",
				},
			}),
			Entry("providerSecretAccessKey missing for provider secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerAccessKeyId": []byte("dummy-id"),
							"userData":            []byte("dummy-user-data"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: true, \nProviderSecretAccessKey: false, \nUserData: true",
				},
			}),
			Entry("userData missing for provider secret", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: true, \nProviderSecretAccessKey: true, \nUserData: false",
				},
			}),
			Entry("Validation for providerSpec fails. Missing AMI & Region.", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: []byte("{\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Error while validating ProviderSpec [AMI is required field Region is required field]",
				},
			}),
			Entry("Invalid region that doesn't exist", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"" + mockclient.FailAtRegion + "\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Region doesn't exist while trying to create session",
				},
			}),
			Entry("Invalid image ID that doesn't exist", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: []byte("{\"ami\":\"" + mockclient.FailQueryAtDescribeImages + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Couldn't find image with given ID",
				},
			}),
			Entry("Name tag cannot be set on AWS instances", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\",\"Name\":\"dummy\"}}"),
						Secrets:      providerSecret,
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
						MachineName:  "test",
						ProviderSpec: []byte("{\"ami\":\"" + mockclient.FailQueryAtRunInstances + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Couldn't run instance with given ID",
				},
			}),

		)
	})


	Describe("#DeleteMachine", func() {
		type setup struct {
			createMachineRequest *driver.CreateMachineRequest
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
				p := NewPlugin("tcp://127.0.0.1:8080")
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachinePlugin(p, mockPluginSPIImpl)

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
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						MachineName:  "test",
						Secrets:      providerSecret,
						ProviderSpec: providerSpec,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: false, \nProviderSecretAccessKey: true",
				},
			}),
			Entry("providerSecretAccessKey missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerAccessKeyId": []byte("dummy-id"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: true, \nProviderSecretAccessKey: false",
				},
			}),
			Entry("Termination of instance that doesn't exist on provider", &data{
				setup: setup{},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						MachineName:  "test",
						Secrets:      providerSecret,
						ProviderSpec: providerSpec,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     true,
					errMessage:            "rpc error: code = NotFound desc = AWS plugin is returning no VM instances backing this machine object",
				},
			}),
			Entry("Termination of instance that doesn't exist on provider", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: []byte("{\"ami\":\"" + mockclient.SetInstanceID + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"" + mockclient.FailQueryAtTerminateInstances + "\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						MachineName:  "test",
						ProviderSpec: []byte("{\"ami\":\"" + mockclient.SetInstanceID + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"" + mockclient.FailQueryAtTerminateInstances + "\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = InvalidInstanceID.Malformed: \ncaused by: Termination of instance errorred out",
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
				p := NewPlugin("tcp://127.0.0.1:8080")
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachinePlugin(p, mockPluginSPIImpl)

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
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				expect: expect{},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerSecretAccessKey": []byte("dummy-key"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: false, \nProviderSecretAccessKey: true",
				},
			}),
			Entry("providerSecretAccessKey missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						MachineName:  "test",
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerAccessKeyId": []byte("dummy-key"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: true, \nProviderSecretAccessKey: false",
				},
			}),

				TODO: Try to incorporate these changes if feasible
				Entry("Provider-ID is an invalid format", &data{
					setup: setup{
						createMachineRequest: &driver.CreateMachineRequest{
							MachineName:  "test",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
					},
					action: action{
						getMachineRequest: &driver.GetMachineStatusRequest{
							//ProviderID: "aws:eu-west-1:i-0123456789-0",
							MachineName: "test",
							Secrets:     providerSecret,
						},
					},
					expect: expect{
						errToHaveOccurred: true,
						errMessage:        "rpc error: code = Internal desc = Unable to decode provider-ID",
					},
				}),
				Entry("Region doesn't exist", &data{
					setup: setup{
						createMachineRequest: &driver.CreateMachineRequest{
							MachineName:  "test",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
					},
					action: action{
						getMachineRequest: &driver.GetMachineStatusRequest{
							//ProviderID: "aws:///" + mockclient.FailAtRegion + "/i-0123456789-0",
							MachineName: "test",
							Secrets:     providerSecret,
						},
					},
					expect: expect{
						errToHaveOccurred: true,
						errMessage:        "rpc error: code = Internal desc = Region doesn't exist while trying to create session",
					},
				}),
				Entry("Get machine of non-existant instance fails", &data{
					setup: setup{
						createMachineRequest: &driver.CreateMachineRequest{
							MachineName:  "test",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
					},
					action: action{
						getMachineRequest: &driver.GetMachineStatusRequest{
							//ProviderID: "aws:///eu-west-1/i-not-found",
							MachineName: "test",
							Secrets:     providerSecret,
						},
					},
					expect: expect{
						errToHaveOccurred: true,
						errMessage:        "rpc error: code = Internal desc = Couldn't find any instance matching requirement",
					},
				}),
				Entry("Return of empty list of machines for Get", &data{
					setup: setup{
						createMachineRequest: &driver.CreateMachineRequest{
							MachineName:  "test",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
					},
					action: action{
						getMachineRequest: &driver.GetMachineStatusRequest{
							//ProviderID: "aws:///eu-west-1/" + mockclient.ReturnEmptyListAtDescribeInstances,
							MachineName: "test",
							Secrets:     providerSecret,
						},
					},
					expect: expect{
						errToHaveOccurred: false,
						//getMachineResponse: &driver.GetMachineStatusResponse{
						//Exists: false,
						//},
					},
				}),
				Entry("Get request without a create request", &data{
					setup: setup{},
					action: action{
						getMachineRequest: &driver.GetMachineStatusRequest{
							MachineName: "test",
							//ProviderID: "aws:///eu-west-1/i-0123456789-0",
							Secrets: providerSecret,
						},
					},
					expect: expect{
						getMachineResponse: &driver.GetMachineStatusResponse{},
						errToHaveOccurred:  true,
						errMessage:         "rpc error: code = Internal desc = Couldn't find any instance matching requirement",
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
				p := NewPlugin("tcp://127.0.0.1:8080")
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachinePlugin(p, mockPluginSPIImpl)

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
				}
			},
			Entry("Simple Machine List Request", &data{
				setup: setup{
					createMachineRequest: []*driver.CreateMachineRequest{
						&driver.CreateMachineRequest{
							MachineName:  "test-0",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
						&driver.CreateMachineRequest{
							MachineName:  "test-1",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
						&driver.CreateMachineRequest{
							MachineName:  "test-2",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
					},
				},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					listMachineResponse: &driver.ListMachinesResponse{
						MachineList: map[string]string{
							"aws:///eu-west-1/i-0123456789-0": "test-0",
							"aws:///eu-west-1/i-0123456789-1": "test-1",
							"aws:///eu-west-1/i-0123456789-2": "test-2",
						},
					},
				},
			}),
			Entry("Unexpected end of JSON input", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: []byte(""),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = unexpected end of JSON input",
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerSecretAccessKey": []byte("dummy-secret"),
							"userData":                []byte("dummy-user-data"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: false, \nProviderSecretAccessKey: true, \nUserData: true",
				},
			}),
			Entry("providerSecretAccessKey missing for secret", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerAccessKeyId": []byte("dummy-id"),
							"userData":            []byte("dummy-user-data"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: true, \nProviderSecretAccessKey: false, \nUserData: true",
				},
			}),
			Entry("userData missing for secret", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: providerSpec,
						Secrets: map[string][]byte{
							"providerAccessKeyId":     []byte("dummy-id"),
							"providerSecretAccessKey": []byte("dummy-secret"),
						},
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Invalidate Secret Map. Map variables present \nProviderAccessKeyID: true, \nProviderSecretAccessKey: true, \nUserData: false",
				},
			}),
			Entry("Validation for providerSpec fails. Missing AMI & Region.", &data{
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: []byte("{\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Error while validating ProviderSpec [AMI is required field Region is required field]",
				},
			}),
			Entry("Region doesn't exist", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"" + mockclient.FailAtRegion + "\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Region doesn't exist while trying to create session",
				},
			}),
			Entry("Cluster details missing in machine class", &data{
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Error while validating ProviderSpec [Tag is required of the form kubernetes.io/cluster/****]",
				},
			}),
			Entry("Cloud provider returned error while describing instance", &data{
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/" + mockclient.ReturnErrorAtDescribeInstances + "\":\"1\",\"kubernetes.io/role/test\":\"1\"}}"),
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = Cloud provider returned error",
				},
			}),
			Entry("List request without a create request", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
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
				p := NewPlugin("tcp://127.0.0.1:8080")
				mockPluginSPIImpl := &mockclient.MockPluginSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachinePlugin(p, mockPluginSPIImpl)

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
			Entry("Simple GetVolumeIDs request", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecList: []byte("[{\"capacity\":{\"storage\":\"1Gi\"},\"awsElasticBlockStore\":{\"volumeID\":\"aws://eu-east-2b/vol-xxxxyyyyzzzz11112\",\"fsType\":\"ext4\"},\"accessModes\":[\"ReadWriteOnce\"],\"claimRef\":{\"kind\":\"PersistentVolumeClaim\",\"namespace\":\"default\",\"name\":\"www-web-0\",\"uid\":\"0c3b34f8-a494-11e9-b4c3-0e956a869a31\",\"apiVersion\":\"v1\",\"resourceVersion\":\"32423232\"},\"persistentVolumeReclaimPolicy\":\"Delete\",\"storageClassName\":\"default\",\"nodeAffinity\":{\"required\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"failure-domain.beta.kubernetes.io/zone\",\"operator\":\"In\",\"values\":[\"eu-east-2b\"]},{\"key\":\"failure-domain.beta.kubernetes.io/region\",\"operator\":\"In\",\"values\":[\"eu-east-2\"]}]}]}}}]"),
					},
				},
				expect: expect{
					getListOfVolumeIDsForExistingPVsResponse: &driver.GetVolumeIDsResponse{
						VolumeIDs: []string{
							"aws://eu-east-2b/vol-xxxxyyyyzzzz11112",
						},
					},
				},
			}),
			Entry("GetVolumeIDs with multiple pvSpecs request", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecList: []byte("[{\"capacity\":{\"storage\":\"1Gi\"},\"awsElasticBlockStore\":{\"volumeID\":\"aws://eu-east-2b/vol-xxxxyyyyzzzz11112\",\"fsType\":\"ext4\"},\"accessModes\":[\"ReadWriteOnce\"],\"claimRef\":{\"kind\":\"PersistentVolumeClaim\",\"namespace\":\"default\",\"name\":\"www-web-0\",\"uid\":\"0c3b34f8-a494-11e9-b4c3-0e956a869a31\",\"apiVersion\":\"v1\",\"resourceVersion\":\"32423232\"},\"persistentVolumeReclaimPolicy\":\"Delete\",\"storageClassName\":\"default\",\"nodeAffinity\":{\"required\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"failure-domain.beta.kubernetes.io/zone\",\"operator\":\"In\",\"values\":[\"eu-east-2b\"]},{\"key\":\"failure-domain.beta.kubernetes.io/region\",\"operator\":\"In\",\"values\":[\"eu-east-2\"]}]}]}}},{\"capacity\":{\"storage\":\"1Gi\"},\"awsElasticBlockStore\":{\"volumeID\":\"aws://eu-east-2b/vol-xxxxyyyyzzzz11113\",\"fsType\":\"ext4\"},\"accessModes\":[\"ReadWriteOnce\"],\"claimRef\":{\"kind\":\"PersistentVolumeClaim\",\"namespace\":\"default\",\"name\":\"www-web-1\",\"uid\":\"0c3b34f8-a494-11e9-b4c3-0e956a869a31\",\"apiVersion\":\"v1\",\"resourceVersion\":\"32423232\"},\"persistentVolumeReclaimPolicy\":\"Delete\",\"storageClassName\":\"default\",\"nodeAffinity\":{\"required\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"failure-domain.beta.kubernetes.io/zone\",\"operator\":\"In\",\"values\":[\"eu-east-2b\"]},{\"key\":\"failure-domain.beta.kubernetes.io/region\",\"operator\":\"In\",\"values\":[\"eu-east-2\"]}]}]}}}]"),
					},
				},
				expect: expect{
					getListOfVolumeIDsForExistingPVsResponse: &driver.GetVolumeIDsResponse{
						VolumeIDs: []string{
							"aws://eu-east-2b/vol-xxxxyyyyzzzz11112",
							"aws://eu-east-2b/vol-xxxxyyyyzzzz11113",
						},
					},
				},
			}),
			Entry("GetVolumeIDs for Azure pvSpecs request", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecList: []byte("[{\"capacity\":{\"storage\":\"1Gi\"},\"azureDisk\":{\"volumeID\":\"aws://eu-east-2b/vol-xxxxyyyyzzzz11112\",\"fsType\":\"ext4\"},\"accessModes\":[\"ReadWriteOnce\"],\"claimRef\":{\"kind\":\"PersistentVolumeClaim\",\"namespace\":\"default\",\"name\":\"www-web-0\",\"uid\":\"0c3b34f8-a494-11e9-b4c3-0e956a869a31\",\"apiVersion\":\"v1\",\"resourceVersion\":\"32423232\"},\"persistentVolumeReclaimPolicy\":\"Delete\",\"storageClassName\":\"default\",\"nodeAffinity\":{\"required\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"failure-domain.beta.kubernetes.io/zone\",\"operator\":\"In\",\"values\":[\"eu-east-2b\"]},{\"key\":\"failure-domain.beta.kubernetes.io/region\",\"operator\":\"In\",\"values\":[\"eu-east-2\"]}]}]}}}]"),
					},
				},
				expect: expect{
					getListOfVolumeIDsForExistingPVsResponse: &driver.GetVolumeIDsResponse{},
				},
			}),
			Entry("GetVolumeIDs for invalid json input", &data{
				action: action{
					getListOfVolumeIDsForExistingPVsRequest: &driver.GetVolumeIDsRequest{
						PVSpecList: []byte("[{\"capacity\":{\"storage\":\"1Gi\"},\"awsElasticBlockStore\":{\"volumeID\":\"aws://eu-east-2b/vol-xxxxyyyyzzzz11112\",\"fsType\":\"ext4\"},\"accessModes\":[\"ReadWriteOnce\"],\"claimRef\":{\"kind\":\"PersistentVolumeClaim\",\"namespace\":\"default\",\"name\":\"www-web-0\",\"uid\":\"0c3b34f8-a494-11e9-b4c3-0e956a869a31\",\"apiVersion\":\"v1\",\"resourceVersion\":\"32423232\"},\"persistentVolumeReclaimPolicy\":\"Delete\",\"storageClassName\":\"default\",\"nodeAffinity\":{\"required\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"failure-domain.beta.kubernetes.io/zone\",\"operator\":\"In\",\"values\":[\"eu-east-2b\"]},{\"key\":\"failure-domain.beta.kubernetes.io/region\",\"operator\":\"In\"\"values\":[\"eu-east-2\"]}]}]}}}]"),
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "rpc error: code = Internal desc = invalid character '\"' after object key:value pair",
				},
			}),
*/
