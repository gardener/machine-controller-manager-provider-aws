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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/sampleprovider/driver.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package aws

import (
	"github.com/golang/glog"

	cmicommon "github.com/gardener/machine-controller-manager-provider-aws/pkg/cmi-common"
)

// Driver is struct for the driver.
type Driver struct {
	cmiDriver *cmicommon.CMIDriver
	endpoint  string

	ids *cmicommon.DefaultIdentityServer
	ms  *MachineServer
}

const (
	driverName = "cmi-aws-driver"
)

var (
	version = "0.1.0" //TODO- remove or figure out to sync with project-version
)

// NewDriver returns the new driver object.
func NewDriver(endpoint string) *Driver {
	glog.Infof("Driver: %v version: %v", driverName, version)

	d := &Driver{}
	d.endpoint = endpoint
	cmiDriver := cmicommon.NewCMIDriver(driverName, version)
	d.cmiDriver = cmiDriver

	return d
}

// NewMachineServer returns the new MachineServer object.
func NewMachineServer(d *Driver) *MachineServer {
	return &MachineServer{
		DefaultMachineServer: cmicommon.NewDefaultMachineServer(d.cmiDriver),
	}
}

// Run runs forever, it initiates all the gRPC services.
func (d *Driver) Run() {
	s := cmicommon.NewNonBlockingGRPCServer()
	s.Start(d.endpoint,
		nil,
		NewMachineServer(d))
	s.Wait()
}
