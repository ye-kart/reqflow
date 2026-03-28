package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/core/codegen"
	"github.com/ye-kart/reqflow/internal/domain"
)

func newCodegenCommand(_ *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codegen",
		Short: "Generate code snippets for HTTP requests",
		Long:  fmt.Sprintf("Generate code in various languages for an HTTP request.\nSupported languages: %s", strings.Join(codegen.ListLanguages(), ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			lang, _ := cmd.Flags().GetString("lang")
			if lang == "" {
				return fmt.Errorf("--lang is required (available: %s)", strings.Join(codegen.ListLanguages(), ", "))
			}

			urlStr, _ := cmd.Flags().GetString("url")
			if urlStr == "" {
				return fmt.Errorf("--url is required")
			}

			method, _ := cmd.Flags().GetString("method")
			if method == "" {
				method = "GET"
			}

			req := domain.HTTPRequest{
				Method: domain.HTTPMethod(strings.ToUpper(method)),
				URL:    urlStr,
			}

			// Parse headers from persistent flags.
			headers, _ := cmd.Flags().GetStringSlice("header")
			for _, h := range headers {
				key, value, ok := parseHeader(h)
				if ok {
					req.Headers = append(req.Headers, domain.Header{Key: key, Value: value})
				}
			}

			// Parse body.
			data, _ := cmd.Flags().GetString("data")
			if data != "" {
				req.Body = []byte(data)
			}

			code, err := codegen.Generate(lang, req)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), code)
			return nil
		},
	}

	cmd.Flags().String("lang", "", "target language (python, javascript, go, java, ruby, php, csharp, curl)")
	cmd.Flags().String("url", "", "request URL")
	cmd.Flags().String("method", "GET", "HTTP method")
	cmd.Flags().StringP("data", "d", "", "request body")

	return cmd
}

// addCodegenFlag adds the --codegen flag for generating code instead of executing.
func addCodegenFlag(cmd *cobra.Command) {
	cmd.Flags().String("codegen", "", "generate code instead of executing (python, javascript, go, java, ruby, php, csharp, curl)")
}

// printCodegen generates code for the given language and request config, then prints it.
func printCodegen(cmd *cobra.Command, lang string, config domain.RequestConfig) error {
	// Convert RequestConfig to HTTPRequest for the codegen package.
	req := domain.HTTPRequest{
		Method:  config.Method,
		URL:     config.URL,
		Headers: config.Headers,
		Body:    config.Body,
	}

	code, err := codegen.Generate(lang, req)
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), code)
	return nil
}
