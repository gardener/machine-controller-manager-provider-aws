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

package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// encodeMachineID encodes a given provider-ID as per it's provider ID
func encodeProviderID(region, providerID string) string {
	return fmt.Sprintf("aws:///%s/%s", region, providerID)
}

// decodeRegionAndProviderID extracts region and provider ID
func decodeRegionAndProviderID(id string) (string, string, error) {
	splitProviderID := strings.Split(id, "/")
	if len(splitProviderID) < 2 {
		err := fmt.Errorf("Unable to decode provider-ID")
		return "", "", err
	}
	return splitProviderID[len(splitProviderID)-2], splitProviderID[len(splitProviderID)-1], nil
}

// Helper function to create SVC
func (ms *MachinePlugin) createSVC(secrets *api.Secrets, region string) (ec2iface.EC2API, error) {
	session, err := ms.SPI.NewSession(*secrets, region)
	if err != nil {
		return nil, err
	}
	svc := ms.SPI.NewEC2API(session)
	return svc, nil
}

//pluginSPIImpl is the real implementation of PluginSPI interface that makes the calls to the AWS SDK.
type pluginSPIImpl struct{}

// NewSession starts a new AWS session
func (ms *pluginSPIImpl) NewSession(Secrets api.Secrets, region string) (*awssession.Session, error) {
	var (
		err     error
		session *awssession.Session
		config  *aws.Config
	)

	accessKeyID := strings.TrimSpace(Secrets.ProviderAccessKeyID)
	secretAccessKey := strings.TrimSpace(Secrets.ProviderSecretAccessKey)

	if accessKeyID != "" && secretAccessKey != "" {
		config = &aws.Config{
			Region: aws.String(region),
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			},
			)}

	} else {
		config = &aws.Config{
			Region: aws.String(region),
		}
	}
	session, err = awssession.NewSession(config)
	return session, err
}

// NewEC2API Returns a EC2API object
func (ms *pluginSPIImpl) NewEC2API(session *session.Session) ec2iface.EC2API {
	service := ec2.New(session)
	return service
}
