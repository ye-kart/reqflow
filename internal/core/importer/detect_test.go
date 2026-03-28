package importer

import (
	"testing"
)

func TestDetectFormat_Postman(t *testing.T) {
	data := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": []
	}`)

	got := DetectFormat(data)
	if got != "postman" {
		t.Errorf("DetectFormat() = %q, want %q", got, "postman")
	}
}

func TestDetectFormat_OpenAPI_JSON(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {}
	}`)

	got := DetectFormat(data)
	if got != "openapi" {
		t.Errorf("DetectFormat() = %q, want %q", got, "openapi")
	}
}

func TestDetectFormat_OpenAPI_YAML(t *testing.T) {
	data := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths: {}
`)

	got := DetectFormat(data)
	if got != "openapi" {
		t.Errorf("DetectFormat() = %q, want %q", got, "openapi")
	}
}

func TestDetectFormat_HAR(t *testing.T) {
	data := []byte(`{
		"log": {
			"version": "1.2",
			"entries": []
		}
	}`)

	got := DetectFormat(data)
	if got != "har" {
		t.Errorf("DetectFormat() = %q, want %q", got, "har")
	}
}

func TestDetectFormat_Insomnia(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"__export_format": 4,
		"resources": []
	}`)

	got := DetectFormat(data)
	if got != "insomnia" {
		t.Errorf("DetectFormat() = %q, want %q", got, "insomnia")
	}
}

func TestDetectFormat_Curl(t *testing.T) {
	data := []byte(`curl -X GET https://example.com/api`)

	got := DetectFormat(data)
	if got != "curl" {
		t.Errorf("DetectFormat() = %q, want %q", got, "curl")
	}
}

func TestDetectFormat_Unknown(t *testing.T) {
	data := []byte(`just some random text`)

	got := DetectFormat(data)
	if got != "unknown" {
		t.Errorf("DetectFormat() = %q, want %q", got, "unknown")
	}
}

func TestImport_AutoDetectPostman(t *testing.T) {
	data := []byte(`{
		"info": {
			"name": "Auto Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Test Request",
				"request": {
					"method": "GET",
					"url": {"raw": "https://example.com"}
				}
			}
		]
	}`)

	col, err := Import(data)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if col.Name != "Auto Test" {
		t.Errorf("Name = %q, want %q", col.Name, "Auto Test")
	}
}

func TestImport_AutoDetectOpenAPI(t *testing.T) {
	data := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "OpenAPI Auto", "version": "1.0"},
		"paths": {
			"/test": {
				"get": {
					"summary": "Test",
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`)

	col, err := Import(data)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if col.Name != "OpenAPI Auto" {
		t.Errorf("Name = %q, want %q", col.Name, "OpenAPI Auto")
	}
}

func TestImport_AutoDetectHAR(t *testing.T) {
	data := []byte(`{
		"log": {
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://example.com",
						"headers": []
					}
				}
			]
		}
	}`)

	col, err := Import(data)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if len(col.Requests) != 1 {
		t.Errorf("Requests len = %d, want 1", len(col.Requests))
	}
}

func TestImport_AutoDetectInsomnia(t *testing.T) {
	data := []byte(`{
		"_type": "export",
		"__export_format": 4,
		"resources": [
			{
				"_id": "wrk_1",
				"_type": "workspace",
				"name": "Auto"
			}
		]
	}`)

	col, err := Import(data)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if col.Name != "Auto" {
		t.Errorf("Name = %q, want %q", col.Name, "Auto")
	}
}

func TestImport_AutoDetectCurl(t *testing.T) {
	// curl import returns a collection with one request.
	data := []byte(`curl -X POST -d '{"key":"val"}' https://example.com/api`)

	col, err := Import(data)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if len(col.Requests) != 1 {
		t.Fatalf("Requests len = %d, want 1", len(col.Requests))
	}
	if col.Requests[0].Config.URL != "https://example.com/api" {
		t.Errorf("URL = %q, want %q", col.Requests[0].Config.URL, "https://example.com/api")
	}
}

func TestImport_UnknownFormat(t *testing.T) {
	_, err := Import([]byte(`random garbage`))
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}
