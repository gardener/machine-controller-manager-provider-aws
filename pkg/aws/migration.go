/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.

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

// Package aws contains the cloud provider specific implementations to manage machines
package aws

import (
	"encoding/json"

	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// ProviderAWS string const to identify AWS provider
	ProviderAWS = "AWS"
)

// fillUpMachineClass copies over the fields from ProviderMachineClass to MachineClass
func fillUpMachineClass(awsMachineClass *v1alpha1.AWSMachineClass, machineClass *v1alpha1.MachineClass) error {

	// Prepare the IAM struct
	iam := api.AWSIAMProfileSpec{
		ARN:  awsMachineClass.Spec.IAM.ARN,
		Name: awsMachineClass.Spec.IAM.Name,
	}

	// Prepare the providerSpec struct
	providerSpec := &api.AWSProviderSpec{
		APIVersion:        api.V1alpha1,
		AMI:               awsMachineClass.Spec.AMI,
		BlockDevices:      []api.AWSBlockDeviceMappingSpec{},
		EbsOptimized:      awsMachineClass.Spec.EbsOptimized,
		IAM:               iam,
		KeyName:           &awsMachineClass.Spec.KeyName,
		MachineType:       awsMachineClass.Spec.MachineType,
		Monitoring:        awsMachineClass.Spec.Monitoring,
		NetworkInterfaces: []api.AWSNetworkInterfaceSpec{},
		Region:            awsMachineClass.Spec.Region,
		Tags:              awsMachineClass.Spec.Tags,
		SpotPrice:         awsMachineClass.Spec.SpotPrice,
	}

	// Add BlockDevices
	for _, awsBlockDevice := range awsMachineClass.Spec.BlockDevices {
		blockDevice := api.AWSBlockDeviceMappingSpec{
			DeviceName: awsBlockDevice.DeviceName,
			Ebs: api.AWSEbsBlockDeviceSpec{
				DeleteOnTermination: awsBlockDevice.Ebs.DeleteOnTermination,
				Encrypted:           awsBlockDevice.Ebs.Encrypted,
				Iops:                awsBlockDevice.Ebs.Iops,
				KmsKeyID:            awsBlockDevice.Ebs.KmsKeyID,
				SnapshotID:          awsBlockDevice.Ebs.SnapshotID,
				VolumeSize:          awsBlockDevice.Ebs.VolumeSize,
				VolumeType:          awsBlockDevice.Ebs.VolumeType,
			},
			NoDevice:    awsBlockDevice.NoDevice,
			VirtualName: awsBlockDevice.VirtualName,
		}
		providerSpec.BlockDevices = append(providerSpec.BlockDevices, blockDevice)
	}

	// Add NetworkInterfaces
	for _, awsNetworkInterface := range awsMachineClass.Spec.NetworkInterfaces {
		networkInterface := api.AWSNetworkInterfaceSpec{
			AssociatePublicIPAddress: awsNetworkInterface.AssociatePublicIPAddress,
			DeleteOnTermination:      awsNetworkInterface.DeleteOnTermination,
			Description:              awsNetworkInterface.Description,
			SecurityGroupIDs:         awsNetworkInterface.SecurityGroupIDs,
			SubnetID:                 awsNetworkInterface.SubnetID,
		}
		providerSpec.NetworkInterfaces = append(providerSpec.NetworkInterfaces, networkInterface)
	}

	// Marshal providerSpec into Raw Bytes
	providerSpecMarshal, err := json.Marshal(providerSpec)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	// Migrate finalizers, labels, annotations
	machineClass.Name = awsMachineClass.Name
	machineClass.Labels = awsMachineClass.Labels
	machineClass.Annotations = awsMachineClass.Annotations
	machineClass.Finalizers = awsMachineClass.Finalizers
	machineClass.ProviderSpec = runtime.RawExtension{
		Raw: providerSpecMarshal,
	}
	machineClass.SecretRef = awsMachineClass.Spec.SecretRef
	machineClass.CredentialsSecretRef = awsMachineClass.Spec.CredentialsSecretRef
	machineClass.Provider = ProviderAWS

	return nil
}
