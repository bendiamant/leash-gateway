// Simplified gRPC service implementation for Phase 1
package module

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// ModuleHostServer is the server API for ModuleHost service.
type ModuleHostServer interface {
	ProcessRequest(context.Context, *ProcessRequestRequest) (*ProcessRequestResponse, error)
	Health(context.Context, *HealthRequest) (*HealthResponse, error)
	mustEmbedUnimplementedModuleHostServer()
}

// UnimplementedModuleHostServer must be embedded to have forward compatible implementations.
type UnimplementedModuleHostServer struct{}

func (UnimplementedModuleHostServer) ProcessRequest(context.Context, *ProcessRequestRequest) (*ProcessRequestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProcessRequest not implemented")
}

func (UnimplementedModuleHostServer) Health(context.Context, *HealthRequest) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}

func (UnimplementedModuleHostServer) mustEmbedUnimplementedModuleHostServer() {}

// RegisterModuleHostServer registers the ModuleHost service
func RegisterModuleHostServer(s grpc.ServiceRegistrar, srv ModuleHostServer) {
	s.RegisterService(&ModuleHost_ServiceDesc, srv)
}

func _ModuleHost_ProcessRequest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProcessRequestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ModuleHostServer).ProcessRequest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/leash.module.v1.ModuleHost/ProcessRequest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ModuleHostServer).ProcessRequest(ctx, req.(*ProcessRequestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ModuleHost_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HealthRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ModuleHostServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/leash.module.v1.ModuleHost/Health",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ModuleHostServer).Health(ctx, req.(*HealthRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ModuleHost_ServiceDesc is the grpc.ServiceDesc for ModuleHost service.
var ModuleHost_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "leash.module.v1.ModuleHost",
	HandlerType: (*ModuleHostServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ProcessRequest",
			Handler:    _ModuleHost_ProcessRequest_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _ModuleHost_Health_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/module.proto",
}
