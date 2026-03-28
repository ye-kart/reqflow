package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/core/exporter"
	"github.com/ye-kart/reqflow/internal/domain"
)

func newExportCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export requests to external formats",
	}

	cmd.AddCommand(newExportCurlCommand())
	cmd.AddCommand(newExportCollectionCommand(a))
	return cmd
}

func newExportCurlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "curl",
		Short: "Export the last request as a cURL command",
		Long:  "Print the equivalent curl command for the last executed request.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("no request to export; use --curl flag on request commands instead")
		},
	}
	return cmd
}

func newExportCollectionCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collection <name>",
		Short: "Export a collection to an external format",
		Long:  "Export a reqflow collection to Postman, OpenAPI, or cURL format.",
		Args:  cobra.ExactArgs(1),
		RunE:  makeExportCollectionRunE(a),
	}

	cmd.Flags().String("format", "postman", "export format (postman, openapi, curl)")
	cmd.Flags().String("collection-dir", defaultCollectionDir(), "directory containing collection files")
	return cmd
}

func makeExportCollectionRunE(a *app.App) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		colName := args[0]
		colDir, _ := cmd.Flags().GetString("collection-dir")
		format, _ := cmd.Flags().GetString("format")

		colPath := filepath.Join(colDir, colName+".yaml")
		col, err := a.Storage.ReadCollection(colPath)
		if err != nil {
			return fmt.Errorf("reading collection %q: %w", colName, err)
		}

		w := cmd.OutOrStdout()

		switch format {
		case "postman":
			data, err := exporter.ExportPostman(col)
			if err != nil {
				return fmt.Errorf("exporting to Postman: %w", err)
			}
			fmt.Fprint(w, string(data))

		case "openapi":
			data, err := exporter.ExportOpenAPI(col)
			if err != nil {
				return fmt.Errorf("exporting to OpenAPI: %w", err)
			}
			fmt.Fprint(w, string(data))

		case "curl":
			requests := collectAllExportRequests(col)
			for i, r := range requests {
				if i > 0 {
					fmt.Fprintln(w)
					fmt.Fprintln(w, "---")
					fmt.Fprintln(w)
				}
				fmt.Fprintf(w, "# %s\n", r.Name)
				fmt.Fprintln(w, exporter.ExportCurl(r.Config))
			}

		default:
			return fmt.Errorf("unsupported export format: %q (use postman, openapi, or curl)", format)
		}

		return nil
	}
}

// printCurlExport writes the curl equivalent of the config to the command output.
func printCurlExport(cmd *cobra.Command, config domain.RequestConfig) error {
	w := cmd.OutOrStdout()
	curlStr := exporter.ExportCurl(config)
	fmt.Fprintln(w, curlStr)
	return nil
}

func collectAllExportRequests(c domain.Collection) []domain.SavedRequest {
	var all []domain.SavedRequest
	all = append(all, c.Requests...)
	for _, f := range c.Folders {
		all = append(all, collectFolderExportRequests(f)...)
	}
	return all
}

func collectFolderExportRequests(f domain.Folder) []domain.SavedRequest {
	var all []domain.SavedRequest
	all = append(all, f.Requests...)
	for _, sub := range f.Folders {
		all = append(all, collectFolderExportRequests(sub)...)
	}
	return all
}
