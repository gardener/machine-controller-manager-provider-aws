// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"

	"github.com/gardener/machine-controller-manager-provider-aws/pkg/mockclient"
)

const (
	awsAccessKeyIDIsMissing               = "machine codes error: code = [InvalidArgument] message = [Error while validating ProviderSpec secretRef.AWSAccessKeyID: Required value: Mention atleast providerAccessKeyId or accessKeyID]"
	awsSecretAccessKeyIsMissing           = "machine codes error: code = [InvalidArgument] message = [Error while validating ProviderSpec secretRef.AWSSecretAccessKey: Required value: Mention atleast providerSecretAccessKey or secretAccessKey]"
	awsSecretAccessKeyNUserDataAreMissing = "machine codes error: code = [InvalidArgument] message = [Error while validating ProviderSpec [secretRef.AWSSecretAccessKey: Required value: Mention atleast providerSecretAccessKey or secretAccessKey, secretRef.userData: Required value: Mention userData]]"
	regionNAMIMissing                     = "machine codes error: code = [InvalidArgument] message = [Error while validating ProviderSpec [providerSpec.ami: Required value: AMI is required, providerSpec.region: Required value: Region is required]]"
	userDataIsMissing                     = "machine codes error: code = [InvalidArgument] message = [Error while validating ProviderSpec secretRef.userData: Required value: Mention userData]"
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
	annotations := map[string]string{
		awsPlacement: `{ "affinity": "host", "availabilityZone": "eu-west-1a", "tenancy": "host"}`,
	}

	Describe("#CreateMachine", func() {
		type setup struct {
			maxElapsedTimeForRetry time.Duration
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
				var temp time.Duration

				if data.setup.maxElapsedTimeForRetry != 0 {
					temp = maxElapsedTimeInBackoff
					maxElapsedTimeInBackoff = data.setup.maxElapsedTimeForRetry
				}
				response, err := ms.CreateMachine(ctx, data.action.machineRequest)

				maxElapsedTimeInBackoff = temp

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
				setup: setup{},
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
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
			Entry("Simple Machine Creation Request with missing provider in MachineClass", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClassWithProvider(providerSpec, "azure"),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [InvalidArgument] message = [Requested for Provider 'azure', we only support 'AWS']",
				},
			}),
			Entry("Machine creation request with IAM ARN", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"arn\":\"some-arn\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"capacityReservation\":{\"capacityReservationId\":\"cr-05c28b843c05abcde\",\"capacityReservationResourceGroupArn\":\"arn:aws:resource-groups:us-west-1:123456789012:group/my-test-cr-group\"},\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [InvalidArgument] message = [Error while validating ProviderSpec providerSpec.capacityReservation: Required value: CapacityReservationResourceGroupArn or CapacityReservationId are optional but only one should be used]",
				},
			}),
			Entry("Machine creation request for an AWS Capacity Reservation Group with capacityReservationPreference only", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"capacityReservation\":{\"capacityReservationPreference\":\"open\"},\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
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
			Entry("Machine creation request for capacity reservations with capacityReservationId", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"" + mockclient.FailAtRegion + "\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        fmt.Sprintf("machine codes error: code = [Internal] message = [%s]", mockclient.AWSInvalidRegionError),
				},
			}),
			Entry("Placement object with affinity, tenancy and availablityZone set", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, annotations),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
					machineResponse: &driver.CreateMachineResponse{
						ProviderID: "aws:///eu-west-1/i-0123456789-0/placement={affinity:host,availabilityZone:eu-west-1a,tenancy:host}",
						NodeName:   "ip-0",
					},
				},
			}),
			Entry("Invalid image ID that doesn't exist", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte(fmt.Sprintf("{\"ami\":\"%s\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}", mockclient.FailQueryAtDescribeImages))),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        fmt.Sprintf("machine codes error: code = [Internal] message = [%s]", mockclient.AWSImageNotFoundError),
				},
			}),
			Entry("Name tag cannot be set on AWS instances", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
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
			Entry("RunInstance call fails with error code as Internal", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte(fmt.Sprintf("{\"ami\":\"%s\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}", mockclient.FailQueryAtRunInstances))),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        fmt.Sprintf("machine codes error: code = [Internal] message = [%s]", mockclient.AWSInternalErrorForRunInstances),
				},
			}),
			Entry("RunInstance call fails with error code as InsufficientCapacity", &data{
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte(fmt.Sprintf("{\"ami\":\"%s\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"%s\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}", mockclient.FailQueryAtRunInstances, mockclient.InsufficientCapacity))),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        fmt.Sprintf("machine codes error: code = [%s] message = [%s]", codes.ResourceExhausted, mockclient.AWSInsufficientCapacityError),
				},
			}),
			Entry("Should Fail when APIs are not consistent for 10sec(in real situation its 5min)", &data{
				setup: setup{
					maxElapsedTimeForRetry: 10 * time.Second,
				},
				action: action{
					machineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte("{\"ami\":\"" + mockclient.SetInstanceID + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"" + mockclient.InconsistencyInAPIs + "\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [Internal] message = [creation of VM \"aws:///eu-west-1/i-instance-doesnt-exist\" failed, timed out waiting for eventual consistency. Multiple VMs backing machine obj might spawn, they will be orphan collected]",
				},
			}),
		)
	})

	Describe("#GetPlacementObject", func() {
		type setup struct {
			objectmeta v1.ObjectMeta
		}
		type action struct {
			createMachineRequest *driver.CreateMachineRequest
		}
		type expect struct {
			placementobj      *ec2.Placement
			errToHaveOccurred bool
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {
				data.action.createMachineRequest.Machine.Spec.NodeTemplateSpec.ObjectMeta.Annotations = data.setup.objectmeta.Annotations

				obj, err := getPlacementObj(data.action.createMachineRequest)

				if data.expect.errToHaveOccurred {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(obj).To(Equal(data.expect.placementobj))
				}
			},
			Entry("when Availability zone is set and affinity and tenancy set to host", &data{
				setup: setup{
					objectmeta: v1.ObjectMeta{
						Annotations: map[string]string{
							awsPlacement: `{ "affinity": "host", "availabilityZone": "eu-west-1a", "tenancy": "host"}`,
						},
					},
				},
				action: action{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
					placementobj: &ec2.Placement{
						AvailabilityZone: getStringPtr("eu-west-1a"),
						Affinity:         getStringPtr(ec2.AffinityHost),
						Tenancy:          getStringPtr(ec2.TenancyHost),
					},
				},
			}),
			Entry("when Availability zone is not set but affinity and tenancy set to host", &data{
				setup: setup{
					objectmeta: v1.ObjectMeta{
						Annotations: map[string]string{
							awsPlacement: `{ "affinity": "host", "tenancy": "host"}`,
						},
					},
				},
				action: action{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					placementobj: &ec2.Placement{
						Affinity: getStringPtr(ec2.AffinityHost),
						Tenancy:  getStringPtr(ec2.TenancyHost),
					},
				},
			}),
			Entry("when awsPlacement key is not there", &data{
				setup: setup{
					objectmeta: v1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
				action: action{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("when awsPlacement key is there but value not there", &data{
				setup: setup{
					objectmeta: v1.ObjectMeta{
						Annotations: map[string]string{
							awsPlacement: "",
						},
					},
				},
				action: action{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("when awsPlacement key and value is there but value is {} only", &data{
				setup: setup{
					objectmeta: v1.ObjectMeta{
						Annotations: map[string]string{
							awsPlacement: "{}",
						},
					},
				},
				action: action{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("when Availability zone is set and affinity and tenancy set to value default", &data{
				setup: setup{
					objectmeta: v1.ObjectMeta{
						Annotations: map[string]string{
							awsPlacement: `{ "affinity": "default", "availabilityZone": "eu-west-1a", "tenancy": "default"}`,
						},
					},
				},
				action: action{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
					placementobj: &ec2.Placement{
						Affinity:         getStringPtr(ec2.AffinityDefault),
						AvailabilityZone: getStringPtr("eu-west-1a"),
						Tenancy:          getStringPtr(ec2.TenancyDefault),
					},
				},
			}),
			Entry("when just hostId and partitionNumber is set", &data{
				setup: setup{
					objectmeta: v1.ObjectMeta{
						Annotations: map[string]string{
							awsPlacement: `{ "hostId": "h-0123b456af7f89123", "partitionNumber": 7}`,
						},
					},
				},
				action: action{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
					placementobj: &ec2.Placement{
						HostId:          getStringPtr("h-0123b456af7f89123"),
						PartitionNumber: getIntPtrForString("7"),
					},
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
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     false,
				},
			}),
			Entry("Simple Machine Delete Request with wrong provider in MachineClass", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClassWithProvider(providerSpec, "azure"),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [InvalidArgument] message = [Requested for Provider 'azure', we only support 'AWS']",
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0, nil),
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
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0, nil),
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
			Entry("Termination of instance that doesn't exist on provider but machine obj has providerID", &data{
				setup: setup{},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     false,
				},
			}),
			Entry("Another case for Termination of instance that doesn't exist on provider but machine obj has providerID", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte("{\"ami\":\"" + mockclient.SetInstanceID + "\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"" + mockclient.FailQueryAtTerminateInstances + "\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/shoot--test\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: false,
				},
			}),
			Entry("Termination of machine obj without providerID and backing instance", &data{
				setup: setup{},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(-1, nil),
						MachineClass: newMachineClass([]byte("{\"ami\":\"ami-123456789\",\"blockDevices\":[{\"ebs\":{\"volumeSize\":50,\"volumeType\":\"gp2\"}}],\"iam\":{\"name\":\"test-iam\"},\"keyName\":\"test-ssh-publickey\",\"machineType\":\"m4.large\",\"networkInterfaces\":[{\"securityGroupIDs\":[\"sg-00002132323\"],\"subnetID\":\"subnet-123456\"}],\"region\":\"eu-west-1\",\"tags\":{\"kubernetes.io/cluster/" + mockclient.ReturnErrorAtDescribeInstances + "\":\"1\",\"kubernetes.io/role/test\":\"1\"}}")),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					deleteMachineResponse: &driver.DeleteMachineResponse{},
					errToHaveOccurred:     true,
					errMessage:            fmt.Sprintf("machine codes error: code = [Internal] message = [%s]", mockclient.AWSInternalErrorForDescribeInstances),
				},
			}),
			Entry("Termination of machine with any backing instance but no providerID", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					deleteMachineRequest: &driver.DeleteMachineRequest{
						// Setting machineIndex to -1 to simulate no providerID
						Machine:      newMachine(-1, nil),
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

				// if there is a create machine request by the test case then create the machine
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
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				expect: expect{},
			}),
			Entry("Simple Machine Get Request with unsupported provider in MachineClass", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClassWithProvider(providerSpec, "azure"),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [InvalidArgument] message = [Requested for Provider 'azure', we only support 'AWS']",
				},
			}),
			Entry("providerAccessKeyId missing for secret", &data{
				setup: setup{
					createMachineRequest: &driver.CreateMachineRequest{
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0, nil),
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
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0, nil),
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
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(0, nil),
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
						Machine:      newMachine(0, nil),
						MachineClass: newMachineClass(providerSpec),
						Secret:       providerSecret,
					},
				},
				action: action{
					getMachineRequest: &driver.GetMachineStatusRequest{
						Machine:      newMachine(-1, nil),
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
						Machine:      newMachine(0, nil),
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
					// Expect(listResponse.MachineList).To(Equal(data.expect.listMachineResponse))
				}
			},
			Entry("Simple Machine List Request", &data{
				setup: setup{
					createMachineRequest: []*driver.CreateMachineRequest{
						{
							Machine:      newMachine(0, nil),
							MachineClass: newMachineClass(providerSpec),
							Secret:       providerSecret,
						},
						{
							Machine:      newMachine(1, nil),
							MachineClass: newMachineClass(providerSpec),
							Secret:       providerSecret,
						},
						{
							Machine:      newMachine(2, nil),
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

			Entry("List Machine Request with unsupported provider in MachineClass", &data{
				setup: setup{},
				action: action{
					listMachineRequest: &driver.ListMachinesRequest{
						MachineClass: newMachineClassWithProvider(providerSpec, "azure"),
						Secret:       providerSecret,
					},
				},
				expect: expect{
					errToHaveOccurred: true,
					errMessage:        "machine codes error: code = [InvalidArgument] message = [Requested for Provider 'azure', we only support 'AWS']",
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
					errMessage:        fmt.Sprintf("machine codes error: code = [Internal] message = [%s]", mockclient.AWSInvalidRegionError),
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
					errMessage:        "machine codes error: code = [InvalidArgument] message = [Error while validating ProviderSpec providerSpec.tags[]: Required value: Tag required of the form kubernetes.io/cluster/****]",
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
					errMessage:        fmt.Sprintf("machine codes error: code = [Internal] message = [%s]", mockclient.AWSInternalErrorForDescribeInstances),
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
})
