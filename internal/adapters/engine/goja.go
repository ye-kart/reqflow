package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/ye-kart/reqflow/internal/core/validation"
	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// GojaEngine implements driven.ScriptEngine using the Goja JavaScript runtime.
type GojaEngine struct{}

// NewGojaEngine creates a new GojaEngine.
func NewGojaEngine() *GojaEngine {
	return &GojaEngine{}
}

// Execute runs a JavaScript script with the given context and returns results.
func (e *GojaEngine) Execute(script string, ctx driven.ScriptContext) (driven.ScriptResult, error) {
	if script == "" {
		return driven.ScriptResult{
			UpdatedVariables: copyVars(ctx.Variables),
		}, nil
	}

	vm := goja.New()

	// Shared state collectors.
	var consoleOutput []string
	var testResults []domain.TestResult
	variables := copyVars(ctx.Variables)

	// Track whether the request was modified.
	requestModified := false
	reqURL := ctx.Request.URL
	reqMethod := string(ctx.Request.Method)
	reqHeaders := headersToMap(ctx.Request.Headers)

	// Register console.log.
	consoleObj := vm.NewObject()
	_ = consoleObj.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, arg := range call.Arguments {
			parts[i] = fmt.Sprintf("%v", arg.Export())
		}
		consoleOutput = append(consoleOutput, strings.Join(parts, " "))
		return goja.Undefined()
	})
	vm.Set("console", consoleObj)

	// Build pm object.
	pm := vm.NewObject()

	// pm.variables
	pmVars := vm.NewObject()
	_ = pmVars.Set("get", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		val, ok := variables[key]
		if !ok {
			return goja.Undefined()
		}
		return vm.ToValue(val)
	})
	_ = pmVars.Set("set", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		val := call.Argument(1).String()
		variables[key] = val
		return goja.Undefined()
	})
	_ = pm.Set("variables", pmVars)

	// pm.request (as a proxy-like object with dynamic property access)
	pmRequest := vm.NewObject()
	_ = pmRequest.Set("url", reqURL)
	_ = pmRequest.Set("method", reqMethod)
	_ = pmRequest.Set("headers", reqHeaders)
	_ = pm.Set("request", pmRequest)

	// pm.response
	if ctx.Response != nil {
		pmResponse := vm.NewObject()
		_ = pmResponse.Set("code", ctx.Response.StatusCode)
		_ = pmResponse.Set("responseTime", ctx.Response.Duration.Milliseconds())

		respHeaders := headersToMap(ctx.Response.Headers)
		_ = pmResponse.Set("headers", respHeaders)

		bodyBytes := ctx.Response.Body
		_ = pmResponse.Set("text", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(string(bodyBytes))
		})
		_ = pmResponse.Set("json", func(call goja.FunctionCall) goja.Value {
			var parsed interface{}
			if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
				panic(vm.NewGoError(fmt.Errorf("failed to parse response body as JSON: %w", err)))
			}
			return vm.ToValue(parsed)
		})

		_ = pm.Set("response", pmResponse)
	}

	// pm.test
	_ = pm.Set("test", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			testResults = append(testResults, domain.TestResult{
				Name:   name,
				Passed: false,
				Error:  "second argument to pm.test must be a function",
			})
			return goja.Undefined()
		}

		// Run the test function, catching assertion errors.
		_, err := fn(goja.Undefined())
		if err != nil {
			testResults = append(testResults, domain.TestResult{
				Name:   name,
				Passed: false,
				Error:  err.Error(),
			})
		} else {
			testResults = append(testResults, domain.TestResult{
				Name:   name,
				Passed: true,
			})
		}
		return goja.Undefined()
	})

	// pm.expect
	_ = pm.Set("expect", func(call goja.FunctionCall) goja.Value {
		actual := call.Argument(0).Export()
		return buildExpectation(vm, actual)
	})

	vm.Set("pm", pm)

	// Run the script.
	_, err := vm.RunString(script)
	if err != nil {
		return driven.ScriptResult{}, fmt.Errorf("script execution failed: %w", err)
	}

	// Collect updated request if the script modified pm.request fields.
	// Re-read from the JS object to detect changes.
	var updatedRequest *domain.RequestConfig
	pmReqVal := pm.Get("request")
	if pmReqVal != nil && pmReqVal != goja.Undefined() {
		pmReqObj := pmReqVal.ToObject(vm)
		newURL := pmReqObj.Get("url").String()
		newHeaders := exportHeaders(vm, pmReqObj.Get("headers"))

		if newURL != ctx.Request.URL || !headersEqual(newHeaders, ctx.Request.Headers) {
			requestModified = true
		}

		if requestModified {
			updated := ctx.Request
			updated.URL = newURL
			updated.Headers = newHeaders
			updatedRequest = &updated
		}
	}
	// Suppress unused variable warning.
	_ = requestModified

	return driven.ScriptResult{
		UpdatedVariables: variables,
		UpdatedRequest:   updatedRequest,
		TestResults:      testResults,
		Console:          consoleOutput,
	}, nil
}

// buildExpectation creates a chainable assertion object: pm.expect(val).to.eql(...) etc.
func buildExpectation(vm *goja.Runtime, actual interface{}) goja.Value {
	obj := vm.NewObject()

	// .to is itself an object with assertion methods
	to := vm.NewObject()

	// .to.eql(expected)
	_ = to.Set("eql", makeEqlFunc(vm, actual))
	// .to.equal(expected) - alias
	_ = to.Set("equal", makeEqlFunc(vm, actual))
	// .to.include(str)
	_ = to.Set("include", makeIncludeFunc(vm, actual))

	// .to.be is an object with above/below
	be := vm.NewObject()
	_ = be.Set("above", makeAboveFunc(vm, actual))
	_ = be.Set("below", makeBelowFunc(vm, actual))
	_ = to.Set("be", be)

	// .to.have is an object with property/status/jsonSchema
	have := vm.NewObject()
	_ = have.Set("property", makePropertyFunc(vm, actual))
	_ = have.Set("status", makeStatusFunc(vm, actual))
	_ = have.Set("jsonSchema", makeJsonSchemaFunc(vm, actual))
	_ = to.Set("have", have)

	_ = obj.Set("to", to)
	return obj
}

func makeEqlFunc(vm *goja.Runtime, actual interface{}) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).Export()
		if !deepEqual(actual, expected) {
			panic(vm.NewGoError(fmt.Errorf("expected %v to equal %v", actual, expected)))
		}
		return goja.Undefined()
	}
}

func makeIncludeFunc(vm *goja.Runtime, actual interface{}) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).String()
		actualStr := fmt.Sprintf("%v", actual)
		if !strings.Contains(actualStr, expected) {
			panic(vm.NewGoError(fmt.Errorf("expected '%v' to include '%v'", actual, expected)))
		}
		return goja.Undefined()
	}
}

func makeAboveFunc(vm *goja.Runtime, actual interface{}) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).Export()
		a := toFloat64(actual)
		b := toFloat64(expected)
		if a <= b {
			panic(vm.NewGoError(fmt.Errorf("expected %v to be above %v", actual, expected)))
		}
		return goja.Undefined()
	}
}

func makeBelowFunc(vm *goja.Runtime, actual interface{}) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).Export()
		a := toFloat64(actual)
		b := toFloat64(expected)
		if a >= b {
			panic(vm.NewGoError(fmt.Errorf("expected %v to be below %v", actual, expected)))
		}
		return goja.Undefined()
	}
}

func makePropertyFunc(vm *goja.Runtime, actual interface{}) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		propName := call.Argument(0).String()
		m, ok := actual.(map[string]interface{})
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("expected value to be an object, got %T", actual)))
		}
		if _, exists := m[propName]; !exists {
			panic(vm.NewGoError(fmt.Errorf("expected object to have property '%s'", propName)))
		}
		return goja.Undefined()
	}
}

func makeStatusFunc(vm *goja.Runtime, actual interface{}) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).Export()
		if !deepEqual(actual, expected) {
			panic(vm.NewGoError(fmt.Errorf("expected status %v, got %v", expected, actual)))
		}
		return goja.Undefined()
	}
}

// deepEqual compares two values for equality, handling numeric type coercion.
func deepEqual(a, b interface{}) bool {
	// Handle numeric comparison: Goja may return int64 or float64.
	af, aIsNum := toFloat64Safe(a)
	bf, bIsNum := toFloat64Safe(b)
	if aIsNum && bIsNum {
		return af == bf
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func toFloat64(v interface{}) float64 {
	f, _ := toFloat64Safe(v)
	return f
}

func toFloat64Safe(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func copyVars(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func headersToMap(headers []domain.Header) map[string]interface{} {
	m := make(map[string]interface{}, len(headers))
	for _, h := range headers {
		m[h.Key] = h.Value
	}
	return m
}

func exportHeaders(vm *goja.Runtime, val goja.Value) []domain.Header {
	if val == nil || val == goja.Undefined() || val == goja.Null() {
		return nil
	}

	obj := val.ToObject(vm)
	keys := obj.Keys()
	headers := make([]domain.Header, 0, len(keys))
	for _, key := range keys {
		v := obj.Get(key)
		if v != nil && v != goja.Undefined() {
			headers = append(headers, domain.Header{Key: key, Value: v.String()})
		}
	}
	return headers
}

func makeJsonSchemaFunc(vm *goja.Runtime, actual interface{}) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		schemaObj := call.Argument(0).Export()

		// Marshal actual data to JSON.
		dataBytes, err := json.Marshal(actual)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("failed to marshal data for schema validation: %w", err)))
		}

		// Marshal schema object to JSON.
		schemaBytes, err := json.Marshal(schemaObj)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("failed to marshal schema: %w", err)))
		}

		valid, errs, err := validation.ValidateJSONSchema(dataBytes, schemaBytes)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("schema validation error: %w", err)))
		}
		if !valid {
			panic(vm.NewGoError(fmt.Errorf("JSON Schema validation failed: %s", strings.Join(errs, "; "))))
		}

		return goja.Undefined()
	}
}

func headersEqual(a []domain.Header, b []domain.Header) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]string, len(a))
	for _, h := range a {
		am[h.Key] = h.Value
	}
	for _, h := range b {
		if am[h.Key] != h.Value {
			return false
		}
	}
	return true
}
