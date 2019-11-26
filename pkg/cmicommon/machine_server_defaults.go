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
// It implements the cmi.MachineClient interface
type DefaultMachineServer struct{}

// CreateMachine method handles default machine creation request
func (ms *DefaultMachineServer) CreateMachine(ctx context.Context, req *cmi.CreateMachineRequest) (*cmi.CreateMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("Create machine request has been recieved for %q", req.MachineName)
	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteMachine method handles default machine deletion request
func (ms *DefaultMachineServer) DeleteMachine(ctx context.Context, req *cmi.DeleteMachineRequest) (*cmi.DeleteMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("Delete machine request has been recieved for %q", req.MachineName)
	return nil, status.Error(codes.Unimplemented, "")
}

// GetMachineStatus method handles default machine get request
func (ms *DefaultMachineServer) GetMachineStatus(ctx context.Context, req *cmi.GetMachineStatusRequest) (*cmi.GetMachineStatusResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("Get machine request has been recieved for %q", req.MachineName)
	return nil, status.Error(codes.Unimplemented, "")
}

// ListMachines method handles default machines list request
func (ms *DefaultMachineServer) ListMachines(ctx context.Context, req *cmi.ListMachinesRequest) (*cmi.ListMachinesResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("List machines request has been recieved for %q", req.ProviderSpec)
	return nil, status.Error(codes.Unimplemented, "")
}

// ShutDownMachine method handles default machines shutdown request
func (ms *DefaultMachineServer) ShutDownMachine(ctx context.Context, req *cmi.ShutDownMachineRequest) (*cmi.ShutDownMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("ShutDown machine request has been recieved for %q", req.MachineName)
	return nil, status.Error(codes.Unimplemented, "")
}

// GetVolumeIDs method handles default getPVIDs request
func (ms *DefaultMachineServer) GetVolumeIDs(ctx context.Context, req *cmi.GetVolumeIDsRequest) (*cmi.GetVolumeIDsResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("GetListOfVolumeIDsForExistingPVs request has been recieved for %v", req.PVSpecList)
	return nil, status.Error(codes.Unimplemented, "")
}
