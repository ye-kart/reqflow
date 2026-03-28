package domain

import "time"

// TestReport holds the results of a single test suite (e.g., one workflow step or request).
type TestReport struct {
	SuiteName string
	Results   []TestResult
	Passed    int
	Failed    int
	Total     int
	Duration  time.Duration
}

// TestSuiteResult aggregates multiple test reports into a single result.
type TestSuiteResult struct {
	Suites   []TestReport
	Total    int
	Passed   int
	Failed   int
	Duration time.Duration
}
