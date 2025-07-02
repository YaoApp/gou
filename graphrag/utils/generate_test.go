package utils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Name with dashes - should fail",
			inputName:   "user-documents",
			expectError: true,
			errorMsg:    "invalid collection name format",
		},
		{
			name:        "Name with spaces - should fail",
			inputName:   "User Documents",
			expectError: true,
			errorMsg:    "invalid collection name format",
		},
		{
			name:        "Empty name - should fail",
			inputName:   "",
			expectError: true,
			errorMsg:    "collection name cannot be empty",
		},
		{
			name:        "Name with special characters - should fail",
			inputName:   "test-collection@123",
			expectError: true,
			errorMsg:    "invalid collection name format",
		},
		{
			name:      "Valid name with numbers",
			inputName: "collection123",
		},
		{
			name:      "Valid name with underscore",
			inputName: "test_collection",
		},
		{
			name:      "Valid name with underscores and numbers",
			inputName: "user_documents_123",
		},
		{
			name:      "Valid simple name",
			inputName: "documents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.inputName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGenCollectionID(t *testing.T) {
	prefixes := []string{"collection", "user", "doc", "test"}
	ids := make(map[string]bool)

	for _, prefix := range prefixes {
		for i := 0; i < 5; i++ {
			id := GenCollectionID(prefix)

			// Check that ID starts with prefix
			if !strings.HasPrefix(id, prefix) {
				t.Errorf("ID should start with prefix '%s', got: %s", prefix, id)
			}

			// Check uniqueness
			if ids[id] {
				t.Errorf("Duplicate ID generated: %s", id)
			}
			ids[id] = true

			t.Logf("Generated ID for prefix '%s': %s", prefix, id)
		}
	}
}

func TestGetCollectionIDs(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Name with dashes - should fail",
			inputName:   "user-documents",
			expectError: true,
			errorMsg:    "invalid collection name format",
		},
		{
			name:        "Name with spaces - should fail",
			inputName:   "User Documents",
			expectError: true,
			errorMsg:    "invalid collection name format",
		},
		{
			name:        "Empty name - should fail",
			inputName:   "",
			expectError: true,
			errorMsg:    "collection name cannot be empty",
		},
		{
			name:        "Name with special characters - should fail",
			inputName:   "test-collection@123",
			expectError: true,
			errorMsg:    "invalid collection name format",
		},
		{
			name:        "Name with dashes and numbers - should fail",
			inputName:   "Test-Collection-123",
			expectError: true,
			errorMsg:    "invalid collection name format",
		},
		{
			name:      "Valid name with underscore",
			inputName: "test_collection",
		},
		{
			name:      "Valid name with underscores and numbers",
			inputName: "Test_Collection_123",
		},
		{
			name:      "Valid simple name",
			inputName: "documents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetCollectionIDs(tt.inputName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check that all IDs are generated
			if result.Vector == "" {
				t.Error("Vector should not be empty")
			}
			if result.Graph == "" {
				t.Error("Graph should not be empty")
			}
			if result.Store == "" {
				t.Error("Store should not be empty")
			}

			// Check that Vector contains "vector"
			if !strings.Contains(result.Vector, "vector") {
				t.Errorf("Vector should contain 'vector', got: %s", result.Vector)
			}

			// Check that Graph contains "graph"
			if !strings.Contains(result.Graph, "graph") {
				t.Errorf("Graph should contain 'graph', got: %s", result.Graph)
			}

			// Check that Store contains "store"
			if !strings.Contains(result.Store, "store") {
				t.Errorf("Store should contain 'store', got: %s", result.Store)
			}

			// Check format: name_suffix
			expectedVector := fmt.Sprintf("%s_vector", strings.ToLower(strings.TrimSpace(tt.inputName)))
			expectedGraph := fmt.Sprintf("%s_graph", strings.ToLower(strings.TrimSpace(tt.inputName)))
			expectedStore := fmt.Sprintf("%s_store", strings.ToLower(strings.TrimSpace(tt.inputName)))

			if result.Vector != expectedVector {
				t.Errorf("Expected Vector: %s, got: %s", expectedVector, result.Vector)
			}
			if result.Graph != expectedGraph {
				t.Errorf("Expected Graph: %s, got: %s", expectedGraph, result.Graph)
			}
			if result.Store != expectedStore {
				t.Errorf("Expected Store: %s, got: %s", expectedStore, result.Store)
			}

			t.Logf("Generated Vector: %s", result.Vector)
			t.Logf("Generated Graph: %s", result.Graph)
			t.Logf("Generated Store: %s", result.Store)
		})
	}
}

func TestGenDocID(t *testing.T) {
	// Generate multiple IDs to ensure uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenDocID()

		// Check if it's a valid UUID
		if !IsValidUUID(id) {
			t.Errorf("Generated Doc ID is not a valid UUID: %s", id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate Doc ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenChunkID(t *testing.T) {
	// Generate multiple IDs to ensure uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenChunkID()

		// Check if it's a valid UUID
		if !IsValidUUID(id) {
			t.Errorf("Generated Chunk ID is not a valid UUID: %s", id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate Chunk ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenShortID(t *testing.T) {
	// Generate multiple IDs to ensure uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenShortID()

		// Check format (should contain underscore)
		if !strings.Contains(id, "_") {
			t.Errorf("Short ID should contain underscore, got: %s", id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate Short ID generated: %s", id)
		}
		ids[id] = true

		// Check length (should be reasonably short)
		if len(id) > 20 {
			t.Errorf("Short ID too long: %s (length: %d)", id, len(id))
		}
	}
}

func TestBatchGenDocIDs(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  int
	}{
		{
			name:  "Generate 5 IDs",
			count: 5,
			want:  5,
		},
		{
			name:  "Generate 0 IDs",
			count: 0,
			want:  0,
		},
		{
			name:  "Generate negative count",
			count: -1,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BatchGenDocIDs(tt.count)

			if len(result) != tt.want {
				t.Errorf("Expected %d IDs, got %d", tt.want, len(result))
			}

			// Check all IDs are valid UUIDs and unique
			seen := make(map[string]bool)
			for _, id := range result {
				if !IsValidUUID(id) {
					t.Errorf("Generated ID is not a valid UUID: %s", id)
				}
				if seen[id] {
					t.Errorf("Duplicate ID in batch: %s", id)
				}
				seen[id] = true
			}
		})
	}
}

func TestBatchGenChunkIDs(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  int
	}{
		{
			name:  "Generate 3 IDs",
			count: 3,
			want:  3,
		},
		{
			name:  "Generate 0 IDs",
			count: 0,
			want:  0,
		},
		{
			name:  "Generate negative count",
			count: -5,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BatchGenChunkIDs(tt.count)

			if len(result) != tt.want {
				t.Errorf("Expected %d IDs, got %d", tt.want, len(result))
			}

			// Check all IDs are valid UUIDs and unique
			seen := make(map[string]bool)
			for _, id := range result {
				if !IsValidUUID(id) {
					t.Errorf("Generated ID is not a valid UUID: %s", id)
				}
				if seen[id] {
					t.Errorf("Duplicate ID in batch: %s", id)
				}
				seen[id] = true
			}
		})
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "Valid UUID v4",
			id:   uuid.New().String(),
			want: true,
		},
		{
			name: "Valid UUID with uppercase",
			id:   "550E8400-E29B-41D4-A716-446655440000",
			want: true,
		},
		{
			name: "Invalid UUID - too short",
			id:   "123456",
			want: false,
		},
		{
			name: "Invalid UUID - wrong format",
			id:   "not-a-uuid",
			want: false,
		},
		{
			name: "Empty string",
			id:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidUUID(tt.id)
			if result != tt.want {
				t.Errorf("IsValidUUID(%s) = %v, want %v", tt.id, result, tt.want)
			}
		})
	}
}

func TestGetCollectionIDsUniqueness(t *testing.T) {
	// Test that GetCollectionIDs with same name produces consistent results
	vectorName := "test_collection"

	result1, err1 := GetCollectionIDs(vectorName)
	if err1 != nil {
		t.Fatalf("Unexpected error: %v", err1)
	}

	result2, err2 := GetCollectionIDs(vectorName)
	if err2 != nil {
		t.Fatalf("Unexpected error: %v", err2)
	}

	// Should produce same results for same input
	if result1.Vector != result2.Vector {
		t.Errorf("GetCollectionIDs should be deterministic for Vector, got: %s vs %s", result1.Vector, result2.Vector)
	}
	if result1.Graph != result2.Graph {
		t.Errorf("GetCollectionIDs should be deterministic for Graph, got: %s vs %s", result1.Graph, result2.Graph)
	}
	if result1.Store != result2.Store {
		t.Errorf("GetCollectionIDs should be deterministic for Store, got: %s vs %s", result1.Store, result2.Store)
	}
}

func TestExtractCollectionIDFromVectorName(t *testing.T) {
	tests := []struct {
		name        string
		vectorName  string
		expected    string
		description string
	}{
		{
			name:        "Valid vector name with suffix",
			vectorName:  "user_docs_vector",
			expected:    "user_docs",
			description: "Should extract collection ID from valid vector name",
		},
		{
			name:        "Vector name without suffix",
			vectorName:  "user_docs",
			expected:    "",
			description: "Should return empty string if no _vector suffix (strict mode)",
		},
		{
			name:        "Complex collection name",
			vectorName:  "my_test_collection_123_vector",
			expected:    "my_test_collection_123",
			description: "Should handle complex collection names",
		},
		{
			name:        "Empty string",
			vectorName:  "",
			expected:    "",
			description: "Should handle empty string",
		},
		{
			name:        "Only suffix",
			vectorName:  "_vector",
			expected:    "",
			description: "Should handle edge case of only suffix",
		},
		{
			name:        "Whitespace handling",
			vectorName:  "  test_collection_vector  ",
			expected:    "test_collection",
			description: "Should trim whitespace and extract correctly",
		},
		{
			name:        "Multiple vector suffixes",
			vectorName:  "test_vector_vector",
			expected:    "test_vector",
			description: "Should only remove the last _vector suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCollectionIDFromVectorName(tt.vectorName)
			if result != tt.expected {
				t.Errorf("ExtractCollectionIDFromVectorName(%q) = %q, expected %q. %s",
					tt.vectorName, result, tt.expected, tt.description)
			}
		})
	}
}

func TestExtractCollectionIDFromGraphName(t *testing.T) {
	tests := []struct {
		name        string
		graphName   string
		expected    string
		description string
	}{
		{
			name:        "Valid graph name with suffix",
			graphName:   "user_docs_graph",
			expected:    "user_docs",
			description: "Should extract collection ID from valid graph name",
		},
		{
			name:        "Graph name without suffix",
			graphName:   "user_docs",
			expected:    "",
			description: "Should return empty string if no _graph suffix (strict mode)",
		},
		{
			name:        "Complex collection name",
			graphName:   "my_test_collection_123_graph",
			expected:    "my_test_collection_123",
			description: "Should handle complex collection names",
		},
		{
			name:        "Empty string",
			graphName:   "",
			expected:    "",
			description: "Should handle empty string",
		},
		{
			name:        "Only suffix",
			graphName:   "_graph",
			expected:    "",
			description: "Should handle edge case of only suffix",
		},
		{
			name:        "Whitespace handling",
			graphName:   "  test_collection_graph  ",
			expected:    "test_collection",
			description: "Should trim whitespace and extract correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCollectionIDFromGraphName(tt.graphName)
			if result != tt.expected {
				t.Errorf("ExtractCollectionIDFromGraphName(%q) = %q, expected %q. %s",
					tt.graphName, result, tt.expected, tt.description)
			}
		})
	}
}

func TestExtractCollectionIDFromStoreName(t *testing.T) {
	tests := []struct {
		name        string
		storeName   string
		expected    string
		description string
	}{
		{
			name:        "Valid store name with suffix",
			storeName:   "user_docs_store",
			expected:    "user_docs",
			description: "Should extract collection ID from valid store name",
		},
		{
			name:        "Store name without suffix",
			storeName:   "user_docs",
			expected:    "",
			description: "Should return empty string if no _store suffix (strict mode)",
		},
		{
			name:        "Complex collection name",
			storeName:   "my_test_collection_123_store",
			expected:    "my_test_collection_123",
			description: "Should handle complex collection names",
		},
		{
			name:        "Empty string",
			storeName:   "",
			expected:    "",
			description: "Should handle empty string",
		},
		{
			name:        "Only suffix",
			storeName:   "_store",
			expected:    "",
			description: "Should handle edge case of only suffix",
		},
		{
			name:        "Whitespace handling",
			storeName:   "  test_collection_store  ",
			expected:    "test_collection",
			description: "Should trim whitespace and extract correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCollectionIDFromStoreName(tt.storeName)
			if result != tt.expected {
				t.Errorf("ExtractCollectionIDFromStoreName(%q) = %q, expected %q. %s",
					tt.storeName, result, tt.expected, tt.description)
			}
		})
	}
}

func TestRoundTripCollectionIDs(t *testing.T) {
	// Test that GetCollectionIDs and Extract functions are proper inverses
	testNames := []string{
		"user_docs",
		"my_test_collection",
		"simple",
		"complex_name_with_underscores_123",
		"TEST_MIXED_CASE", // This will be converted to lowercase
	}

	for _, originalName := range testNames {
		t.Run(fmt.Sprintf("RoundTrip_%s", originalName), func(t *testing.T) {
			// Generate collection IDs
			ids, err := GetCollectionIDs(originalName)
			if err != nil {
				t.Fatalf("GetCollectionIDs failed for %s: %v", originalName, err)
			}

			// Extract back the collection IDs
			extractedFromVector := ExtractCollectionIDFromVectorName(ids.Vector)
			extractedFromGraph := ExtractCollectionIDFromGraphName(ids.Graph)
			extractedFromStore := ExtractCollectionIDFromStoreName(ids.Store)

			// GetCollectionIDs converts to lowercase, so we need to compare with lowercase
			expectedName := strings.ToLower(originalName)

			// All extractions should give us back the original collection name
			if extractedFromVector != expectedName {
				t.Errorf("Vector round-trip failed: %s -> %s -> %s", originalName, ids.Vector, extractedFromVector)
			}
			if extractedFromGraph != expectedName {
				t.Errorf("Graph round-trip failed: %s -> %s -> %s", originalName, ids.Graph, extractedFromGraph)
			}
			if extractedFromStore != expectedName {
				t.Errorf("Store round-trip failed: %s -> %s -> %s", originalName, ids.Store, extractedFromStore)
			}

			t.Logf("Round-trip success for '%s': Vector=%s, Graph=%s, Store=%s",
				originalName, ids.Vector, ids.Graph, ids.Store)
		})
	}
}
