package domain_test

import (
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestTestReport_Fields(t *testing.T) {
	report := domain.TestReport{
		SuiteName: "User API",
		Results: []domain.TestResult{
			{Name: "Status code is 200", Passed: true},
			{Name: "Body contains name", Passed: false, Error: "not found"},
		},
		Passed:   1,
		Failed:   1,
		Total:    2,
		Duration: 15 * time.Millisecond,
	}

	if report.SuiteName != "User API" {
		t.Errorf("SuiteName = %q, want %q", report.SuiteName, "User API")
	}
	if report.Total != 2 {
		t.Errorf("Total = %d, want 2", report.Total)
	}
	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Failed = %d, want 1", report.Failed)
	}
	if len(report.Results) != 2 {
		t.Errorf("Results count = %d, want 2", len(report.Results))
	}
	if report.Duration != 15*time.Millisecond {
		t.Errorf("Duration = %v, want 15ms", report.Duration)
	}
}

func TestTestSuiteResult_Fields(t *testing.T) {
	suite := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{SuiteName: "Suite A", Total: 3, Passed: 2, Failed: 1},
			{SuiteName: "Suite B", Total: 2, Passed: 2, Failed: 0},
		},
		Total:    5,
		Passed:   4,
		Failed:   1,
		Duration: 100 * time.Millisecond,
	}

	if len(suite.Suites) != 2 {
		t.Errorf("Suites count = %d, want 2", len(suite.Suites))
	}
	if suite.Total != 5 {
		t.Errorf("Total = %d, want 5", suite.Total)
	}
	if suite.Passed != 4 {
		t.Errorf("Passed = %d, want 4", suite.Passed)
	}
	if suite.Failed != 1 {
		t.Errorf("Failed = %d, want 1", suite.Failed)
	}
	if suite.Duration != 100*time.Millisecond {
		t.Errorf("Duration = %v, want 100ms", suite.Duration)
	}
}
