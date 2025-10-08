// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	corev1 "k8s.io/api/core/v1"

	"github.com/aws/aws-sdk-go-v2/config"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	"github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/interfaces"
)

// ClientProvider is the real implementation of CPI interface that provides a client to make calls against the API
type ClientProvider struct{}

// NewConfig returns the config used to create a new EC2 Client set-up with the provided values.
func (cp *ClientProvider) NewConfig(ctx context.Context, secret *corev1.Secret, region string) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	if workloadIdentityTokenFile, ok := secret.Data["workloadIdentityTokenFile"]; ok {
		webIDProvider := stscreds.NewWebIdentityRoleProvider(
			sts.NewFromConfig(cfg),
			string(secret.Data["roleARN"]),
			stscreds.IdentityTokenFile(workloadIdentityTokenFile),
		)
		cfg, err := config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(aws.NewCredentialsCache(webIDProvider)),
		)
		if err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	accessKeyID := extractCredentialsFromData(secret.Data, api.AWSAccessKeyID, api.AWSAlternativeAccessKeyID)
	secretAccessKey := extractCredentialsFromData(secret.Data, api.AWSSecretAccessKey, api.AWSAlternativeSecretAccessKey)

	if accessKeyID != "" && secretAccessKey != "" {
		cfg, err := config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""))),
		)
		if err != nil {
			return nil, err
		}
		return &cfg, nil

	}

	return &cfg, nil
}

// NewEC2Client Returns an EC2Client object
func (cp *ClientProvider) NewEC2Client(config *aws.Config) interfaces.Ec2Client {
	client := ec2.NewFromConfig(*config)
	return client
}

// extractCredentialsFromData extracts and trims a value from the given data map. The first key that exists is being
// returned, otherwise, the next key is tried, etc. If no key exists then an empty string is returned.
func extractCredentialsFromData(data map[string][]byte, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			return strings.TrimSpace(string(val))
		}
	}
	return ""
}
