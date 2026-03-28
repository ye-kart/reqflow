package engine_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/adapters/engine"
	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

func newEngine() driven.ScriptEngine {
	return engine.NewGojaEngine()
}

func TestExecuteEmptyScript_ReturnsEmptyResult(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute("", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.TestResults) != 0 {
		t.Errorf("expected no test results, got %d", len(result.TestResults))
	}
	if len(result.Console) != 0 {
		t.Errorf("expected no console output, got %d", len(result.Console))
	}
}

func TestPmVariablesGet_ReturnsVariableValue(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{"host": "example.com"},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`
		var val = pm.variables.get("host");
		console.log(val);
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "example.com" {
		t.Errorf("expected console [example.com], got %v", result.Console)
	}
}

func TestPmVariablesSet_UpdatesVariable(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`pm.variables.set("token", "abc123");`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UpdatedVariables["token"] != "abc123" {
		t.Errorf("expected token=abc123, got %v", result.UpdatedVariables)
	}
}

func TestPmResponseCode_ReturnsStatusCode(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 201,
		},
	}

	result, err := eng.Execute(`console.log(pm.response.code);`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "201" {
		t.Errorf("expected console [201], got %v", result.Console)
	}
}

func TestPmResponseJson_ParsesBody(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 200,
			Body:       []byte(`{"name":"test","count":42}`),
		},
	}

	result, err := eng.Execute(`
		var data = pm.response.json();
		console.log(data.name);
		console.log(data.count);
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 2 {
		t.Fatalf("expected 2 console entries, got %d: %v", len(result.Console), result.Console)
	}
	if result.Console[0] != "test" {
		t.Errorf("expected name=test, got %s", result.Console[0])
	}
	if result.Console[1] != "42" {
		t.Errorf("expected count=42, got %s", result.Console[1])
	}
}

func TestPmResponseText_ReturnsBodyString(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 200,
			Body:       []byte("hello world"),
		},
	}

	result, err := eng.Execute(`console.log(pm.response.text());`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "hello world" {
		t.Errorf("expected console [hello world], got %v", result.Console)
	}
}

func TestPmResponseHeaders_ReturnsHeaders(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 200,
			Headers: []domain.Header{
				{Key: "Content-Type", Value: "application/json"},
			},
		},
	}

	result, err := eng.Execute(`console.log(pm.response.headers["Content-Type"]);`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "application/json" {
		t.Errorf("expected console [application/json], got %v", result.Console)
	}
}

func TestPmResponseResponseTime_ReturnsDurationMs(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 200,
			Duration:   150 * time.Millisecond,
		},
	}

	result, err := eng.Execute(`console.log(pm.response.responseTime);`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "150" {
		t.Errorf("expected console [150], got %v", result.Console)
	}
}

func TestPmTest_PassingAssertion(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 200,
		},
	}

	result, err := eng.Execute(`
		pm.test("status is 200", function() {
			pm.expect(pm.response.code).to.eql(200);
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.TestResults) != 1 {
		t.Fatalf("expected 1 test result, got %d", len(result.TestResults))
	}
	if result.TestResults[0].Name != "status is 200" {
		t.Errorf("expected test name 'status is 200', got %s", result.TestResults[0].Name)
	}
	if !result.TestResults[0].Passed {
		t.Errorf("expected test to pass, but it failed: %s", result.TestResults[0].Error)
	}
}

func TestPmTest_FailingAssertion(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 500,
		},
	}

	result, err := eng.Execute(`
		pm.test("status is 200", function() {
			pm.expect(pm.response.code).to.eql(200);
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.TestResults) != 1 {
		t.Fatalf("expected 1 test result, got %d", len(result.TestResults))
	}
	if result.TestResults[0].Passed {
		t.Error("expected test to fail, but it passed")
	}
	if result.TestResults[0].Error == "" {
		t.Error("expected error message for failing test")
	}
}

func TestPmExpectToEql_Works(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`
		pm.test("values equal", function() {
			pm.expect("hello").to.eql("hello");
		});
		pm.test("values not equal", function() {
			pm.expect("hello").to.eql("world");
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.TestResults) != 2 {
		t.Fatalf("expected 2 test results, got %d", len(result.TestResults))
	}
	if !result.TestResults[0].Passed {
		t.Error("expected first test to pass")
	}
	if result.TestResults[1].Passed {
		t.Error("expected second test to fail")
	}
}

func TestPmExpectToEqual_IsAliasForEql(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`
		pm.test("equal alias", function() {
			pm.expect(42).to.equal(42);
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TestResults[0].Passed {
		t.Error("expected test to pass using .to.equal()")
	}
}

func TestPmExpectToBeAbove_Works(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`
		pm.test("10 above 5", function() {
			pm.expect(10).to.be.above(5);
		});
		pm.test("3 above 5 fails", function() {
			pm.expect(3).to.be.above(5);
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TestResults[0].Passed {
		t.Errorf("expected 10 > 5 to pass")
	}
	if result.TestResults[1].Passed {
		t.Errorf("expected 3 > 5 to fail")
	}
}

func TestPmExpectToBeBelow_Works(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`
		pm.test("3 below 5", function() {
			pm.expect(3).to.be.below(5);
		});
		pm.test("10 below 5 fails", function() {
			pm.expect(10).to.be.below(5);
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TestResults[0].Passed {
		t.Errorf("expected 3 < 5 to pass")
	}
	if result.TestResults[1].Passed {
		t.Errorf("expected 10 < 5 to fail")
	}
}

func TestPmExpectToInclude_Works(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`
		pm.test("includes substring", function() {
			pm.expect("hello world").to.include("world");
		});
		pm.test("does not include", function() {
			pm.expect("hello world").to.include("xyz");
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TestResults[0].Passed {
		t.Errorf("expected include to pass")
	}
	if result.TestResults[1].Passed {
		t.Errorf("expected include to fail")
	}
}

func TestPmExpectToHaveProperty_Works(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 200,
			Body:       []byte(`{"name":"test","items":[1,2]}`),
		},
	}

	result, err := eng.Execute(`
		var data = pm.response.json();
		pm.test("has name property", function() {
			pm.expect(data).to.have.property("name");
		});
		pm.test("has missing property", function() {
			pm.expect(data).to.have.property("missing");
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TestResults[0].Passed {
		t.Errorf("expected property check to pass")
	}
	if result.TestResults[1].Passed {
		t.Errorf("expected missing property check to fail")
	}
}

func TestPmExpectToHaveStatus_Works(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
		Response: &domain.HTTPResponse{
			StatusCode: 201,
		},
	}

	result, err := eng.Execute(`
		pm.test("status is 201", function() {
			pm.expect(pm.response.code).to.have.status(201);
		});
		pm.test("status is not 200", function() {
			pm.expect(pm.response.code).to.have.status(200);
		});
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TestResults[0].Passed {
		t.Errorf("expected status 201 check to pass")
	}
	if result.TestResults[1].Passed {
		t.Errorf("expected status 200 check to fail")
	}
}

func TestConsoleLog_CapturesOutput(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	result, err := eng.Execute(`
		console.log("first");
		console.log("second");
		console.log("number:", 42);
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 3 {
		t.Fatalf("expected 3 console entries, got %d: %v", len(result.Console), result.Console)
	}
	if result.Console[0] != "first" {
		t.Errorf("expected first, got %s", result.Console[0])
	}
	if result.Console[1] != "second" {
		t.Errorf("expected second, got %s", result.Console[1])
	}
	if result.Console[2] != "number: 42" {
		t.Errorf("expected 'number: 42', got %s", result.Console[2])
	}
}

func TestScriptError_ReturnsError(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request:   domain.RequestConfig{},
	}

	_, err := eng.Execute(`throw new Error("boom");`, ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected error to contain 'boom', got: %v", err)
	}
}

func TestPmRequestURL_ReturnsURL(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request: domain.RequestConfig{
			URL: "https://example.com/api",
		},
	}

	result, err := eng.Execute(`console.log(pm.request.url);`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "https://example.com/api" {
		t.Errorf("expected [https://example.com/api], got %v", result.Console)
	}
}

func TestPmRequestMethod_ReturnsMethod(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request: domain.RequestConfig{
			Method: domain.MethodPost,
		},
	}

	result, err := eng.Execute(`console.log(pm.request.method);`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "POST" {
		t.Errorf("expected [POST], got %v", result.Console)
	}
}

func TestPmRequestHeaders_ReturnsHeaders(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request: domain.RequestConfig{
			Headers: []domain.Header{
				{Key: "Authorization", Value: "Bearer tok"},
			},
		},
	}

	result, err := eng.Execute(`console.log(pm.request.headers["Authorization"]);`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Console) != 1 || result.Console[0] != "Bearer tok" {
		t.Errorf("expected [Bearer tok], got %v", result.Console)
	}
}

func TestPreRequestScript_ModifiesRequest(t *testing.T) {
	eng := newEngine()
	ctx := driven.ScriptContext{
		Variables: map[string]string{},
		Request: domain.RequestConfig{
			Method: domain.MethodGet,
			URL:    "https://example.com/api",
			Headers: []domain.Header{
				{Key: "Accept", Value: "text/plain"},
			},
		},
	}

	result, err := eng.Execute(`
		pm.request.url = "https://example.com/api/v2";
		pm.request.headers["Accept"] = "application/json";
		pm.request.headers["X-Custom"] = "value";
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UpdatedRequest == nil {
		t.Fatal("expected updated request, got nil")
	}
	if result.UpdatedRequest.URL != "https://example.com/api/v2" {
		t.Errorf("expected URL https://example.com/api/v2, got %s", result.UpdatedRequest.URL)
	}

	headers := make(map[string]string)
	for _, h := range result.UpdatedRequest.Headers {
		headers[h.Key] = h.Value
	}
	if headers["Accept"] != "application/json" {
		t.Errorf("expected Accept=application/json, got %s", headers["Accept"])
	}
	if headers["X-Custom"] != "value" {
		t.Errorf("expected X-Custom=value, got %s", headers["X-Custom"])
	}
}
