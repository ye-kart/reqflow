package driven

import "github.com/ye-kart/reqflow/internal/domain"

// ScriptContext provides data accessible to pre-request and post-response scripts.
type ScriptContext struct {
	Variables map[string]string
	Request   domain.RequestConfig
	Response  *domain.HTTPResponse // nil for pre-request scripts
}

// ScriptResult holds the outputs produced by script execution.
type ScriptResult struct {
	UpdatedVariables map[string]string
	UpdatedRequest   *domain.RequestConfig // only for pre-request scripts
	TestResults      []domain.TestResult
	Console          []string // console.log output
}

// ScriptEngine is the driven port for executing JavaScript scripts.
type ScriptEngine interface {
	Execute(script string, ctx ScriptContext) (ScriptResult, error)
}
