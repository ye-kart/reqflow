package exporter_test

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/core/exporter"
	"github.com/ye-kart/reqflow/internal/domain"
)

func TestExportJUnit_GeneratesValidXML(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "User API",
				Results: []domain.TestResult{
					{Name: "Status code is 200", Passed: true},
				},
				Passed:   1,
				Failed:   0,
				Total:    1,
				Duration: 2 * time.Millisecond,
			},
		},
		Total:    1,
		Passed:   1,
		Failed:   0,
		Duration: 2 * time.Millisecond,
	}

	data, err := exporter.ExportJUnit(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid XML.
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Errorf("expected XML declaration, got:\n%s", string(data))
	}

	// Verify it parses as XML.
	var testsuites struct {
		XMLName xml.Name `xml:"testsuites"`
	}
	if err := xml.Unmarshal(data, &testsuites); err != nil {
		t.Errorf("invalid XML: %v", err)
	}
}

func TestExportJUnit_IncludesFailuresWithMessages(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "User API",
				Results: []domain.TestResult{
					{Name: "Status code is 200", Passed: true},
					{Name: "Body contains user name", Passed: false, Error: "Expected: John, Actual: Jane"},
				},
				Passed:   1,
				Failed:   1,
				Total:    2,
				Duration: 15 * time.Millisecond,
			},
		},
		Total:    2,
		Passed:   1,
		Failed:   1,
		Duration: 15 * time.Millisecond,
	}

	data, err := exporter.ExportJUnit(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xmlStr := string(data)

	if !strings.Contains(xmlStr, `name="Body contains user name"`) {
		t.Errorf("expected failure test name in XML, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, "Expected: John, Actual: Jane") {
		t.Errorf("expected failure message in XML, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<failure") {
		t.Errorf("expected <failure> element in XML, got:\n%s", xmlStr)
	}
}

func TestExportJUnit_TestCountMatches(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "Suite A",
				Results: []domain.TestResult{
					{Name: "test 1", Passed: true},
					{Name: "test 2", Passed: true},
					{Name: "test 3", Passed: false, Error: "fail"},
				},
				Passed:   2,
				Failed:   1,
				Total:    3,
				Duration: 10 * time.Millisecond,
			},
		},
		Total:    3,
		Passed:   2,
		Failed:   1,
		Duration: 10 * time.Millisecond,
	}

	data, err := exporter.ExportJUnit(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the XML and verify counts.
	type failure struct {
		Message string `xml:"message,attr"`
	}
	type testcase struct {
		Name    string   `xml:"name,attr"`
		Time    string   `xml:"time,attr"`
		Failure *failure `xml:"failure"`
	}
	type testsuite struct {
		Name     string     `xml:"name,attr"`
		Tests    int        `xml:"tests,attr"`
		Failures int        `xml:"failures,attr"`
		Cases    []testcase `xml:"testcase"`
	}
	type testsuites struct {
		Tests    int         `xml:"tests,attr"`
		Failures int         `xml:"failures,attr"`
		Suites   []testsuite `xml:"testsuite"`
	}

	var ts testsuites
	if err := xml.Unmarshal(data, &ts); err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	if ts.Tests != 3 {
		t.Errorf("testsuites tests=%d, want 3", ts.Tests)
	}
	if ts.Failures != 1 {
		t.Errorf("testsuites failures=%d, want 1", ts.Failures)
	}
	if len(ts.Suites) != 1 {
		t.Fatalf("expected 1 testsuite, got %d", len(ts.Suites))
	}
	if ts.Suites[0].Tests != 3 {
		t.Errorf("testsuite tests=%d, want 3", ts.Suites[0].Tests)
	}
	if ts.Suites[0].Failures != 1 {
		t.Errorf("testsuite failures=%d, want 1", ts.Suites[0].Failures)
	}
	if len(ts.Suites[0].Cases) != 3 {
		t.Fatalf("expected 3 testcases, got %d", len(ts.Suites[0].Cases))
	}

	// Verify the failing test has a failure element.
	failingCase := ts.Suites[0].Cases[2]
	if failingCase.Failure == nil {
		t.Fatal("expected failure element on failing test case")
	}
	if failingCase.Failure.Message != "fail" {
		t.Errorf("expected failure message 'fail', got %q", failingCase.Failure.Message)
	}

	// Verify passing tests have no failure element.
	if ts.Suites[0].Cases[0].Failure != nil {
		t.Error("expected no failure element on passing test case")
	}
}

func TestExportJUnit_MultipleSuites(t *testing.T) {
	result := domain.TestSuiteResult{
		Suites: []domain.TestReport{
			{
				SuiteName: "Suite A",
				Results: []domain.TestResult{
					{Name: "test a1", Passed: true},
				},
				Passed: 1, Failed: 0, Total: 1,
				Duration: 5 * time.Millisecond,
			},
			{
				SuiteName: "Suite B",
				Results: []domain.TestResult{
					{Name: "test b1", Passed: false, Error: "nope"},
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

	data, err := exporter.ExportJUnit(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xmlStr := string(data)
	if !strings.Contains(xmlStr, `name="Suite A"`) {
		t.Errorf("expected Suite A in XML, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `name="Suite B"`) {
		t.Errorf("expected Suite B in XML, got:\n%s", xmlStr)
	}
}
