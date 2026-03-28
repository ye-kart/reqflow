package output

import (
	"fmt"
	"io"

	"github.com/ye-kart/reqflow/internal/core/graphql"
)

// FormatGraphQLResponse writes a pretty-formatted GraphQL response to the
// writer, showing data and errors separately. Errors are highlighted in red
// unless noColor is true.
func FormatGraphQLResponse(w io.Writer, resp graphql.GraphQLResponse, noColor bool) error {
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		fmt.Fprintln(w, bold("Data:"))
		fmt.Fprintln(w, prettyPrintBody(resp.Data))
	}

	if resp.HasErrors() {
		if len(resp.Data) > 0 && string(resp.Data) != "null" {
			fmt.Fprintln(w) // separator between data and errors
		}

		header := "Errors:"
		if !noColor {
			header = red(bold(header))
		}
		fmt.Fprintln(w, header)

		for i, gqlErr := range resp.Errors {
			prefix := fmt.Sprintf("  %d. ", i+1)
			msg := gqlErr.Message
			if !noColor {
				msg = red(msg)
			}
			fmt.Fprintf(w, "%s%s\n", prefix, msg)

			for _, loc := range gqlErr.Locations {
				fmt.Fprintf(w, "     at line %d, column %d\n", loc.Line, loc.Column)
			}

			if len(gqlErr.Path) > 0 {
				fmt.Fprintf(w, "     path: %v\n", gqlErr.Path)
			}
		}
	}

	return nil
}
