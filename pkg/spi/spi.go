// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package spi

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	corev1 "k8s.io/api/core/v1"
)

// SessionProviderInterface provides an interface to deal with AWS provider session
type SessionProviderInterface interface {
	NewSession(*corev1.Secret, string) (*session.Session, error)
	NewEC2API(*session.Session) ec2iface.EC2API
}
