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
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/csi-common/server.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package cmicommon

import (
	"net"
	"os"
	"sync"

	"github.com/golang/glog"
	"google.golang.org/grpc"

	"github.com/gardener/machine-spec/lib/go/cmi"
)

// NonBlockingGRPCServer defines Non blocking GRPC server interfaces
type NonBlockingGRPCServer interface {
	// Start services at the endpoint
	Start(endpoint string, ids cmi.IdentityServer, ms cmi.MachineServer)
	// Waits for the service to stop
	Wait()
	// Stops the service gracefully
	Stop()
	// Stops the service forcefully
	ForceStop()
}

// NewNonBlockingGRPCServer returns an empty NonBlockingGRPCServer
func NewNonBlockingGRPCServer() NonBlockingGRPCServer {
	return &nonBlockingGRPCServer{}
}

// NonBlocking server
type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

// Start starts a nonBlockingGRPCServer
func (s *nonBlockingGRPCServer) Start(endpoint string, ids cmi.IdentityServer, ms cmi.MachineServer) {
	s.wg.Add(1)
	go s.serve(endpoint, ids, ms)

	return
}

// Wait adds a wait on the waitgroup of the nonBlockingGRPCServer
func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

// Stop gracefully stops the nonBlockingGRPCServer
func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

// ForceStop force stops the nonBlockingGRPCServer
func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

// serve listens to requests on the given endpoint
func (s *nonBlockingGRPCServer) serve(endpoint string, ids cmi.IdentityServer, ms cmi.MachineServer) {
	proto, addr, err := ParseEndpoint(endpoint)
	if err != nil {
		glog.Fatal(err.Error())
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			glog.Fatalf("Failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		glog.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	s.server = server

	if ids != nil {
		cmi.RegisterIdentityServer(server, ids)
	}
	if ms != nil {
		cmi.RegisterMachineServer(server, ms)
	}

	glog.V(1).Infof("Listening for connections on address: %#v", listener.Addr())
	server.Serve(listener)
}
