package datafile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ye-kart/reqflow/internal/core/datafile"
)

// --- CSV Parser Tests ---

func TestParseCSV_ValidWithHeaders(t *testing.T) {
	input := []byte("name,email,role\nJohn,john@example.com,admin\nJane,jane@example.com,user\n")
	rows, err := datafile.ParseCSV(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows count = %d, want 2", len(rows))
	}

	if rows[0]["name"] != "John" {
		t.Errorf("row[0][name] = %q, want %q", rows[0]["name"], "John")
	}
	if rows[0]["email"] != "john@example.com" {
		t.Errorf("row[0][email] = %q, want %q", rows[0]["email"], "john@example.com")
	}
	if rows[0]["role"] != "admin" {
		t.Errorf("row[0][role] = %q, want %q", rows[0]["role"], "admin")
	}
	if rows[1]["name"] != "Jane" {
		t.Errorf("row[1][name] = %q, want %q", rows[1]["name"], "Jane")
	}
	if rows[1]["email"] != "jane@example.com" {
		t.Errorf("row[1][email] = %q, want %q", rows[1]["email"], "jane@example.com")
	}
	if rows[1]["role"] != "user" {
		t.Errorf("row[1][role] = %q, want %q", rows[1]["role"], "user")
	}
}

func TestParseCSV_Empty(t *testing.T) {
	rows, err := datafile.ParseCSV([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows count = %d, want 0", len(rows))
	}
}

func TestParseCSV_HeadersOnly(t *testing.T) {
	input := []byte("name,email,role\n")
	rows, err := datafile.ParseCSV(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows count = %d, want 0 for headers-only CSV", len(rows))
	}
}

func TestParseCSV_QuotedCommas(t *testing.T) {
	input := []byte("name,address\nJohn,\"123 Main St, Apt 4\"\n")
	rows, err := datafile.ParseCSV(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows count = %d, want 1", len(rows))
	}
	if rows[0]["address"] != "123 Main St, Apt 4" {
		t.Errorf("row[0][address] = %q, want %q", rows[0]["address"], "123 Main St, Apt 4")
	}
}

func TestParseCSV_InconsistentColumns(t *testing.T) {
	input := []byte("name,email\nJohn,john@example.com,extra\n")
	_, err := datafile.ParseCSV(input)
	if err == nil {
		t.Fatal("expected error for inconsistent column count, got nil")
	}
}

// --- JSON Parser Tests ---

func TestParseJSON_ValidArray(t *testing.T) {
	input := []byte(`[
		{"name": "John", "email": "john@example.com", "role": "admin"},
		{"name": "Jane", "email": "jane@example.com", "role": "user"}
	]`)
	rows, err := datafile.ParseJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows count = %d, want 2", len(rows))
	}
	if rows[0]["name"] != "John" {
		t.Errorf("row[0][name] = %q, want %q", rows[0]["name"], "John")
	}
	if rows[1]["role"] != "user" {
		t.Errorf("row[1][role] = %q, want %q", rows[1]["role"], "user")
	}
}

func TestParseJSON_EmptyArray(t *testing.T) {
	rows, err := datafile.ParseJSON([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows count = %d, want 0", len(rows))
	}
}

func TestParseJSON_NonArrayTopLevel(t *testing.T) {
	_, err := datafile.ParseJSON([]byte(`{"name": "John"}`))
	if err == nil {
		t.Fatal("expected error for non-array JSON, got nil")
	}
}

func TestParseJSON_NestedValuesStringified(t *testing.T) {
	input := []byte(`[{"name": "John", "meta": {"age": 30, "active": true}}]`)
	rows, err := datafile.ParseJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows count = %d, want 1", len(rows))
	}
	if rows[0]["name"] != "John" {
		t.Errorf("row[0][name] = %q, want %q", rows[0]["name"], "John")
	}
	// Nested object should be JSON-stringified
	meta := rows[0]["meta"]
	if meta == "" {
		t.Fatal("row[0][meta] is empty, want JSON string")
	}
}

func TestParseJSON_NumbersConvertedToStrings(t *testing.T) {
	input := []byte(`[{"id": 42, "score": 3.14, "active": true}]`)
	rows, err := datafile.ParseJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows count = %d, want 1", len(rows))
	}
	if rows[0]["id"] != "42" {
		t.Errorf("row[0][id] = %q, want %q", rows[0]["id"], "42")
	}
	if rows[0]["score"] != "3.14" {
		t.Errorf("row[0][score] = %q, want %q", rows[0]["score"], "3.14")
	}
	if rows[0]["active"] != "true" {
		t.Errorf("row[0][active] = %q, want %q", rows[0]["active"], "true")
	}
}

// --- ParseFile Tests ---

func TestParseFile_CSVExtension(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "data.csv")
	os.WriteFile(csvPath, []byte("name,role\nJohn,admin\n"), 0644)

	rows, err := datafile.ParseFile(csvPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows count = %d, want 1", len(rows))
	}
	if rows[0]["name"] != "John" {
		t.Errorf("row[0][name] = %q, want %q", rows[0]["name"], "John")
	}
}

func TestParseFile_JSONExtension(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "data.json")
	os.WriteFile(jsonPath, []byte(`[{"name": "Jane", "role": "user"}]`), 0644)

	rows, err := datafile.ParseFile(jsonPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows count = %d, want 1", len(rows))
	}
	if rows[0]["name"] != "Jane" {
		t.Errorf("row[0][name] = %q, want %q", rows[0]["name"], "Jane")
	}
}

func TestParseFile_UnknownExtension(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "data.xml")
	os.WriteFile(path, []byte("<data/>"), 0644)

	_, err := datafile.ParseFile(path)
	if err == nil {
		t.Fatal("expected error for unknown extension, got nil")
	}
}

func TestParseFile_NonexistentFile(t *testing.T) {
	_, err := datafile.ParseFile("/nonexistent/data.csv")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}
