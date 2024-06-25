// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.20.3
// source: query.proto

package pb

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

// BridgeServiceClient is the client API for BridgeService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BridgeServiceClient interface {
	// Getters
	// / Get api version
	CheckAPI(ctx context.Context, in *CheckAPIRequest, opts ...grpc.CallOption) (*CheckAPIResponse, error)
	// / Get bridges for the destination address both in L1 and L2
	GetBridges(ctx context.Context, in *GetBridgesRequest, opts ...grpc.CallOption) (*GetBridgesResponse, error)
	// / Get the merkle proof for the specific deposit
	GetProof(ctx context.Context, in *GetProofRequest, opts ...grpc.CallOption) (*GetProofResponse, error)
	// / Get the merkle proof for the specific deposit and GER
	GetProofByGER(ctx context.Context, in *GetProofByGERRequest, opts ...grpc.CallOption) (*GetProofResponse, error)
	// / Get the specific deposit
	GetBridge(ctx context.Context, in *GetBridgeRequest, opts ...grpc.CallOption) (*GetBridgeResponse, error)
	// / Get claims for the specific smart contract address both in L1 and L2
	GetClaims(ctx context.Context, in *GetClaimsRequest, opts ...grpc.CallOption) (*GetClaimsResponse, error)
	// / Get token wrapped for the specific smart contract address both in L1 and L2
	GetTokenWrapped(ctx context.Context, in *GetTokenWrappedRequest, opts ...grpc.CallOption) (*GetTokenWrappedResponse, error)
}

type bridgeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBridgeServiceClient(cc grpc.ClientConnInterface) BridgeServiceClient {
	return &bridgeServiceClient{cc}
}

func (c *bridgeServiceClient) CheckAPI(ctx context.Context, in *CheckAPIRequest, opts ...grpc.CallOption) (*CheckAPIResponse, error) {
	out := new(CheckAPIResponse)
	err := c.cc.Invoke(ctx, "/bridge.v1.BridgeService/CheckAPI", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bridgeServiceClient) GetBridges(ctx context.Context, in *GetBridgesRequest, opts ...grpc.CallOption) (*GetBridgesResponse, error) {
	out := new(GetBridgesResponse)
	err := c.cc.Invoke(ctx, "/bridge.v1.BridgeService/GetBridges", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bridgeServiceClient) GetProof(ctx context.Context, in *GetProofRequest, opts ...grpc.CallOption) (*GetProofResponse, error) {
	out := new(GetProofResponse)
	err := c.cc.Invoke(ctx, "/bridge.v1.BridgeService/GetProof", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bridgeServiceClient) GetProofByGER(ctx context.Context, in *GetProofByGERRequest, opts ...grpc.CallOption) (*GetProofResponse, error) {
	out := new(GetProofResponse)
	err := c.cc.Invoke(ctx, "/bridge.v1.BridgeService/GetProofByGER", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bridgeServiceClient) GetBridge(ctx context.Context, in *GetBridgeRequest, opts ...grpc.CallOption) (*GetBridgeResponse, error) {
	out := new(GetBridgeResponse)
	err := c.cc.Invoke(ctx, "/bridge.v1.BridgeService/GetBridge", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bridgeServiceClient) GetClaims(ctx context.Context, in *GetClaimsRequest, opts ...grpc.CallOption) (*GetClaimsResponse, error) {
	out := new(GetClaimsResponse)
	err := c.cc.Invoke(ctx, "/bridge.v1.BridgeService/GetClaims", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bridgeServiceClient) GetTokenWrapped(ctx context.Context, in *GetTokenWrappedRequest, opts ...grpc.CallOption) (*GetTokenWrappedResponse, error) {
	out := new(GetTokenWrappedResponse)
	err := c.cc.Invoke(ctx, "/bridge.v1.BridgeService/GetTokenWrapped", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BridgeServiceServer is the server API for BridgeService service.
// All implementations must embed UnimplementedBridgeServiceServer
// for forward compatibility
type BridgeServiceServer interface {
	// Getters
	// / Get api version
	CheckAPI(context.Context, *CheckAPIRequest) (*CheckAPIResponse, error)
	// / Get bridges for the destination address both in L1 and L2
	GetBridges(context.Context, *GetBridgesRequest) (*GetBridgesResponse, error)
	// / Get the merkle proof for the specific deposit
	GetProof(context.Context, *GetProofRequest) (*GetProofResponse, error)
	// / Get the merkle proof for the specific deposit and GER
	GetProofByGER(context.Context, *GetProofByGERRequest) (*GetProofResponse, error)
	// / Get the specific deposit
	GetBridge(context.Context, *GetBridgeRequest) (*GetBridgeResponse, error)
	// / Get claims for the specific smart contract address both in L1 and L2
	GetClaims(context.Context, *GetClaimsRequest) (*GetClaimsResponse, error)
	// / Get token wrapped for the specific smart contract address both in L1 and L2
	GetTokenWrapped(context.Context, *GetTokenWrappedRequest) (*GetTokenWrappedResponse, error)
	mustEmbedUnimplementedBridgeServiceServer()
}

// UnimplementedBridgeServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBridgeServiceServer struct {
}

func (UnimplementedBridgeServiceServer) CheckAPI(context.Context, *CheckAPIRequest) (*CheckAPIResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckAPI not implemented")
}
func (UnimplementedBridgeServiceServer) GetBridges(context.Context, *GetBridgesRequest) (*GetBridgesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBridges not implemented")
}
func (UnimplementedBridgeServiceServer) GetProof(context.Context, *GetProofRequest) (*GetProofResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProof not implemented")
}
func (UnimplementedBridgeServiceServer) GetProofByGER(context.Context, *GetProofByGERRequest) (*GetProofResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProofByGER not implemented")
}
func (UnimplementedBridgeServiceServer) GetBridge(context.Context, *GetBridgeRequest) (*GetBridgeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBridge not implemented")
}
func (UnimplementedBridgeServiceServer) GetClaims(context.Context, *GetClaimsRequest) (*GetClaimsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetClaims not implemented")
}
func (UnimplementedBridgeServiceServer) GetTokenWrapped(context.Context, *GetTokenWrappedRequest) (*GetTokenWrappedResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTokenWrapped not implemented")
}
func (UnimplementedBridgeServiceServer) mustEmbedUnimplementedBridgeServiceServer() {}

// UnsafeBridgeServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BridgeServiceServer will
// result in compilation errors.
type UnsafeBridgeServiceServer interface {
	mustEmbedUnimplementedBridgeServiceServer()
}

func RegisterBridgeServiceServer(s grpc.ServiceRegistrar, srv BridgeServiceServer) {
	s.RegisterService(&BridgeService_ServiceDesc, srv)
}

func _BridgeService_CheckAPI_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckAPIRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BridgeServiceServer).CheckAPI(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bridge.v1.BridgeService/CheckAPI",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BridgeServiceServer).CheckAPI(ctx, req.(*CheckAPIRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BridgeService_GetBridges_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBridgesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BridgeServiceServer).GetBridges(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bridge.v1.BridgeService/GetBridges",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BridgeServiceServer).GetBridges(ctx, req.(*GetBridgesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BridgeService_GetProof_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetProofRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BridgeServiceServer).GetProof(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bridge.v1.BridgeService/GetProof",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BridgeServiceServer).GetProof(ctx, req.(*GetProofRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BridgeService_GetProofByGER_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetProofByGERRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BridgeServiceServer).GetProofByGER(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bridge.v1.BridgeService/GetProofByGER",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BridgeServiceServer).GetProofByGER(ctx, req.(*GetProofByGERRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BridgeService_GetBridge_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBridgeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BridgeServiceServer).GetBridge(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bridge.v1.BridgeService/GetBridge",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BridgeServiceServer).GetBridge(ctx, req.(*GetBridgeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BridgeService_GetClaims_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetClaimsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BridgeServiceServer).GetClaims(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bridge.v1.BridgeService/GetClaims",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BridgeServiceServer).GetClaims(ctx, req.(*GetClaimsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BridgeService_GetTokenWrapped_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTokenWrappedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BridgeServiceServer).GetTokenWrapped(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bridge.v1.BridgeService/GetTokenWrapped",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BridgeServiceServer).GetTokenWrapped(ctx, req.(*GetTokenWrappedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BridgeService_ServiceDesc is the grpc.ServiceDesc for BridgeService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BridgeService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "bridge.v1.BridgeService",
	HandlerType: (*BridgeServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CheckAPI",
			Handler:    _BridgeService_CheckAPI_Handler,
		},
		{
			MethodName: "GetBridges",
			Handler:    _BridgeService_GetBridges_Handler,
		},
		{
			MethodName: "GetProof",
			Handler:    _BridgeService_GetProof_Handler,
		},
		{
			MethodName: "GetProofByGER",
			Handler:    _BridgeService_GetProofByGER_Handler,
		},
		{
			MethodName: "GetBridge",
			Handler:    _BridgeService_GetBridge_Handler,
		},
		{
			MethodName: "GetClaims",
			Handler:    _BridgeService_GetClaims_Handler,
		},
		{
			MethodName: "GetTokenWrapped",
			Handler:    _BridgeService_GetTokenWrapped_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "query.proto",
}
