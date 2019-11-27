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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/csi-common/utils.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package cmicommon

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// ParseEndpoint parses the given endpoint string into the protocol and address
func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("Invalid endpoint: %v", ep)
}

// NewDefaultIdentityServer returns the default Identity server object
func NewDefaultIdentityServer(plugin *DefaultPlugin) *DefaultIdentityServer {
	return &DefaultIdentityServer{
		Plugin: plugin,
	}
}

// NewDefaultMachineServer returns a default machine server object
func NewDefaultMachineServer(d *DefaultPlugin) *DefaultMachineServer {
	return &DefaultMachineServer{}
}

// logGRPC logs various GRPC requests and response
func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	glog.V(3).Infof("GRPC call: %s", info.FullMethod)
	glog.V(5).Infof("GRPC request: %+v", req)
	resp, err := handler(ctx, req)
	if err != nil {
		glog.Errorf("GRPC call: %s, Error: %v", info.FullMethod, err)
	} else {
		glog.V(5).Infof("GRPC response: %+v", resp)
	}
	return resp, err
}
