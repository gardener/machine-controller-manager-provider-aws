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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/nfs/driver.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	cmicommon "github.com/gardener/machine-controller-manager-provider-aws/pkg/cmi-common"
	"github.com/golang/glog"
)

// MachineServer contains the machine server info
// It implements the cmi.MachineClient interface
// It also implements the driverSPI interface
type MachineServer struct {
	*cmicommon.DefaultMachineServer
	SPI DriverSPI
}

// DriverSPI provides an interface to deal with cloud provider session
type DriverSPI interface {
	NewSession(api.Secrets, string) (*session.Session, error)
	NewEC2API(*session.Session) ec2iface.EC2API
}

// Driver returns the new provider details
type Driver struct {
	// CMIDriver contains details about the CMIDriver object
	CMIDriver *cmicommon.CMIDriver
	// Contains the endpoint details on which the driver is open for connections
	endpoint string
	// Identity server attached to the driver
	ids *cmicommon.DefaultIdentityServer
	// Machine Server attached to the driver
	ms *MachineServer
}

const (
	driverName = "cmi-aws-driver"
)

var (
	version = "0.1.0"
)

// NewDriver returns a newly created driver object
func NewDriver(endpoint string) *Driver {
	glog.V(1).Infof("Driver: %v version: %v", driverName, version)

	d := &Driver{}

	d.endpoint = endpoint

	CMIDriver := cmicommon.NewCMIDriver(driverName, version)
	// TODO MachineService Capabilities
	// cmiDriver.AddControllerServiceCapabilities([]cmi.ControllerServiceCapability_RPC_Type{cmi.ControllerServiceCapability_RPC_UNKNOWN})
	d.CMIDriver = CMIDriver

	return d
}

// NewMachineServer returns a new machineserver
func NewMachineServer(d *Driver, spi DriverSPI) *MachineServer {
	return &MachineServer{
		DefaultMachineServer: cmicommon.NewDefaultMachineServer(d.CMIDriver),
		SPI:                  spi,
	}
}

// Run starts a new gRPC server to start the driver
func (d *Driver) Run() {
	s := cmicommon.NewNonBlockingGRPCServer()
	s.Start(d.endpoint,
		nil,
		NewMachineServer(d, &driverSPIImpl{}))
	s.Wait()
}
