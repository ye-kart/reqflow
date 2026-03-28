package http

import (
	"context"
	"fmt"

	"github.com/ye-kart/reqflow/internal/core/auth"
	"github.com/ye-kart/reqflow/internal/core/request"
	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// Option configures an Executor.
type Option func(*Executor)

// WithScriptEngine sets the JavaScript script engine for pre/post scripts.
func WithScriptEngine(se driven.ScriptEngine) Option {
	return func(e *Executor) {
		e.scriptEngine = se
	}
}

// Executor orchestrates HTTP request execution by tying core logic to adapters.
type Executor struct {
	httpClient   driven.HTTPClient
	scriptEngine driven.ScriptEngine
}

// NewExecutor creates a new Executor with the given HTTP client and options.
func NewExecutor(hc driven.HTTPClient, opts ...Option) *Executor {
	e := &Executor{httpClient: hc}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// BuildRequest builds a fully resolved HTTP request from config without sending it.
// This applies variable substitution and auth but does not make an HTTP call.
func (e *Executor) BuildRequest(config domain.RequestConfig, vars map[string]string) (domain.HTTPRequest, error) {
	req, err := request.BuildRequest(config, vars)
	if err != nil {
		return domain.HTTPRequest{}, fmt.Errorf("building request: %w", err)
	}

	if config.Auth != nil {
		req, err = auth.Apply(req, config.Auth)
		if err != nil {
			return domain.HTTPRequest{}, fmt.Errorf("applying auth: %w", err)
		}
	}

	return req, nil
}

// Execute builds a request from config, applies auth, sends it, and returns the result.
func (e *Executor) Execute(ctx context.Context, config domain.RequestConfig, vars map[string]string) (domain.ExecutionResult, error) {
	if vars == nil {
		vars = make(map[string]string)
	}

	var allConsole []string
	var allTestResults []domain.TestResult

	// Run pre-request script if present and engine available.
	if config.PreScript != "" && e.scriptEngine != nil {
		scriptCtx := driven.ScriptContext{
			Variables: vars,
			Request:   config,
			Response:  nil,
		}
		scriptResult, err := e.scriptEngine.Execute(config.PreScript, scriptCtx)
		if err != nil {
			return domain.ExecutionResult{}, fmt.Errorf("pre-request script: %w", err)
		}

		// Apply script modifications.
		vars = scriptResult.UpdatedVariables
		if scriptResult.UpdatedRequest != nil {
			config = *scriptResult.UpdatedRequest
		}
		allConsole = append(allConsole, scriptResult.Console...)
		allTestResults = append(allTestResults, scriptResult.TestResults...)
	}

	req, err := e.BuildRequest(config, vars)
	if err != nil {
		return domain.ExecutionResult{}, err
	}

	// Send the request via the HTTP client adapter.
	resp, err := e.httpClient.Do(ctx, req)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("executing request: %w", err)
	}

	// Run post-response script if present and engine available.
	if config.PostScript != "" && e.scriptEngine != nil {
		scriptCtx := driven.ScriptContext{
			Variables: vars,
			Request:   config,
			Response:  &resp,
		}
		scriptResult, err := e.scriptEngine.Execute(config.PostScript, scriptCtx)
		if err != nil {
			return domain.ExecutionResult{}, fmt.Errorf("post-response script: %w", err)
		}

		allConsole = append(allConsole, scriptResult.Console...)
		allTestResults = append(allTestResults, scriptResult.TestResults...)
	}

	return domain.ExecutionResult{
		Request:       req,
		Response:      resp,
		TestResults:   allTestResults,
		ScriptConsole: allConsole,
	}, nil
}
