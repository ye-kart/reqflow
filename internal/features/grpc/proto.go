package grpc

import (
	"fmt"
	"path/filepath"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
)

// ServiceInfo describes a gRPC service discovered from a proto file or reflection.
type ServiceInfo struct {
	Name    string       // fully qualified service name
	Methods []MethodInfo // methods in the service
}

// MethodInfo describes a single gRPC method.
type MethodInfo struct {
	Name       string               // method name
	InputType  string               // fully qualified input message type
	OutputType string               // fully qualified output message type
	Descriptor *desc.MethodDescriptor // the underlying descriptor (for making calls)
}

// ParseProtoFile parses a .proto file and returns all services and methods.
func ParseProtoFile(protoPath string) ([]ServiceInfo, error) {
	dir := filepath.Dir(protoPath)
	base := filepath.Base(protoPath)

	parser := protoparse.Parser{
		ImportPaths: []string{dir},
	}

	fds, err := parser.ParseFiles(base)
	if err != nil {
		return nil, fmt.Errorf("parsing proto file %s: %w", protoPath, err)
	}

	var services []ServiceInfo
	for _, fd := range fds {
		for _, sd := range fd.GetServices() {
			svc := ServiceInfo{
				Name: sd.GetFullyQualifiedName(),
			}
			for _, md := range sd.GetMethods() {
				svc.Methods = append(svc.Methods, MethodInfo{
					Name:       md.GetName(),
					InputType:  md.GetInputType().GetFullyQualifiedName(),
					OutputType: md.GetOutputType().GetFullyQualifiedName(),
					Descriptor: md,
				})
			}
			services = append(services, svc)
		}
	}

	return services, nil
}
