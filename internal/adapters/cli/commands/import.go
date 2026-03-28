package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/adapters/cli/output"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/core/importer"
	"github.com/ye-kart/reqflow/internal/domain"
)

func newImportCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import requests from external formats",
	}

	cmd.AddCommand(newImportCurlCommand(a))
	cmd.AddCommand(newImportFileCommand(a))
	return cmd
}

func newImportCurlCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "curl <curl-command>",
		Short: "Import and execute a cURL command",
		Long:  "Parse a cURL command string and execute the equivalent HTTP request.",
		Args:  cobra.ExactArgs(1),
		RunE:  makeImportCurlRunE(a),
	}

	cmd.Flags().Bool("dry-run", false, "parse and display the request without executing")
	return cmd
}

func makeImportCurlRunE(a *app.App) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		curlCmd := args[0]

		config, err := importer.ParseCurl(curlCmd)
		if err != nil {
			return fmt.Errorf("failed to parse curl command: %w", err)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			return printDryRun(cmd, config)
		}

		// Execute the request.
		timeout := config.Timeout
		if timeout == 0 {
			t, _ := cmd.Flags().GetDuration("timeout")
			timeout = resolveTimeout(t)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		result, err := a.HTTPExecutor.Execute(ctx, config, nil)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		outputFmt, _ := cmd.Flags().GetString("output")
		noColor, _ := cmd.Flags().GetBool("no-color")
		formatter := output.New(domain.OutputFormat(outputFmt), noColor)
		return formatter.FormatResponse(os.Stdout, result.Response)
	}
}

func newImportFileCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <path>",
		Short: "Import a collection from a file (auto-detects format)",
		Long:  "Import requests from a Postman, OpenAPI, HAR, Insomnia, or cURL file.",
		Args:  cobra.ExactArgs(1),
		RunE:  makeImportFileRunE(a),
	}

	cmd.Flags().String("format", "", "explicitly set format (postman, openapi, har, insomnia, curl)")
	cmd.Flags().String("save", "", "save imported collection with given name")
	cmd.Flags().String("collection-dir", defaultCollectionDir(), "directory for saving collections")
	return cmd
}

func makeImportFileRunE(a *app.App) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")

		var col domain.Collection
		if format != "" {
			col, err = importer.ImportWithFormat(data, format)
		} else {
			col, err = importer.Import(data)
		}
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		// Print summary.
		w := cmd.OutOrStdout()
		totalRequests := countRequests(col)
		totalFolders := countFolders(col)

		fmt.Fprintf(w, "Imported: %s\n", col.Name)
		fmt.Fprintf(w, "  %d request(s)", totalRequests)
		if totalFolders > 0 {
			fmt.Fprintf(w, ", %d folder(s)", totalFolders)
		}
		fmt.Fprintln(w)

		// Optionally save as a reqflow collection.
		saveName, _ := cmd.Flags().GetString("save")
		if saveName != "" {
			colDir, _ := cmd.Flags().GetString("collection-dir")
			if err := os.MkdirAll(colDir, 0755); err != nil {
				return fmt.Errorf("creating collection dir: %w", err)
			}
			colPath := filepath.Join(colDir, saveName+".yaml")
			if err := a.Storage.WriteCollection(colPath, col); err != nil {
				return fmt.Errorf("saving collection: %w", err)
			}
			fmt.Fprintf(w, "Saved as collection %q\n", saveName)
		}

		return nil
	}
}

func printDryRun(cmd *cobra.Command, config domain.RequestConfig) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Method:  %s\n", config.Method)
	fmt.Fprintf(w, "URL:     %s\n", config.URL)
	if len(config.Headers) > 0 {
		fmt.Fprintln(w, "Headers:")
		for _, h := range config.Headers {
			fmt.Fprintf(w, "  %s: %s\n", h.Key, h.Value)
		}
	}
	if len(config.Body) > 0 {
		fmt.Fprintf(w, "Body:    %s\n", string(config.Body))
	}
	if config.Auth != nil {
		fmt.Fprintf(w, "Auth:    %s\n", config.Auth.Type)
	}
	return nil
}

func countRequests(col domain.Collection) int {
	n := len(col.Requests)
	for _, f := range col.Folders {
		n += countFolderRequests(f)
	}
	return n
}

func countFolderRequests(f domain.Folder) int {
	n := len(f.Requests)
	for _, sub := range f.Folders {
		n += countFolderRequests(sub)
	}
	return n
}

func countFolders(col domain.Collection) int {
	n := len(col.Folders)
	for _, f := range col.Folders {
		n += countSubFolders(f)
	}
	return n
}

func countSubFolders(f domain.Folder) int {
	n := len(f.Folders)
	for _, sub := range f.Folders {
		n += countSubFolders(sub)
	}
	return n
}
