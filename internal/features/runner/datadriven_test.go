package runner_test

import (
	"context"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/features/runner"
)

func TestRunWithData_SingleIteration(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{"message":"ok"}`)},
		},
	}

	r := runner.New(client)
	wf := domain.Workflow{
		Name: "single-iter",
		Steps: []domain.Step{
			{
				Name:   "greet",
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users/{{name}}",
				Assert: []domain.Assertion{
					{Field: "status", Operator: "==", Expected: 200},
				},
			},
		},
	}

	dataRows := []map[string]string{
		{"name": "John"},
	}

	result, err := r.RunWithData(context.Background(), wf, nil, dataRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalIterations != 1 {
		t.Errorf("TotalIterations = %d, want 1", result.TotalIterations)
	}
	if result.Passed != 1 {
		t.Errorf("Passed = %d, want 1", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
	if len(result.Iterations) != 1 {
		t.Fatalf("len(Iterations) = %d, want 1", len(result.Iterations))
	}
	if result.Iterations[0].Iteration != 0 {
		t.Errorf("Iteration[0].Iteration = %d, want 0", result.Iterations[0].Iteration)
	}
}

func TestRunWithData_TwoIterations(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{"message":"ok"}`)},
			{StatusCode: 200, Body: []byte(`{"message":"ok"}`)},
		},
	}

	r := runner.New(client)
	wf := domain.Workflow{
		Name: "two-iter",
		Steps: []domain.Step{
			{
				Name:   "greet",
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users/{{name}}",
				Assert: []domain.Assertion{
					{Field: "status", Operator: "==", Expected: 200},
				},
			},
		},
	}

	dataRows := []map[string]string{
		{"name": "John"},
		{"name": "Jane"},
	}

	result, err := r.RunWithData(context.Background(), wf, nil, dataRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalIterations != 2 {
		t.Errorf("TotalIterations = %d, want 2", result.TotalIterations)
	}
	if result.Passed != 2 {
		t.Errorf("Passed = %d, want 2", result.Passed)
	}
	if len(result.Iterations) != 2 {
		t.Fatalf("len(Iterations) = %d, want 2", len(result.Iterations))
	}

	// Verify each iteration carries its own variables
	if result.Iterations[0].Variables["name"] != "John" {
		t.Errorf("iter[0].Variables[name] = %q, want %q", result.Iterations[0].Variables["name"], "John")
	}
	if result.Iterations[1].Variables["name"] != "Jane" {
		t.Errorf("iter[1].Variables[name] = %q, want %q", result.Iterations[1].Variables["name"], "Jane")
	}

	// Client should have been called twice
	if client.calls != 2 {
		t.Errorf("HTTP client calls = %d, want 2", client.calls)
	}
}

func TestRunWithData_VariablesAvailableInSteps(t *testing.T) {
	var capturedURLs []string
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		onDo: func(req domain.HTTPRequest) {
			capturedURLs = append(capturedURLs, req.URL)
		},
	}

	r := runner.New(client)
	wf := domain.Workflow{
		Name: "var-test",
		Steps: []domain.Step{
			{
				Name:   "fetch user",
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users/{{name}}/role/{{role}}",
			},
		},
	}

	dataRows := []map[string]string{
		{"name": "John", "role": "admin"},
		{"name": "Jane", "role": "user"},
	}

	result, err := r.RunWithData(context.Background(), wf, nil, dataRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalIterations != 2 {
		t.Fatalf("TotalIterations = %d, want 2", result.TotalIterations)
	}

	if len(capturedURLs) != 2 {
		t.Fatalf("captured URLs = %d, want 2", len(capturedURLs))
	}
	if capturedURLs[0] != "https://api.example.com/users/John/role/admin" {
		t.Errorf("URL[0] = %q, want John/admin URL", capturedURLs[0])
	}
	if capturedURLs[1] != "https://api.example.com/users/Jane/role/user" {
		t.Errorf("URL[1] = %q, want Jane/user URL", capturedURLs[1])
	}
}

func TestRunWithData_IterationAndCountVars(t *testing.T) {
	var capturedURLs []string
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		onDo: func(req domain.HTTPRequest) {
			capturedURLs = append(capturedURLs, req.URL)
		},
	}

	r := runner.New(client)
	wf := domain.Workflow{
		Name: "meta-vars",
		Steps: []domain.Step{
			{
				Name:   "meta",
				Method: domain.MethodGet,
				URL:    "https://api.example.com/iter/{{iteration}}/of/{{iterationCount}}",
			},
		},
	}

	dataRows := []map[string]string{
		{"name": "John"},
		{"name": "Jane"},
		{"name": "Bob"},
	}

	_, err := r.RunWithData(context.Background(), wf, nil, dataRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedURLs) != 3 {
		t.Fatalf("captured URLs = %d, want 3", len(capturedURLs))
	}
	if capturedURLs[0] != "https://api.example.com/iter/0/of/3" {
		t.Errorf("URL[0] = %q, want iter/0/of/3", capturedURLs[0])
	}
	if capturedURLs[1] != "https://api.example.com/iter/1/of/3" {
		t.Errorf("URL[1] = %q, want iter/1/of/3", capturedURLs[1])
	}
	if capturedURLs[2] != "https://api.example.com/iter/2/of/3" {
		t.Errorf("URL[2] = %q, want iter/2/of/3", capturedURLs[2])
	}
}

func TestRunWithData_MergesInitialVars(t *testing.T) {
	var capturedURLs []string
	client := &capturingHTTPClient{
		response: domain.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)},
		onDo: func(req domain.HTTPRequest) {
			capturedURLs = append(capturedURLs, req.URL)
		},
	}

	r := runner.New(client)
	wf := domain.Workflow{
		Name: "merge-test",
		Steps: []domain.Step{
			{
				Name:   "merged",
				Method: domain.MethodGet,
				URL:    "https://{{host}}/users/{{name}}",
			},
		},
	}

	initialVars := map[string]string{"host": "api.example.com"}
	dataRows := []map[string]string{
		{"name": "John"},
	}

	_, err := r.RunWithData(context.Background(), wf, initialVars, dataRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedURLs) != 1 {
		t.Fatalf("captured URLs = %d, want 1", len(capturedURLs))
	}
	if capturedURLs[0] != "https://api.example.com/users/John" {
		t.Errorf("URL[0] = %q, want merged URL", capturedURLs[0])
	}
}

func TestRunWithData_FailedIterationCountsCorrectly(t *testing.T) {
	client := &mockHTTPClient{
		responses: []domain.HTTPResponse{
			{StatusCode: 200, Body: []byte(`{}`)},
			{StatusCode: 404, Body: []byte(`{}`)},
		},
	}

	r := runner.New(client)
	wf := domain.Workflow{
		Name: "fail-count",
		Steps: []domain.Step{
			{
				Name:   "check",
				Method: domain.MethodGet,
				URL:    "https://api.example.com/users/{{name}}",
				Assert: []domain.Assertion{
					{Field: "status", Operator: "==", Expected: 200},
				},
			},
		},
	}

	dataRows := []map[string]string{
		{"name": "John"},
		{"name": "Missing"},
	}

	result, err := r.RunWithData(context.Background(), wf, nil, dataRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed != 1 {
		t.Errorf("Passed = %d, want 1", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
	if result.TotalIterations != 2 {
		t.Errorf("TotalIterations = %d, want 2", result.TotalIterations)
	}
}

