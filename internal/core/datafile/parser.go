package datafile

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseCSV parses CSV data where the first row contains headers used as
// variable keys. Each subsequent row becomes one iteration's variable map.
func ParseCSV(data []byte) ([]map[string]string, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1 // allow variable field counts initially

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}

	expectedCols := len(headers)
	var rows []map[string]string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}

		if len(record) != expectedCols {
			return nil, fmt.Errorf("row %d has %d columns, expected %d", len(rows)+1, len(record), expectedCols)
		}

		row := make(map[string]string, expectedCols)
		for i, header := range headers {
			row[header] = record[i]
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// ParseJSON parses a JSON array of objects. Each object becomes one
// iteration's variable map. Non-string values are converted to strings.
func ParseJSON(data []byte) ([]map[string]string, error) {
	var raw []map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		// Check if it's a non-array top-level value
		var single interface{}
		if json.Unmarshal(data, &single) == nil {
			return nil, fmt.Errorf("data file must contain a JSON array, got %T", single)
		}
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	rows := make([]map[string]string, 0, len(raw))
	for _, obj := range raw {
		row := make(map[string]string, len(obj))
		for k, v := range obj {
			row[k] = stringify(v)
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// ParseFile reads a file and auto-detects the format by extension.
// Supported extensions: .csv, .json.
func ParseFile(path string) ([]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading data file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return ParseCSV(data)
	case ".json":
		return ParseJSON(data)
	default:
		return nil, fmt.Errorf("unsupported data file extension %q (use .csv or .json)", ext)
	}
}

// stringify converts an arbitrary value to a string representation.
func stringify(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// Use strconv to avoid trailing zeros for integers
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	default:
		// For nested objects/arrays, marshal back to JSON
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}
