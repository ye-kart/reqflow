package commands

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/features/loadtest"
)

func newPerfCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perf <url>",
		Short: "Run a performance/load test against a URL",
		Long: `Run a load test by sending concurrent requests to a URL.

Configurable virtual users (VUs), duration, and optional ramp-up period.
Displays real-time progress and a summary with latency percentiles.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]

			method, _ := cmd.Flags().GetString("method")
			vus, _ := cmd.Flags().GetInt("vus")
			duration, _ := cmd.Flags().GetDuration("duration")
			rampUp, _ := cmd.Flags().GetDuration("ramp-up")
			data, _ := cmd.Flags().GetString("data")

			// Parse headers from the persistent -H flag.
			headerSlice, _ := cmd.Flags().GetStringSlice("header")
			headers := make(map[string]string)
			for _, h := range headerSlice {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}

			var body []byte
			if data != "" {
				body = []byte(data)
			}

			cfg := loadtest.Config{
				URL:      url,
				Method:   strings.ToUpper(method),
				Headers:  headers,
				Body:     body,
				VUs:      vus,
				Duration: duration,
				RampUp:   rampUp,
			}

			w := cmd.OutOrStdout()

			// Progress channel for real-time updates.
			progress := make(chan loadtest.Snapshot, 64)
			done := make(chan struct{})
			go func() {
				for snap := range progress {
					printProgress(w, snap)
				}
				close(done)
			}()

			result, err := a.LoadTestEngine.Run(cmd.Context(), cfg, progress)
			close(progress)
			<-done

			if err != nil {
				return err
			}

			printResult(w, result)
			return nil
		},
	}

	cmd.Flags().String("method", "GET", "HTTP method")
	cmd.Flags().Int("vus", 10, "number of virtual users (concurrent connections)")
	cmd.Flags().Duration("duration", 30*time.Second, "test duration")
	cmd.Flags().Duration("ramp-up", 0, "time to ramp up to full VU count")
	cmd.Flags().StringP("data", "d", "", "request body")

	return cmd
}

func printProgress(w io.Writer, snap loadtest.Snapshot) {
	fmt.Fprintf(w, "\r  [%s] VUs: %d | Requests: %d | RPS: %.1f",
		snap.Elapsed.Round(time.Second), snap.VUs, snap.Requests, snap.RPS)
}

func printResult(w io.Writer, r loadtest.Result) {
	fmt.Fprintf(w, "\n\n")
	fmt.Fprintf(w, "  Load Test Results\n")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 40))
	fmt.Fprintf(w, "  Total Requests:   %d\n", r.TotalRequests)
	fmt.Fprintf(w, "  Successes:        %d\n", r.Successes)
	fmt.Fprintf(w, "  Failures:         %d\n", r.Failures)
	fmt.Fprintf(w, "  Error Rate:       %.2f%%\n", r.ErrorRate*100)
	fmt.Fprintf(w, "  Duration:         %s\n", r.Duration.Round(time.Millisecond))
	fmt.Fprintf(w, "  Requests/sec:     %.2f\n", r.RequestsPerSec)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  Latency\n")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 40))
	fmt.Fprintf(w, "  Average:          %s\n", r.Average.Round(time.Microsecond))
	fmt.Fprintf(w, "  Min:              %s\n", r.Min.Round(time.Microsecond))
	fmt.Fprintf(w, "  Max:              %s\n", r.Max.Round(time.Microsecond))
	fmt.Fprintf(w, "  P50:              %s\n", r.P50.Round(time.Microsecond))
	fmt.Fprintf(w, "  P90:              %s\n", r.P90.Round(time.Microsecond))
	fmt.Fprintf(w, "  P95:              %s\n", r.P95.Round(time.Microsecond))
	fmt.Fprintf(w, "  P99:              %s\n", r.P99.Round(time.Microsecond))
}
