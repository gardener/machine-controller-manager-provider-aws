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
	"net/url"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	corev1 "k8s.io/api/core/v1"
)

// awsVolumeRegMatch represents Regex Match for AWS volume.
var awsVolumeRegMatch = regexp.MustCompile("^vol-[^/]*$")

// encodeInstanceID encodes a given instanceID as per it's providerID
func encodeInstanceID(region, instanceID string) string {
	return fmt.Sprintf("aws:///%s/%s", region, instanceID)
}

// decodeRegionAndInstanceID extracts region and instanceID
func decodeRegionAndInstanceID(id string) (string, string, error) {
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

// kubernetesVolumeIDToEBSVolumeID translates Kubernetes volume ID to EBS volume ID
// KubernetsVolumeID forms:
//  * aws://<zone>/<awsVolumeId>
//  * aws:///<awsVolumeId>
//  * <awsVolumeId>
// EBS Volume ID form:
//  * vol-<alphanumberic>
func kubernetesVolumeIDToEBSVolumeID(kubernetesID string) (string, error) {
	// name looks like aws://availability-zone/awsVolumeId

	// The original idea of the URL-style name was to put the AZ into the
	// host, so we could find the AZ immediately from the name without
	// querying the API.  But it turns out we don't actually need it for
	// multi-AZ clusters, as we put the AZ into the labels on the PV instead.
	// However, if in future we want to support multi-AZ cluster
	// volume-awareness without using PersistentVolumes, we likely will
	// want the AZ in the host.
	if !strings.HasPrefix(kubernetesID, "aws://") {
		// Assume a bare aws volume id (vol-1234...)
		return kubernetesID, nil
	}
	url, err := url.Parse(kubernetesID)
	if err != nil {
		return "", fmt.Errorf("invalid disk name (%s): %v", kubernetesID, err)
	}
	if url.Scheme != "aws" {
		return "", fmt.Errorf("invalid scheme for AWS volume (%s)", kubernetesID)
	}

	awsID := url.Path
	awsID = strings.Trim(awsID, "/")

	// We sanity check the resulting volume; the two known formats are
	// vol-12345678 and vol-12345678abcdef01
	if !awsVolumeRegMatch.MatchString(awsID) {
		return "", fmt.Errorf("invalid format for AWS volume (%s)", kubernetesID)
	}

	return awsID, nil
}
