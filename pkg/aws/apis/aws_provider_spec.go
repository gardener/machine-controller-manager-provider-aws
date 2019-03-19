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

package api

//AWSProviderSpec is the spec to be used while parsing the gRPC calls.
type AWSProviderSpec struct {
	//TODO: figure out a better way to present objectmeta and typemeta without importing k8s.io
	APIVersion        string                      `json:"apiVersion,omitempty"`
	AMI               string                      `json:"ami,omitempty"`
	BlockDevices      []AWSBlockDeviceMappingSpec `json:"blockDevices,omitempty"`
	EbsOptimized      bool                        `json:"ebsOptimized,omitempty"`
	IAM               AWSIAMProfileSpec           `json:"iam,omitempty"`
	MachineType       string                      `json:"machineType,omitempty"`
	KeyName           string                      `json:"keyName,omitempty"`
	Monitoring        bool                        `json:"monitoring,omitempty"`
	NetworkInterfaces []AWSNetworkInterfaceSpec   `json:"networkInterfaces,omitempty"`
	Region            string                      `json:"region,omitempty"`
	Tags              map[string]string           `json:"tags,omitempty"`
}

// AWSBlockDeviceMappingSpec stores info about AWS block device mappings
type AWSBlockDeviceMappingSpec struct {

	// The device name exposed to the machine (for example, /dev/sdh or xvdh).
	DeviceName string `json:"deviceName,omitempty"`

	// Parameters used to automatically set up EBS volumes when the machine is
	// launched.
	Ebs AWSEbsBlockDeviceSpec `json:"ebs,omitempty"`

	// Suppresses the specified device included in the block device mapping of the
	// AMI.
	NoDevice string `json:"noDevice,omitempty"`

	// The virtual device name (ephemeralN). Machine store volumes are numbered
	// starting from 0. An machine type with 2 available machine store volumes
	// can specify mappings for ephemeral0 and ephemeral1.The number of available
	// machine store volumes depends on the machine type. After you connect to
	// the machine, you must mount the volume.
	//
	// Constraints: For M3 machines, you must specify machine store volumes in
	// the block device mapping for the machine. When you launch an M3 machine,
	// we ignore any machine store volumes specified in the block device mapping
	// for the AMI.
	VirtualName string `json:"virtualName,omitempty"`
}

// AWSEbsBlockDeviceSpec describes a block device for an EBS volume.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/ec2-2016-11-15/EbsBlockDevice
type AWSEbsBlockDeviceSpec struct {

	// Indicates whether the EBS volume is deleted on machine termination.
	DeleteOnTermination bool `json:"deleteOnTermination,omitempty"`

	// Indicates whether the EBS volume is encrypted. Encrypted Amazon EBS volumes
	// may only be attached to machines that support Amazon EBS encryption.
	Encrypted bool `json:"encrypted,omitempty"`

	// The number of I/O operations per second (IOPS) that the volume supports.
	// For io1, this represents the number of IOPS that are provisioned for the
	// volume. For gp2, this represents the baseline performance of the volume and
	// the rate at which the volume accumulates I/O credits for bursting. For more
	// information about General Purpose SSD baseline performance, I/O credits,
	// and bursting, see Amazon EBS Volume Types (http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSVolumeTypes.html)
	// in the Amazon Elastic Compute Cloud User Guide.
	//
	// Constraint: Range is 100-20000 IOPS for io1 volumes and 100-10000 IOPS for
	// gp2 volumes.
	//
	// Condition: This parameter is required for requests to create io1 volumes;
	// it is not used in requests to create gp2, st1, sc1, or standard volumes.
	Iops int64 `json:"iops,omitempty"`

	// The size of the volume, in GiB.
	//
	// Constraints: 1-16384 for General Purpose SSD (gp2), 4-16384 for Provisioned
	// IOPS SSD (io1), 500-16384 for Throughput Optimized HDD (st1), 500-16384 for
	// Cold HDD (sc1), and 1-1024 for Magnetic (standard) volumes. If you specify
	// a snapshot, the volume size must be equal to or larger than the snapshot
	// size.
	//
	// Default: If you're creating the volume from a snapshot and don't specify
	// a volume size, the default is the snapshot size.
	VolumeSize int64 `json:"volumeSize,omitempty"`

	// The volume type: gp2, io1, st1, sc1, or standard.
	//
	// Default: standard
	VolumeType string `json:"volumeType,omitempty"`
}

// AWSIAMProfileSpec describes an IAM machine profile.
type AWSIAMProfileSpec struct {
	// The Amazon Resource Name (ARN) of the machine profile.
	ARN string `json:"arn,omitempty"`

	// The name of the machine profile.
	Name string `json:"name,omitempty"`
}

// AWSNetworkInterfaceSpec describes a network interface.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/ec2-2016-11-15/MachineAWSNetworkInterfaceSpecification
type AWSNetworkInterfaceSpec struct {

	// Indicates whether to assign a public IPv4 address to an machine you launch
	// in a VPC. The public IP address can only be assigned to a network interface
	// for eth0, and can only be assigned to a new network interface, not an existing
	// one. You cannot specify more than one network interface in the request. If
	// launching into a default subnet, the default value is true.
	AssociatePublicIPAddress bool `json:"associatePublicIPAddress,omitempty"`

	// If set to true, the interface is deleted when the machine is terminated.
	// You can specify true only if creating a new network interface when launching
	// an machine.
	DeleteOnTermination bool `json:"deleteOnTermination,omitempty"`

	// The description of the network interface. Applies only if creating a network
	// interface when launching an machine.
	Description string `json:"description,omitempty"`

	// The IDs of the security groups for the network interface. Applies only if
	// creating a network interface when launching an machine.
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`

	// The ID of the subnet associated with the network string. Applies only if
	// creating a network interface when launching an machine.
	SubnetID string `json:"subnetID,omitempty"`
}

// Secrets stores the cloud-provider specific sensitive-information.
type Secrets struct {
	// AWS access key id (base64 encoded)
	ProviderAccessKeyID string `json:"providerAccessKeyId,omitempty"`
	// AWS secret access key (base64 encoded)
	ProviderSecretAccessKey string `json:"providerSecretAccessKey,omitempty"`
	// AWS cloud config file (base64 encoded)
	UserData string `json:"userData,omitempty"`
}
