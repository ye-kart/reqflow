package runner

import (
	"context"
	"strconv"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

// RunWithData executes a workflow once per data row, merging each row's
// variables into the initial vars. Exposes $iteration (0-indexed) and
// $iterationCount as dynamic variables for each iteration.
func (r *Runner) RunWithData(ctx context.Context, wf domain.Workflow, initialVars map[string]string, dataRows []map[string]string) (domain.DataDrivenResult, error) {
	start := time.Now()

	iterationCount := strconv.Itoa(len(dataRows))

	result := domain.DataDrivenResult{
		TotalIterations: len(dataRows),
	}

	for i, row := range dataRows {
		// Build merged vars: initial vars + data row + iteration metadata
		vars := make(map[string]string)
		for k, v := range initialVars {
			vars[k] = v
		}
		for k, v := range row {
			vars[k] = v
		}
		vars["iteration"] = strconv.Itoa(i)
		vars["iterationCount"] = iterationCount

		wfResult, err := r.Run(ctx, wf, vars)
		if err != nil {
			return domain.DataDrivenResult{}, err
		}

		iterResult := domain.IterationResult{
			Iteration: i,
			Variables: row,
			Result:    wfResult,
		}
		result.Iterations = append(result.Iterations, iterResult)

		if wfResult.TotalFailed > 0 {
			result.Failed++
		} else {
			result.Passed++
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}
