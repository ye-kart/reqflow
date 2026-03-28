package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// DetectFormat examines the raw data and returns the detected format name:
// "postman", "openapi", "har", "insomnia", "curl", or "unknown".
func DetectFormat(data []byte) string {
	trimmed := bytes.TrimSpace(data)

	// JSON-based formats: try to detect from structure.
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var raw map[string]json.RawMessage
		if json.Unmarshal(trimmed, &raw) == nil {
			return detectJSONFormat(raw)
		}
	}

	// YAML-based: check for OpenAPI marker.
	text := string(trimmed)
	if strings.Contains(text, "openapi:") || strings.Contains(text, "swagger:") {
		return "openapi"
	}

	// cURL command.
	if strings.HasPrefix(text, "curl ") || strings.HasPrefix(text, "curl\t") {
		return "curl"
	}

	return "unknown"
}

func detectJSONFormat(raw map[string]json.RawMessage) string {
	// Postman: has "info" and "item" keys, or info.schema contains "postman".
	if _, hasInfo := raw["info"]; hasInfo {
		if _, hasItem := raw["item"]; hasItem {
			return "postman"
		}
	}

	// OpenAPI: has "openapi" or "swagger" key.
	if _, has := raw["openapi"]; has {
		return "openapi"
	}
	if _, has := raw["swagger"]; has {
		return "openapi"
	}

	// HAR: has "log" key with "entries".
	if logRaw, has := raw["log"]; has {
		var logObj map[string]json.RawMessage
		if json.Unmarshal(logRaw, &logObj) == nil {
			if _, hasEntries := logObj["entries"]; hasEntries {
				return "har"
			}
		}
	}

	// Insomnia: has "_type": "export" and "resources".
	if typeRaw, has := raw["_type"]; has {
		var t string
		if json.Unmarshal(typeRaw, &t) == nil && t == "export" {
			if _, hasResources := raw["resources"]; hasResources {
				return "insomnia"
			}
		}
	}

	return "unknown"
}

// Import auto-detects the format of the input data and parses it into a
// domain.Collection.
func Import(data []byte) (domain.Collection, error) {
	format := DetectFormat(data)
	return ImportWithFormat(data, format)
}

// ImportWithFormat parses data using the specified format name.
func ImportWithFormat(data []byte, format string) (domain.Collection, error) {
	switch format {
	case "postman":
		return ParsePostmanCollection(data)
	case "openapi":
		return ParseOpenAPI(data)
	case "har":
		return ParseHAR(data)
	case "insomnia":
		return ParseInsomnia(data)
	case "curl":
		return importCurl(data)
	default:
		return domain.Collection{}, fmt.Errorf("unknown or unsupported format: %q", format)
	}
}

func importCurl(data []byte) (domain.Collection, error) {
	config, err := ParseCurl(string(data))
	if err != nil {
		return domain.Collection{}, err
	}

	return domain.Collection{
		Name: "cURL Import",
		Requests: []domain.SavedRequest{
			{
				Name:   deriveRequestName(string(config.Method), config.URL),
				Config: config,
			},
		},
	}, nil
}
