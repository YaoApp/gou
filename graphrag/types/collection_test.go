package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSerializeCollection(t *testing.T) {
	tests := []struct {
		name        string
		collection  Collection
		expectError bool
		description string
	}{
		{
			name: "Valid basic collection",
			collection: Collection{
				ID: "test_collection",
				Metadata: map[string]interface{}{
					"type":      "document",
					"category":  "research",
					"count":     10,
					"is_active": true,
				},
			},
			expectError: false,
			description: "Should serialize a basic collection successfully",
		},
		{
			name: "Collection with nil metadata",
			collection: Collection{
				ID:       "test_nil_metadata",
				Metadata: nil,
			},
			expectError: false,
			description: "Should handle nil metadata",
		},
		{
			name: "Collection with empty metadata",
			collection: Collection{
				ID:       "test_empty_metadata",
				Metadata: map[string]interface{}{},
			},
			expectError: false,
			description: "Should handle empty metadata",
		},
		{
			name: "Collection with complex metadata",
			collection: Collection{
				ID: "complex_collection",
				Metadata: map[string]interface{}{
					"nested": map[string]interface{}{
						"level1": map[string]interface{}{
							"level2": "deep_value",
							"number": 42,
						},
						"array": []interface{}{1, 2, "three"},
					},
					"float_val": 3.14159,
					"bool_val":  false,
				},
			},
			expectError: false,
			description: "Should handle complex nested metadata",
		},
		{
			name: "Empty collection ID",
			collection: Collection{
				ID: "",
				Metadata: map[string]interface{}{
					"test": "value",
				},
			},
			expectError: false,
			description: "Should serialize even with empty ID (validation is separate)",
		},
		{
			name: "Collection with unserializable metadata (function)",
			collection: Collection{
				ID: "error_collection",
				Metadata: map[string]interface{}{
					"function": func() {},
				},
			},
			expectError: true,
			description: "Should return error for unserializable data types",
		},
		{
			name: "Collection with unserializable metadata (channel)",
			collection: Collection{
				ID: "error_collection_channel",
				Metadata: map[string]interface{}{
					"channel": make(chan int),
				},
			},
			expectError: true,
			description: "Should return error for channel type in metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SerializeCollection(tt.collection)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. %s", tt.description)
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v. %s", err, tt.description)
				return
			}

			// Verify the result is valid JSON
			var testUnmarshal map[string]interface{}
			if err := json.Unmarshal([]byte(result), &testUnmarshal); err != nil {
				t.Errorf("Serialized result is not valid JSON: %v", err)
				return
			}

			// Verify the ID is preserved
			if id, ok := testUnmarshal["id"].(string); ok {
				if id != tt.collection.ID {
					t.Errorf("ID not preserved: expected %s, got %s", tt.collection.ID, id)
				}
			}

			t.Logf("Serialized successfully: %s", result)
		})
	}
}

func TestDeserializeCollection(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectedID  string
		description string
	}{
		{
			name:        "Valid JSON with basic data",
			jsonData:    `{"id":"test_collection","metadata":{"type":"document","count":5}}`,
			expectError: false,
			expectedID:  "test_collection",
			description: "Should deserialize valid JSON successfully",
		},
		{
			name:        "Valid JSON with null metadata",
			jsonData:    `{"id":"test-null","metadata":null}`,
			expectError: false,
			expectedID:  "test-null",
			description: "Should handle null metadata",
		},
		{
			name:        "Valid JSON with empty metadata",
			jsonData:    `{"id":"test-empty","metadata":{}}`,
			expectError: false,
			expectedID:  "test-empty",
			description: "Should handle empty metadata object",
		},
		{
			name:        "Valid JSON with complex nested data",
			jsonData:    `{"id":"complex","metadata":{"nested":{"level1":{"level2":"value"},"array":[1,2,"three"]},"float":3.14}}`,
			expectError: false,
			expectedID:  "complex",
			description: "Should handle complex nested JSON",
		},
		{
			name:        "Invalid JSON - malformed",
			jsonData:    `{"id":"test","metadata":{"invalid":}`,
			expectError: true,
			description: "Should return error for malformed JSON",
		},
		{
			name:        "Invalid JSON - empty string",
			jsonData:    "",
			expectError: true,
			description: "Should return error for empty string",
		},
		{
			name:        "Invalid JSON - not an object",
			jsonData:    `"just a string"`,
			expectError: true,
			description: "Should return error for non-object JSON",
		},
		{
			name:        "Invalid JSON - array instead of object",
			jsonData:    `[{"id":"test"}]`,
			expectError: true,
			description: "Should return error for array JSON",
		},
		{
			name:        "Valid minimal JSON",
			jsonData:    `{"id":"minimal"}`,
			expectError: false,
			expectedID:  "minimal",
			description: "Should handle minimal valid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeCollection(tt.jsonData)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. %s", tt.description)
					return
				}
				if !strings.Contains(err.Error(), "failed to deserialize collection") {
					t.Errorf("Error message should contain 'failed to deserialize collection', got: %s", err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v. %s", err, tt.description)
				return
			}

			// Verify the ID is correct
			if result.ID != tt.expectedID {
				t.Errorf("Expected ID %s, got %s", tt.expectedID, result.ID)
			}

			t.Logf("Deserialized successfully: ID=%s, Metadata=%+v", result.ID, result.Metadata)
		})
	}
}

func TestValidateCollection(t *testing.T) {
	tests := []struct {
		name        string
		collection  Collection
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name: "Valid collection with basic data",
			collection: Collection{
				ID: "valid-collection",
				Metadata: map[string]interface{}{
					"type": "document",
				},
			},
			expectError: false,
			description: "Should validate a basic valid collection",
		},
		{
			name: "Invalid collection - empty ID",
			collection: Collection{
				ID: "",
				Metadata: map[string]interface{}{
					"type": "document",
				},
			},
			expectError: true,
			errorMsg:    "collection ID cannot be empty",
			description: "Should return error for empty ID",
		},
		{
			name: "Valid collection with VectorConfig",
			collection: Collection{
				ID: "vector-collection",
				VectorConfig: &VectorStoreConfig{
					CollectionName: "test-vector",
					Dimension:      512,
					Distance:       DistanceCosine,
					IndexType:      IndexTypeHNSW,
				},
			},
			expectError: false,
			description: "Should validate collection with valid VectorConfig",
		},
		{
			name: "Invalid collection - invalid VectorConfig",
			collection: Collection{
				ID: "invalid-vector-collection",
				VectorConfig: &VectorStoreConfig{
					CollectionName: "", // Invalid - empty collection name
					Dimension:      512,
					Distance:       DistanceCosine,
					IndexType:      IndexTypeHNSW,
				},
			},
			expectError: true,
			errorMsg:    "invalid vector config",
			description: "Should return error for invalid VectorConfig",
		},

		{
			name: "Valid collection with VectorConfig",
			collection: Collection{
				ID: "full-collection",
				VectorConfig: &VectorStoreConfig{
					CollectionName: "test-both",
					Dimension:      256,
					Distance:       DistanceEuclidean,
					IndexType:      IndexTypeIVF,
				},
			},
			expectError: false,
			description: "Should validate collection with VectorConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCollection(tt.collection)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. %s", tt.description)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v. %s", err, tt.description)
			}
		})
	}
}

func TestCloneCollection(t *testing.T) {
	tests := []struct {
		name        string
		original    Collection
		expectError bool
		description string
	}{
		{
			name: "Clone basic collection",
			original: Collection{
				ID: "original-collection",
				Metadata: map[string]interface{}{
					"type":   "document",
					"count":  42,
					"active": true,
				},
			},
			expectError: false,
			description: "Should clone a basic collection successfully",
		},
		{
			name: "Clone collection with nil metadata",
			original: Collection{
				ID:       "nil-metadata",
				Metadata: nil,
			},
			expectError: false,
			description: "Should clone collection with nil metadata",
		},
		{
			name: "Clone collection with complex nested metadata",
			original: Collection{
				ID: "complex-collection",
				Metadata: map[string]interface{}{
					"nested": map[string]interface{}{
						"level1": map[string]interface{}{
							"level2": "deep_value",
							"array":  []interface{}{1, 2, "three", true},
						},
						"number": 123.456,
					},
					"simple": "value",
				},
			},
			expectError: false,
			description: "Should clone collection with complex nested metadata",
		},
		{
			name: "Clone collection with VectorConfig",
			original: Collection{
				ID: "vector-collection",
				Metadata: map[string]interface{}{
					"type": "vector",
				},
				VectorConfig: &VectorStoreConfig{
					CollectionName: "test-clone-vector",
					Dimension:      768,
					Distance:       DistanceCosine,
					IndexType:      IndexTypeHNSW,
				},
			},
			expectError: false,
			description: "Should clone collection with VectorConfig",
		},

		{
			name: "Clone collection with unserializable metadata",
			original: Collection{
				ID: "error-collection",
				Metadata: map[string]interface{}{
					"function": func() {},
				},
			},
			expectError: true,
			description: "Should return error when cloning collection with unserializable metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned, err := CloneCollection(tt.original)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. %s", tt.description)
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v. %s", err, tt.description)
				return
			}

			// Verify the clone has the same ID
			if cloned.ID != tt.original.ID {
				t.Errorf("Clone ID mismatch: expected %s, got %s", tt.original.ID, cloned.ID)
			}

			// Verify metadata is deeply cloned
			if tt.original.Metadata != nil {
				if cloned.Metadata == nil {
					t.Error("Cloned metadata should not be nil when original is not nil")
					return
				}

				// Check that modifying cloned metadata doesn't affect original
				if len(cloned.Metadata) > 0 {
					// Add a new key to cloned metadata
					cloned.Metadata["__test_clone__"] = "modified"

					// Original should not have this key
					if _, exists := tt.original.Metadata["__test_clone__"]; exists {
						t.Error("Modifying cloned metadata affected original - not a deep copy")
					}
				}
			}

			// Verify VectorConfig is cloned (if it exists)
			if tt.original.VectorConfig != nil {
				if cloned.VectorConfig == nil {
					t.Error("Cloned VectorConfig should not be nil when original is not nil")
				} else {
					// Verify it's a different pointer (deep copy)
					if &tt.original.VectorConfig == &cloned.VectorConfig {
						t.Error("VectorConfig should be deeply cloned, not just copied by reference")
					}
					// Verify values are the same
					if cloned.VectorConfig.CollectionName != tt.original.VectorConfig.CollectionName {
						t.Errorf("VectorConfig CollectionName mismatch: expected %s, got %s",
							tt.original.VectorConfig.CollectionName, cloned.VectorConfig.CollectionName)
					}
				}
			}

			t.Logf("Successfully cloned collection: %s", cloned.ID)
		})
	}
}

func TestRoundTripSerializeDeserialize(t *testing.T) {
	// Test that SerializeCollection and DeserializeCollection are proper inverses
	testCollections := []Collection{
		{
			ID: "simple-roundtrip",
			Metadata: map[string]interface{}{
				"type": "test",
			},
		},
		{
			ID: "complex-roundtrip",
			Metadata: map[string]interface{}{
				"nested": map[string]interface{}{
					"level1": map[string]interface{}{
						"level2": "value",
						"number": 42,
						"bool":   true,
					},
					"array": []interface{}{1, "two", 3.14, false},
				},
				"simple_string": "test",
				"simple_number": 123,
				"simple_bool":   true,
				"simple_float":  3.14159,
			},
		},
		{
			ID:       "nil-metadata-roundtrip",
			Metadata: nil,
		},
		{
			ID:       "empty-metadata-roundtrip",
			Metadata: map[string]interface{}{},
		},
	}

	for i, original := range testCollections {
		t.Run(fmt.Sprintf("RoundTrip_%d_%s", i+1, original.ID), func(t *testing.T) {
			// Serialize
			serialized, err := SerializeCollection(original)
			if err != nil {
				t.Fatalf("Serialization failed: %v", err)
			}

			// Deserialize
			deserialized, err := DeserializeCollection(serialized)
			if err != nil {
				t.Fatalf("Deserialization failed: %v", err)
			}

			// Verify round-trip integrity
			if deserialized.ID != original.ID {
				t.Errorf("ID round-trip failed: expected %s, got %s", original.ID, deserialized.ID)
			}

			// For metadata comparison, we need to handle the case where nil becomes empty map
			if original.Metadata == nil {
				// After JSON round-trip, nil metadata might become nil or empty map
				// Both are acceptable
				if len(deserialized.Metadata) > 0 {
					t.Errorf("Nil metadata should remain nil or become empty, got: %+v", deserialized.Metadata)
				}
			} else {
				if deserialized.Metadata == nil {
					t.Error("Non-nil metadata should not become nil after round-trip")
				}
			}

			t.Logf("Round-trip successful for collection: %s", original.ID)
		})
	}
}
