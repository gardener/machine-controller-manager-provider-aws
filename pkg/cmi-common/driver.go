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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/csi-common/driver.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package cmicommon

import (
	"github.com/golang/glog"
)

// CMIDriver object is used to store the driver details
type CMIDriver struct {
	name    string
	version string
	/*
		TODO Add controller service capability handler
		cap     []*cmi.ControllerServiceCapability
	*/
}

// NewCMIDriver creates a new CMIDriver object and returns the same
func NewCMIDriver(name string, v string) *CMIDriver {
	if name == "" {
		glog.Errorf("Driver name missing")
		return nil
	}

	// TODO version format and validation
	if len(v) == 0 {
		glog.Errorf("Version argument missing")
		return nil
	}

	driver := CMIDriver{
		name:    name,
		version: v,
	}

	return &driver
}
