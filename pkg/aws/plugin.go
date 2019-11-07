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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/nfs/plugin.go

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

const pluginName = "cmi-aws-plugin"

var version = "0.1.0"

// NewPlugin returns a newly created plugin object
func NewPlugin(endpoint string) *Plugin {
	glog.V(1).Infof("Plugin: %v version: %v", pluginName, version)

	p := &Plugin{}
	p.endpoint = endpoint
	cmiPlugin := cmicommon.NewDefaultPlugin(pluginName, version)

	// TODO MachineService Capabilities
	// cmiPlugin.AddControllerServiceCapabilities([]cmi.ControllerServiceCapability_RPC_Type{cmi.ControllerServiceCapability_RPC_UNKNOWN})
	p.CMIPlugin = cmiPlugin

	return p
}

// Run starts a new gRPC server to start the plugin
func (p *Plugin) Run() {
	s := cmicommon.NewNonBlockingGRPCServer()
	s.Start(
		p.endpoint,
		NewIdentityPlugin(p, &pluginSPIImpl{}),
		NewMachinePlugin(p, &pluginSPIImpl{}),
	)
	s.Wait()
}

// PluginSPI provides an interface to deal with cloud provider session
type PluginSPI interface {
	NewSession(api.Secrets, string) (*session.Session, error)
	NewEC2API(*session.Session) ec2iface.EC2API
}

// MachinePlugin implements the cmi.MachineServer
// It also implements the pluginSPI interface
type MachinePlugin struct {
	*cmicommon.DefaultMachineServer
	SPI PluginSPI
}

// NewMachinePlugin returns a new MachinePlugin
func NewMachinePlugin(p *Plugin, spi PluginSPI) *MachinePlugin {
	return &MachinePlugin{
		DefaultMachineServer: cmicommon.NewDefaultMachineServer(p.CMIPlugin),
		SPI:                  spi,
	}
}

// IdentityPlugin implements the cmi.IdentityServer clients
type IdentityPlugin struct {
	*cmicommon.DefaultIdentityServer
}

// NewIdentityPlugin returns a new IdentityPlugin
func NewIdentityPlugin(p *Plugin, spi PluginSPI) *IdentityPlugin {
	return &IdentityPlugin{
		DefaultIdentityServer: cmicommon.NewDefaultIdentityServer(p.CMIPlugin),
	}
}

// Plugin returns the new provider details
type Plugin struct {
	// CMIPlugin contains details about the CMIPlugin object
	CMIPlugin *cmicommon.DefaultPlugin
	// Contains the endpoint details on which the plugin is open for connections
	endpoint string
	// Identity server attached to the plugin
	ids *cmicommon.DefaultIdentityServer
	// Machine Server attached to the plugin
	ms *cmicommon.DefaultMachineServer
}
