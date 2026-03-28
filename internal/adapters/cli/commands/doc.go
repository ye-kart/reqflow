package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/core/exporter"
)

func newDocCommand(a *app.App) *cobra.Command {
	var (
		format    string
		outputFile string
	)

	cmd := &cobra.Command{
		Use:   "doc <collection>",
		Short: "Generate API documentation from a collection",
		Long:  "Generate API documentation in Markdown, HTML, or OpenAPI format from a collection file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("collection-dir")
			name := args[0]
			path := filepath.Join(dir, name+".yaml")

			col, err := a.Storage.ReadCollection(path)
			if err != nil {
				return fmt.Errorf("reading collection %q: %w", name, err)
			}

			var data []byte

			switch format {
			case "markdown", "md":
				data, err = exporter.ExportMarkdown(col)
			case "html":
				data, err = exporter.ExportHTML(col)
			case "openapi":
				data, err = exporter.ExportOpenAPI(col)
			default:
				return fmt.Errorf("unsupported format %q: use markdown, html, or openapi", format)
			}

			if err != nil {
				return fmt.Errorf("generating %s documentation: %w", format, err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0o644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
				cmd.Printf("Documentation written to %s\n", outputFile)
				return nil
			}

			cmd.Print(string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "output format (markdown, html, openapi)")
	cmd.Flags().StringVar(&outputFile, "doc-output", "", "write output to file instead of stdout")
	cmd.Flags().String("collection-dir", defaultCollectionDir(), "directory containing collection files")

	return cmd
}
