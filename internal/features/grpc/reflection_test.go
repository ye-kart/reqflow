package grpc_test

import (
	"context"
	"testing"

	grpcFeature "github.com/ye-kart/reqflow/internal/features/grpc"
)

func TestListServices(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	caller := grpcFeature.NewCaller()
	services, err := caller.ListServices(context.Background(), addr, true)
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	if len(services) == 0 {
		t.Fatal("expected at least one service, got none")
	}

	found := false
	for _, s := range services {
		if s == "grpc.testing.TestService" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find 'grpc.testing.TestService' in %v", services)
	}
}

func TestDescribeService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	caller := grpcFeature.NewCaller()
	info, err := caller.DescribeService(context.Background(), addr, "grpc.testing.TestService", true)
	if err != nil {
		t.Fatalf("DescribeService failed: %v", err)
	}

	if info.Name != "grpc.testing.TestService" {
		t.Errorf("expected service name 'grpc.testing.TestService', got %q", info.Name)
	}
	if len(info.Methods) == 0 {
		t.Fatal("expected at least one method")
	}

	found := false
	for _, m := range info.Methods {
		if m.Name == "Echo" {
			found = true
			if m.InputType == "" {
				t.Error("expected non-empty input type")
			}
			if m.OutputType == "" {
				t.Error("expected non-empty output type")
			}
		}
	}
	if !found {
		t.Errorf("expected to find 'Echo' method in %v", info.Methods)
	}
}

func TestListServices_ConnectionError(t *testing.T) {
	caller := grpcFeature.NewCaller()
	_, err := caller.ListServices(context.Background(), "127.0.0.1:1", true)
	if err == nil {
		t.Fatal("expected error for bad address, got nil")
	}
}

func TestDescribeService_NotFound(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	caller := grpcFeature.NewCaller()
	_, err := caller.DescribeService(context.Background(), addr, "nonexistent.Service", true)
	if err == nil {
		t.Fatal("expected error for nonexistent service, got nil")
	}
}
