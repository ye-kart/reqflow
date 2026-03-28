package domain

// TestResult holds the outcome of a pm.test() call in a script.
type TestResult struct {
	Name   string
	Passed bool
	Error  string
}
