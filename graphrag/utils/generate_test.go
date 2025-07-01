package utils

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenCollectionIDs(t *testing.T) {
	tests := []struct {
		name        string
		vectorName  string
		expectEmpty bool
	}{
		{
			name:       "Normal vector name",
			vectorName: "user_documents",
		},
		{
			name:       "Name with spaces",
			vectorName: "User Documents",
		},
		{
			name:       "Empty name",
			vectorName: "",
		},
		{
			name:       "Name with special characters",
			vectorName: "test-collection@123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenCollectionIDs(tt.vectorName)

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

			// Check that Graph uses Vector as prefix
			if !strings.HasPrefix(result.Graph, result.Vector) {
				t.Errorf("Graph should use Vector as prefix, got Vector: %s, Graph: %s",
					result.Vector, result.Graph)
			}

			// Check that Store uses Vector as prefix
			if !strings.HasPrefix(result.Store, result.Vector) {
				t.Errorf("Store should use Vector as prefix, got Vector: %s, Store: %s",
					result.Vector, result.Store)
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

func TestCollectionIDsUniqueness(t *testing.T) {
	// Generate multiple collection IDs with same name to ensure timestamp uniqueness
	vectorName := "test_collection"
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		result := GenCollectionIDs(vectorName)

		if ids[result.Vector] {
			t.Errorf("Duplicate Vector generated: %s", result.Vector)
		}
		if ids[result.Graph] {
			t.Errorf("Duplicate Graph generated: %s", result.Graph)
		}
		if ids[result.Store] {
			t.Errorf("Duplicate Store generated: %s", result.Store)
		}

		ids[result.Vector] = true
		ids[result.Graph] = true
		ids[result.Store] = true
	}
}
