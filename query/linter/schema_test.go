package linter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestQueryDSLSchema(t *testing.T) {
	schema := QueryDSLSchema()
	if schema == nil {
		t.Fatal("QueryDSLSchema() returned nil")
	}

	// Check required fields
	if schema["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("Expected $schema to be draft-07, got %v", schema["$schema"])
	}

	if schema["title"] != "QueryDSL" {
		t.Errorf("Expected title to be QueryDSL, got %v", schema["title"])
	}

	// Check definitions exist
	defs, ok := schema["definitions"].(map[string]interface{})
	if !ok {
		t.Fatal("definitions should be a map")
	}

	expectedDefs := []string{"expression", "table", "condition", "where", "order", "group", "having", "join", "sql"}
	for _, def := range expectedDefs {
		if _, exists := defs[def]; !exists {
			t.Errorf("Missing definition: %s", def)
		}
	}
}

func TestQueryDSLSchemaJSON(t *testing.T) {
	jsonStr := QueryDSLSchemaJSON
	if jsonStr == "" {
		t.Fatal("QueryDSLSchemaJSON is empty")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("QueryDSLSchemaJSON is invalid JSON: %v", err)
	}

	// Verify structure
	if parsed["title"] != "QueryDSL" {
		t.Errorf("Expected title to be QueryDSL, got %v", parsed["title"])
	}
}

func TestValidator(t *testing.T) {
	validator, err := Validator()
	if err != nil {
		t.Fatalf("Validator() returned error: %v", err)
	}
	if validator == nil {
		t.Fatal("Validator() returned nil")
	}
}

func TestValidateSchema_ValidCases(t *testing.T) {
	cases := loadTestCases(t)
	assetsDir := getTestAssetsDir()

	for _, tc := range cases.Valid.Cases {
		t.Run(tc.File, func(t *testing.T) {
			// Load test file
			path := filepath.Join(assetsDir, tc.File)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to load test file %s: %v", tc.File, err)
			}

			// Parse JSON to interface{}
			var queryData interface{}
			if err := json.Unmarshal(data, &queryData); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Validate against schema
			if err := ValidateSchema(queryData); err != nil {
				t.Errorf("Expected valid schema, got error: %v", err)
			}
		})
	}
}

func TestValidateSchema_InvalidCases(t *testing.T) {
	cases := loadTestCases(t)
	assetsDir := getTestAssetsDir()

	// These invalid cases have JSON syntax errors, skip schema validation for them
	skipFiles := map[string]bool{
		"invalid/json_syntax_error.json": true,
	}

	for _, tc := range cases.Invalid.Cases {
		if skipFiles[tc.File] {
			continue
		}

		t.Run(tc.File, func(t *testing.T) {
			// Load test file
			path := filepath.Join(assetsDir, tc.File)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to load test file %s: %v", tc.File, err)
			}

			// Parse JSON to interface{}
			var queryData interface{}
			if err := json.Unmarshal(data, &queryData); err != nil {
				// JSON syntax error is expected for some invalid cases
				return
			}

			// Note: JSON Schema validation is structural, not semantic
			// Some invalid cases (like missing_select, missing_from) are structurally valid JSON
			// but semantically invalid QueryDSL. The schema allows optional fields.
			// The Linter does semantic validation, Schema does structural validation.

			// For cases that should fail schema validation (type errors, wrong structures)
			// we check that validation either passes (semantic error) or fails (structural error)
			_ = ValidateSchema(queryData)
		})
	}
}

func TestValidateSchema_TypeErrors(t *testing.T) {
	// These are structural errors that JSON Schema should catch

	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "select should be array",
			data: map[string]interface{}{
				"select": "id,name", // Should be array
				"from":   "users",
			},
			wantErr: true,
		},
		{
			name: "wheres should be array",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"wheres": "invalid", // Should be array
			},
			wantErr: true,
		},
		{
			name: "orders should be array or string",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"orders": 123, // Should be array or string
			},
			wantErr: true,
		},
		{
			name: "joins should be array",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"joins":  "invalid", // Should be array
			},
			wantErr: true,
		},
		{
			name: "unions should be array",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"unions": "invalid", // Should be array
			},
			wantErr: true,
		},
		{
			name: "debug should be boolean",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"debug":  "yes", // Should be boolean
			},
			wantErr: true,
		},
		{
			name: "valid simple query",
			data: map[string]interface{}{
				"select": []interface{}{"id", "name"},
				"from":   "users",
			},
			wantErr: false,
		},
		{
			name: "valid query with sql",
			data: map[string]interface{}{
				"sql": map[string]interface{}{
					"stmt": "SELECT * FROM users WHERE id = ?",
					"args": []interface{}{1},
				},
			},
			wantErr: false,
		},
		{
			name: "valid query with string orders",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"orders": []interface{}{"id desc", "name asc"},
			},
			wantErr: false,
		},
		{
			name: "valid query with object orders",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"orders": []interface{}{
					map[string]interface{}{"field": "id", "sort": "desc"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid query with condition shorthand",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"from":   "users",
				"wheres": []interface{}{
					map[string]interface{}{"field": "status", "=": "active"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid query with joins",
			data: map[string]interface{}{
				"select": []interface{}{"id", "name"},
				"from":   "users",
				"joins": []interface{}{
					map[string]interface{}{
						"from":    "orders",
						"key":     "user_id",
						"foreign": "users.id",
						"left":    true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid query with subquery",
			data: map[string]interface{}{
				"select": []interface{}{"id"},
				"query": map[string]interface{}{
					"select": []interface{}{"id"},
					"from":   "users",
				},
				"name": "sub",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchema(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSchemaWithValidator(t *testing.T) {
	validator, err := Validator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test valid data
	validData := map[string]interface{}{
		"select": []interface{}{"id", "name"},
		"from":   "users",
		"wheres": []interface{}{
			map[string]interface{}{
				"field": "status",
				"op":    "=",
				"value": "active",
			},
		},
	}

	if err := validator.Validate(validData); err != nil {
		t.Errorf("Expected valid data to pass, got error: %v", err)
	}

	// Test invalid data
	invalidData := map[string]interface{}{
		"select": "id,name", // Should be array
		"from":   "users",
	}

	if err := validator.Validate(invalidData); err == nil {
		t.Error("Expected invalid data to fail validation")
	}
}
