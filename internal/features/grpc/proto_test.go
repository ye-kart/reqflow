package grpc_test

import (
	"os"
	"path/filepath"
	"testing"

	grpcFeature "github.com/ye-kart/reqflow/internal/features/grpc"
)

func TestParseProtoFile_DiscoversServices(t *testing.T) {
	// Write a temporary .proto file.
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "test.proto")
	protoContent := `syntax = "proto3";
package mypackage;

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string greeting = 1;
}

service Greeter {
  rpc SayHello(HelloRequest) returns (HelloResponse);
  rpc SayGoodbye(HelloRequest) returns (HelloResponse);
}
`
	if err := os.WriteFile(protoPath, []byte(protoContent), 0644); err != nil {
		t.Fatalf("failed to write proto file: %v", err)
	}

	services, err := grpcFeature.ParseProtoFile(protoPath)
	if err != nil {
		t.Fatalf("ParseProtoFile failed: %v", err)
	}

	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}

	svc := services[0]
	if svc.Name != "mypackage.Greeter" {
		t.Errorf("expected service name 'mypackage.Greeter', got %q", svc.Name)
	}
	if len(svc.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(svc.Methods))
	}

	methodNames := map[string]bool{}
	for _, m := range svc.Methods {
		methodNames[m.Name] = true
	}
	if !methodNames["SayHello"] {
		t.Error("expected method 'SayHello'")
	}
	if !methodNames["SayGoodbye"] {
		t.Error("expected method 'SayGoodbye'")
	}
}

func TestParseProtoFile_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "bad.proto")
	if err := os.WriteFile(protoPath, []byte("not valid proto content {{{"), 0644); err != nil {
		t.Fatalf("failed to write proto file: %v", err)
	}

	_, err := grpcFeature.ParseProtoFile(protoPath)
	if err == nil {
		t.Fatal("expected error for invalid proto file, got nil")
	}
}

func TestParseProtoFile_FileNotFound(t *testing.T) {
	_, err := grpcFeature.ParseProtoFile("/nonexistent/path/to.proto")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestParseProtoFile_MultipleServices(t *testing.T) {
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "multi.proto")
	protoContent := `syntax = "proto3";
package multi;

message Request { string data = 1; }
message Response { string data = 1; }

service ServiceA {
  rpc MethodA(Request) returns (Response);
}

service ServiceB {
  rpc MethodB(Request) returns (Response);
}
`
	if err := os.WriteFile(protoPath, []byte(protoContent), 0644); err != nil {
		t.Fatalf("failed to write proto file: %v", err)
	}

	services, err := grpcFeature.ParseProtoFile(protoPath)
	if err != nil {
		t.Fatalf("ParseProtoFile failed: %v", err)
	}

	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
}
