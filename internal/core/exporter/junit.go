package exporter

import (
	"encoding/xml"
	"fmt"

	"github.com/ye-kart/reqflow/internal/domain"
)

// junitTestSuites is the root element of a JUnit XML report.
type junitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Time     string           `xml:"time,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name    string        `xml:"name,attr"`
	Time    string        `xml:"time,attr"`
	Failure *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
}

// ExportJUnit generates JUnit XML from a TestSuiteResult for CI integration.
func ExportJUnit(result domain.TestSuiteResult) ([]byte, error) {
	root := junitTestSuites{
		Tests:    result.Total,
		Failures: result.Failed,
		Time:     fmt.Sprintf("%.3f", result.Duration.Seconds()),
	}

	for _, suite := range result.Suites {
		js := junitTestSuite{
			Name:     suite.SuiteName,
			Tests:    suite.Total,
			Failures: suite.Failed,
			Time:     fmt.Sprintf("%.3f", suite.Duration.Seconds()),
		}

		for _, tr := range suite.Results {
			tc := junitTestCase{
				Name: tr.Name,
				Time: "0.000",
			}
			if !tr.Passed {
				tc.Failure = &junitFailure{
					Message: tr.Error,
				}
			}
			js.Cases = append(js.Cases, tc)
		}

		root.Suites = append(root.Suites, js)
	}

	data, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JUnit XML: %w", err)
	}

	xmlHeader := []byte(xml.Header)
	return append(xmlHeader, data...), nil
}
