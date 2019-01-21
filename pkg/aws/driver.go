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

type driver struct {
	cmiDriver *cmicommon.CMIDriver
	endpoint  string

	ids *cmicommon.DefaultIdentityServer
	ms  *machineServer
}

const (
	driverName = "cmi-aws-driver"
)

var (
	version = "1.0.0"
)

func NewDriver(endpoint string) *driver {
	glog.Infof("Driver: %v version: %v", driverName, version)

	d := &driver{}

	d.endpoint = endpoint

	cmiDriver := cmicommon.NewCMIDriver(driverName, version)
	// TODO MachineService Capabilities
	// cmiDriver.AddControllerServiceCapabilities([]cmi.ControllerServiceCapability_RPC_Type{cmi.ControllerServiceCapability_RPC_UNKNOWN})

	d.cmiDriver = cmiDriver

	return d
}

func NewMachineServer(d *driver) *machineServer {
	return &machineServer{
		DefaultMachineServer: cmicommon.NewDefaultMachineServer(d.cmiDriver),
	}
}

func (d *driver) Run() {
	s := cmicommon.NewNonBlockingGRPCServer()
	s.Start(d.endpoint,
		nil,
		NewMachineServer(d))
	s.Wait()
}
