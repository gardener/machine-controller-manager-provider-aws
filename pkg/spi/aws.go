// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package spi

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/sts"
	corev1 "k8s.io/api/core/v1"

	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
)

// PluginSPIImpl is the real implementation of SPI interface that makes the calls to the AWS SDK.
type PluginSPIImpl struct{}

// NewSession starts a new AWS session
func (ms *PluginSPIImpl) NewSession(secret *corev1.Secret, region string) (*session.Session, error) {
	var config = &aws.Config{
		Region: aws.String(region),
	}

	if workloadIdentityTokenFile, ok := secret.Data["workloadIdentityTokenFile"]; ok {
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		webIDProvider := stscreds.NewWebIdentityRoleProviderWithOptions(
			sts.New(sess),
			string(secret.Data["roleARN"]),
			secret.Namespace,
			stscreds.FetchTokenPath(workloadIdentityTokenFile),
		)
		creds, err := webIDProvider.Retrieve()
		if err != nil {
			return nil, err
		}
		cc := credentials.NewStaticCredentialsFromCreds(creds)
		config := &aws.Config{
			Region:      aws.String(region),
			Credentials: cc,
		}
		return session.NewSession(config)
	}

	accessKeyID := extractCredentialsFromData(secret.Data, api.AWSAccessKeyID, api.AWSAlternativeAccessKeyID)
	secretAccessKey := extractCredentialsFromData(secret.Data, api.AWSSecretAccessKey, api.AWSAlternativeSecretAccessKey)

	if accessKeyID != "" && secretAccessKey != "" {
		config = &aws.Config{
			Region: aws.String(region),
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			}),
		}
	}

	return session.NewSession(config)
}

// NewEC2API Returns a EC2API object
func (ms *PluginSPIImpl) NewEC2API(session *session.Session) ec2iface.EC2API {
	service := ec2.New(session)
	return service
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
