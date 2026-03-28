package output_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/adapters/cli/output"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestWorkflowToTestSuite_ConvertsAssertions(t *testing.T) {
	wfResult := domain.WorkflowResult{
		Name: "Test Workflow",
		Steps: []domain.StepResult{
			{
				StepName: "Get User",
				Assertions: []domain.AssertionResult{
					{
						Assertion: domain.Assertion{Field: "status", Operator: "==", Expected: 200},
						Passed:    true,
					},
					{
						Assertion: domain.Assertion{Field: "body.name", Operator: "==", Expected: "John"},
						Passed:    false,
						Actual:    "Jane",
						Message:   `body.name: expected "John", got "Jane"`,
					},
				},
				Duration: 10 * time.Millisecond,
			},
		},
		TotalPassed: 1,
		TotalFailed: 1,
		Duration:    10 * time.Millisecond,
	}

	suite := output.WorkflowToTestSuite(wfResult)

	if suite.Total != 2 {
		t.Errorf("Total = %d, want 2", suite.Total)
	}
	if suite.Passed != 1 {
		t.Errorf("Passed = %d, want 1", suite.Passed)
	}
	if suite.Failed != 1 {
		t.Errorf("Failed = %d, want 1", suite.Failed)
	}
	if len(suite.Suites) != 1 {
		t.Fatalf("Suites count = %d, want 1", len(suite.Suites))
	}
	if suite.Suites[0].SuiteName != "Get User" {
		t.Errorf("SuiteName = %q, want %q", suite.Suites[0].SuiteName, "Get User")
	}
	if len(suite.Suites[0].Results) != 2 {
		t.Fatalf("Results count = %d, want 2", len(suite.Suites[0].Results))
	}
	if !suite.Suites[0].Results[0].Passed {
		t.Error("expected first result to pass")
	}
	if suite.Suites[0].Results[1].Passed {
		t.Error("expected second result to fail")
	}
}

func TestWorkflowToTestSuite_HandlesStepErrors(t *testing.T) {
	wfResult := domain.WorkflowResult{
		Name: "Error Workflow",
		Steps: []domain.StepResult{
			{
				StepName: "Broken Step",
				Error:    fmt.Errorf("connection refused"),
				Duration: 5 * time.Millisecond,
			},
		},
		TotalPassed: 0,
		TotalFailed: 0,
		Duration:    5 * time.Millisecond,
	}

	suite := output.WorkflowToTestSuite(wfResult)

	if suite.Total != 1 {
		t.Errorf("Total = %d, want 1", suite.Total)
	}
	if suite.Failed != 1 {
		t.Errorf("Failed = %d, want 1", suite.Failed)
	}
	if len(suite.Suites) != 1 {
		t.Fatalf("Suites count = %d, want 1", len(suite.Suites))
	}
	if len(suite.Suites[0].Results) != 1 {
		t.Fatalf("Results count = %d, want 1", len(suite.Suites[0].Results))
	}
	if suite.Suites[0].Results[0].Passed {
		t.Error("expected error step to register as failed")
	}
}

func TestCollectionToTestSuite_ConvertsResults(t *testing.T) {
	colResult := domain.CollectionRunResult{
		CollectionName: "My Collection",
		Results: []domain.RequestRunResult{
			{
				RequestName: "Get Users",
				Passed:      true,
				Duration:    5 * time.Millisecond,
			},
			{
				RequestName: "Get Missing",
				Passed:      false,
				Response:    domain.HTTPResponse{StatusCode: 404},
				Duration:    3 * time.Millisecond,
			},
		},
		Passed:   1,
		Failed:   1,
		Duration: 8 * time.Millisecond,
	}

	suite := output.CollectionToTestSuite(colResult)

	if suite.Total != 2 {
		t.Errorf("Total = %d, want 2", suite.Total)
	}
	if suite.Passed != 1 {
		t.Errorf("Passed = %d, want 1", suite.Passed)
	}
	if suite.Failed != 1 {
		t.Errorf("Failed = %d, want 1", suite.Failed)
	}
	if len(suite.Suites) != 1 {
		t.Fatalf("Suites count = %d, want 1", len(suite.Suites))
	}
	if suite.Suites[0].SuiteName != "My Collection" {
		t.Errorf("SuiteName = %q, want %q", suite.Suites[0].SuiteName, "My Collection")
	}
}
