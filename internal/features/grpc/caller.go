package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// bytesCodec is a gRPC codec that passes raw bytes through without
// marshalling/unmarshalling. This is used per-call via grpc.ForceCodec.
type bytesCodec struct{}

func (bytesCodec) Marshal(v interface{}) ([]byte, error) {
	if b, ok := v.(*[]byte); ok {
		return *b, nil
	}
	return nil, fmt.Errorf("bytesCodec: unsupported type %T", v)
}

func (bytesCodec) Unmarshal(data []byte, v interface{}) error {
	if b, ok := v.(*[]byte); ok {
		*b = append((*b)[:0], data...)
		return nil
	}
	return fmt.Errorf("bytesCodec: unsupported type %T", v)
}

func (bytesCodec) Name() string { return "proto" }

// Ensure bytesCodec implements encoding.Codec.
var _ encoding.Codec = bytesCodec{}

// CallOptions configures a gRPC call.
type CallOptions struct {
	Address       string            // host:port
	Method        string            // "package.Service/Method"
	Proto         string            // path to .proto file (optional)
	Data          string            // JSON request body
	Headers       map[string]string // gRPC metadata
	Plaintext     bool              // no TLS
	UseReflection bool              // use gRPC reflection
}

// CallResult holds the result of a gRPC call.
type CallResult struct {
	Response json.RawMessage   // JSON-encoded response
	Headers  map[string]string // response headers (metadata)
	Trailers map[string]string // response trailers
	Status   string            // gRPC status code name
	Duration time.Duration     // call duration
}

// Caller executes gRPC calls.
type Caller struct{}

// NewCaller creates a new Caller instance.
func NewCaller() *Caller {
	return &Caller{}
}

// Call performs a unary gRPC call based on the provided options.
func (c *Caller) Call(ctx context.Context, opts CallOptions) (CallResult, error) {
	start := time.Now()

	conn, err := dial(ctx, opts.Address, opts.Plaintext)
	if err != nil {
		return CallResult{}, fmt.Errorf("connecting to %s: %w", opts.Address, err)
	}
	defer conn.Close()

	// Resolve the method descriptor.
	methodDesc, err := c.resolveMethod(ctx, conn, opts)
	if err != nil {
		return CallResult{}, fmt.Errorf("resolving method %s: %w", opts.Method, err)
	}

	// Build the request message from JSON and serialize to protobuf bytes.
	reqMsg := dynamic.NewMessage(methodDesc.GetInputType())
	if opts.Data != "" {
		if err := reqMsg.UnmarshalJSON([]byte(opts.Data)); err != nil {
			return CallResult{}, fmt.Errorf("unmarshalling request JSON: %w", err)
		}
	}
	reqBytes, err := reqMsg.Marshal()
	if err != nil {
		return CallResult{}, fmt.Errorf("marshalling request: %w", err)
	}

	// Set up outgoing metadata (headers).
	if len(opts.Headers) > 0 {
		md := metadata.New(opts.Headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// Build the full gRPC method path: /package.Service/Method
	serviceName, methodName, _ := splitMethod(opts.Method)
	fullMethod := "/" + serviceName + "/" + methodName

	// Invoke the RPC using raw bytes codec to avoid marshalling conflicts
	// between jhump/protoreflect dynamic messages and the standard proto codec.
	var respBytes []byte
	var respHeaders metadata.MD
	var respTrailers metadata.MD

	err = conn.Invoke(ctx, fullMethod, &reqBytes, &respBytes,
		grpc.ForceCodec(bytesCodec{}),
		grpc.Header(&respHeaders),
		grpc.Trailer(&respTrailers),
	)

	duration := time.Since(start)

	result := CallResult{
		Headers:  flattenMetadata(respHeaders),
		Trailers: flattenMetadata(respTrailers),
		Duration: duration,
	}

	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			result.Status = st.Code().String()
			return result, fmt.Errorf("rpc error: code = %s desc = %s", st.Code(), st.Message())
		}
		return result, fmt.Errorf("rpc call failed: %w", err)
	}

	// Unmarshal the raw response bytes into a dynamic message and convert to JSON.
	respMsg := dynamic.NewMessage(methodDesc.GetOutputType())
	if err := respMsg.Unmarshal(respBytes); err != nil {
		return result, fmt.Errorf("unmarshalling response: %w", err)
	}
	respJSON, err := respMsg.MarshalJSON()
	if err != nil {
		return result, fmt.Errorf("marshalling response to JSON: %w", err)
	}
	result.Response = respJSON
	result.Status = "OK"

	return result, nil
}

// resolveMethod gets the method descriptor either via reflection or from a proto file.
func (c *Caller) resolveMethod(ctx context.Context, conn *grpc.ClientConn, opts CallOptions) (*desc.MethodDescriptor, error) {
	serviceName, methodName, err := splitMethod(opts.Method)
	if err != nil {
		return nil, err
	}

	if opts.Proto != "" {
		services, err := ParseProtoFile(opts.Proto)
		if err != nil {
			return nil, fmt.Errorf("parsing proto file: %w", err)
		}
		for _, svc := range services {
			if svc.Name == serviceName {
				for _, m := range svc.Methods {
					if m.Name == methodName {
						return m.Descriptor, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("method %s not found in proto file %s", opts.Method, opts.Proto)
	}

	if opts.UseReflection {
		refClient := grpcreflect.NewClientAuto(ctx, conn)
		defer refClient.Reset()

		svcDesc, err := refClient.ResolveService(serviceName)
		if err != nil {
			return nil, fmt.Errorf("resolving service %s via reflection: %w", serviceName, err)
		}
		md := svcDesc.FindMethodByName(methodName)
		if md == nil {
			return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
		}
		return md, nil
	}

	return nil, fmt.Errorf("either --proto or --reflection must be specified to resolve method %s", opts.Method)
}

// splitMethod splits "package.Service/Method" into service name and method name.
func splitMethod(fullMethod string) (string, string, error) {
	parts := strings.SplitN(fullMethod, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid method format %q, expected 'package.Service/Method'", fullMethod)
	}
	return parts[0], parts[1], nil
}

// dial creates a gRPC client connection.
func dial(ctx context.Context, address string, plaintext bool) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	if plaintext {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// flattenMetadata converts gRPC metadata to a flat map (first value only).
func flattenMetadata(md metadata.MD) map[string]string {
	if md == nil {
		return nil
	}
	result := make(map[string]string, len(md))
	for k, v := range md {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}
