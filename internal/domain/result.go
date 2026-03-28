package domain

// ExecutionResult holds the request and response from executing an HTTP call.
type ExecutionResult struct {
	Request       HTTPRequest
	Response      HTTPResponse
	TestResults   []TestResult
	ScriptConsole []string
}
