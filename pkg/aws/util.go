// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"

	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	backoff "github.com/cenkalti/backoff/v4"
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
		err := fmt.Errorf("unable to decode provider-ID")
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

// Function returns true only if error code equals codes.NotFound
func isNotFoundError(err error) bool {
	errorStatus, ok := status.FromError(err)
	if ok && errorStatus.Code() == codes.NotFound {
		return true
	}
	return false
}

// kubernetesVolumeIDToEBSVolumeID translates Kubernetes volume ID to EBS volume ID
// KubernetsVolumeID forms:
//   - aws://<zone>/<awsVolumeId>
//   - aws:///<awsVolumeId>
//   - <awsVolumeId>
//
// EBS Volume ID form:
//   - vol-<alphanumberic>
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

func getStringPtr(s string) *string {
	return &s
}
func getIntPtrForString(s string) *int64 {
	var num int64
	num, _ = strconv.ParseInt(s, 10, 64)
	return &num
}

func retryWithExponentialBackOff(operation backoff.Operation, _ time.Duration) error {
	expBackOffObj := backoff.NewExponentialBackOff()
	expBackOffObj.MaxElapsedTime = maxElapsedTimeInBackoff
	if err := backoff.Retry(operation, expBackOffObj); err != nil {
		return err
	}

	return nil
}
