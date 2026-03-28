package validation_test

import (
	"testing"

	"github.com/ye-kart/reqflow/internal/core/validation"
)

func TestValidateJSONSchema_ValidObject(t *testing.T) {
	data := []byte(`{"name": "John", "age": 30}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Errorf("expected valid, got errors: %v", errs)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateJSONSchema_MissingRequiredField(t *testing.T) {
	data := []byte(`{"name": "John"}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for missing required field")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_WrongType(t *testing.T) {
	data := []byte(`{"name": 42}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for wrong type")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_NumberOutOfRange(t *testing.T) {
	data := []byte(`{"age": 150}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"age": {"type": "integer", "minimum": 0, "maximum": 120}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for number out of range")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_NumberBelowMinimum(t *testing.T) {
	data := []byte(`{"age": -5}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"age": {"type": "number", "minimum": 0}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for number below minimum")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_StringTooShort(t *testing.T) {
	data := []byte(`{"name": "ab"}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 3}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for string too short")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_StringTooLong(t *testing.T) {
	data := []byte(`{"name": "abcdefghijk"}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "maxLength": 5}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for string too long")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_EnumValueNotInList(t *testing.T) {
	data := []byte(`{"status": "unknown"}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"status": {"type": "string", "enum": ["active", "inactive", "pending"]}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for enum value not in list")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_EnumValueValid(t *testing.T) {
	data := []byte(`{"status": "active"}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"status": {"type": "string", "enum": ["active", "inactive", "pending"]}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Errorf("expected valid, got errors: %v", errs)
	}
}

func TestValidateJSONSchema_NestedObjectValidation(t *testing.T) {
	data := []byte(`{"user": {"name": "John", "address": {"city": 123}}}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"address": {
						"type": "object",
						"properties": {
							"city": {"type": "string"}
						}
					}
				}
			}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for nested wrong type")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_ValidNestedObject(t *testing.T) {
	data := []byte(`{"user": {"name": "John", "address": {"city": "NYC"}}}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"address": {
						"type": "object",
						"properties": {
							"city": {"type": "string"}
						}
					}
				}
			}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Errorf("expected valid, got errors: %v", errs)
	}
}

func TestValidateJSONSchema_ArrayType(t *testing.T) {
	data := []byte(`{"items": "not-an-array"}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"items": {"type": "array"}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for non-array value with array type")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_NullType(t *testing.T) {
	data := []byte(`{"value": null}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"value": {"type": "null"}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Errorf("expected valid for null value, got errors: %v", errs)
	}
}

func TestValidateJSONSchema_BooleanType(t *testing.T) {
	data := []byte(`{"active": true}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"active": {"type": "boolean"}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Errorf("expected valid for boolean, got errors: %v", errs)
	}
}

func TestValidateJSONSchema_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json}`)
	schema := []byte(`{"type": "object"}`)

	_, _, err := validation.ValidateJSONSchema(data, schema)
	if err == nil {
		t.Error("expected error for invalid JSON data")
	}
}

func TestValidateJSONSchema_InvalidSchema(t *testing.T) {
	data := []byte(`{"name": "John"}`)
	schema := []byte(`{invalid schema}`)

	_, _, err := validation.ValidateJSONSchema(data, schema)
	if err == nil {
		t.Error("expected error for invalid JSON schema")
	}
}

func TestValidateJSONSchema_TopLevelTypeCheck(t *testing.T) {
	data := []byte(`"just a string"`)
	schema := []byte(`{"type": "object"}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for string when object expected")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateJSONSchema_NumberType_AcceptsFloat(t *testing.T) {
	data := []byte(`{"price": 19.99}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"price": {"type": "number"}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Errorf("expected valid for float with number type, got errors: %v", errs)
	}
}

func TestValidateJSONSchema_IntegerType_RejectsFloat(t *testing.T) {
	data := []byte(`{"count": 3.5}`)
	schema := []byte(`{
		"type": "object",
		"properties": {
			"count": {"type": "integer"}
		}
	}`)

	valid, errs, err := validation.ValidateJSONSchema(data, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid for float with integer type")
	}
	if len(errs) == 0 {
		t.Error("expected at least one error")
	}
}
