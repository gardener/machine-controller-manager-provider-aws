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

package cmicommon

import (
	"github.com/gardener/machine-spec/lib/go/cmi"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DefaultIdentityServer contains the server identity
// Implements the cmi.IdentityServer
type DefaultIdentityServer struct {
	Plugin *DefaultPlugin
}

// GetPluginInfo returns the Server details
func (ids *DefaultIdentityServer) GetPluginInfo(ctx context.Context, req *cmi.GetPluginInfoRequest) (*cmi.GetPluginInfoResponse, error) {
	glog.V(5).Infof("Using default GetPluginInfo")

	if ids.Plugin.Name == "" {
		return nil, status.Error(codes.Unavailable, "Plugin name not configured")
	}

	if ids.Plugin.Version == "" {
		return nil, status.Error(codes.Unavailable, "Plugin is missing version")
	}

	return &cmi.GetPluginInfoResponse{
		Name:    ids.Plugin.Name,
		Version: ids.Plugin.Version,
	}, nil
}

// Probe tries to probe the server and returns a response
func (ids *DefaultIdentityServer) Probe(ctx context.Context, req *cmi.ProbeRequest) (*cmi.ProbeResponse, error) {
	return &cmi.ProbeResponse{}, nil
}

// GetPluginCapabilities gets capabilities of the plugin
func (ids *DefaultIdentityServer) GetPluginCapabilities(ctx context.Context, req *cmi.GetPluginCapabilitiesRequest) (*cmi.GetPluginCapabilitiesResponse, error) {
	glog.V(3).Infof("Using default GetPluginCapabilities")
	return &cmi.GetPluginCapabilitiesResponse{
		Capabilities: []*cmi.PluginCapability{},
	}, nil
}
