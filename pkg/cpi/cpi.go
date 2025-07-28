// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/interfaces"
	corev1 "k8s.io/api/core/v1"
)

// ClientProviderInterface provides an interface to set-up and instantiate (EC2) Clients
type ClientProviderInterface interface {
	NewConfig(context.Context, *corev1.Secret, string) (*aws.Config, error)
	NewEC2Client(*aws.Config) interfaces.Ec2Client
}
