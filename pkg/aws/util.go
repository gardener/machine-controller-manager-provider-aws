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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
)

// Helper function to create SVC
func createSVC(Secrets api.Secrets, region string) *ec2.EC2 {

	accessKeyID := strings.TrimSpace(Secrets.ProviderAccessKeyID)
	secretAccessKey := strings.TrimSpace(Secrets.ProviderSecretAccessKey)

	if accessKeyID != "" && secretAccessKey != "" {
		return ec2.New(session.New(&aws.Config{
			Region: aws.String(region),
			Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			}),
		}))
	}

	return ec2.New(session.New(&aws.Config{
		Region: aws.String(region),
	}))
}

func encodeMachineID(region, machineID string) string {
	return fmt.Sprintf("aws:///%s/%s", region, machineID)
}

func decodeRegionAndMachineID(id string) (string, string, error) {
	splitProviderID := strings.Split(id, "/")
	if len(splitProviderID) < 2 {
		err := fmt.Errorf("Unable to decode provider-ID")
		return "", "", err
	}
	return splitProviderID[len(splitProviderID)-2], splitProviderID[len(splitProviderID)-1], nil
}
