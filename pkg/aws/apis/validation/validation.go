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

// Package validation - validation is used to validate cloud specific ProviderSpec for AWS
package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	awsapi "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	corev1 "k8s.io/api/core/v1"
)

const nameFmt string = `[-a-z0-9]+`
const nameMaxLength int = 63

var nameRegexp = regexp.MustCompile("^" + nameFmt + "$")

// ValidateAWSProviderSpec validates AWS provider spec
func ValidateAWSProviderSpec(spec *awsapi.AWSProviderSpec, secret *corev1.Secret) []error {
	var allErrs []error

	if "" == spec.AMI {
		allErrs = append(allErrs, fmt.Errorf("AMI is required field"))
	}
	if "" == spec.Region {
		allErrs = append(allErrs, fmt.Errorf("Region is required field"))
	}
	if "" == spec.MachineType {
		allErrs = append(allErrs, fmt.Errorf("MachineType is required field"))
	}
	if "" == spec.IAM.Name {
		allErrs = append(allErrs, fmt.Errorf("IAM Name is required field"))
	}
	if "" == spec.KeyName {
		allErrs = append(allErrs, fmt.Errorf("KeyName is required field"))
	}

	allErrs = append(allErrs, validateBlockDevices(spec.BlockDevices)...)
	allErrs = append(allErrs, validateNetworkInterfaces(spec.NetworkInterfaces)...)
	allErrs = append(allErrs, ValidateSecret(secret)...)
	allErrs = append(allErrs, validateSpecTags(spec.Tags)...)

	return allErrs
}

func validateSpecTags(tags map[string]string) []error {
	var allErrs []error
	clusterName := ""
	nodeRole := ""

	for key := range tags {
		if strings.Contains(key, "kubernetes.io/cluster/") {
			clusterName = key
		} else if strings.Contains(key, "kubernetes.io/role/") {
			nodeRole = key
		}
	}

	if clusterName == "" {
		allErrs = append(allErrs, fmt.Errorf("Tag is required of the form kubernetes.io/cluster/****"))
	}
	if nodeRole == "" {
		allErrs = append(allErrs, fmt.Errorf("Tag is required of the form kubernetes.io/role/****"))
	}
	return allErrs
}

func validateBlockDevices(blockDevices []awsapi.AWSBlockDeviceMappingSpec) []error {

	var allErrs []error

	if len(blockDevices) > 1 {
		allErrs = append(allErrs, fmt.Errorf("Can only specify one (root) block device"))
	} else if len(blockDevices) == 1 {
		if blockDevices[0].Ebs.VolumeSize <= 0 {
			allErrs = append(allErrs, fmt.Errorf("Please mention a valid ebs volume size"))
		}
		if blockDevices[0].Ebs.VolumeType == "" {
			allErrs = append(allErrs, fmt.Errorf("Please mention a valid ebs volume type"))
		} else if blockDevices[0].Ebs.VolumeType == "io1" && blockDevices[0].Ebs.Iops <= 0 {
			allErrs = append(allErrs, fmt.Errorf("Please mention a valid ebs volume iops"))
		}
	}
	return allErrs
}

func validateNetworkInterfaces(networkInterfaces []awsapi.AWSNetworkInterfaceSpec) []error {
	var allErrs []error
	if len(networkInterfaces) == 0 {
		allErrs = append(allErrs, fmt.Errorf("Mention at least one NetworkInterface"))
	} else {
		for i := range networkInterfaces {
			if "" == networkInterfaces[i].SubnetID {
				allErrs = append(allErrs, fmt.Errorf("SubnetID is required"))
			}

			if 0 == len(networkInterfaces[i].SecurityGroupIDs) {
				allErrs = append(allErrs, fmt.Errorf("Mention at least one securityGroupID"))
			} else {
				for j := range networkInterfaces[i].SecurityGroupIDs {
					if "" == networkInterfaces[i].SecurityGroupIDs[j] {
						output := strings.Join([]string{"securityGroupIDs cannot be blank for networkInterface:", strconv.Itoa(i), " securityGroupID:", strconv.Itoa(j)}, "")

						allErrs = append(allErrs, fmt.Errorf(output))

					}
				}
			}
		}
	}
	return allErrs
}

// ValidateSecret makes sure that the supplied secrets contains the required fields
func ValidateSecret(secret *corev1.Secret) []error {
	var allErrs []error
	if secret == nil {
		allErrs = append(allErrs, fmt.Errorf("SecretReference is Nil"))
	} else {
		if "" == string(secret.Data["providerAccessKeyId"]) {
			allErrs = append(allErrs, fmt.Errorf("Secret providerAccessKeyId is required field"))
		}
		if "" == string(secret.Data["providerSecretAccessKey"]) {
			allErrs = append(allErrs, fmt.Errorf("Secret providerSecretAccessKey is required field"))
		}

		if "" == string(secret.Data["userData"]) {
			allErrs = append(allErrs, fmt.Errorf("Secret userData is required field"))
		}
	}
	return allErrs
}
