package domain

import "time"

// IterationResult holds the outcome of executing a workflow for one data row.
type IterationResult struct {
	Iteration int
	Variables map[string]string // data for this iteration
	Result    WorkflowResult
}

// DataDrivenResult holds the aggregate outcome of a data-driven workflow run.
type DataDrivenResult struct {
	Iterations      []IterationResult
	TotalIterations int
	Passed          int
	Failed          int
	Duration        time.Duration
}
