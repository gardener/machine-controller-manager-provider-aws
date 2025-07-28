// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"k8s.io/utils/ptr"

	"github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/interfaces"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"

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

// Helper function to create Client
func (d *Driver) createClient(ctx context.Context, secret *corev1.Secret, region string) (interfaces.Ec2Client, error) {
	config, err := d.CPI.NewConfig(ctx, secret, region)
	if err != nil {
		return nil, err
	}
	client := d.CPI.NewEC2Client(config)
	return client, nil
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
func getInt32PtrForString(s string) *int32 {
	num, _ := strconv.ParseInt(s, 10, 32)
	return ptr.To(int32(num))
}

func retryWithExponentialBackOff(operation backoff.OperationWithData[*ec2types.Instance], _ time.Duration) (*ec2types.Instance, error) {
	expBackOffObj := backoff.NewExponentialBackOff()
	expBackOffObj.MaxElapsedTime = maxElapsedTimeInBackoff
	return backoff.RetryWithData(operation, expBackOffObj)
}
