package output

import (
	"fmt"
	"io"

	"github.com/ye-kart/reqflow/internal/domain"
)

// FormatTestReport writes a BDD-style test report to the given writer.
func FormatTestReport(w io.Writer, report domain.TestSuiteResult, noColor bool) error {
	green := "\033[32m"
	red := "\033[31m"
	bold := "\033[1m"
	reset := "\033[0m"
	dim := "\033[2m"

	if noColor {
		green = ""
		red = ""
		bold = ""
		reset = ""
		dim = ""
	}

	checkMark := green + "\u2713" + reset
	crossMark := red + "\u2717" + reset

	for _, suite := range report.Suites {
		fmt.Fprintf(w, "%s%s%s\n", bold, suite.SuiteName, reset)

		for _, tr := range suite.Results {
			if tr.Passed {
				fmt.Fprintf(w, "  %s %s\n", checkMark, tr.Name)
			} else {
				fmt.Fprintf(w, "  %s %s\n", crossMark, tr.Name)
				if tr.Error != "" {
					fmt.Fprintf(w, "    %s%s%s\n", dim, tr.Error, reset)
				}
			}
		}

		fmt.Fprintln(w)
	}

	// Summary line.
	fmt.Fprintf(w, "  %s%d passed%s, %s%d failed%s %s(%s)%s\n",
		green, report.Passed, reset,
		red, report.Failed, reset,
		dim, report.Duration.Round(1*1e6), reset, // round to ms
	)

	return nil
}
