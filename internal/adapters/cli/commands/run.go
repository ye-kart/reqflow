package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/core/datafile"
	"github.com/ye-kart/reqflow/internal/core/variable"
	"github.com/ye-kart/reqflow/internal/core/workflow"
	"github.com/ye-kart/reqflow/internal/domain"
)

func newRunCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <workflow.yaml>",
		Short: "Execute a workflow",
		Long:  "Execute a workflow defined in a YAML file with sequential HTTP steps, variable extraction, and assertions.",
		Args:  cobra.ExactArgs(1),
		RunE:  makeRunWorkflowE(a),
	}

	cmd.Flags().String("data", "", "data file for iteration (CSV or JSON)")

	return cmd
}

func makeRunWorkflowE(a *app.App) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		// Read the workflow file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading workflow file: %w", err)
		}

		// Parse the workflow
		wf, err := workflow.Parse(data)
		if err != nil {
			return fmt.Errorf("parsing workflow: %w", err)
		}

		// Load environment variables if specified
		vars := make(map[string]string)
		envName, _ := cmd.Flags().GetString("env")
		if envName == "" {
			envName = wf.Env
		}
		if envName != "" && a.Storage != nil {
			envDir, _ := cmd.Flags().GetString("env-dir")
			envPath := filepath.Join(envDir, envName+".yaml")
			env, err := a.Storage.ReadEnvironment(envPath)
			if err != nil {
				return fmt.Errorf("loading environment %q: %w", envName, err)
			}
			vars = variable.Resolve(env.Variables)
		}

		w := cmd.OutOrStdout()

		// Dry-run mode: validate and show workflow summary
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			return formatDryRunWorkflow(w, wf)
		}

		// Execute the workflow
		if a.Runner == nil {
			return fmt.Errorf("workflow runner not initialized")
		}

		timeout, _ := cmd.Flags().GetDuration("timeout")
		ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout(timeout))
		defer cancel()

		outputFmt, _ := cmd.Flags().GetString("output")
		noColor, _ := cmd.Flags().GetBool("no-color")

		// Data-driven mode
		dataPath, _ := cmd.Flags().GetString("data")
		if dataPath != "" {
			dataRows, err := datafile.ParseFile(dataPath)
			if err != nil {
				return fmt.Errorf("parsing data file: %w", err)
			}

			ddResult, err := a.Runner.RunWithData(ctx, wf, vars, dataRows)
			if err != nil {
				return fmt.Errorf("running data-driven workflow: %w", err)
			}

			if outputFmt == "json" {
				return formatDataDrivenJSON(w, ddResult)
			}
			return formatDataDrivenPretty(w, ddResult, noColor)
		}

		result, err := a.Runner.Run(ctx, wf, vars)
		if err != nil {
			return fmt.Errorf("running workflow: %w", err)
		}

		// Format output
		if outputFmt == "json" {
			return formatWorkflowJSON(w, result)
		}

		return formatWorkflowPretty(w, result, noColor)
	}
}

func formatDryRunWorkflow(w io.Writer, wf domain.Workflow) error {
	fmt.Fprintf(w, "Workflow: %s\n", wf.Name)
	if wf.Env != "" {
		fmt.Fprintf(w, "Environment: %s\n", wf.Env)
	}
	fmt.Fprintf(w, "Steps: %d\n\n", len(wf.Steps))

	for i, step := range wf.Steps {
		fmt.Fprintf(w, "  %d. [%s] %s %s\n", i+1, step.Name, step.Method, step.URL)
		if len(step.Headers) > 0 {
			fmt.Fprintf(w, "     Headers: %d\n", len(step.Headers))
		}
		if step.Body != nil {
			fmt.Fprintf(w, "     Body: yes\n")
		}
		if len(step.Extract) > 0 {
			fmt.Fprintf(w, "     Extract: %s\n", strings.Join(mapKeys(step.Extract), ", "))
		}
		if len(step.Assert) > 0 {
			fmt.Fprintf(w, "     Assertions: %d\n", len(step.Assert))
		}
	}

	return nil
}

type wfJSONAssertionResult struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Expected interface{} `json:"expected,omitempty"`
	Actual   interface{} `json:"actual,omitempty"`
	Passed   bool        `json:"passed"`
	Message  string      `json:"message,omitempty"`
}

type wfJSONStepResult struct {
	StepName   string                   `json:"step_name"`
	Assertions []wfJSONAssertionResult  `json:"assertions,omitempty"`
	Extracted  map[string]string        `json:"extracted,omitempty"`
	Error      string                   `json:"error,omitempty"`
	Duration   string                   `json:"duration"`
}

type wfJSONResult struct {
	Name        string             `json:"name"`
	Steps       []wfJSONStepResult `json:"steps"`
	TotalPassed int                `json:"total_passed"`
	TotalFailed int                `json:"total_failed"`
	Duration    string             `json:"duration"`
}

func formatWorkflowJSON(w io.Writer, result domain.WorkflowResult) error {
	jr := wfJSONResult{
		Name:        result.Name,
		TotalPassed: result.TotalPassed,
		TotalFailed: result.TotalFailed,
		Duration:    result.Duration.String(),
	}

	for _, sr := range result.Steps {
		jsr := wfJSONStepResult{
			StepName:  sr.StepName,
			Extracted: sr.Extracted,
			Duration:  sr.Duration.String(),
		}
		if sr.Error != nil {
			jsr.Error = sr.Error.Error()
		}
		for _, ar := range sr.Assertions {
			jsr.Assertions = append(jsr.Assertions, wfJSONAssertionResult{
				Field:    ar.Assertion.Field,
				Operator: ar.Assertion.Operator,
				Expected: ar.Assertion.Expected,
				Actual:   ar.Actual,
				Passed:   ar.Passed,
				Message:  ar.Message,
			})
		}
		jr.Steps = append(jr.Steps, jsr)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jr)
}

func formatWorkflowPretty(w io.Writer, result domain.WorkflowResult, noColor bool) error {
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

	fmt.Fprintf(w, "%sWorkflow: %s%s\n\n", bold, result.Name, reset)

	for i, sr := range result.Steps {
		icon := green + "PASS" + reset
		if sr.Error != nil {
			icon = red + "ERROR" + reset
		} else {
			for _, ar := range sr.Assertions {
				if !ar.Passed {
					icon = red + "FAIL" + reset
					break
				}
			}
		}

		fmt.Fprintf(w, "  %d. [%s] %s %s(%s)%s\n", i+1, icon, sr.StepName, dim, sr.Duration.Round(time.Millisecond), reset)

		if sr.Error != nil {
			fmt.Fprintf(w, "     %sError: %s%s\n", red, sr.Error, reset)
			continue
		}

		for _, ar := range sr.Assertions {
			if ar.Passed {
				fmt.Fprintf(w, "     %s✓%s %s %s %v\n", green, reset, ar.Assertion.Field, ar.Assertion.Operator, ar.Assertion.Expected)
			} else {
				fmt.Fprintf(w, "     %s✗%s %s (got: %v)\n", red, reset, ar.Message, ar.Actual)
			}
		}

		if len(sr.Extracted) > 0 {
			for k, v := range sr.Extracted {
				fmt.Fprintf(w, "     %s→ %s = %s%s\n", dim, k, v, reset)
			}
		}
	}

	fmt.Fprintln(w)
	passStr := fmt.Sprintf("%d passed", result.TotalPassed)
	failStr := fmt.Sprintf("%d failed", result.TotalFailed)
	if result.TotalFailed > 0 {
		fmt.Fprintf(w, "%s%s%s, %s%s%s %s(%s)%s\n", red, failStr, reset, green, passStr, reset, dim, result.Duration.Round(time.Millisecond), reset)
	} else {
		fmt.Fprintf(w, "%s%s%s %s(%s)%s\n", green, passStr, reset, dim, result.Duration.Round(time.Millisecond), reset)
	}

	return nil
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// --- Data-driven output formatters ---

type ddJSONIterationResult struct {
	Iteration int               `json:"iteration"`
	Variables map[string]string `json:"variables"`
	Result    wfJSONResult      `json:"result"`
}

type ddJSONResult struct {
	Iterations      []ddJSONIterationResult `json:"iterations"`
	TotalIterations int                     `json:"total_iterations"`
	Passed          int                     `json:"passed"`
	Failed          int                     `json:"failed"`
	Duration        string                  `json:"duration"`
}

func formatDataDrivenJSON(w io.Writer, result domain.DataDrivenResult) error {
	jr := ddJSONResult{
		TotalIterations: result.TotalIterations,
		Passed:          result.Passed,
		Failed:          result.Failed,
		Duration:        result.Duration.String(),
	}

	for _, iter := range result.Iterations {
		iterJSON := ddJSONIterationResult{
			Iteration: iter.Iteration,
			Variables: iter.Variables,
			Result: wfJSONResult{
				Name:        iter.Result.Name,
				TotalPassed: iter.Result.TotalPassed,
				TotalFailed: iter.Result.TotalFailed,
				Duration:    iter.Result.Duration.String(),
			},
		}
		for _, sr := range iter.Result.Steps {
			jsr := wfJSONStepResult{
				StepName:  sr.StepName,
				Extracted: sr.Extracted,
				Duration:  sr.Duration.String(),
			}
			if sr.Error != nil {
				jsr.Error = sr.Error.Error()
			}
			for _, ar := range sr.Assertions {
				jsr.Assertions = append(jsr.Assertions, wfJSONAssertionResult{
					Field:    ar.Assertion.Field,
					Operator: ar.Assertion.Operator,
					Expected: ar.Assertion.Expected,
					Actual:   ar.Actual,
					Passed:   ar.Passed,
					Message:  ar.Message,
				})
			}
			iterJSON.Result.Steps = append(iterJSON.Result.Steps, jsr)
		}
		jr.Iterations = append(jr.Iterations, iterJSON)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jr)
}

func formatDataDrivenPretty(w io.Writer, result domain.DataDrivenResult, noColor bool) error {
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

	for _, iter := range result.Iterations {
		varParts := make([]string, 0, len(iter.Variables))
		for k, v := range iter.Variables {
			varParts = append(varParts, fmt.Sprintf("%s=%s", k, v))
		}
		varStr := strings.Join(varParts, ", ")
		fmt.Fprintf(w, "%sIteration %d%s %s(%s)%s\n", bold, iter.Iteration+1, reset, dim, varStr, reset)

		wfResult := iter.Result
		for i, sr := range wfResult.Steps {
			icon := green + "PASS" + reset
			if sr.Error != nil {
				icon = red + "ERROR" + reset
			} else {
				for _, ar := range sr.Assertions {
					if !ar.Passed {
						icon = red + "FAIL" + reset
						break
					}
				}
			}

			fmt.Fprintf(w, "  %d. [%s] %s %s(%s)%s\n", i+1, icon, sr.StepName, dim, sr.Duration.Round(time.Millisecond), reset)

			if sr.Error != nil {
				fmt.Fprintf(w, "     %sError: %s%s\n", red, sr.Error, reset)
				continue
			}

			for _, ar := range sr.Assertions {
				if ar.Passed {
					fmt.Fprintf(w, "     %s✓%s %s %s %v\n", green, reset, ar.Assertion.Field, ar.Assertion.Operator, ar.Assertion.Expected)
				} else {
					fmt.Fprintf(w, "     %s✗%s %s (got: %v)\n", red, reset, ar.Message, ar.Actual)
				}
			}

			if len(sr.Extracted) > 0 {
				for k, v := range sr.Extracted {
					fmt.Fprintf(w, "     %s-> %s = %s%s\n", dim, k, v, reset)
				}
			}
		}
		fmt.Fprintln(w)
	}

	// Summary
	fmt.Fprintf(w, "%s%d iterations%s: ", bold, result.TotalIterations, reset)
	if result.Failed > 0 {
		fmt.Fprintf(w, "%s%d failed%s, %s%d passed%s", red, result.Failed, reset, green, result.Passed, reset)
	} else {
		fmt.Fprintf(w, "%s%d passed%s", green, result.Passed, reset)
	}
	fmt.Fprintf(w, " %s(%s)%s\n", dim, result.Duration.Round(time.Millisecond), reset)

	return nil
}
