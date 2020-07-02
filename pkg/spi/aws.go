package spi

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	corev1 "k8s.io/api/core/v1"
)

// PluginSPIImpl is the real implementation of SPI interface that makes the calls to the AWS SDK.
type PluginSPIImpl struct{}

// NewSession starts a new AWS session
func (ms *PluginSPIImpl) NewSession(secret *corev1.Secret, region string) (*session.Session, error) {
	var (
		err    error
		sess   *session.Session
		config *aws.Config
	)

	accessKeyID := strings.TrimSpace(string(secret.Data["providerAccessKeyId"]))
	secretAccessKey := strings.TrimSpace(string(secret.Data["providerSecretAccessKey"]))

	if accessKeyID != "" && secretAccessKey != "" {
		config = &aws.Config{
			Region: aws.String(region),
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			},
			),
		}
	} else {
		config = &aws.Config{
			Region: aws.String(region),
		}
	}
	sess, err = session.NewSession(config)
	return sess, err
}

// NewEC2API Returns a EC2API object
func (ms *PluginSPIImpl) NewEC2API(session *session.Session) ec2iface.EC2API {
	service := ec2.New(session)
	return service
}
