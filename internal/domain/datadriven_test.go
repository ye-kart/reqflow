package domain_test

import (
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestIterationResult_HoldsIterationData(t *testing.T) {
	ir := domain.IterationResult{
		Iteration: 0,
		Variables: map[string]string{"name": "John"},
		Result: domain.WorkflowResult{
			Name:        "test",
			TotalPassed: 1,
			Duration:    100 * time.Millisecond,
		},
	}

	if ir.Iteration != 0 {
		t.Errorf("Iteration = %d, want 0", ir.Iteration)
	}
	if ir.Variables["name"] != "John" {
		t.Errorf("Variables[name] = %q, want %q", ir.Variables["name"], "John")
	}
	if ir.Result.TotalPassed != 1 {
		t.Errorf("Result.TotalPassed = %d, want 1", ir.Result.TotalPassed)
	}
}

func TestDataDrivenResult_AggregatesSummary(t *testing.T) {
	ddr := domain.DataDrivenResult{
		Iterations: []domain.IterationResult{
			{Iteration: 0, Variables: map[string]string{"name": "John"}},
			{Iteration: 1, Variables: map[string]string{"name": "Jane"}},
		},
		TotalIterations: 2,
		Passed:          1,
		Failed:          1,
		Duration:        200 * time.Millisecond,
	}

	if ddr.TotalIterations != 2 {
		t.Errorf("TotalIterations = %d, want 2", ddr.TotalIterations)
	}
	if ddr.Passed != 1 {
		t.Errorf("Passed = %d, want 1", ddr.Passed)
	}
	if ddr.Failed != 1 {
		t.Errorf("Failed = %d, want 1", ddr.Failed)
	}
	if len(ddr.Iterations) != 2 {
		t.Errorf("len(Iterations) = %d, want 2", len(ddr.Iterations))
	}
}
