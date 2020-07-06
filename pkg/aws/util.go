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
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	corev1 "k8s.io/api/core/v1"
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
func (d *Driver) createSVC(secret *corev1.Secret, region string) (ec2iface.EC2API, error) {
	session, err := d.SPI.NewSession(secret, region)
	if err != nil {
		return nil, err
	}
	svc := d.SPI.NewEC2API(session)
	return svc, nil
}
