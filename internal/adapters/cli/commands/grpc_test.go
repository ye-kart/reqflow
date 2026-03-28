package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/app"
)

func startGRPCTestServer(t *testing.T) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	registerCLITestService(t, srv)
	reflection.Register(srv)

	go func() { _ = srv.Serve(lis) }()

	return lis.Addr().String(), func() {
		srv.GracefulStop()
		lis.Close()
	}
}

func registerCLITestService(t *testing.T, srv *grpc.Server) {
	t.Helper()

	fdProto := &descriptorpb.FileDescriptorProto{
		Name:    strp("grpc_cli_testing.proto"),
		Package: strp("grpc.clitesting"),
		Syntax:  strp("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strp("PingRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strp("value"), Number: intp(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: strp("value")},
				},
			},
			{
				Name: strp("PingResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strp("value"), Number: intp(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: strp("value")},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strp("PingService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strp("Ping"),
						InputType:  strp(".grpc.clitesting.PingRequest"),
						OutputType: strp(".grpc.clitesting.PingResponse"),
					},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdProto, nil)
	if err != nil {
		t.Fatalf("building file descriptor: %v", err)
	}

	if _, regErr := protoregistry.GlobalFiles.FindFileByPath("grpc_cli_testing.proto"); regErr != nil {
		if err := protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
			t.Fatalf("registering file descriptor: %v", err)
		}
		for i := range fd.Messages().Len() {
			md := fd.Messages().Get(i)
			msgType := dynamicpb.NewMessageType(md)
			_ = protoregistry.GlobalTypes.RegisterMessage(msgType)
		}
	}

	reqDesc := fd.Messages().ByName("PingRequest")
	respDesc := fd.Messages().ByName("PingResponse")

	sd := &grpc.ServiceDesc{
		ServiceName: "grpc.clitesting.PingService",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Ping",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					md, _ := metadata.FromIncomingContext(ctx)
					if vals := md.Get("x-test"); len(vals) > 0 {
						_ = grpc.SendHeader(ctx, metadata.Pairs("x-test-echo", vals[0]))
					}

					reqMsg := dynamicpb.NewMessage(reqDesc)
					if err := dec(reqMsg); err != nil {
						return nil, err
					}

					respMsg := dynamicpb.NewMessage(respDesc)
					respMsg.Set(respDesc.Fields().ByName("value"), reqMsg.Get(reqDesc.Fields().ByName("value")))
					return respMsg, nil
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "grpc_cli_testing.proto",
	}

	srv.RegisterService(sd, &struct{}{})
}

func strp(s string) *string  { return &s }
func intp(i int32) *int32    { return &i }

func TestGRPCCommand_Registered(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)

	found := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "grpc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'grpc' subcommand to be registered on root")
	}
}

func TestGRPCCommand_HasSubcommands(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)

	for _, cmd := range root.Commands() {
		if cmd.Name() == "grpc" {
			subs := cmd.Commands()
			subNames := make(map[string]bool)
			for _, s := range subs {
				subNames[s.Name()] = true
			}
			for _, expected := range []string{"call", "list", "describe"} {
				if !subNames[expected] {
					t.Errorf("expected grpc subcommand %q not found", expected)
				}
			}
			return
		}
	}
	t.Error("grpc command not found")
}

func TestGRPCCallCommand(t *testing.T) {
	addr, cleanup := startGRPCTestServer(t)
	defer cleanup()

	a := &app.App{}
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{
		"grpc", "call", addr, "grpc.clitesting.PingService/Ping",
		"--data", `{"value": "pong"}`,
		"--plaintext",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("grpc call failed: %v\noutput: %s", err, buf.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse output as JSON: %v\nraw: %s", err, buf.String())
	}
	if resp["value"] != "pong" {
		t.Errorf("expected value 'pong', got %v", resp["value"])
	}
}

func TestGRPCListCommand(t *testing.T) {
	addr, cleanup := startGRPCTestServer(t)
	defer cleanup()

	a := &app.App{}
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"grpc", "list", addr, "--plaintext"})

	if err := root.Execute(); err != nil {
		t.Fatalf("grpc list failed: %v\noutput: %s", err, buf.String())
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("grpc.clitesting.PingService")) {
		t.Errorf("expected output to contain 'grpc.clitesting.PingService', got: %s", output)
	}
}

func TestGRPCDescribeCommand(t *testing.T) {
	addr, cleanup := startGRPCTestServer(t)
	defer cleanup()

	a := &app.App{}
	root := commands.NewRootCommand(a)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"grpc", "describe", addr, "grpc.clitesting.PingService", "--plaintext"})

	if err := root.Execute(); err != nil {
		t.Fatalf("grpc describe failed: %v\noutput: %s", err, buf.String())
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Ping")) {
		t.Errorf("expected output to contain method 'Ping', got: %s", output)
	}
}
