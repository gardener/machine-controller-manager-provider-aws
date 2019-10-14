/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was copied and modified from the kubernetes-csi/drivers project
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/csi-common/identityserver-default.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package aws

import (
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetPluginInfo returns the Server details
func (ids *MachineNIdentityPlugin) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	glog.Infof("Using GetPluginInfo")

	if ids.Plugin.Name == "" {
		return nil, status.Error(codes.Unavailable, "Plugin name not configured")
	}

	if ids.Plugin.Name == "" {
		return nil, status.Error(codes.Unavailable, "Plugin is missing version")
	}

	return &csi.GetPluginInfoResponse{
		Name:          ids.Plugin.Name,
		VendorVersion: ids.Plugin.Version,
	}, nil
}

// Probe tries to probe the server and returns a response
func (ids *MachineNIdentityPlugin) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

// GetPluginCapabilities gets capabilities of the plugin
func (ids *MachineNIdentityPlugin) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	glog.V(5).Infof("Using default capabilities")
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}
