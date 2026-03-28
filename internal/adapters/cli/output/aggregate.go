package output

import (
	"fmt"

	"github.com/ye-kart/reqflow/internal/domain"
)

// WorkflowToTestSuite converts a WorkflowResult into a TestSuiteResult
// by translating each step's assertions (and errors) into TestResults.
func WorkflowToTestSuite(wf domain.WorkflowResult) domain.TestSuiteResult {
	suite := domain.TestSuiteResult{
		Duration: wf.Duration,
	}

	for _, step := range wf.Steps {
		report := domain.TestReport{
			SuiteName: step.StepName,
			Duration:  step.Duration,
		}

		if step.Error != nil {
			report.Results = append(report.Results, domain.TestResult{
				Name:   step.StepName,
				Passed: false,
				Error:  step.Error.Error(),
			})
			report.Failed = 1
			report.Total = 1
			suite.Failed++
			suite.Total++
			suite.Suites = append(suite.Suites, report)
			continue
		}

		for _, ar := range step.Assertions {
			tr := domain.TestResult{
				Passed: ar.Passed,
			}
			if ar.Passed {
				tr.Name = fmt.Sprintf("%s %s %v", ar.Assertion.Field, ar.Assertion.Operator, ar.Assertion.Expected)
				report.Passed++
			} else {
				tr.Name = fmt.Sprintf("%s %s %v", ar.Assertion.Field, ar.Assertion.Operator, ar.Assertion.Expected)
				tr.Error = ar.Message
				report.Failed++
			}
			report.Results = append(report.Results, tr)
			report.Total++
		}

		suite.Total += report.Total
		suite.Passed += report.Passed
		suite.Failed += report.Failed
		suite.Suites = append(suite.Suites, report)
	}

	return suite
}

// CollectionToTestSuite converts a CollectionRunResult into a TestSuiteResult
// by treating each request as a single pass/fail test.
func CollectionToTestSuite(col domain.CollectionRunResult) domain.TestSuiteResult {
	report := domain.TestReport{
		SuiteName: col.CollectionName,
		Duration:  col.Duration,
	}

	for _, rr := range col.Results {
		tr := domain.TestResult{
			Name:   rr.RequestName,
			Passed: rr.Passed,
		}
		if !rr.Passed {
			if rr.Error != nil {
				tr.Error = rr.Error.Error()
			} else {
				tr.Error = fmt.Sprintf("status %d", rr.Response.StatusCode)
			}
			report.Failed++
		} else {
			report.Passed++
		}
		report.Total++
		report.Results = append(report.Results, tr)
	}

	return domain.TestSuiteResult{
		Suites:   []domain.TestReport{report},
		Total:    report.Total,
		Passed:   report.Passed,
		Failed:   report.Failed,
		Duration: col.Duration,
	}
}
