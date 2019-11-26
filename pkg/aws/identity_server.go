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
	"github.com/gardener/machine-spec/lib/go/cmi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetPluginInfo returns the Server details
func (ids *IdentityPlugin) GetPluginInfo(ctx context.Context, req *cmi.GetPluginInfoRequest) (*cmi.GetPluginInfoResponse, error) {
	glog.V(3).Infof("Using GetPluginInfo")

	if ids.Plugin.Name == "" {
		return nil, status.Error(codes.Internal, "Plugin name not configured")
	}

	if ids.Plugin.Name == "" {
		return nil, status.Error(codes.Internal, "Plugin is missing version")
	}

	return &cmi.GetPluginInfoResponse{
		Name:    ids.Plugin.Name,
		Version: ids.Plugin.Version,
	}, nil
}

// Probe tries to probe the server and returns a response
func (ids *IdentityPlugin) Probe(ctx context.Context, req *cmi.ProbeRequest) (*cmi.ProbeResponse, error) {
	return &cmi.ProbeResponse{}, nil
}

// GetPluginCapabilities gets capabilities of the plugin
func (ids *IdentityPlugin) GetPluginCapabilities(ctx context.Context, req *cmi.GetPluginCapabilitiesRequest) (*cmi.GetPluginCapabilitiesResponse, error) {
	var (
		cmc []*cmi.PluginCapability
		cl  []cmi.PluginCapability_RPC_Type
	)

	cl = []cmi.PluginCapability_RPC_Type{
		cmi.PluginCapability_RPC_CREATE_MACHINE,
		cmi.PluginCapability_RPC_DELETE_MACHINE,
		cmi.PluginCapability_RPC_GET_MACHINE_STATUS,
		cmi.PluginCapability_RPC_LIST_MACHINES,
		cmi.PluginCapability_RPC_SHUTDOWN_MACHINE,
		cmi.PluginCapability_RPC_GET_VOLUME_IDS,
	}

	for _, c := range cl {
		glog.V(4).Infof("Enabling controller service capability: %v", c.String())
		cmc = append(cmc, NewPluginCapability(c))
	}

	return &cmi.GetPluginCapabilitiesResponse{
		Capabilities: cmc,
	}, nil
}

// NewPluginCapability TODO
func NewPluginCapability(cap cmi.PluginCapability_RPC_Type) *cmi.PluginCapability {
	return &cmi.PluginCapability{
		Type: &cmi.PluginCapability_Rpc{
			Rpc: &cmi.PluginCapability_RPC{
				Type: cap,
			},
		},
	}
}
