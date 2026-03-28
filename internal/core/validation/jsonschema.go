package validation

import (
	"encoding/json"
	"fmt"
	"math"
)

// ValidateJSONSchema validates JSON data against a JSON Schema.
// It supports a subset of JSON Schema Draft 4/6/7 covering the most common cases:
// type, required, properties, minimum/maximum, minLength/maxLength, and enum.
func ValidateJSONSchema(data []byte, schema []byte) (bool, []string, error) {
	var dataVal interface{}
	if err := json.Unmarshal(data, &dataVal); err != nil {
		return false, nil, fmt.Errorf("invalid JSON data: %w", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return false, nil, fmt.Errorf("invalid JSON schema: %w", err)
	}

	errs := validate(dataVal, schemaMap, "")
	return len(errs) == 0, errs, nil
}

func validate(data interface{}, schema map[string]interface{}, path string) []string {
	var errs []string

	// Check type constraint.
	if schemaType, ok := schema["type"].(string); ok {
		if typeErr := checkType(data, schemaType, path); typeErr != "" {
			errs = append(errs, typeErr)
			// If the type is wrong, skip further validation for this node.
			return errs
		}
	}

	// Check enum constraint.
	if enumVal, ok := schema["enum"]; ok {
		if enumList, ok := enumVal.([]interface{}); ok {
			if enumErr := checkEnum(data, enumList, path); enumErr != "" {
				errs = append(errs, enumErr)
			}
		}
	}

	// Numeric constraints.
	if numVal, isNum := toNumber(data); isNum {
		if min, ok := schema["minimum"]; ok {
			if minVal, isMinNum := toNumber(min); isMinNum {
				if numVal < minVal {
					errs = append(errs, fmtPath(path, fmt.Sprintf("value %v is less than minimum %v", numVal, minVal)))
				}
			}
		}
		if max, ok := schema["maximum"]; ok {
			if maxVal, isMaxNum := toNumber(max); isMaxNum {
				if numVal > maxVal {
					errs = append(errs, fmtPath(path, fmt.Sprintf("value %v is greater than maximum %v", numVal, maxVal)))
				}
			}
		}
	}

	// String constraints.
	if strVal, ok := data.(string); ok {
		if minLen, ok := schema["minLength"]; ok {
			if minLenVal, isNum := toNumber(minLen); isNum {
				if len(strVal) < int(minLenVal) {
					errs = append(errs, fmtPath(path, fmt.Sprintf("string length %d is less than minLength %d", len(strVal), int(minLenVal))))
				}
			}
		}
		if maxLen, ok := schema["maxLength"]; ok {
			if maxLenVal, isNum := toNumber(maxLen); isNum {
				if len(strVal) > int(maxLenVal) {
					errs = append(errs, fmtPath(path, fmt.Sprintf("string length %d is greater than maxLength %d", len(strVal), int(maxLenVal))))
				}
			}
		}
	}

	// Object constraints.
	if objMap, ok := data.(map[string]interface{}); ok {
		// Check required fields.
		if req, ok := schema["required"]; ok {
			if reqList, ok := req.([]interface{}); ok {
				for _, r := range reqList {
					if fieldName, ok := r.(string); ok {
						if _, exists := objMap[fieldName]; !exists {
							errs = append(errs, fmtPath(path, fmt.Sprintf("missing required field %q", fieldName)))
						}
					}
				}
			}
		}

		// Validate properties recursively.
		if props, ok := schema["properties"]; ok {
			if propsMap, ok := props.(map[string]interface{}); ok {
				for propName, propSchema := range propsMap {
					if propSchemaMap, ok := propSchema.(map[string]interface{}); ok {
						if propVal, exists := objMap[propName]; exists {
							childPath := propName
							if path != "" {
								childPath = path + "." + propName
							}
							childErrs := validate(propVal, propSchemaMap, childPath)
							errs = append(errs, childErrs...)
						}
					}
				}
			}
		}
	}

	return errs
}

func checkType(data interface{}, expectedType string, path string) string {
	actual := jsonType(data)
	switch expectedType {
	case "string":
		if actual != "string" {
			return fmtPath(path, fmt.Sprintf("expected type %q, got %q", expectedType, actual))
		}
	case "number":
		if actual != "number" {
			return fmtPath(path, fmt.Sprintf("expected type %q, got %q", expectedType, actual))
		}
	case "integer":
		if actual != "number" {
			return fmtPath(path, fmt.Sprintf("expected type %q, got %q", expectedType, actual))
		}
		// Additionally check that the number is a whole number.
		if numVal, ok := data.(float64); ok {
			if numVal != math.Trunc(numVal) {
				return fmtPath(path, fmt.Sprintf("expected integer, got float %v", numVal))
			}
		}
	case "boolean":
		if actual != "boolean" {
			return fmtPath(path, fmt.Sprintf("expected type %q, got %q", expectedType, actual))
		}
	case "object":
		if actual != "object" {
			return fmtPath(path, fmt.Sprintf("expected type %q, got %q", expectedType, actual))
		}
	case "array":
		if actual != "array" {
			return fmtPath(path, fmt.Sprintf("expected type %q, got %q", expectedType, actual))
		}
	case "null":
		if actual != "null" {
			return fmtPath(path, fmt.Sprintf("expected type %q, got %q", expectedType, actual))
		}
	}
	return ""
}

func checkEnum(data interface{}, enumList []interface{}, path string) string {
	dataJSON, _ := json.Marshal(data)
	for _, allowed := range enumList {
		allowedJSON, _ := json.Marshal(allowed)
		if string(dataJSON) == string(allowedJSON) {
			return ""
		}
	}
	return fmtPath(path, fmt.Sprintf("value %v is not in enum %v", data, enumList))
}

func jsonType(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return fmt.Sprintf("unknown(%T)", v)
	}
}

func toNumber(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func fmtPath(path, msg string) string {
	if path == "" {
		return msg
	}
	return path + ": " + msg
}
