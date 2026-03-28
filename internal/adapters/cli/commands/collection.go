package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/domain"
)

// defaultCollectionDir returns the default collection directory (~/.reqflow/collections/).
func defaultCollectionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".reqflow/collections"
	}
	return filepath.Join(home, ".reqflow", "collections")
}

func newCollectionCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collection",
		Short: "Manage collections",
		Long:  "Manage collection files for organizing HTTP requests into groups.",
	}

	cmd.PersistentFlags().String("collection-dir", defaultCollectionDir(), "directory containing collection files")

	cmd.AddCommand(newCollectionListCommand(a))
	cmd.AddCommand(newCollectionShowCommand(a))
	cmd.AddCommand(newCollectionCreateCommand(a))
	cmd.AddCommand(newCollectionAddCommand(a))
	cmd.AddCommand(newCollectionRunCommand(a))

	return cmd
}

func newCollectionListCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available collections",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, _ := cmd.Flags().GetString("collection-dir")

			names, err := a.Storage.ListCollections(dir)
			if err != nil {
				return fmt.Errorf("listing collections: %w", err)
			}

			if len(names) == 0 {
				cmd.Println("No collections found.")
				return nil
			}

			for _, name := range names {
				cmd.Println(name)
			}
			return nil
		},
	}
}

func newCollectionShowCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Display collection contents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("collection-dir")
			name := args[0]
			path := filepath.Join(dir, name+".yaml")

			col, err := a.Storage.ReadCollection(path)
			if err != nil {
				return fmt.Errorf("reading collection %q: %w", name, err)
			}

			cmd.Printf("Collection: %s\n", col.Name)
			if col.Description != "" {
				cmd.Printf("Description: %s\n", col.Description)
			}

			if len(col.Requests) > 0 {
				cmd.Println("\nRequests:")
				for _, r := range col.Requests {
					cmd.Printf("  %s %s  %s\n", r.Config.Method, r.Config.URL, r.Name)
				}
			}

			if len(col.Folders) > 0 {
				printFolders(cmd, col.Folders, "")
			}

			return nil
		},
	}
}

func printFolders(cmd *cobra.Command, folders []domain.Folder, indent string) {
	for _, f := range folders {
		cmd.Printf("\n%sFolder: %s\n", indent, f.Name)
		if f.Description != "" {
			cmd.Printf("%s  Description: %s\n", indent, f.Description)
		}
		for _, r := range f.Requests {
			cmd.Printf("%s  %s %s  %s\n", indent, r.Config.Method, r.Config.URL, r.Name)
		}
		if len(f.Folders) > 0 {
			printFolders(cmd, f.Folders, indent+"  ")
		}
	}
}

func newCollectionCreateCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new empty collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("collection-dir")
			name := args[0]
			path := filepath.Join(dir, name+".yaml")

			col := domain.Collection{
				Name: name,
			}

			if err := a.Storage.WriteCollection(path, col); err != nil {
				return fmt.Errorf("creating collection %q: %w", name, err)
			}

			cmd.Printf("Created collection %q\n", name)
			return nil
		},
	}
}

func newCollectionAddCommand(a *app.App) *cobra.Command {
	var method, url string

	cmd := &cobra.Command{
		Use:   "add <collection> <name>",
		Short: "Add a request to a collection",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("collection-dir")
			colName := args[0]
			reqName := args[1]
			path := filepath.Join(dir, colName+".yaml")

			// Read existing collection.
			col, err := a.Storage.ReadCollection(path)
			if err != nil {
				return fmt.Errorf("reading collection %q: %w", colName, err)
			}

			// Add the new request.
			col.Requests = append(col.Requests, domain.SavedRequest{
				Name: reqName,
				Config: domain.RequestConfig{
					Method: domain.HTTPMethod(method),
					URL:    url,
				},
			})

			if err := a.Storage.WriteCollection(path, col); err != nil {
				return fmt.Errorf("writing collection %q: %w", colName, err)
			}

			cmd.Printf("Added request %q to collection %q\n", reqName, colName)
			return nil
		},
	}

	cmd.Flags().StringVar(&method, "method", "GET", "HTTP method")
	cmd.Flags().StringVar(&url, "url", "", "request URL")
	_ = cmd.MarkFlagRequired("url")

	return cmd
}
