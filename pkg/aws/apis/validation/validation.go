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
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const nameFmt string = `[-a-z0-9]+`
const nameMaxLength int = 63

var nameRegexp = regexp.MustCompile("^" + nameFmt + "$")

// ValidateAWSProviderSpec validates AWS provider spec
func ValidateAWSProviderSpec(spec *awsapi.AWSProviderSpec, secret *corev1.Secret, fldPath *field.Path) field.ErrorList {
	var (
		allErrs = field.ErrorList{}
	)

	if "" == spec.AMI {
		allErrs = append(allErrs, field.Required(fldPath.Child("ami"), "AMI is required"))
	}
	if "" == spec.Region {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), "Region is required"))
	}
	if "" == spec.MachineType {
		allErrs = append(allErrs, field.Required(fldPath.Child("machineType"), "MachineType is required"))
	}
	if ("" == spec.IAM.Name && "" == spec.IAM.ARN) || ("" != spec.IAM.Name && "" != spec.IAM.ARN) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("iam"), spec.IAM, "either IAM Name or ARN must be set"))
	}
	if "" == spec.KeyName {
		allErrs = append(allErrs, field.Required(fldPath.Child("keyName"), "KeyName is required"))
	}

	allErrs = append(allErrs, validateBlockDevices(spec.BlockDevices, fldPath.Child("blockDevices"))...)
	allErrs = append(allErrs, validateCapacityReservations(spec.CapacityReservationTarget, fldPath.Child("capacityReservation"))...)
	allErrs = append(allErrs, validateNetworkInterfaces(spec.NetworkInterfaces, fldPath.Child("networkInterfaces"))...)
	allErrs = append(allErrs, ValidateSecret(secret, field.NewPath("secretRef"))...)
	allErrs = append(allErrs, validateSpecTags(spec.Tags, fldPath.Child("tags"))...)

	return allErrs
}

func validateSpecTags(tags map[string]string, fldPath *field.Path) field.ErrorList {
	var (
		allErrs     = field.ErrorList{}
		clusterName = ""
		nodeRole    = ""
	)

	for key := range tags {
		if strings.Contains(key, awsapi.ClusterTagPrefix) {
			clusterName = key
		} else if strings.Contains(key, awsapi.RoleTagPrefix) {
			nodeRole = key
		}
	}

	if clusterName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child(""), "Tag required of the form "+awsapi.ClusterTagPrefix+"****"))
	}
	if nodeRole == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child(""), "Tag required of the form "+awsapi.RoleTagPrefix+"****"))
	}

	return allErrs
}

func validateBlockDevices(blockDevices []awsapi.AWSBlockDeviceMappingSpec, fldPath *field.Path) field.ErrorList {
	var (
		allErrs              = field.ErrorList{}
		rootPartitionCount   = 0
		deviceNames          = make(map[string]int)
		dataDeviceNameRegexp = regexp.MustCompile(awsapi.DataDeviceNameFormat)
	)

	// if blockDevices is empty, AWS will automatically create a root partition
	for i, disk := range blockDevices {
		idxPath := fldPath.Index(i)

		if disk.DeviceName == awsapi.RootDeviceName {
			rootPartitionCount++
		} else if len(blockDevices) > 1 && !dataDeviceNameRegexp.MatchString(disk.DeviceName) {
			// if there are multiple devices, non-root devices are expected to adhere to AWS naming conventions
			allErrs = append(allErrs, field.Invalid(idxPath.Child("deviceName"), disk.DeviceName, utilvalidation.RegexError(fmt.Sprintf("Device name given: %s does not match the expected pattern", disk.DeviceName), awsapi.DataDeviceNameFormat)))
		}

		if _, keyExist := deviceNames[disk.DeviceName]; keyExist {
			deviceNames[disk.DeviceName]++
		} else {
			deviceNames[disk.DeviceName] = 1
		}

		if !contains(awsapi.ValidVolumeTypes, disk.Ebs.VolumeType) {
			allErrs = append(allErrs, field.Required(idxPath.Child("ebs.volumeType"), fmt.Sprintf("Please mention a valid EBS volume type: %v", awsapi.ValidVolumeTypes)))
		}

		if disk.Ebs.VolumeSize <= 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("ebs.volumeSize"), "Please mention a valid EBS volume size"))
		}

		if disk.Ebs.VolumeType == awsapi.VolumeTypeIO1 && disk.Ebs.Iops <= 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("ebs.iops"), "Please mention a valid EBS volume iops"))
		}

	}

	if rootPartitionCount > 1 {
		allErrs = append(allErrs, field.Required(fldPath, "Only one device can be specified as root"))
		// len(blockDevices) > 1 allow backward compatibility when a single disk is provided without DeviceName
	} else if rootPartitionCount == 0 && len(blockDevices) > 1 {
		allErrs = append(allErrs, field.Required(fldPath, "Only one device can be specified as root"))
	}

	for device, number := range deviceNames {
		if number > 1 {
			allErrs = append(allErrs, field.Required(fldPath, fmt.Sprintf("Device name '%s' duplicated %d times, DeviceName must be unique", device, number)))
		}
	}

	return allErrs
}

func validateCapacityReservations(capacityReservation *awsapi.AWSCapacityReservationTargetSpec, fldPath *field.Path) field.ErrorList {
	var (
		allErrs = field.ErrorList{}
	)

	if capacityReservation != nil {
		if capacityReservation.CapacityReservationID != nil && capacityReservation.CapacityReservationResourceGroupArn != nil {
			allErrs = append(allErrs, field.Required(fldPath, "capacityReservationResourceGroupArn or capacityReservationId are optional but only one should be used"))
		}
	}

	return allErrs
}

func validateNetworkInterfaces(networkInterfaces []awsapi.AWSNetworkInterfaceSpec, fldPath *field.Path) field.ErrorList {
	var (
		allErrs = field.ErrorList{}
	)

	if len(networkInterfaces) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child(""), "Mention at least one NetworkInterface"))
	} else {
		for i := range networkInterfaces {
			if "" == networkInterfaces[i].SubnetID {
				allErrs = append(allErrs, field.Required(fldPath.Child("subnetID"), "SubnetID is required"))
			}

			if 0 == len(networkInterfaces[i].SecurityGroupIDs) {
				allErrs = append(allErrs, field.Required(fldPath.Child("securityGroupIDs"), "Mention at least one securityGroupID"))
			} else {
				for j := range networkInterfaces[i].SecurityGroupIDs {
					if "" == networkInterfaces[i].SecurityGroupIDs[j] {
						output := strings.Join([]string{"securityGroupIDs cannot be blank for networkInterface:", strconv.Itoa(i), " securityGroupID:", strconv.Itoa(j)}, "")
						allErrs = append(allErrs, field.Required(fldPath.Child("securityGroupIDs"), output))
					}
				}
			}
		}
	}
	return allErrs
}

// ValidateSecret makes sure that the supplied secrets contains the required fields
func ValidateSecret(secret *corev1.Secret, fldPath *field.Path) field.ErrorList {
	var (
		allErrs = field.ErrorList{}
	)

	if secret == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child(""), "secretRef is required"))
	} else {
		if "" == string(secret.Data[awsapi.AWSAccessKeyID]) && "" == string(secret.Data[awsapi.AWSAlternativeAccessKeyID]) {
			allErrs = append(allErrs, field.Required(fldPath.Child("AWSAccessKeyID"), fmt.Sprintf("Mention atleast %s or %s", awsapi.AWSAccessKeyID, awsapi.AWSAlternativeAccessKeyID)))
		}
		if "" == string(secret.Data[awsapi.AWSSecretAccessKey]) && "" == string(secret.Data[awsapi.AWSAlternativeSecretAccessKey]) {
			allErrs = append(allErrs, field.Required(fldPath.Child("AWSSecretAccessKey"), fmt.Sprintf("Mention atleast %s or %s", awsapi.AWSSecretAccessKey, awsapi.AWSAlternativeSecretAccessKey)))
		}
		if "" == string(secret.Data["userData"]) {
			allErrs = append(allErrs, field.Required(fldPath.Child("userData"), "Mention userData"))
		}
	}

	return allErrs
}

func contains(arr []string, checkValue string) bool {
	for _, value := range arr {
		if value == checkValue {
			return true
		}
	}
	return false
}
