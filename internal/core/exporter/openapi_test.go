package exporter

import (
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestExportOpenAPI_ValidYAMLWithVersion(t *testing.T) {
	col := testCollection()

	data, err := ExportOpenAPI(col)
	if err != nil {
		t.Fatalf("ExportOpenAPI() error = %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "openapi: 3.0") && !strings.Contains(content, "openapi: \"3.0") {
		t.Error("OpenAPI output missing openapi version")
	}
}

func TestExportOpenAPI_HasPaths(t *testing.T) {
	col := testCollection()

	data, err := ExportOpenAPI(col)
	if err != nil {
		t.Fatalf("ExportOpenAPI() error = %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "paths:") {
		t.Error("OpenAPI output missing paths section")
	}
	if !strings.Contains(content, "/health") {
		t.Error("OpenAPI output missing /health path")
	}
	if !strings.Contains(content, "/users") {
		t.Error("OpenAPI output missing /users path")
	}
	if !strings.Contains(content, "/pets") {
		t.Error("OpenAPI output missing /pets path")
	}
}

func TestExportOpenAPI_MethodsMappedCorrectly(t *testing.T) {
	col := testCollection()

	data, err := ExportOpenAPI(col)
	if err != nil {
		t.Fatalf("ExportOpenAPI() error = %v", err)
	}

	content := string(data)

	// /users should have both get and post
	if !strings.Contains(content, "get:") {
		t.Error("OpenAPI output missing GET method mapping")
	}
	if !strings.Contains(content, "post:") {
		t.Error("OpenAPI output missing POST method mapping")
	}
}

func TestExportOpenAPI_HasInfoSection(t *testing.T) {
	col := testCollection()

	data, err := ExportOpenAPI(col)
	if err != nil {
		t.Fatalf("ExportOpenAPI() error = %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "info:") {
		t.Error("OpenAPI output missing info section")
	}
	if !strings.Contains(content, "Pet Store API") {
		t.Error("OpenAPI output missing collection title in info")
	}
}

func TestExportOpenAPI_EmptyCollection(t *testing.T) {
	col := domain.Collection{
		Name: "Empty API",
	}

	data, err := ExportOpenAPI(col)
	if err != nil {
		t.Fatalf("ExportOpenAPI() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "openapi:") {
		t.Error("OpenAPI output missing openapi field for empty collection")
	}
	if !strings.Contains(content, "paths:") {
		t.Error("OpenAPI output missing paths field for empty collection")
	}
}

func TestExportOpenAPI_RequestBodyIncluded(t *testing.T) {
	col := testCollection()

	data, err := ExportOpenAPI(col)
	if err != nil {
		t.Fatalf("ExportOpenAPI() error = %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "requestBody:") {
		t.Error("OpenAPI output missing requestBody for POST request")
	}
}
