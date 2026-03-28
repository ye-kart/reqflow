package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	grpcFeature "github.com/ye-kart/reqflow/internal/features/grpc"
)

func newGRPCCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grpc",
		Short: "Interact with gRPC services",
		Long:  "Send gRPC requests, list services, and describe methods using server reflection or proto files.",
	}

	cmd.AddCommand(newGRPCCallCommand())
	cmd.AddCommand(newGRPCListCommand())
	cmd.AddCommand(newGRPCDescribeCommand())

	return cmd
}

func newGRPCCallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call <address> <method>",
		Short: "Call a gRPC method",
		Long:  `Call a unary gRPC method. Method format: "package.Service/Method".`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			method := args[1]

			data, _ := cmd.Flags().GetString("data")
			proto, _ := cmd.Flags().GetString("proto")
			plaintext, _ := cmd.Flags().GetBool("plaintext")
			headers, _ := cmd.Flags().GetStringSlice("header")

			headerMap := make(map[string]string)
			for _, h := range headers {
				key, value, ok := parseHeader(h)
				if ok {
					headerMap[key] = value
				}
			}

			// Use reflection by default unless a proto file is provided.
			useReflection := proto == ""

			caller := grpcFeature.NewCaller()

			timeout, _ := cmd.Flags().GetDuration("timeout")
			ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
			defer cancel()

			result, err := caller.Call(ctx, grpcFeature.CallOptions{
				Address:       address,
				Method:        method,
				Proto:         proto,
				Data:          data,
				Headers:       headerMap,
				Plaintext:     plaintext,
				UseReflection: useReflection,
			})
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()

			// Pretty-print JSON response.
			var prettyJSON json.RawMessage
			if err := json.Unmarshal(result.Response, &prettyJSON); err == nil {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(prettyJSON)
			}

			// Fallback: write raw response.
			fmt.Fprintln(w, string(result.Response))
			return nil
		},
	}

	cmd.Flags().StringP("data", "d", "", "JSON request body")
	cmd.Flags().String("proto", "", "path to .proto file")
	cmd.Flags().Bool("plaintext", false, "use plaintext connection (no TLS)")

	return cmd
}

func newGRPCListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <address>",
		Short: "List gRPC services via reflection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			plaintext, _ := cmd.Flags().GetBool("plaintext")

			caller := grpcFeature.NewCaller()

			timeout, _ := cmd.Flags().GetDuration("timeout")
			ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
			defer cancel()

			services, err := caller.ListServices(ctx, address, plaintext)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			for _, s := range services {
				fmt.Fprintln(w, s)
			}
			return nil
		},
	}

	cmd.Flags().Bool("plaintext", false, "use plaintext connection (no TLS)")
	return cmd
}

func newGRPCDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <address> <service>",
		Short: "Describe a gRPC service's methods via reflection",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			serviceName := args[1]
			plaintext, _ := cmd.Flags().GetBool("plaintext")

			caller := grpcFeature.NewCaller()

			timeout, _ := cmd.Flags().GetDuration("timeout")
			ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
			defer cancel()

			info, err := caller.DescribeService(ctx, address, serviceName, plaintext)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Service: %s\n", info.Name)
			for _, m := range info.Methods {
				fmt.Fprintf(w, "  rpc %s(%s) returns (%s)\n", m.Name, m.InputType, m.OutputType)
			}
			return nil
		},
	}

	cmd.Flags().Bool("plaintext", false, "use plaintext connection (no TLS)")
	return cmd
}
