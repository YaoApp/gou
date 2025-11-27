package json

import (
	"encoding/json"
	"fmt"

	"github.com/kaptinlin/jsonschema"
)

// Validator wraps a compiled JSON Schema for validation
type Validator struct {
	schema *jsonschema.Schema
}

// NewValidator compiles a JSON Schema and returns a validator
// Returns error if the schema is invalid
//
// Args:
//   - schema: can be map[string]interface{}, []byte, string, or any JSON-serializable type
//
// Usage:
//
//	// From map
//	schemaMap := map[string]interface{}{
//	    "type": "object",
//	    "properties": map[string]interface{}{
//	        "name": map[string]interface{}{"type": "string"},
//	    },
//	    "required": []string{"name"},
//	}
//	validator, err := json.NewValidator(schemaMap)
//
//	// From JSON string
//	validator, err := json.NewValidator(`{"type": "object", "properties": {...}}`)
//
//	// From JSON bytes
//	validator, err := json.NewValidator([]byte(`{"type": "object", ...}`))
func NewValidator(schema interface{}) (*Validator, error) {
	var schemaBytes []byte
	var err error

	// Handle different input types
	switch v := schema.(type) {
	case string:
		// Already a JSON string
		schemaBytes = []byte(v)
	case []byte:
		// Already JSON bytes
		schemaBytes = v
	default:
		// Marshal to JSON
		schemaBytes, err = json.Marshal(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema: %w", err)
		}
	}

	// Compile the schema - this validates the schema structure
	compiler := jsonschema.NewCompiler()
	compiledSchema, err := compiler.Compile(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON Schema: %w", err)
	}

	return &Validator{
		schema: compiledSchema,
	}, nil
}

// Validate validates data against the compiled JSON Schema
// Returns nil if data is valid, error with validation details otherwise
//
// Usage:
//
//	validator, _ := json.NewValidator(schemaMap)
//	data := map[string]interface{}{"name": "John"}
//	if err := validator.Validate(data); err != nil {
//	    log.Printf("Validation failed: %v", err)
//	}
func (v *Validator) Validate(data interface{}) error {
	result := v.schema.Validate(data)
	if !result.IsValid() {
		// Collect all validation errors
		var errMsg string
		for field, err := range result.Errors {
			if errMsg != "" {
				errMsg += "; "
			}
			errMsg += fmt.Sprintf("%s: %s", field, err.Message)
		}
		return fmt.Errorf("validation failed: %s", errMsg)
	}

	return nil
}

// ValidateSchema validates a JSON Schema structure without compiling it
// Returns error if the schema is invalid
func ValidateSchema(schema interface{}) error {
	_, err := NewValidator(schema)
	return err
}

// Validate validates data against a JSON Schema (one-shot validation)
// Returns error if schema is invalid or data doesn't match the schema
//
// Usage:
//
//	err := json.Validate(data, schemaMap)
//	if err != nil {
//	    log.Printf("Validation failed: %v", err)
//	}
func Validate(data interface{}, schema interface{}) error {
	validator, err := NewValidator(schema)
	if err != nil {
		return err
	}
	return validator.Validate(data)
}
