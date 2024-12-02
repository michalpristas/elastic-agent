// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.25.3
// source: control_v1.proto

package proto

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ElasticAgentControlClient is the client API for ElasticAgentControl service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ElasticAgentControlClient interface {
	// Fetches the currently running version of the Elastic Agent.
	Version(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*VersionResponse, error)
	// Fetches the currently status of the Elastic Agent.
	Status(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusResponse, error)
	// Restart restarts the current running Elastic Agent.
	Restart(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*RestartResponse, error)
	// Upgrade starts the upgrade process of Elastic Agent.
	Upgrade(ctx context.Context, in *UpgradeRequest, opts ...grpc.CallOption) (*UpgradeResponse, error)
}

type elasticAgentControlClient struct {
	cc grpc.ClientConnInterface
}

func NewElasticAgentControlClient(cc grpc.ClientConnInterface) ElasticAgentControlClient {
	return &elasticAgentControlClient{cc}
}

func (c *elasticAgentControlClient) Version(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*VersionResponse, error) {
	out := new(VersionResponse)
	err := c.cc.Invoke(ctx, "/proto.ElasticAgentControl/Version", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *elasticAgentControlClient) Status(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, "/proto.ElasticAgentControl/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *elasticAgentControlClient) Restart(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*RestartResponse, error) {
	out := new(RestartResponse)
	err := c.cc.Invoke(ctx, "/proto.ElasticAgentControl/Restart", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *elasticAgentControlClient) Upgrade(ctx context.Context, in *UpgradeRequest, opts ...grpc.CallOption) (*UpgradeResponse, error) {
	out := new(UpgradeResponse)
	err := c.cc.Invoke(ctx, "/proto.ElasticAgentControl/Upgrade", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ElasticAgentControlServer is the server API for ElasticAgentControl service.
// All implementations must embed UnimplementedElasticAgentControlServer
// for forward compatibility
type ElasticAgentControlServer interface {
	// Fetches the currently running version of the Elastic Agent.
	Version(context.Context, *Empty) (*VersionResponse, error)
	// Fetches the currently status of the Elastic Agent.
	Status(context.Context, *Empty) (*StatusResponse, error)
	// Restart restarts the current running Elastic Agent.
	Restart(context.Context, *Empty) (*RestartResponse, error)
	// Upgrade starts the upgrade process of Elastic Agent.
	Upgrade(context.Context, *UpgradeRequest) (*UpgradeResponse, error)
	mustEmbedUnimplementedElasticAgentControlServer()
}

// UnimplementedElasticAgentControlServer must be embedded to have forward compatible implementations.
type UnimplementedElasticAgentControlServer struct {
}

func (UnimplementedElasticAgentControlServer) Version(context.Context, *Empty) (*VersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Version not implemented")
}
func (UnimplementedElasticAgentControlServer) Status(context.Context, *Empty) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedElasticAgentControlServer) Restart(context.Context, *Empty) (*RestartResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Restart not implemented")
}
func (UnimplementedElasticAgentControlServer) Upgrade(context.Context, *UpgradeRequest) (*UpgradeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Upgrade not implemented")
}
func (UnimplementedElasticAgentControlServer) mustEmbedUnimplementedElasticAgentControlServer() {}

// UnsafeElasticAgentControlServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ElasticAgentControlServer will
// result in compilation errors.
type UnsafeElasticAgentControlServer interface {
	mustEmbedUnimplementedElasticAgentControlServer()
}

func RegisterElasticAgentControlServer(s grpc.ServiceRegistrar, srv ElasticAgentControlServer) {
	s.RegisterService(&ElasticAgentControl_ServiceDesc, srv)
}

func _ElasticAgentControl_Version_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ElasticAgentControlServer).Version(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ElasticAgentControl/Version",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ElasticAgentControlServer).Version(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ElasticAgentControl_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ElasticAgentControlServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ElasticAgentControl/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ElasticAgentControlServer).Status(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ElasticAgentControl_Restart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ElasticAgentControlServer).Restart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ElasticAgentControl/Restart",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ElasticAgentControlServer).Restart(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ElasticAgentControl_Upgrade_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpgradeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ElasticAgentControlServer).Upgrade(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ElasticAgentControl/Upgrade",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ElasticAgentControlServer).Upgrade(ctx, req.(*UpgradeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ElasticAgentControl_ServiceDesc is the grpc.ServiceDesc for ElasticAgentControl service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ElasticAgentControl_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.ElasticAgentControl",
	HandlerType: (*ElasticAgentControlServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Version",
			Handler:    _ElasticAgentControl_Version_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _ElasticAgentControl_Status_Handler,
		},
		{
			MethodName: "Restart",
			Handler:    _ElasticAgentControl_Restart_Handler,
		},
		{
			MethodName: "Upgrade",
			Handler:    _ElasticAgentControl_Upgrade_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "control_v1.proto",
}
