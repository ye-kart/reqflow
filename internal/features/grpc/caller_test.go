package grpc_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	grpcFeature "github.com/ye-kart/reqflow/internal/features/grpc"
)

// startTestServer spins up a gRPC server with reflection enabled and returns
// the listener address and a cleanup function.
func startTestServer(t *testing.T) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	registerTestService(t, srv)
	reflection.Register(srv)

	go func() { _ = srv.Serve(lis) }()

	return lis.Addr().String(), func() {
		srv.GracefulStop()
		lis.Close()
	}
}

func registerTestService(t *testing.T, srv *grpc.Server) {
	t.Helper()

	fdProto := &descriptorpb.FileDescriptorProto{
		Name:    sp("grpc_testing.proto"),
		Package: sp("grpc.testing"),
		Syntax:  sp("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: sp("EchoRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: sp("message"), Number: ip(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: sp("message")},
					{Name: sp("code"), Number: ip(2), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: sp("code")},
				},
			},
			{
				Name: sp("EchoResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: sp("message"), Number: ip(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: sp("message")},
					{Name: sp("code"), Number: ip(2), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: sp("code")},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: sp("TestService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       sp("Echo"),
						InputType:  sp(".grpc.testing.EchoRequest"),
						OutputType: sp(".grpc.testing.EchoResponse"),
					},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdProto, nil)
	if err != nil {
		t.Fatalf("building file descriptor: %v", err)
	}

	// Register with global registries so gRPC reflection can discover the service.
	if _, regErr := protoregistry.GlobalFiles.FindFileByPath("grpc_testing.proto"); regErr != nil {
		if err := protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
			t.Fatalf("registering file descriptor: %v", err)
		}
		for i := range fd.Messages().Len() {
			md := fd.Messages().Get(i)
			msgType := dynamicpb.NewMessageType(md)
			_ = protoregistry.GlobalTypes.RegisterMessage(msgType)
		}
	}

	reqDesc := fd.Messages().ByName("EchoRequest")
	respDesc := fd.Messages().ByName("EchoResponse")

	sd := &grpc.ServiceDesc{
		ServiceName: "grpc.testing.TestService",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Echo",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					md, _ := metadata.FromIncomingContext(ctx)
					if vals := md.Get("x-custom-header"); len(vals) > 0 {
						_ = grpc.SendHeader(ctx, metadata.Pairs("x-echo-header", vals[0]))
					}

					reqMsg := dynamicpb.NewMessage(reqDesc)
					if err := dec(reqMsg); err != nil {
						return nil, err
					}

					respMsg := dynamicpb.NewMessage(respDesc)
					respMsg.Set(respDesc.Fields().ByName("message"), reqMsg.Get(reqDesc.Fields().ByName("message")))
					respMsg.Set(respDesc.Fields().ByName("code"), reqMsg.Get(reqDesc.Fields().ByName("code")))

					return respMsg, nil
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "grpc_testing.proto",
	}

	srv.RegisterService(sd, &struct{}{})
}

func sp(s string) *string { return &s }
func ip(i int32) *int32   { return &i }

// Suppress unused import.
var _ = proto.Marshal

// ---------------------------------------------------------------------------
// Caller tests
// ---------------------------------------------------------------------------

func TestCall_UnaryReturnsResponse(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	caller := grpcFeature.NewCaller()
	result, err := caller.Call(context.Background(), grpcFeature.CallOptions{
		Address:       addr,
		Method:        "grpc.testing.TestService/Echo",
		Data:          `{"message": "hello"}`,
		Plaintext:     true,
		UseReflection: true,
	})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.Status != "OK" {
		t.Errorf("expected status OK, got %q", result.Status)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(result.Response, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["message"] != "hello" {
		t.Errorf("expected message 'hello', got %v", resp["message"])
	}
}

func TestCall_MetadataSentAndReceived(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	caller := grpcFeature.NewCaller()
	result, err := caller.Call(context.Background(), grpcFeature.CallOptions{
		Address:       addr,
		Method:        "grpc.testing.TestService/Echo",
		Data:          `{"message": "meta-test"}`,
		Headers:       map[string]string{"x-custom-header": "custom-value"},
		Plaintext:     true,
		UseReflection: true,
	})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.Status != "OK" {
		t.Errorf("expected status OK, got %q", result.Status)
	}

	if result.Headers["x-echo-header"] != "custom-value" {
		t.Errorf("expected echoed header 'custom-value', got %q", result.Headers["x-echo-header"])
	}
}

func TestCall_JSONInputOutputConversion(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	caller := grpcFeature.NewCaller()
	result, err := caller.Call(context.Background(), grpcFeature.CallOptions{
		Address:       addr,
		Method:        "grpc.testing.TestService/Echo",
		Data:          `{"message": "json-test", "code": 42}`,
		Plaintext:     true,
		UseReflection: true,
	})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(result.Response, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["message"] != "json-test" {
		t.Errorf("expected message 'json-test', got %v", resp["message"])
	}
	if resp["code"] != float64(42) {
		t.Errorf("expected code 42, got %v", resp["code"])
	}
}

func TestCall_PlaintextConnection(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	caller := grpcFeature.NewCaller()
	result, err := caller.Call(context.Background(), grpcFeature.CallOptions{
		Address:       addr,
		Method:        "grpc.testing.TestService/Echo",
		Data:          `{"message": "plaintext"}`,
		Plaintext:     true,
		UseReflection: true,
	})
	if err != nil {
		t.Fatalf("Call with plaintext failed: %v", err)
	}

	if result.Status != "OK" {
		t.Errorf("expected status OK, got %q", result.Status)
	}
}

func TestCall_ConnectionErrorHandled(t *testing.T) {
	caller := grpcFeature.NewCaller()
	_, err := caller.Call(context.Background(), grpcFeature.CallOptions{
		Address:       "127.0.0.1:1",
		Method:        "grpc.testing.TestService/Echo",
		Data:          `{"message": "fail"}`,
		Plaintext:     true,
		UseReflection: true,
	})
	if err == nil {
		t.Fatal("expected an error for bad address, got nil")
	}
}

func TestCall_ContextCancellation(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	caller := grpcFeature.NewCaller()
	_, err := caller.Call(ctx, grpcFeature.CallOptions{
		Address:       addr,
		Method:        "grpc.testing.TestService/Echo",
		Data:          `{"message": "cancelled"}`,
		Plaintext:     true,
		UseReflection: true,
	})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
