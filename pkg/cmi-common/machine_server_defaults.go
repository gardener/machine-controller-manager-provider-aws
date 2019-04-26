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
type DefaultMachineServer struct {
	Driver *CMIDriver
}

// NOTE
//
// The basic working of the controller will work with just implementing the CreateMachine() & DeleteMachine() methods.
// You can first implement these two methods and check the working of the controller.
// Once this works you can implement the rest of the methods.
// Implementation of few methods like - ShutDownMachine() are optional, however we highly recommend implementing it as well.

// CreateMachine handles a machine creation request
// REQUIRED METHOD
//
// REQUEST PARAMETERS (cmi.CreateMachineRequest)
// Name                 string              Contains the identification name/tag used to link the machine object with VM on cloud provider
// ProviderSpec         bytes(blob)         Template/Configuration of the machine to be created is given by at the provider
// Secrets              map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.CreateMachineResponse)
// MachineID            string              Unique identification of the VM at the cloud provider. This could be the same/different from req.Name.
//                                          MachineID typically matches with the node.Spec.ProviderID on the node object.
//                                          Eg: gce://project-name/region/vm-machineID
// NodeName             string              Returns the name of the node-object that the VM register's with Kubernetes.
//                                          This could be different from req.Name as well
//
// OPTIONAL IMPLEMENTATION LOGIC
// It is optionally expected by the safety controller to use an identification mechanisms to map the VM Created by a providerSpec.
// These could be done using tag(s)/resource-groups etc.
// This logic is used by safety controller to delete orphan VMs which are not backed by any machine CRD
//
func (ms *DefaultMachineServer) CreateMachine(ctx context.Context, req *cmi.CreateMachineRequest) (*cmi.CreateMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("Create machine request has been recieved for %q", req.Name)
	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteMachine handles a machine deletion request
// REQUIRED METHOD
//
// REQUEST PARAMETERS (cmi.DeleteMachineRequest)
// MachineID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
func (ms *DefaultMachineServer) DeleteMachine(ctx context.Context, req *cmi.DeleteMachineRequest) (*cmi.DeleteMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("Delete machine request has been recieved for %q", req.MachineID)
	return nil, status.Error(codes.Unimplemented, "")
}

// GetMachine handles a machine details fetching request
//
// REQUEST PARAMETERS (cmi.GetMachineRequest)
// MachineID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.GetMachineResponse)
// Exists           bool                Returns a boolean value which is set to true when it exists on the cloud provider
// Status           enum                Contains the status of the machine on the cloud provider mapped to the enum values - {Unknown, Stopped, Running}
//
func (ms *DefaultMachineServer) GetMachine(ctx context.Context, req *cmi.GetMachineRequest) (*cmi.GetMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("Get machine request has been recieved for %q", req.MachineID)
	return nil, status.Error(codes.Unimplemented, "")
}

// ListMachines lists all the machines possibilly created by a providerSpec
// Identifying machines created by a given providerSpec depends on the OPTIONAL IMPLEMENTATION LOGIC
// you have used to identify machines created by a providerSpec. It could be tags/resource-groups etc
//
// REQUEST PARAMETERS (cmi.ListMachinesRequest)
// ProviderSpec     bytes(blob)         Template/Configuration of the machine that wouldn've been created by this ProviderSpec (Machine Class)
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
// RESPONSE PARAMETERS (cmi.ListMachinesResponse)
// MachineList      map<string,string>  A map containing the keys as the MachineID and value as the MachineName
//                                      for all machine's who where possibilly created by this ProviderSpec
//
func (ms *DefaultMachineServer) ListMachines(ctx context.Context, req *cmi.ListMachinesRequest) (*cmi.ListMachinesResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("List machines request has been recieved for %q", req.ProviderSpec)
	return nil, status.Error(codes.Unimplemented, "")
}

// ShutDownMachine handles a machine shutdown/power-off/stop request
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (cmi.ShutDownMachineRequest)
// MachineID        string              Contains the unique identification of the VM at the cloud provider
// Secrets          map<string,bytes>   (Optional) Contains a map from string to string contains any cloud specific secrets that can be used by the provider
//
func (ms *DefaultMachineServer) ShutDownMachine(ctx context.Context, req *cmi.ShutDownMachineRequest) (*cmi.ShutDownMachineResponse, error) {
	// Log messages to track start of request
	glog.V(2).Infof("ShutDown machine request has been recieved for %q", req.MachineID)
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (ms *DefaultMachineServer) ControllerGetCapabilities(ctx context.Context, req *cmi.ControllerGetCapabilitiesRequest) (*cmi.ControllerGetCapabilitiesResponse, error) {
	glog.V(5).Infof("Using default ControllerGetCapabilities")

	// TODO: Update later to return default caps.
	return &cmi.ControllerGetCapabilitiesResponse{}, nil
}
