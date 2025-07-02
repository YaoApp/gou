package graphrag

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/log"
)

// TestCreateCollection tests the CreateCollection function
func TestCreateCollection(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+graph", "vector+system", "complete"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Fatalf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create GraphRag instance: %v", err)
			}

			// Test cases for CreateCollection
			testCases := []struct {
				name        string
				collection  types.Collection
				expectError bool
				description string
			}{
				{
					name: "Valid_collection_with_ID",
					collection: types.Collection{
						ID: "test-collection-001",
						Metadata: map[string]interface{}{
							"type":        "test",
							"description": "Test collection",
						},
						VectorConfig: &types.VectorStoreConfig{
							Dimension: 128,
							Distance:  types.DistanceCosine,
							IndexType: types.IndexTypeHNSW,
						},
					},
					expectError: false,
					description: "Should create collection with provided ID",
				},
				{
					name: "Valid_collection_without_ID",
					collection: types.Collection{
						Metadata: map[string]interface{}{
							"type": "auto-generated",
						},
						VectorConfig: &types.VectorStoreConfig{
							Dimension: 128,
							Distance:  types.DistanceCosine,
							IndexType: types.IndexTypeHNSW,
						},
					},
					expectError: false,
					description: "Should create collection with auto-generated ID",
				},
				{
					name: "Collection_with_graph_config",
					collection: types.Collection{
						ID: "test-collection-graph",
						Metadata: map[string]interface{}{
							"type": "graph",
						},
						VectorConfig: &types.VectorStoreConfig{
							Dimension: 128,
							Distance:  types.DistanceCosine,
							IndexType: types.IndexTypeHNSW,
						},
					},
					expectError: false,
					description: "Should create collection with graph config",
				},
				{
					name: "Collection_with_nil_metadata",
					collection: types.Collection{
						ID:       "test-collection-nil-meta",
						Metadata: nil,
						VectorConfig: &types.VectorStoreConfig{
							Dimension: 128,
							Distance:  types.DistanceCosine,
							IndexType: types.IndexTypeHNSW,
						},
					},
					expectError: false,
					description: "Should handle nil metadata",
				},
				{
					name: "Collection_without_vector_config",
					collection: types.Collection{
						ID: "test-collection-no-vector",
						Metadata: map[string]interface{}{
							"type": "no-vector",
						},
					},
					expectError: false,
					description: "Should handle collection without vector config",
				},
			}

			// Keep track of created collections for cleanup
			var createdCollections []string

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					// Create collection
					collectionID, err := g.CreateCollection(ctx, tc.collection)

					if tc.expectError {
						if err == nil {
							t.Errorf("Expected error but got none. %s", tc.description)
						}
						return
					}

					if err != nil {
						t.Errorf("Unexpected error: %v. %s", err, tc.description)
						return
					}

					// Verify collection ID
					if collectionID == "" {
						t.Error("Collection ID should not be empty")
						return
					}

					// Add to cleanup list
					createdCollections = append(createdCollections, collectionID)

					// Verify collection exists
					exists, err := g.CollectionExists(ctx, collectionID)
					if err != nil {
						t.Errorf("Failed to check collection existence: %v", err)
						return
					}
					if !exists {
						t.Error("Collection should exist after creation")
						return
					}

					t.Logf("Successfully created collection: %s", collectionID)
				})
			}

			// Test duplicate collection creation
			t.Run("Duplicate_collection_creation", func(t *testing.T) {
				ctx := context.Background()
				collection := types.Collection{
					ID: "duplicate-test",
					Metadata: map[string]interface{}{
						"type": "duplicate",
					},
					VectorConfig: &types.VectorStoreConfig{
						Dimension: 128,
						Distance:  types.DistanceCosine,
						IndexType: types.IndexTypeHNSW,
					},
				}

				// Create first collection
				collectionID, err := g.CreateCollection(ctx, collection)
				if err != nil {
					t.Fatalf("Failed to create first collection: %v", err)
				}
				createdCollections = append(createdCollections, collectionID)

				// Try to create duplicate
				_, err = g.CreateCollection(ctx, collection)
				if err == nil {
					t.Error("Expected error for duplicate collection creation")
				}
			})

			// Cleanup created collections
			t.Cleanup(func() {
				ctx := context.Background()
				for _, collectionID := range createdCollections {
					_, err := g.RemoveCollection(ctx, collectionID)
					if err != nil {
						t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
					}
				}
			})
		})
	}
}

// TestRemoveCollection tests the RemoveCollection function
func TestRemoveCollection(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+graph", "vector+system", "complete"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Fatalf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create GraphRag instance: %v", err)
			}

			ctx := context.Background()

			// Test cases for RemoveCollection
			testCases := []struct {
				name            string
				setupCollection func() string
				collectionID    string
				expectRemoved   bool
				expectError     bool
				description     string
			}{
				{
					name: "Remove_existing_collection",
					setupCollection: func() string {
						collection := types.Collection{
							ID: "remove-test-001",
							Metadata: map[string]interface{}{
								"type": "remove-test",
							},
							VectorConfig: &types.VectorStoreConfig{
								Dimension: 128,
								Distance:  types.DistanceCosine,
								IndexType: types.IndexTypeHNSW,
							},
						}
						id, err := g.CreateCollection(ctx, collection)
						if err != nil {
							t.Fatalf("Failed to setup collection: %v", err)
						}
						return id
					},
					expectRemoved: true,
					expectError:   false,
					description:   "Should remove existing collection",
				},
				{
					name:          "Remove_nonexistent_collection",
					collectionID:  "nonexistent-collection",
					expectRemoved: false,
					expectError:   false,
					description:   "Should return false for nonexistent collection",
				},
				{
					name:          "Remove_with_empty_ID",
					collectionID:  "",
					expectRemoved: false,
					expectError:   true,
					description:   "Should return error for empty collection ID",
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Setup collection if needed
					collectionID := tc.collectionID
					if tc.setupCollection != nil {
						collectionID = tc.setupCollection()
					}

					// Remove collection
					removed, err := g.RemoveCollection(ctx, collectionID)

					if tc.expectError {
						if err == nil {
							t.Errorf("Expected error but got none. %s", tc.description)
						}
						return
					}

					if err != nil {
						t.Errorf("Unexpected error: %v. %s", err, tc.description)
						return
					}

					if removed != tc.expectRemoved {
						t.Errorf("Expected removed=%v, got %v. %s", tc.expectRemoved, removed, tc.description)
						return
					}

					// Verify collection doesn't exist if it was removed
					if removed {
						exists, err := g.CollectionExists(ctx, collectionID)
						if err != nil {
							t.Errorf("Failed to check collection existence after removal: %v", err)
							return
						}
						if exists {
							t.Error("Collection should not exist after removal")
							return
						}
					}

					t.Logf("Successfully tested removal: %s (removed=%v)", collectionID, removed)
				})
			}
		})
	}
}

// TestCollectionExists tests the CollectionExists function
func TestCollectionExists(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+system", "complete"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Fatalf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create GraphRag instance: %v", err)
			}

			ctx := context.Background()

			// Test cases for CollectionExists
			testCases := []struct {
				name            string
				setupCollection func() string
				collectionID    string
				expectExists    bool
				expectError     bool
				description     string
			}{
				{
					name: "Existing_collection",
					setupCollection: func() string {
						collection := types.Collection{
							ID: "exists-test-001",
							Metadata: map[string]interface{}{
								"type": "exists-test",
							},
							VectorConfig: &types.VectorStoreConfig{
								Dimension: 128,
								Distance:  types.DistanceCosine,
								IndexType: types.IndexTypeHNSW,
							},
						}
						id, err := g.CreateCollection(ctx, collection)
						if err != nil {
							t.Fatalf("Failed to setup collection: %v", err)
						}
						return id
					},
					expectExists: true,
					expectError:  false,
					description:  "Should return true for existing collection",
				},
				{
					name:         "Nonexistent_collection",
					collectionID: "nonexistent-collection-exists",
					expectExists: false,
					expectError:  false,
					description:  "Should return false for nonexistent collection",
				},
			}

			var createdCollections []string

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Setup collection if needed
					collectionID := tc.collectionID
					if tc.setupCollection != nil {
						collectionID = tc.setupCollection()
						createdCollections = append(createdCollections, collectionID)
					}

					// Check collection existence
					exists, err := g.CollectionExists(ctx, collectionID)

					if tc.expectError {
						if err == nil {
							t.Errorf("Expected error but got none. %s", tc.description)
						}
						return
					}

					if err != nil {
						t.Errorf("Unexpected error: %v. %s", err, tc.description)
						return
					}

					if exists != tc.expectExists {
						t.Errorf("Expected exists=%v, got %v. %s", tc.expectExists, exists, tc.description)
						return
					}

					t.Logf("Successfully tested existence: %s (exists=%v)", collectionID, exists)
				})
			}

			// Cleanup created collections
			t.Cleanup(func() {
				for _, collectionID := range createdCollections {
					_, err := g.RemoveCollection(ctx, collectionID)
					if err != nil {
						t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
					}
				}
			})
		})
	}
}

// TestGetCollections tests the GetCollections function and related helper functions
func TestGetCollections(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+system", "complete"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Fatalf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create GraphRag instance: %v", err)
			}

			ctx := context.Background()

			// Setup test collections
			testCollections := []types.Collection{
				{
					ID: "get-test-001",
					Metadata: map[string]interface{}{
						"type":     "document",
						"category": "research",
						"count":    10,
					},
					VectorConfig: &types.VectorStoreConfig{
						Dimension: 128,
						Distance:  types.DistanceCosine,
						IndexType: types.IndexTypeHNSW,
					},
				},
				{
					ID: "get-test-002",
					Metadata: map[string]interface{}{
						"type":     "document",
						"category": "blog",
						"count":    5,
					},
					VectorConfig: &types.VectorStoreConfig{
						Dimension: 128,
						Distance:  types.DistanceCosine,
						IndexType: types.IndexTypeHNSW,
					},
				},
				{
					ID: "get-test-003",
					Metadata: map[string]interface{}{
						"type":     "image",
						"category": "research",
						"count":    3,
					},
					VectorConfig: &types.VectorStoreConfig{
						Dimension: 128,
						Distance:  types.DistanceCosine,
						IndexType: types.IndexTypeHNSW,
					},
				},
			}

			var createdCollections []string

			// Create test collections
			for _, collection := range testCollections {
				id, err := g.CreateCollection(ctx, collection)
				if err != nil {
					t.Fatalf("Failed to create test collection %s: %v", collection.ID, err)
				}
				createdCollections = append(createdCollections, id)
			}

			// Test cases for GetCollections
			testCases := []struct {
				name            string
				filter          map[string]interface{}
				expectedCount   int
				expectedMinimum int // Minimum expected count (for cases where system collections might exist)
				description     string
			}{
				{
					name:            "Get_all_collections",
					filter:          nil,
					expectedMinimum: 3,
					description:     "Should return all collections",
				},
				{
					name: "Filter_by_type_document",
					filter: map[string]interface{}{
						"type": "document",
					},
					expectedCount: 2,
					description:   "Should return collections with type=document",
				},
				{
					name: "Filter_by_type_image",
					filter: map[string]interface{}{
						"type": "image",
					},
					expectedCount: 1,
					description:   "Should return collections with type=image",
				},
				{
					name: "Filter_by_category_research",
					filter: map[string]interface{}{
						"category": "research",
					},
					expectedCount: 2,
					description:   "Should return collections with category=research",
				},
				{
					name: "Filter_by_nonexistent_value",
					filter: map[string]interface{}{
						"type": "nonexistent",
					},
					expectedCount: 0,
					description:   "Should return no collections for nonexistent filter",
				},
				{
					name: "Filter_by_multiple_criteria",
					filter: map[string]interface{}{
						"type":     "document",
						"category": "research",
					},
					expectedCount: 1,
					description:   "Should return collections matching multiple criteria",
				},
				{
					name:            "Empty_filter",
					filter:          map[string]interface{}{},
					expectedMinimum: 3,
					description:     "Should return all collections for empty filter",
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Get collections with filter
					collections, err := g.GetCollections(ctx, tc.filter)
					if err != nil {
						t.Errorf("Unexpected error: %v. %s", err, tc.description)
						return
					}

					// Check count
					if tc.expectedCount > 0 {
						if len(collections) != tc.expectedCount {
							t.Errorf("Expected %d collections, got %d. %s", tc.expectedCount, len(collections), tc.description)
							return
						}
					} else if tc.expectedMinimum > 0 {
						if len(collections) < tc.expectedMinimum {
							t.Errorf("Expected at least %d collections, got %d. %s", tc.expectedMinimum, len(collections), tc.description)
							return
						}
					}

					// Verify filter criteria
					if len(tc.filter) > 0 {
						for _, collection := range collections {
							for key, expectedValue := range tc.filter {
								if collection.Metadata == nil {
									t.Errorf("Collection %s has nil metadata but should match filter", collection.ID)
									continue
								}
								actualValue, exists := collection.Metadata[key]
								if !exists {
									t.Errorf("Collection %s missing metadata key %s", collection.ID, key)
									continue
								}
								if actualValue != expectedValue {
									t.Errorf("Collection %s metadata %s: expected %v, got %v", collection.ID, key, expectedValue, actualValue)
									continue
								}
							}
						}
					}

					t.Logf("Successfully retrieved %d collections with filter %v", len(collections), tc.filter)
				})
			}

			// Cleanup created collections
			t.Cleanup(func() {
				for _, collectionID := range createdCollections {
					_, err := g.RemoveCollection(ctx, collectionID)
					if err != nil {
						t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
					}
				}
			})
		})
	}
}

// TestEnsureSystemCollection tests the ensureSystemCollection function indirectly
func TestEnsureSystemCollection(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector+system", "complete"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Fatalf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create GraphRag instance: %v", err)
			}

			ctx := context.Background()

			// Test ensureSystemCollection by creating a collection
			// This will internally call ensureSystemCollection
			collection := types.Collection{
				ID: "system-test-001",
				Metadata: map[string]interface{}{
					"type": "system-test",
				},
				VectorConfig: &types.VectorStoreConfig{
					Dimension: 128,
					Distance:  types.DistanceCosine,
					IndexType: types.IndexTypeHNSW,
				},
			}

			// Create collection (this will test ensureSystemCollection)
			collectionID, err := g.CreateCollection(ctx, collection)
			if err != nil {
				t.Errorf("Failed to create collection (ensureSystemCollection test): %v", err)
				return
			}

			// Verify system collection exists
			systemExists, err := g.Vector.CollectionExists(ctx, g.System)
			if err != nil {
				t.Errorf("Failed to check system collection existence: %v", err)
			} else if !systemExists {
				t.Error("System collection should exist after creating a collection")
			}

			// Cleanup
			t.Cleanup(func() {
				_, err := g.RemoveCollection(ctx, collectionID)
				if err != nil {
					t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
				}
			})

			t.Logf("Successfully tested ensureSystemCollection via CreateCollection")
		})
	}
}

// TestCollectionErrorHandling tests various error conditions
func TestCollectionErrorHandling(t *testing.T) {
	configs := GetTestConfigs()

	t.Run("Invalid_config", func(t *testing.T) {
		config := configs["invalid"]
		if config == nil {
			t.Fatal("Invalid config not found")
		}

		// Try to create GraphRag instance with invalid config
		_, err := New(config)
		if err == nil {
			t.Error("Expected error for invalid config")
		}
	})

	t.Run("Nil_vector_store", func(t *testing.T) {
		// Create config with nil vector store
		config := &Config{
			Vector: nil,
			Logger: log.StandardLogger(),
		}

		// Should fail to create GraphRag instance with nil vector store
		_, err := New(config)
		if err == nil {
			t.Error("Expected error when creating GraphRag instance with nil vector store")
		}

		if !strings.Contains(err.Error(), "vector store is required") {
			t.Errorf("Expected error message to contain 'vector store is required', got: %s", err.Error())
		}
	})
}

// TestExtractCollectionID tests the utility function indirectly
func TestExtractCollectionID(t *testing.T) {
	testCases := []struct {
		vectorName  string
		expectedID  string
		description string
	}{
		{
			vectorName:  "test_collection_vector",
			expectedID:  "test_collection",
			description: "Should extract ID from vector collection name",
		},
		{
			vectorName:  "simple_vector",
			expectedID:  "simple",
			description: "Should extract ID from simple name",
		},
		{
			vectorName:  "invalid_name",
			expectedID:  "",
			description: "Should return empty for invalid name format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := utils.ExtractCollectionIDFromVectorName(tc.vectorName)
			if result != tc.expectedID {
				t.Errorf("Expected %s, got %s. %s", tc.expectedID, result, tc.description)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkCreateCollection(b *testing.B) {
	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		b.Fatal("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		b.Fatalf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use UUID to ensure unique collection ID for each iteration
		collection := types.Collection{
			ID: utils.GenDocID(), // This generates a unique UUID
			Metadata: map[string]interface{}{
				"type":  "benchmark",
				"index": i,
			},
			VectorConfig: &types.VectorStoreConfig{
				Dimension: 128,
				Distance:  types.DistanceCosine,
				IndexType: types.IndexTypeHNSW,
			},
		}

		_, err := g.CreateCollection(ctx, collection)
		if err != nil {
			b.Errorf("Failed to create collection in benchmark: %v", err)
		}
	}
}

func BenchmarkCollectionExists(b *testing.B) {
	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		b.Fatal("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		b.Fatalf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	// Create a test collection with unique ID
	collection := types.Collection{
		ID: utils.GenDocID(), // Use unique UUID
		Metadata: map[string]interface{}{
			"type": "benchmark",
		},
		VectorConfig: &types.VectorStoreConfig{
			Dimension: 128,
			Distance:  types.DistanceCosine,
			IndexType: types.IndexTypeHNSW,
		},
	}

	collectionID, err := g.CreateCollection(ctx, collection)
	if err != nil {
		b.Fatalf("Failed to create test collection: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := g.CollectionExists(ctx, collectionID)
		if err != nil {
			b.Errorf("Failed to check collection existence in benchmark: %v", err)
		}
	}

	// Cleanup
	_, err = g.RemoveCollection(ctx, collectionID)
	if err != nil {
		b.Logf("Warning: Failed to cleanup benchmark collection: %v", err)
	}
}
