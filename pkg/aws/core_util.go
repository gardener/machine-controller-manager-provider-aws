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

package aws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	validation "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis/validation"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	// AWSMachineClassKind for AWSMachineClass
	AWSMachineClassKind = "AWSMachineClass"

	// MachineClassKind for MachineClass
	MachineClassKind = "MachineClass"
)

// decodeProviderSpecAndSecret converts request parameters to api.ProviderSpec & api.Secrets
func decodeProviderSpecAndSecret(machineClass *v1alpha1.MachineClass, secret *corev1.Secret) (*api.AWSProviderSpec, error) {
	var (
		providerSpec *api.AWSProviderSpec
	)

	// Extract providerSpec
	if machineClass == nil {
		return nil, status.Error(codes.Internal, "MachineClass ProviderSpec is nil")
	}

	err := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Validate the Spec and Secrets
	ValidationErr := validation.ValidateAWSProviderSpec(providerSpec, secret)
	if ValidationErr != nil {
		err = fmt.Errorf("Error while validating ProviderSpec %v", ValidationErr)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return providerSpec, nil
}

// getInstancesFromMachineName extracts AWS Instance object from given machine name
func (d *Driver) getInstancesFromMachineName(machineName string, providerSpec *api.AWSProviderSpec, secret *corev1.Secret) ([]*ec2.Instance, error) {
	var (
		clusterName string
		nodeRole    string
		instances   []*ec2.Instance
	)

	svc, err := d.createSVC(secret, providerSpec.Region)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for key := range providerSpec.Tags {
		if strings.Contains(key, "kubernetes.io/cluster/") {
			clusterName = key
		} else if strings.Contains(key, "kubernetes.io/role/") {
			nodeRole = key
		}
	}

	input := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(machineName),
				},
			},
			{
				Name: aws.String("tag-key"),
				Values: []*string{
					&clusterName,
				},
			},
			{
				Name: aws.String("tag-key"),
				Values: []*string{
					&nodeRole,
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("pending"),
					aws.String("running"),
					aws.String("stopping"),
					aws.String("stopped"),
				},
			},
		},
	}

	runResult, err := svc.DescribeInstances(&input)
	if err != nil {
		klog.Errorf("AWS plugin is returning error while describe instances request is sent: %s", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, reservation := range runResult.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance)
		}
	}
	if len(instances) == 0 {
		errMessage := fmt.Sprintf("AWS plugin is returning no VM instances backing this machine object")
		return nil, status.Error(codes.NotFound, errMessage)
	}

	return instances, nil
}
