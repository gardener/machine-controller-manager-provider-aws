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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/csi-common/nodeserver-default.go

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

// DefaultMachineServer contains the machine server info
type DefaultMachineServer struct {
	Driver *CMIDriver
}

// CreateMachine is the default method used to create a machine
func (cs *DefaultMachineServer) CreateMachine(ctx context.Context, req *cmi.CreateMachineRequest) (*cmi.CreateMachineResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListMachines is the default method used to list machines
// Returns a VM matching the machineID, but when the machineID is an empty string
// then it returns all matching instances in terms of map[string]string
func (cs *DefaultMachineServer) ListMachines(ctx context.Context, req *cmi.ListMachinesRequest) (*cmi.ListMachinesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteMachine is the default method used to delete a machine
func (cs *DefaultMachineServer) DeleteMachine(ctx context.Context, req *cmi.DeleteMachineRequest) (*cmi.DeleteMachineResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *DefaultMachineServer) ControllerGetCapabilities(ctx context.Context, req *cmi.ControllerGetCapabilitiesRequest) (*cmi.ControllerGetCapabilitiesResponse, error) {
	glog.V(5).Infof("Using default ControllerGetCapabilities")

	// TODO: Update later to return default caps.
	return &cmi.ControllerGetCapabilitiesResponse{}, nil
}
