package grpc

import (
	"context"
	"fmt"

	"github.com/jhump/protoreflect/grpcreflect"
)

// ListServices uses gRPC server reflection to list available services.
// It filters out internal gRPC reflection services.
func (c *Caller) ListServices(ctx context.Context, address string, plaintext bool) ([]string, error) {
	conn, err := dial(ctx, address, plaintext)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", address, err)
	}
	defer conn.Close()

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	services, err := refClient.ListServices()
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	// Filter out internal reflection services.
	var filtered []string
	for _, s := range services {
		if s == "grpc.reflection.v1alpha.ServerReflection" ||
			s == "grpc.reflection.v1.ServerReflection" {
			continue
		}
		filtered = append(filtered, s)
	}

	return filtered, nil
}

// DescribeService uses gRPC server reflection to describe a service's methods.
func (c *Caller) DescribeService(ctx context.Context, address, serviceName string, plaintext bool) (ServiceInfo, error) {
	conn, err := dial(ctx, address, plaintext)
	if err != nil {
		return ServiceInfo{}, fmt.Errorf("connecting to %s: %w", address, err)
	}
	defer conn.Close()

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	svcDesc, err := refClient.ResolveService(serviceName)
	if err != nil {
		return ServiceInfo{}, fmt.Errorf("resolving service %s: %w", serviceName, err)
	}

	info := ServiceInfo{
		Name: svcDesc.GetFullyQualifiedName(),
	}
	for _, md := range svcDesc.GetMethods() {
		info.Methods = append(info.Methods, MethodInfo{
			Name:       md.GetName(),
			InputType:  md.GetInputType().GetFullyQualifiedName(),
			OutputType: md.GetOutputType().GetFullyQualifiedName(),
			Descriptor: md,
		})
	}

	return info, nil
}
