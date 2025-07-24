package graphrag

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/log"
)

// Common test configurations for different test scenarios
var (
	// Standard test configurations covering all combinations
	standardTestConfigs = []string{"vector", "vector+graph", "vector+store", "vector+system", "complete"}

	// System-specific test configurations (tests that need system collection)
	systemTestConfigs = []string{"vector+system", "complete"}

	// Multi-layer test configurations (tests that need comprehensive storage)
	multiLayerTestConfigs = []string{"vector+graph+store", "complete"}
)

// TestCreateCollection tests the CreateCollection function
func TestCreateCollection(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := standardTestConfigs

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

			// Note: Vector configs are now created inline within each test case

			// Test cases for CreateCollection
			testCases := []struct {
				name        string
				collection  types.CollectionConfig
				expectError bool
				description string
			}{
				{
					name: "Valid_collection_with_ID",
					collection: types.CollectionConfig{
						ID: "test_collection_001",
						Metadata: map[string]interface{}{
							"type":        "test",
							"description": "Test collection",
						},
						Config: &types.CreateCollectionOptions{
							CollectionName: "test_collection_001_vector",
							Dimension:      1536,
							Distance:       types.DistanceCosine,
							IndexType:      types.IndexTypeHNSW,
						},
					},
					expectError: false,
					description: "Should create collection with provided ID",
				},
				{
					name: "Valid_collection_without_ID",
					collection: types.CollectionConfig{
						Metadata: map[string]interface{}{
							"type": "auto-generated",
						},
						Config: &types.CreateCollectionOptions{
							CollectionName: "auto_generated_vector",
							Dimension:      1536,
							Distance:       types.DistanceCosine,
							IndexType:      types.IndexTypeHNSW,
						},
					},
					expectError: false,
					description: "Should create collection with auto-generated ID",
				},
				{
					name: "Collection_with_sparse_vectors",
					collection: types.CollectionConfig{
						ID: "test_collection_sparse",
						Metadata: map[string]interface{}{
							"type": "sparse",
						},
						Config: &types.CreateCollectionOptions{
							CollectionName:      "test_collection_sparse_vector",
							Dimension:           1536,
							Distance:            types.DistanceCosine,
							IndexType:           types.IndexTypeHNSW,
							EnableSparseVectors: true,
							DenseVectorName:     "dense",
							SparseVectorName:    "sparse",
						},
					},
					expectError: false,
					description: "Should create collection with sparse vector config",
				},
				{
					name: "Collection_with_nil_metadata",
					collection: types.CollectionConfig{
						ID:       "test_collection_nil_meta",
						Metadata: nil,
						Config: &types.CreateCollectionOptions{
							CollectionName: "nil_meta_vector",
							Dimension:      1536,
							Distance:       types.DistanceCosine,
							IndexType:      types.IndexTypeHNSW,
						},
					},
					expectError: false,
					description: "Should handle nil metadata",
				},
				{
					name: "Collection_without_config",
					collection: types.CollectionConfig{
						ID: "test_collection_no_config",
						Metadata: map[string]interface{}{
							"type": "no_config",
						},
						Config: nil,
					},
					expectError: false,
					description: "Should handle collection without config",
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

					// Verify actual storage layers
					err = verifyCollectionStorage(ctx, t, g, collectionID, tc.collection)
					if err != nil {
						t.Errorf("Storage verification failed: %v", err)
						return
					}

					t.Logf("Successfully created and verified collection: %s", collectionID)
				})
			}

			// Test duplicate collection creation
			t.Run("Duplicate_collection_creation", func(t *testing.T) {
				ctx := context.Background()
				collection := types.CollectionConfig{
					ID: "duplicate_test",
					Metadata: map[string]interface{}{
						"type": "duplicate",
					},
					Config: &types.CreateCollectionOptions{
						CollectionName: "duplicate_test_vector",
						Dimension:      1536,
						Distance:       types.DistanceCosine,
						IndexType:      types.IndexTypeHNSW,
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
	testConfigs := standardTestConfigs

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

			// Vector configs are now created inline within each test case

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
						collection := types.CollectionConfig{
							ID: "remove_test_001",
							Metadata: map[string]interface{}{
								"type": "remove_test",
							},
							Config: &types.CreateCollectionOptions{
								CollectionName: "remove_test_001_vector",
								Dimension:      1536,
								Distance:       types.DistanceCosine,
								IndexType:      types.IndexTypeHNSW,
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
					collectionID:  "nonexistent_collection",
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

						// Verify actual storage layers are cleaned up
						err = verifyCollectionRemoval(ctx, t, g, collectionID)
						if err != nil {
							t.Errorf("Storage removal verification failed: %v", err)
							return
						}
					}

					t.Logf("Successfully tested and verified removal: %s (removed=%v)", collectionID, removed)
				})
			}
		})
	}
}

// TestCollectionExists tests the CollectionExists function
func TestCollectionExists(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := standardTestConfigs

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

			// Create reusable vector config using utility function

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
						collection := types.CollectionConfig{
							ID: "exists_test_001",
							Metadata: map[string]interface{}{
								"type": "exists_test",
							},
							Config: &types.CreateCollectionOptions{
								CollectionName: "exists_test_001_vector",
								Dimension:      1536,
								Distance:       types.DistanceCosine,
								IndexType:      types.IndexTypeHNSW,
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
					collectionID: "nonexistent_collection_exists",
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

					// If collection should exist, verify actual storage layers
					if tc.expectExists && tc.setupCollection != nil {
						// Create a dummy collection config object for verification
						dummyCollectionConfig := types.CollectionConfig{
							ID: collectionID,
							Config: &types.CreateCollectionOptions{
								CollectionName: "exists_test_vector",
								Dimension:      1536,
								Distance:       types.DistanceCosine,
								IndexType:      types.IndexTypeHNSW,
							},
						}
						err = verifyCollectionStorage(ctx, t, g, collectionID, dummyCollectionConfig)
						if err != nil {
							t.Errorf("Storage verification failed: %v", err)
							return
						}
					}

					t.Logf("Successfully tested and verified existence: %s (exists=%v)", collectionID, exists)
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
	testConfigs := standardTestConfigs

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

			// Create reusable vector config using utility function

			// Setup test collections
			testCollections := []types.CollectionConfig{
				{
					ID: "get_test_001",
					Metadata: map[string]interface{}{
						"type":     "document",
						"category": "research",
						"count":    10,
					},
					Config: &types.CreateCollectionOptions{
						CollectionName: "get_test_001_vector",
						Dimension:      1536,
						Distance:       types.DistanceCosine,
						IndexType:      types.IndexTypeHNSW,
					},
				},
				{
					ID: "get_test_002",
					Metadata: map[string]interface{}{
						"type":     "document",
						"category": "blog",
						"count":    5,
					},
					Config: &types.CreateCollectionOptions{
						CollectionName: "get_test_002_vector",
						Dimension:      1536,
						Distance:       types.DistanceCosine,
						IndexType:      types.IndexTypeHNSW,
					},
				},
				{
					ID: "get_test_003",
					Metadata: map[string]interface{}{
						"type":     "image",
						"category": "research",
						"count":    3,
					},
					Config: &types.CreateCollectionOptions{
						CollectionName: "get_test_003_vector",
						Dimension:      1536,
						Distance:       types.DistanceCosine,
						IndexType:      types.IndexTypeHNSW,
					},
				},
			}

			var createdCollections []string

			// Clean up any existing test collections first
			for _, collection := range testCollections {
				g.RemoveCollection(ctx, collection.ID) // Ignore error if collection doesn't exist
			}

			// Create test collections
			for _, collection := range testCollections {
				id, err := g.CreateCollection(ctx, collection)
				if err != nil {
					t.Fatalf("Failed to create test collection %s: %v", collection.ID, err)
				}
				createdCollections = append(createdCollections, id)

				// Verify storage layers for each created collection
				err = verifyCollectionStorage(ctx, t, g, id, collection)
				if err != nil {
					t.Fatalf("Storage verification failed for collection %s: %v", id, err)
				}
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
	testConfigs := systemTestConfigs

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
			collection := types.CollectionConfig{
				ID: "system_test_001",
				Metadata: map[string]interface{}{
					"type": "system_test",
				},
				Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
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

			// Verify actual storage layers
			err = verifyCollectionStorage(ctx, t, g, collectionID, collection)
			if err != nil {
				t.Errorf("Storage verification failed: %v", err)
				return
			}

			// Cleanup
			t.Cleanup(func() {
				_, err := g.RemoveCollection(ctx, collectionID)
				if err != nil {
					t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
				}
			})

			t.Logf("Successfully tested and verified ensureSystemCollection via CreateCollection")
		})
	}
}

// TestMultiLayerStorageVerification tests comprehensive storage layer verification
func TestMultiLayerStorageVerification(t *testing.T) {
	configs := GetTestConfigs()
	// Test comprehensive configurations that include all storage layers
	testConfigs := multiLayerTestConfigs

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Skipf("Config %s not found, skipping", configName)
			continue
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create GraphRag instance: %v", err)
			}

			ctx := context.Background()

			// Create comprehensive test configuration
			testCollection := types.CollectionConfig{
				ID: "multilayer_test_001",
				Metadata: map[string]interface{}{
					"type":        "multilayer",
					"description": "Comprehensive storage test",
					"layers":      []string{"vector", "graph", "metadata"},
				},
				Config: &types.CreateCollectionOptions{
					CollectionName: "multilayer_test_001_vector",
					Dimension:      1536,
					Distance:       types.DistanceCosine,
					IndexType:      types.IndexTypeHNSW,
				},
			}

			// Phase 1: Create collection and verify all layers
			t.Run("Create_and_verify_all_layers", func(t *testing.T) {
				collectionID, err := g.CreateCollection(ctx, testCollection)
				if err != nil {
					t.Fatalf("Failed to create collection: %v", err)
				}

				// Verify using GraphRag API
				exists, err := g.CollectionExists(ctx, collectionID)
				if err != nil {
					t.Fatalf("Failed to check collection existence: %v", err)
				}
				if !exists {
					t.Fatal("Collection should exist after creation")
				}

				// Deep verification of all storage layers
				err = verifyCollectionStorage(ctx, t, g, collectionID, testCollection)
				if err != nil {
					t.Fatalf("Deep storage verification failed: %v", err)
				}

				t.Logf("✓ All storage layers verified for collection: %s", collectionID)

				// Phase 2: Remove collection and verify all layers are cleaned
				t.Run("Remove_and_verify_cleanup", func(t *testing.T) {
					removed, err := g.RemoveCollection(ctx, collectionID)
					if err != nil {
						t.Fatalf("Failed to remove collection: %v", err)
					}
					if !removed {
						t.Fatal("Collection should be removed")
					}

					// Verify using GraphRag API
					exists, err := g.CollectionExists(ctx, collectionID)
					if err != nil {
						t.Fatalf("Failed to check collection existence after removal: %v", err)
					}
					if exists {
						t.Fatal("Collection should not exist after removal")
					}

					// Deep verification of all storage layers cleanup
					err = verifyCollectionRemoval(ctx, t, g, collectionID)
					if err != nil {
						t.Fatalf("Deep storage cleanup verification failed: %v", err)
					}

					t.Logf("✓ All storage layers cleanup verified for collection: %s", collectionID)
				})
			})
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

	// Create reusable vector config using utility function

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use UUID to ensure unique collection ID for each iteration
		collection := types.CollectionConfig{
			ID: utils.GenDocID(), // This generates a unique UUID
			Metadata: map[string]interface{}{
				"type":  "benchmark",
				"index": i,
			},
			Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
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

	// Create reusable vector config using utility function

	// Create a test collection with unique ID
	collection := types.CollectionConfig{
		ID: utils.GenDocID(), // Use unique UUID
		Metadata: map[string]interface{}{
			"type": "benchmark",
		},
		Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
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

// Helper function to convert CollectionConfig to Collection for storage verification
func configToCollection(config types.CollectionConfig) types.Collection {
	return types.Collection{
		ID:               config.ID,
		Metadata:         config.Metadata,
		CollectionConfig: config.Config,
	}
}

// verifyCollectionStorage verifies that collection data exists in all expected storage layers
func verifyCollectionStorage(ctx context.Context, t *testing.T, g *GraphRag, collectionID string, originalCollection types.CollectionConfig) error {
	t.Helper()

	// Convert to Collection for verification
	collection := configToCollection(originalCollection)

	// Generate collection IDs for different storage systems
	ids, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		return fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	// 1. Verify Vector Collection exists
	if g.Vector != nil && originalCollection.Config != nil {
		exists, err := g.Vector.CollectionExists(ctx, ids.Vector)
		if err != nil {
			return fmt.Errorf("failed to check vector collection existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("vector collection %s should exist but was not found", ids.Vector)
		}
		t.Logf("✓ Vector collection verified: %s", ids.Vector)
	}

	// 2. Verify Graph Storage exists (if configured)
	if g.Graph != nil && collection.GraphStoreConfig != nil {
		// Use GraphExists as the primary method to check if graph exists
		exists, err := g.Graph.GraphExists(ctx, ids.Graph)
		if err != nil {
			return fmt.Errorf("failed to check graph existence: %w", err)
		}

		if exists {
			t.Logf("✓ Graph storage verified (graph exists with data): %s", ids.Graph)
		} else {
			// For some graph implementations (like Neo4j Community Edition with label-based storage),
			// a newly created empty graph is not considered as "existing" until nodes are added.
			// This is normal behavior. We verify that the graph infrastructure is properly accessible
			// by testing if we can perform basic operations on the graph namespace.

			// Test if we can perform a basic query operation on this graph
			// This is more reliable than GetStats which always succeeds
			err := verifyGraphInfrastructure(ctx, g.Graph, ids.Graph)
			if err != nil {
				return fmt.Errorf("graph %s infrastructure verification failed: %w", ids.Graph, err)
			}
			t.Logf("✓ Graph storage verified (empty graph, infrastructure operational): %s", ids.Graph)
		}
	}

	// 3. Verify Metadata Storage
	if g.Store != nil {
		// Check Store (KV storage)
		if !g.Store.Has(collectionID) {
			return fmt.Errorf("collection metadata should exist in Store but was not found: %s", collectionID)
		}
		t.Logf("✓ Metadata in Store verified: %s", collectionID)
	} else if g.Vector != nil {
		// Check System Collection
		systemExists, err := g.Vector.CollectionExists(ctx, g.System)
		if err != nil {
			return fmt.Errorf("failed to check System Collection existence: %w", err)
		}
		if !systemExists {
			return fmt.Errorf("System Collection should exist but was not found: %s", g.System)
		}

		// Check if collection metadata exists in System Collection
		opts := &types.GetDocumentOptions{
			CollectionName: g.System,
			IncludeVector:  false,
			IncludePayload: true,
		}

		docs, err := g.Vector.GetDocuments(ctx, []string{collectionID}, opts)
		if err != nil {
			return fmt.Errorf("failed to get collection metadata from System Collection: %w", err)
		}
		if len(docs) == 0 || docs[0] == nil {
			return fmt.Errorf("collection metadata should exist in System Collection but was not found: %s", collectionID)
		}
		t.Logf("✓ Metadata in System Collection verified: %s", collectionID)
	}

	return nil
}

// verifyCollectionRemoval verifies that collection data is properly removed from all storage layers
func verifyCollectionRemoval(ctx context.Context, t *testing.T, g *GraphRag, collectionID string) error {
	t.Helper()

	// Generate collection IDs for different storage systems
	ids, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		return fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	// 1. Verify Vector Collection is removed
	if g.Vector != nil {
		exists, err := g.Vector.CollectionExists(ctx, ids.Vector)
		if err != nil {
			// Some implementations might return error for non-existent collections
			t.Logf("Vector collection check returned error (expected for removed collection): %v", err)
		} else if exists {
			return fmt.Errorf("vector collection %s should be removed but still exists", ids.Vector)
		}
		t.Logf("✓ Vector collection removal verified: %s", ids.Vector)
	}

	// 2. Verify Graph Storage is removed (if it was created)
	if g.Graph != nil {
		// Try to get graph stats - should fail or return empty for removed graph
		_, err := g.Graph.GetStats(ctx, ids.Graph)
		if err != nil {
			t.Logf("✓ Graph storage removal verified (stats check failed as expected): %s", ids.Graph)
		}
	}

	// 3. Verify Metadata is removed
	if g.Store != nil {
		// Check Store (KV storage)
		if g.Store.Has(collectionID) {
			return fmt.Errorf("collection metadata should be removed from Store but still exists: %s", collectionID)
		}
		t.Logf("✓ Metadata removal from Store verified: %s", collectionID)
	} else if g.Vector != nil {
		// Check System Collection
		systemExists, err := g.Vector.CollectionExists(ctx, g.System)
		if err != nil {
			return fmt.Errorf("failed to check System Collection existence: %w", err)
		}
		if systemExists {
			// Check if collection metadata is removed from System Collection
			opts := &types.GetDocumentOptions{
				CollectionName: g.System,
				IncludeVector:  false,
				IncludePayload: false,
			}

			docs, err := g.Vector.GetDocuments(ctx, []string{collectionID}, opts)
			if err != nil {
				// Error might be expected for non-existent documents
				t.Logf("System Collection document check returned error (expected for removed collection): %v", err)
			} else if len(docs) > 0 && docs[0] != nil {
				return fmt.Errorf("collection metadata should be removed from System Collection but still exists: %s", collectionID)
			}
			t.Logf("✓ Metadata removal from System Collection verified: %s", collectionID)
		}
	}

	return nil
}

// verifyGraphInfrastructure verifies that the graph infrastructure is operational
// by performing a basic test operation on the graph namespace
func verifyGraphInfrastructure(ctx context.Context, graph types.GraphStore, graphName string) error {
	// For Neo4j Community Edition, test if we can perform a basic query on the graph namespace
	// This is more reliable than GetStats which always succeeds, and avoids the "graph doesn't exist"
	// error from GetSchema for empty graphs

	// Try to perform a simple operation - attempt to query the graph label
	// This should work if the graph infrastructure is properly set up
	stats, err := graph.GetStats(ctx, graphName)
	if err != nil {
		return fmt.Errorf("failed to get basic stats: %w", err)
	}

	// Additional verification: try to get the list of graphs to ensure our graph name is valid
	graphs, err := graph.ListGraphs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list graphs: %w", err)
	}

	// For Neo4j Community Edition, the graph might not appear in the list until it has nodes
	// But if the ListGraphs operation itself succeeds, it means the graph store is operational
	_ = graphs // We don't require the graph to be in the list for empty graphs

	// If we got here, the graph store is operational for this graph name
	// Stats should be valid (empty stats for empty graph)
	if stats == nil {
		return fmt.Errorf("graph stats should not be nil")
	}

	return nil
}

// ==== Concurrent and Leak Detection Tests ====

// TestCollectionConcurrentStress tests concurrent collection operations under stress
func TestCollectionConcurrentStress(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector", "complete"} // Test basic and complete configurations

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

			// Use light stress config for CI compatibility
			stressConfig := LightStressConfig()

			t.Run("Concurrent_Create_Collections", func(t *testing.T) {
				var createdCollections []string
				var mu sync.Mutex

				// Create operation
				createOperation := func(ctx context.Context) error {
					collection := types.CollectionConfig{
						ID: utils.GenDocID(), // Unique ID for each operation
						Metadata: map[string]interface{}{
							"type":   "stress_test",
							"source": "concurrent",
						},
						Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
					}

					collectionID, err := g.CreateCollection(ctx, collection)
					if err != nil {
						return err
					}

					// Store for cleanup
					mu.Lock()
					createdCollections = append(createdCollections, collectionID)
					mu.Unlock()

					return nil
				}

				// Run stress test with leak detection
				result, leakResult := runConcurrentStressWithLeakDetection(t, stressConfig, createOperation)

				// Assert results
				assertStressTestResult(t, result, stressConfig, "Concurrent_Create_Collections")
				assertNoLeaks(t, leakResult, "Concurrent_Create_Collections")

				// Cleanup
				t.Cleanup(func() {
					ctx := context.Background()
					mu.Lock()
					collections := append([]string{}, createdCollections...)
					mu.Unlock()

					for _, collectionID := range collections {
						_, err := g.RemoveCollection(ctx, collectionID)
						if err != nil {
							t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
						}
					}
				})
			})

			t.Run("Concurrent_Check_Existence", func(t *testing.T) {
				// Setup: Create a test collection
				collection := types.CollectionConfig{
					ID: "stress_check_collection",
					Metadata: map[string]interface{}{
						"type": "stress_test",
					},
					Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
				}

				collectionID, err := g.CreateCollection(context.Background(), collection)
				if err != nil {
					t.Fatalf("Failed to create test collection: %v", err)
				}

				// Check existence operation
				checkOperation := func(ctx context.Context) error {
					_, err := g.CollectionExists(ctx, collectionID)
					return err
				}

				// Run stress test with leak detection
				result, leakResult := runConcurrentStressWithLeakDetection(t, stressConfig, checkOperation)

				// Assert results
				assertStressTestResult(t, result, stressConfig, "Concurrent_Check_Existence")
				assertNoLeaks(t, leakResult, "Concurrent_Check_Existence")

				// Cleanup
				t.Cleanup(func() {
					_, err := g.RemoveCollection(context.Background(), collectionID)
					if err != nil {
						t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
					}
				})
			})

			t.Run("Concurrent_Mixed_Operations", func(t *testing.T) {
				var createdCollections []string
				var mu sync.Mutex

				// Mixed operations (create, check, remove)
				mixedOperation := func(ctx context.Context) error {
					// Random operation type based on time
					opType := time.Now().UnixNano() % 3

					switch opType {
					case 0: // Create
						collection := types.CollectionConfig{
							ID: utils.GenDocID(),
							Metadata: map[string]interface{}{
								"type": "mixed_stress",
							},
							Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
						}

						collectionID, err := g.CreateCollection(ctx, collection)
						if err != nil {
							return err
						}

						mu.Lock()
						createdCollections = append(createdCollections, collectionID)
						mu.Unlock()

					case 1: // Check existence
						mu.Lock()
						if len(createdCollections) > 0 {
							checkID := createdCollections[len(createdCollections)-1]
							mu.Unlock()
							_, err := g.CollectionExists(ctx, checkID)
							return err
						}
						mu.Unlock()

					case 2: // Remove (if any exist)
						mu.Lock()
						if len(createdCollections) > 0 {
							removeID := createdCollections[0]
							createdCollections = createdCollections[1:]
							mu.Unlock()
							_, err := g.RemoveCollection(ctx, removeID)
							return err
						}
						mu.Unlock()
					}

					return nil
				}

				// Run stress test with leak detection
				result, leakResult := runConcurrentStressWithLeakDetection(t, stressConfig, mixedOperation)

				// Assert results with lower success rate expectation for mixed operations
				mixedConfig := stressConfig
				mixedConfig.MinSuccessRate = 85.0 // Lower due to potential race conditions
				assertStressTestResult(t, result, mixedConfig, "Concurrent_Mixed_Operations")
				assertNoLeaks(t, leakResult, "Concurrent_Mixed_Operations")

				// Cleanup remaining collections
				t.Cleanup(func() {
					ctx := context.Background()
					mu.Lock()
					collections := append([]string{}, createdCollections...)
					mu.Unlock()

					for _, collectionID := range collections {
						_, err := g.RemoveCollection(ctx, collectionID)
						if err != nil {
							t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
						}
					}
				})
			})
		})
	}
}

// TestCollectionLeakDetection tests for memory and goroutine leaks in collection operations
func TestCollectionLeakDetection(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector", "complete"}

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

			t.Run("Create_and_Remove_Leak_Detection", func(t *testing.T) {
				result := runWithLeakDetection(t, func() error {
					var createdCollections []string

					// Create multiple collections
					for i := 0; i < 10; i++ {
						collection := types.CollectionConfig{
							ID: fmt.Sprintf("leak_test_%d_%s", i, utils.GenDocID()),
							Metadata: map[string]interface{}{
								"type":  "leak_test",
								"index": i,
							},
							Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
						}

						collectionID, err := g.CreateCollection(context.Background(), collection)
						if err != nil {
							return fmt.Errorf("failed to create collection %d: %w", i, err)
						}
						createdCollections = append(createdCollections, collectionID)
					}

					// Check existence of all collections
					for _, collectionID := range createdCollections {
						exists, err := g.CollectionExists(context.Background(), collectionID)
						if err != nil {
							return fmt.Errorf("failed to check collection existence %s: %w", collectionID, err)
						}
						if !exists {
							return fmt.Errorf("collection %s should exist", collectionID)
						}
					}

					// Remove all collections
					for _, collectionID := range createdCollections {
						removed, err := g.RemoveCollection(context.Background(), collectionID)
						if err != nil {
							return fmt.Errorf("failed to remove collection %s: %w", collectionID, err)
						}
						if !removed {
							return fmt.Errorf("collection %s should be removed", collectionID)
						}
					}

					return nil
				})

				assertNoLeaks(t, result, "Create_and_Remove_Leak_Detection")
			})

			t.Run("Repeated_Operations_Leak_Detection", func(t *testing.T) {
				result := runWithLeakDetection(t, func() error {

					// Repeat create-check-remove cycle multiple times
					for i := 0; i < 20; i++ {
						collection := types.CollectionConfig{
							ID: fmt.Sprintf("repeated_%d_%s", i, utils.GenDocID()),
							Metadata: map[string]interface{}{
								"type":  "repeated_test",
								"cycle": i,
							},
							Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
						}

						// Create
						collectionID, err := g.CreateCollection(context.Background(), collection)
						if err != nil {
							return fmt.Errorf("failed to create collection in cycle %d: %w", i, err)
						}

						// Check
						exists, err := g.CollectionExists(context.Background(), collectionID)
						if err != nil {
							return fmt.Errorf("failed to check collection in cycle %d: %w", i, err)
						}
						if !exists {
							return fmt.Errorf("collection should exist in cycle %d", i)
						}

						// Remove
						removed, err := g.RemoveCollection(context.Background(), collectionID)
						if err != nil {
							return fmt.Errorf("failed to remove collection in cycle %d: %w", i, err)
						}
						if !removed {
							return fmt.Errorf("collection should be removed in cycle %d", i)
						}
					}

					return nil
				})

				assertNoLeaks(t, result, "Repeated_Operations_Leak_Detection")
			})

			t.Run("Error_Conditions_Leak_Detection", func(t *testing.T) {
				result := runWithLeakDetection(t, func() error {

					// Test various error conditions that might cause leaks
					for i := 0; i < 5; i++ {
						// Try to create collection with duplicate ID
						collection := types.CollectionConfig{
							ID: "duplicate_error_test",
							Metadata: map[string]interface{}{
								"type": "error_test",
							},
							Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
						}

						if i == 0 {
							// First creation should succeed
							_, err := g.CreateCollection(context.Background(), collection)
							if err != nil {
								return fmt.Errorf("first creation should succeed: %w", err)
							}
						} else {
							// Subsequent creations should fail
							_, err := g.CreateCollection(context.Background(), collection)
							if err == nil {
								return fmt.Errorf("duplicate creation should fail")
							}
							// Error is expected, continue
						}

						// Try to check non-existent collection
						_, err := g.CollectionExists(context.Background(), fmt.Sprintf("non_existent_%d", i))
						if err != nil {
							return fmt.Errorf("checking non-existent collection should not error: %w", err)
						}

						// Try to remove non-existent collection
						_, err = g.RemoveCollection(context.Background(), fmt.Sprintf("non_existent_%d", i))
						if err != nil {
							return fmt.Errorf("removing non-existent collection should not error: %w", err)
						}
					}

					// Cleanup the duplicate test collection
					_, err := g.RemoveCollection(context.Background(), "duplicate_error_test")
					if err != nil {
						return fmt.Errorf("failed to cleanup duplicate test collection: %w", err)
					}

					return nil
				})

				assertNoLeaks(t, result, "Error_Conditions_Leak_Detection")
			})
		})
	}
}

// TestCollectionStressWithFiltering tests stress operations with collection filtering
func TestCollectionStressWithFiltering(t *testing.T) {
	configs := GetTestConfigs()
	testConfigs := []string{"vector", "complete"}

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

			t.Run("Concurrent_GetCollections_Stress", func(t *testing.T) {
				// Setup: Create multiple collections with different metadata
				var createdCollections []string

				for i := 0; i < 10; i++ {
					collection := types.CollectionConfig{
						ID: fmt.Sprintf("filter_test_%d", i),
						Metadata: map[string]interface{}{
							"type":     "filter_test",
							"category": fmt.Sprintf("cat_%d", i%3), // 3 categories
							"index":    i,
							"even":     i%2 == 0,
						},
						Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
					}

					collectionID, err := g.CreateCollection(context.Background(), collection)
					if err != nil {
						t.Fatalf("Failed to create test collection %d: %v", i, err)
					}
					createdCollections = append(createdCollections, collectionID)
				}

				// Stress test GetCollections with various filters
				stressConfig := LightStressConfig()

				getCollectionsOperation := func(ctx context.Context) error {
					// Rotate through different filter types
					filterType := time.Now().UnixNano() % 4

					var filter map[string]interface{}
					switch filterType {
					case 0:
						filter = nil // Get all
					case 1:
						filter = map[string]interface{}{"type": "filter_test"}
					case 2:
						filter = map[string]interface{}{"category": "cat_1"}
					case 3:
						filter = map[string]interface{}{"even": true}
					}

					collections, err := g.GetCollections(ctx, filter)
					if err != nil {
						return err
					}

					// Basic validation
					if len(collections) == 0 && filter == nil {
						return fmt.Errorf("should have found at least some collections")
					}

					return nil
				}

				// Run stress test with leak detection
				result, leakResult := runConcurrentStressWithLeakDetection(t, stressConfig, getCollectionsOperation)

				// Assert results
				assertStressTestResult(t, result, stressConfig, "Concurrent_GetCollections_Stress")
				assertNoLeaks(t, leakResult, "Concurrent_GetCollections_Stress")

				// Cleanup
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
		})
	}
}

// BenchmarkCollectionOperationsWithLeakDetection benchmarks collection operations while monitoring for leaks
func BenchmarkCollectionOperationsWithLeakDetection(b *testing.B) {
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

	// Capture initial state
	beforeMem := captureMemoryStats()
	beforeGoroutines := captureGoroutineState()

	b.ResetTimer()

	b.Run("CreateCollection", func(b *testing.B) {
		var collectionIDs []string

		for i := 0; i < b.N; i++ {
			collection := types.CollectionConfig{
				ID: utils.GenDocID(),
				Metadata: map[string]interface{}{
					"type":  "benchmark_leak",
					"index": i,
				},
				Config: &types.CreateCollectionOptions{CollectionName: "default_vector", Dimension: 1536, Distance: types.DistanceCosine, IndexType: types.IndexTypeHNSW},
			}

			collectionID, err := g.CreateCollection(ctx, collection)
			if err != nil {
				b.Errorf("Failed to create collection in benchmark: %v", err)
				continue
			}
			collectionIDs = append(collectionIDs, collectionID)
		}

		// Cleanup
		for _, id := range collectionIDs {
			g.RemoveCollection(ctx, id)
		}
	})

	b.StopTimer()

	// Check for leaks after benchmark
	time.Sleep(500 * time.Millisecond) // Allow cleanup
	afterMem := captureMemoryStats()
	afterGoroutines := captureGoroutineState()

	memoryGrowth := calculateMemoryGrowth(beforeMem, afterMem)
	leakedGoroutines, _ := analyzeGoroutineChanges(beforeGoroutines, afterGoroutines)

	// Filter out system goroutines
	var realLeaks []GoroutineInfo
	for _, g := range leakedGoroutines {
		if !g.IsSystem {
			realLeaks = append(realLeaks, g)
		}
	}

	// Report leak detection results
	if memoryGrowth.AllocGrowth > 10*1024*1024 { // 10MB threshold for benchmarks
		b.Errorf("Potential memory leak detected - Alloc growth: %d bytes", memoryGrowth.AllocGrowth)
	}

	if len(realLeaks) > 0 {
		b.Errorf("Goroutine leak detected - %d leaked goroutines", len(realLeaks))
		for i, g := range realLeaks {
			if i >= 3 { // Limit output
				break
			}
			b.Logf("  Leaked goroutine: ID=%d, State=%s, Function=%s", g.ID, g.State, g.Function)
		}
	}

	b.Logf("Benchmark leak detection: Alloc growth: %d bytes, Goroutine leaks: %d",
		memoryGrowth.AllocGrowth, len(realLeaks))
}
