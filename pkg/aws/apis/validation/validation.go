// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package validation - validation is used to validate cloud specific ProviderSpec for AWS
package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	awsapi "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
)

// ValidateAWSProviderSpec validates AWS provider spec
func ValidateAWSProviderSpec(spec *awsapi.AWSProviderSpec, secret *corev1.Secret, fldPath *field.Path) field.ErrorList {
	var (
		allErrs = field.ErrorList{}
	)

	if spec.AMI == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("ami"), "AMI is required"))
	}
	if spec.Region == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), "Region is required"))
	}
	if spec.MachineType == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("machineType"), "MachineType is required"))
	}
	if (spec.IAM.Name == "" && spec.IAM.ARN == "") || (spec.IAM.Name != "" && spec.IAM.ARN != "") {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("iam"), spec.IAM, "either IAM Name or ARN must be set"))
	}

	allErrs = append(allErrs, validateBlockDevices(spec.BlockDevices, fldPath.Child("blockDevices"))...)
	allErrs = append(allErrs, validateCapacityReservations(spec.CapacityReservationTarget, fldPath.Child("capacityReservation"))...)
	allErrs = append(allErrs, validateNetworkInterfaces(spec.NetworkInterfaces, fldPath.Child("networkInterfaces"))...)
	allErrs = append(allErrs, ValidateSecret(secret, field.NewPath("secretRef"))...)
	allErrs = append(allErrs, validateSpecTags(spec.Tags, fldPath.Child("tags"))...)
	allErrs = append(allErrs, validateInstanceMetadata(spec.InstanceMetadataOptions, fldPath.Child("instanceMetadata"))...)
	allErrs = append(allErrs, validateCPUOptions(spec.CPUOptions, fldPath.Child(("cpuOptions")))...)

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

		deviceNames[disk.DeviceName] += 1

		if !slices.Contains(awsapi.ValidVolumeTypes, disk.Ebs.VolumeType) {
			allErrs = append(allErrs, field.Required(idxPath.Child("ebs.volumeType"), fmt.Sprintf("Please mention a valid EBS volume type: %v", awsapi.ValidVolumeTypes)))
		}

		if disk.Ebs.VolumeSize <= 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("ebs.volumeSize"), "Please mention a valid EBS volume size"))
		}

		if disk.Ebs.Iops < 0 || (disk.Ebs.VolumeType == awsapi.VolumeTypeIO1 && disk.Ebs.Iops == 0) {
			allErrs = append(allErrs, field.Required(idxPath.Child("ebs.iops"), "Please mention a valid EBS volume iops"))
		}

		// validate throughput
		if disk.Ebs.Throughput != nil && *disk.Ebs.Throughput <= 0 {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("ebs.throughput"), *disk.Ebs.Throughput, "Throughput should be a positive value"))
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
		if capacityReservation.CapacityReservationPreference != nil {
			if capacityReservation.CapacityReservationID != nil || capacityReservation.CapacityReservationResourceGroupArn != nil {
				allErrs = append(allErrs, field.Required(fldPath, "CapacityReservationPreference cannot be set when also providing a CapacityReservationID or CapacityReservationResourceGroupArn"))
			}
		} else if capacityReservation.CapacityReservationID != nil && capacityReservation.CapacityReservationResourceGroupArn != nil {
			allErrs = append(allErrs, field.Required(fldPath, "CapacityReservationResourceGroupArn or CapacityReservationId are optional but only one should be used"))
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
			if networkInterfaces[i].SubnetID == "" {
				allErrs = append(allErrs, field.Required(fldPath.Child("subnetID"), "SubnetID is required"))
			}

			if len(networkInterfaces[i].SecurityGroupIDs) == 0 {
				allErrs = append(allErrs, field.Required(fldPath.Child("securityGroupIDs"), "Mention at least one securityGroupID"))
			} else {
				for j := range networkInterfaces[i].SecurityGroupIDs {
					if networkInterfaces[i].SecurityGroupIDs[j] == "" {
						output := strings.Join([]string{"securityGroupIDs cannot be blank for networkInterface:", strconv.Itoa(i), " securityGroupID:", strconv.Itoa(j)}, "")
						allErrs = append(allErrs, field.Required(fldPath.Child("securityGroupIDs"), output))
					}
				}
			}
		}
	}
	return allErrs
}

func validateInstanceMetadata(metadata *awsapi.InstanceMetadataOptions, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if metadata == nil {
		return allErrs
	}

	if metadata.HTTPPutResponseHopLimit != nil && (*metadata.HTTPPutResponseHopLimit < 0 || *metadata.HTTPPutResponseHopLimit > 64) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("httpPutResponseHopLimit"), *metadata.HTTPPutResponseHopLimit, "Only values between 0 and 64, both included, are accepted"))
	}

	if metadata.HTTPEndpoint != nil {
		allErrs = append(allErrs, validateStringValues(fldPath.Child("httpEndpoint"), *metadata.HTTPEndpoint, []string{awsapi.HTTPEndpointDisabled, awsapi.HTTPEndpointEnabled})...)
	}

	if metadata.HTTPTokens != nil {
		allErrs = append(allErrs, validateStringValues(fldPath.Child("httpTokens"), *metadata.HTTPTokens, []string{awsapi.HTTPTokensRequired, awsapi.HTTPTokensOptional})...)
	}

	return allErrs
}

func validateCPUOptions(cpuOptions *awsapi.CPUOptions, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if cpuOptions == nil {
		return allErrs
	}

	if cpuOptions.CoreCount == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("coreCount"), "CoreCount is required"))
	}

	if cpuOptions.ThreadsPerCore == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("threadsPerCore"), "ThreadsPerCore is required"))
	}

	if threadsPerCore := *cpuOptions.ThreadsPerCore; threadsPerCore > 2 || threadsPerCore < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("threadsPerCore"), threadsPerCore, "ThreadsPerCore must be either '1' or '2'"))
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
	} else if workloadIdentityTokenFile, ok := secret.Data["workloadIdentityTokenFile"]; ok {
		if len(workloadIdentityTokenFile) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("workloadIdentityTokenFile"), "Workload identity token file is required"))
		}

		if roleARN, ok := secret.Data["roleARN"]; !ok || len(roleARN) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("roleARN"), "Role ARN is required when workload identity is used"))
		}
		if string(secret.Data["userData"]) == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("userData"), "Mention userData"))
		}
	} else {
		if string(secret.Data[awsapi.AWSAccessKeyID]) == "" && string(secret.Data[awsapi.AWSAlternativeAccessKeyID]) == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("AWSAccessKeyID"), fmt.Sprintf("Mention atleast %s or %s", awsapi.AWSAccessKeyID, awsapi.AWSAlternativeAccessKeyID)))
		}
		if string(secret.Data[awsapi.AWSSecretAccessKey]) == "" && string(secret.Data[awsapi.AWSAlternativeSecretAccessKey]) == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("AWSSecretAccessKey"), fmt.Sprintf("Mention atleast %s or %s", awsapi.AWSSecretAccessKey, awsapi.AWSAlternativeSecretAccessKey)))
		}
		if string(secret.Data["userData"]) == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("userData"), "Mention userData"))
		}
	}

	return allErrs
}

func validateStringValues(fld *field.Path, s string, accepted []string) field.ErrorList {
	if slices.Contains(accepted, s) {
		return field.ErrorList{}
	}

	return field.ErrorList{field.Invalid(fld, s, fmt.Sprintf("Accepted values: %v", accepted))}
}
