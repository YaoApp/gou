package json

import (
	"testing"
)

func TestNewValidator(t *testing.T) {
	tests := []struct {
		name    string
		schema  interface{}
		wantErr bool
	}{
		{
			name: "Valid schema from string",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "number"}
				},
				"required": ["name"]
			}`,
			wantErr: false,
		},
		{
			name: "Valid schema from map",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid schema from bytes",
			schema: []byte(`{
				"type": "string",
				"minLength": 1
			}`),
			wantErr: false,
		},
		{
			name:    "Invalid schema - empty",
			schema:  `{}`,
			wantErr: false, // Empty schema is technically valid
		},
		{
			name:    "Invalid schema - malformed JSON",
			schema:  `{invalid}`,
			wantErr: true,
		},
		{
			name: "Invalid schema - missing type keyword",
			schema: map[string]interface{}{
				"properties": "not an object",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewValidator(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewValidator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && validator == nil {
				t.Errorf("NewValidator() returned nil validator")
			}
			if !tt.wantErr && validator.schema == nil {
				t.Errorf("NewValidator() validator.schema is nil")
			}
		})
	}
}

func TestValidator_Validate(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "number", "minimum": 0},
		},
		"required": []interface{}{"name"},
	}

	validator, err := NewValidator(schema)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "Valid data",
			data: map[string]interface{}{
				"name": "test",
				"age":  25,
			},
			wantErr: false,
		},
		{
			name: "Valid data - only required fields",
			data: map[string]interface{}{
				"name": "test",
			},
			wantErr: false,
		},
		{
			name: "Invalid data - missing required field",
			data: map[string]interface{}{
				"age": 25,
			},
			wantErr: true,
		},
		{
			name: "Invalid data - wrong type for name",
			data: map[string]interface{}{
				"name": 123,
				"age":  25,
			},
			wantErr: true,
		},
		{
			name: "Invalid data - negative age",
			data: map[string]interface{}{
				"name": "test",
				"age":  -1,
			},
			wantErr: true,
		},
		{
			name: "Invalid data - age is string",
			data: map[string]interface{}{
				"name": "test",
				"age":  "25",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  interface{}
		wantErr bool
	}{
		{
			name: "Valid schema",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid schema - string type",
			schema: `{
				"type": "string",
				"minLength": 1
			}`,
			wantErr: false,
		},
		{
			name:    "Invalid schema - malformed",
			schema:  `{invalid`,
			wantErr: true,
		},
		{
			name: "Invalid schema - properties not object",
			schema: map[string]interface{}{
				"type":       "object",
				"properties": "should be object",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchema(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateData(t *testing.T) {
	tests := []struct {
		name    string
		schema  interface{}
		data    interface{}
		wantErr bool
	}{
		{
			name: "Valid data - simple string",
			schema: map[string]interface{}{
				"type": "string",
			},
			data:    "hello",
			wantErr: false,
		},
		{
			name: "Valid data - object",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
					"age":  map[string]interface{}{"type": "number"},
				},
			},
			data: map[string]interface{}{
				"name": "test",
				"age":  25,
			},
			wantErr: false,
		},
		{
			name: "Invalid data - type mismatch",
			schema: map[string]interface{}{
				"type": "string",
			},
			data:    123,
			wantErr: true,
		},
		{
			name: "Invalid schema - properties not object",
			schema: map[string]interface{}{
				"type":       "object",
				"properties": "invalid",
			},
			data:    "test",
			wantErr: true,
		},
		{
			name: "Valid data - array",
			schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "number",
				},
			},
			data:    []interface{}{1, 2, 3},
			wantErr: false,
		},
		{
			name: "Invalid data - array with wrong item type",
			schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "number",
				},
			},
			data:    []interface{}{1, "two", 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.data, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
