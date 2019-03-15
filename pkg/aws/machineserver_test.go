package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/mockclient"
	cmipb "github.com/gardener/machine-spec/lib/go/cmi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("MachineServer", func() {

	// Some initializations
	providerSpec := []byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")
	providerSecret := map[string][]byte{
		"providerAccessKeyId":     []byte("dummy-id"),
		"providerSecretAccessKey": []byte("dummy-secret"),
		"userData":                []byte("dummy-user-data"),
	}

	Describe("#CreateMachine", func() {
		type setup struct {
		}
		type action struct {
			machineRequest *cmipb.CreateMachineRequest
		}
		type expect struct {
			machineResponse   *cmipb.CreateMachineResponse
			errToHaveOccurred bool
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				d := NewDriver("tcp://127.0.0.1:8080")
				mockDriverSPIImpl := &mockclient.MockDriverSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachineServer(d, mockDriverSPIImpl)

				ctx := context.Background()
				response, err := ms.CreateMachine(ctx, data.action.machineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(data.expect.machineResponse.MachineID).To(Equal(response.MachineID))
					Expect(data.expect.machineResponse.NodeName).To(Equal(response.NodeName))
				}
			},
			Entry("Simple Machine Creation Request", &data{
				action: action{
					machineRequest: &cmipb.CreateMachineRequest{
						Name:         "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					machineResponse: &cmipb.CreateMachineResponse{
						MachineID: "aws:///eu-west-1/i-0123456789-0",
						NodeName:  "ip-0",
					},
					errToHaveOccurred: false,
				},
			}),
			// TODO add more tests
		)
	})

	Describe("#GetMachine", func() {
		type setup struct {
			createMachineRequest *cmipb.CreateMachineRequest
		}
		type action struct {
			getMachineRequest *cmipb.GetMachineRequest
		}
		type expect struct {
			getMachineResponse *cmipb.GetMachineResponse
			errToHaveOccurred  bool
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				d := NewDriver("tcp://127.0.0.1:8080")
				mockDriverSPIImpl := &mockclient.MockDriverSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachineServer(d, mockDriverSPIImpl)

				ctx := context.Background()
				createResponse, err := ms.CreateMachine(ctx, data.setup.createMachineRequest)
				Expect(err).ToNot(HaveOccurred())

				getResponse, err := ms.GetMachine(ctx, &cmipb.GetMachineRequest{
					MachineID: createResponse.MachineID,
					Secrets:   providerSecret,
				})

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(data.expect.getMachineResponse.Exists).To(Equal(getResponse.Exists))
				}
			},
			Entry("Simple Machine Get Request", &data{
				setup: setup{
					createMachineRequest: &cmipb.CreateMachineRequest{
						Name:         "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					getMachineResponse: &cmipb.GetMachineResponse{
						Exists: true,
					},
				},
			}),
			// TODO add more tests
		)
	})

	Describe("#DeleteMachine", func() {
		type setup struct {
			createMachineRequest *cmipb.CreateMachineRequest
		}
		type action struct {
			deleteMachineRequest *cmipb.DeleteMachineRequest
		}
		type expect struct {
			deleteMachineResponse *cmipb.DeleteMachineResponse
			errToHaveOccurred     bool
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				d := NewDriver("tcp://127.0.0.1:8080")
				mockDriverSPIImpl := &mockclient.MockDriverSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachineServer(d, mockDriverSPIImpl)

				ctx := context.Background()
				createResponse, err := ms.CreateMachine(ctx, data.setup.createMachineRequest)
				Expect(err).ToNot(HaveOccurred())

				_, err = ms.DeleteMachine(ctx, &cmipb.DeleteMachineRequest{
					MachineID: createResponse.MachineID,
					Secrets:   providerSecret,
				})

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("Simple Machine Delete Request", &data{
				setup: setup{
					createMachineRequest: &cmipb.CreateMachineRequest{
						Name:         "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &cmipb.DeleteMachineResponse{},
				},
			}),
			// TODO add more tests
		)
	})

	Describe("#ShutDownMachine", func() {
		type setup struct {
			createMachineRequest *cmipb.CreateMachineRequest
		}
		type action struct {
			shutDownMachineRequest *cmipb.ShutDownMachineRequest
		}
		type expect struct {
			shutDownMachineResponse *cmipb.ShutDownMachineResponse
			errToHaveOccurred       bool
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				d := NewDriver("tcp://127.0.0.1:8080")
				mockDriverSPIImpl := &mockclient.MockDriverSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachineServer(d, mockDriverSPIImpl)

				ctx := context.Background()
				createResponse, err := ms.CreateMachine(ctx, data.setup.createMachineRequest)
				Expect(err).ToNot(HaveOccurred())

				_, err = ms.ShutDownMachine(ctx, &cmipb.ShutDownMachineRequest{
					MachineID: createResponse.MachineID,
					Secrets:   providerSecret,
				})

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("Simple Machine Shutdown Request", &data{
				setup: setup{
					createMachineRequest: &cmipb.CreateMachineRequest{
						Name:         "test",
						ProviderSpec: providerSpec,
						Secrets:      providerSecret,
					},
				},
				expect: expect{
					shutDownMachineResponse: &cmipb.ShutDownMachineResponse{},
				},
			}),
			// TODO add more tests
		)
	})

	Describe("#ListMachines", func() {
		type setup struct {
			createMachineRequest []*cmipb.CreateMachineRequest
		}
		type action struct {
			listMachineRequest *cmipb.ListMachinesRequest
		}
		type expect struct {
			listMachineResponse *cmipb.ListMachinesResponse
			errToHaveOccurred   bool
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				d := NewDriver("tcp://127.0.0.1:8080")
				mockDriverSPIImpl := &mockclient.MockDriverSPIImpl{FakeInstances: make([]ec2.Instance, 0)}
				ms := NewMachineServer(d, mockDriverSPIImpl)

				ctx := context.Background()

				for _, createReq := range data.setup.createMachineRequest {
					_, err := ms.CreateMachine(ctx, createReq)
					Expect(err).ToNot(HaveOccurred())
				}

				listResponse, err := ms.ListMachines(ctx, &cmipb.ListMachinesRequest{
					ProviderSpec: data.setup.createMachineRequest[0].ProviderSpec,
					Secrets:      providerSecret,
				})

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(len(listResponse.MachineList)).To(Equal(3))

				}
			},
			Entry("Simple Machine List Request", &data{
				setup: setup{
					createMachineRequest: []*cmipb.CreateMachineRequest{
						&cmipb.CreateMachineRequest{
							Name:         "test-0",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
						&cmipb.CreateMachineRequest{
							Name:         "test-1",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
						&cmipb.CreateMachineRequest{
							Name:         "test-2",
							ProviderSpec: providerSpec,
							Secrets:      providerSecret,
						},
					},
				},
			}),
			// TODO add more tests
		)
	})
})
