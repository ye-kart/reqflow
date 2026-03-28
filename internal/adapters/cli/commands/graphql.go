package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/adapters/cli/output"
	"github.com/ye-kart/reqflow/internal/app"
	coregraphql "github.com/ye-kart/reqflow/internal/core/graphql"
	"github.com/ye-kart/reqflow/internal/core/variable"
	"github.com/ye-kart/reqflow/internal/domain"
)

func newGraphQLCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graphql <url>",
		Short: "Send a GraphQL query",
		Long:  "Send a GraphQL query as an HTTP POST request to the given URL.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]

			// Read query from --query or --query-file.
			query, _ := cmd.Flags().GetString("query")
			queryFile, _ := cmd.Flags().GetString("query-file")

			if query == "" && queryFile == "" {
				return fmt.Errorf("either --query or --query-file is required")
			}

			if queryFile != "" {
				data, err := os.ReadFile(queryFile)
				if err != nil {
					return fmt.Errorf("reading query file: %w", err)
				}
				query = string(data)
			}

			// Parse variables.
			var vars map[string]interface{}
			variablesStr, _ := cmd.Flags().GetString("variables")
			if variablesStr != "" {
				if err := json.Unmarshal([]byte(variablesStr), &vars); err != nil {
					return fmt.Errorf("invalid --variables JSON: %w", err)
				}
			}

			// Parse operation name.
			operationName, _ := cmd.Flags().GetString("operation-name")

			// Build GraphQL body.
			gqlReq := coregraphql.GraphQLRequest{
				Query:         query,
				Variables:     vars,
				OperationName: operationName,
			}

			body, err := coregraphql.BuildGraphQLBody(gqlReq)
			if err != nil {
				return fmt.Errorf("building graphql body: %w", err)
			}

			// Build HTTP request config.
			config := domain.RequestConfig{
				Method:      domain.MethodPost,
				URL:         url,
				Body:        body,
				ContentType: "application/json",
			}

			// Parse timeout.
			timeout, _ := cmd.Flags().GetDuration("timeout")
			if timeout > 0 {
				config.Timeout = timeout
			}

			// Parse headers from persistent flags.
			cfg := configFromContext(cmd.Context())
			var configHeaders []domain.Header
			if cfg != nil {
				configHeaders = cfg.DefaultHeaders
			}
			headers, _ := cmd.Flags().GetStringSlice("header")
			var cliHeaders []domain.Header
			for _, h := range headers {
				key, value, ok := parseHeader(h)
				if ok {
					cliHeaders = append(cliHeaders, domain.Header{Key: key, Value: value})
				}
			}
			config.Headers = mergeHeaders(configHeaders, cliHeaders)

			// Parse auth flags.
			authConfig, err := parseAuthFlags(cmd)
			if err != nil {
				return err
			}
			config.Auth = authConfig

			// Parse cookie flags.
			noCookies, _ := cmd.Flags().GetBool("no-cookies")
			if noCookies {
				config.NoCookies = true
			}

			// Load environment variables if -e flag is set.
			var envVars map[string]string
			envName, _ := cmd.Flags().GetString("env")
			if envName != "" && a.Storage != nil {
				envDir, _ := cmd.Flags().GetString("env-dir")
				envPath := filepath.Join(envDir, envName+".yaml")
				env, err := a.Storage.ReadEnvironment(envPath)
				if err != nil {
					return fmt.Errorf("loading environment %q: %w", envName, err)
				}
				envVars = variable.Resolve(env.Variables)
			}

			// Execute request.
			noColor, _ := cmd.Flags().GetBool("no-color")
			w := cmd.OutOrStdout()

			ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
			defer cancel()

			result, err := a.HTTPExecutor.Execute(ctx, config, envVars)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}

			// Parse GraphQL response.
			gqlResp, err := coregraphql.ParseGraphQLResponse(result.Response.Body)
			if err != nil {
				// Fall back to regular output if not valid GraphQL response.
				outputFmt, _ := cmd.Flags().GetString("output")
				formatter := output.New(domain.OutputFormat(outputFmt), noColor)
				return formatter.FormatResponse(w, result.Response)
			}

			// Format GraphQL response.
			return output.FormatGraphQLResponse(w, gqlResp, noColor)
		},
	}

	// GraphQL-specific flags.
	cmd.Flags().String("query", "", "GraphQL query string")
	cmd.Flags().String("query-file", "", "path to a file containing the GraphQL query")
	cmd.Flags().String("variables", "", "JSON string of GraphQL variables")
	cmd.Flags().String("operation-name", "", "GraphQL operation name")

	// Reuse common flags.
	addAuthFlags(cmd)
	addCookieFlags(cmd)

	return cmd
}
