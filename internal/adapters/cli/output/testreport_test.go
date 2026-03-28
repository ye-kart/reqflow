package output_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/adapters/cli/output"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestFormatTestReport_AllPassed(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "User API",
				Results: []domain.TestResult{
					{Name: "Status code is 200", Passed: true},
					{Name: "Response time < 500ms", Passed: true},
				},
				Passed:   2,
				Failed:   0,
				Total:    2,
				Duration: 15 * time.Millisecond,
			},
		},
		Total:    2,
		Passed:   2,
		Failed:   0,
		Duration: 15 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := output.FormatTestReport(&buf, result, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	// Check for checkmarks (no-color mode uses plain text symbols).
	if !strings.Contains(out, "Status code is 200") {
		t.Errorf("expected output to contain test name, got:\n%s", out)
	}
	if !strings.Contains(out, "Response time < 500ms") {
		t.Errorf("expected output to contain test name, got:\n%s", out)
	}
	if !strings.Contains(out, "2 passed") {
		t.Errorf("expected summary '2 passed', got:\n%s", out)
	}
	if !strings.Contains(out, "0 failed") {
		t.Errorf("expected summary '0 failed', got:\n%s", out)
	}
}

func TestFormatTestReport_SomeFailed(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "User API",
				Results: []domain.TestResult{
					{Name: "Status code is 200", Passed: true},
					{Name: "Body contains user name", Passed: false, Error: `Expected: "John", Actual: "Jane"`},
					{Name: "Response time < 500ms", Passed: true},
				},
				Passed:   2,
				Failed:   1,
				Total:    3,
				Duration: 15 * time.Millisecond,
			},
		},
		Total:    3,
		Passed:   2,
		Failed:   1,
		Duration: 15 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := output.FormatTestReport(&buf, result, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	// Verify failed test shows expected/actual.
	if !strings.Contains(out, "Body contains user name") {
		t.Errorf("expected failed test name in output, got:\n%s", out)
	}
	if !strings.Contains(out, `Expected: "John", Actual: "Jane"`) {
		t.Errorf("expected error details in output, got:\n%s", out)
	}
	if !strings.Contains(out, "2 passed") {
		t.Errorf("expected summary '2 passed', got:\n%s", out)
	}
	if !strings.Contains(out, "1 failed") {
		t.Errorf("expected summary '1 failed', got:\n%s", out)
	}
}

func TestFormatTestReport_NoColor(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "Test",
				Results: []domain.TestResult{
					{Name: "passes", Passed: true},
				},
				Passed:   1,
				Failed:   0,
				Total:    1,
				Duration: 5 * time.Millisecond,
			},
		},
		Total:    1,
		Passed:   1,
		Failed:   0,
		Duration: 5 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := output.FormatTestReport(&buf, result, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	// No ANSI escape codes should be present.
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no color codes in no-color mode, got:\n%s", out)
	}
}

func TestFormatTestReport_WithColor(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "Test",
				Results: []domain.TestResult{
					{Name: "passes", Passed: true},
					{Name: "fails", Passed: false, Error: "nope"},
				},
				Passed:   1,
				Failed:   1,
				Total:    2,
				Duration: 5 * time.Millisecond,
			},
		},
		Total:    2,
		Passed:   1,
		Failed:   1,
		Duration: 5 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := output.FormatTestReport(&buf, result, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	// Should contain ANSI color codes.
	if !strings.Contains(out, "\033[") {
		t.Errorf("expected color codes in color mode, got:\n%s", out)
	}
}

func TestFormatTestReport_MultipleSuites(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "Suite A",
				Results: []domain.TestResult{
					{Name: "test 1", Passed: true},
				},
				Passed: 1, Failed: 0, Total: 1,
				Duration: 5 * time.Millisecond,
			},
			{
				SuiteName: "Suite B",
				Results: []domain.TestResult{
					{Name: "test 2", Passed: false, Error: "fail reason"},
				},
				Passed: 0, Failed: 1, Total: 1,
				Duration: 10 * time.Millisecond,
			},
		},
		Total:    2,
		Passed:   1,
		Failed:   1,
		Duration: 15 * time.Millisecond,
	}

	var buf bytes.Buffer
	err := output.FormatTestReport(&buf, result, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Suite A") {
		t.Errorf("expected Suite A in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Suite B") {
		t.Errorf("expected Suite B in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 passed") {
		t.Errorf("expected '1 passed' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "1 failed") {
		t.Errorf("expected '1 failed' in summary, got:\n%s", out)
	}
}
