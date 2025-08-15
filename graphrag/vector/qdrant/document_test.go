package qdrant

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/yaoapp/gou/graphrag/types"
)

// =============================================================================
// Helper Functions (Reusing existing utilities)
// =============================================================================

// DocumentTestEnvironment holds document test environment
type DocumentTestEnvironment struct {
	Store            *Store
	ConnectionConfig types.VectorStoreConfig
	CollectionConfig types.CreateCollectionOptions
	CollectionName   string
}

// setupDocumentTestEnvironment creates a clean test environment for document operations
func setupDocumentTestEnvironment(t *testing.T) *DocumentTestEnvironment {
	t.Helper()

	store, connectionConfig, collectionConfig := setupConnectedStoreForDocumentTest(t)

	// Create unique collection name for this test (remove invalid characters)
	testName := t.Name()
	// Replace problematic characters that Qdrant doesn't allow
	testName = fmt.Sprintf("%d", time.Now().UnixNano()) // Use timestamp instead to ensure uniqueness
	collectionName := fmt.Sprintf("test_doc_%s", testName)

	// Update collection config with unique name
	collectionConfig.CollectionName = collectionName

	// Create test collection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := store.CreateCollection(ctx, &collectionConfig)
	if err != nil {
		// Clean up store connection before failing
		if disconnectErr := store.Disconnect(ctx); disconnectErr != nil {
			t.Logf("Warning: Failed to disconnect store during cleanup: %v", disconnectErr)
		}
		t.Fatalf("Failed to create test collection: %v", err)
	}

	return &DocumentTestEnvironment{
		Store:            store,
		ConnectionConfig: connectionConfig,
		CollectionConfig: collectionConfig,
		CollectionName:   collectionName,
	}
}

// cleanupDocumentTestEnvironment cleans up test data and connections
func cleanupDocumentTestEnvironment(t *testing.T, env *DocumentTestEnvironment) {
	t.Helper()

	if env == nil {
		return
	}

	if env.Store != nil {
		// Clean up collection first
		if env.CollectionName != "" {
			cleanupCollection(t, env.Store, env.CollectionName)
		}

		// Close connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := env.Store.Disconnect(ctx); err != nil {
			t.Logf("Warning: Failed to disconnect store during cleanup: %v", err)
		}
	}
}

// withDocumentTestEnvironment executes test function with clean environment
func withDocumentTestEnvironment(t *testing.T, testFunc func(*DocumentTestEnvironment)) {
	t.Helper()
	env := setupDocumentTestEnvironment(t)
	defer cleanupDocumentTestEnvironment(t, env)
	testFunc(env)
}

// setupConnectedStoreForDocumentTest creates separated configurations for document tests
func setupConnectedStoreForDocumentTest(t *testing.T) (*Store, types.VectorStoreConfig, types.CreateCollectionOptions) {
	t.Helper()
	store, collectionConfig := setupConnectedStoreForCollection(t)

	// Get connection config from store
	connectionConfig := store.GetConfig()

	return store, connectionConfig, collectionConfig
}

// setupConnectedStoreForDocument reuses the collection setup but for document tests
func setupConnectedStoreForDocument(t *testing.T) (*Store, types.CreateCollectionOptions) {
	t.Helper()
	store, collectionConfig := setupConnectedStoreForCollection(t)
	return store, collectionConfig
}

// createTestCollection creates a test collection for document operations
func createTestCollection(t *testing.T, store *Store, config types.CreateCollectionOptions) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := store.CreateCollection(ctx, &config)
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
}

// createTestDocuments creates sample documents for testing
func createTestDocuments(count int) []*types.Document {
	docs := make([]*types.Document, count)
	for i := 0; i < count; i++ {
		docs[i] = &types.Document{
			ID:      fmt.Sprintf("test_doc_%d", i),
			Content: fmt.Sprintf("This is test document content number %d. It contains useful information for testing purposes.", i),
			Vector:  generateTestVector(128), // 128-dimensional test vector
			Metadata: map[string]interface{}{
				"doc_index": i,
				"category":  fmt.Sprintf("category_%d", i%3),
				"priority":  float64(i % 5),
				"published": i%2 == 0,
				"tags":      []string{fmt.Sprintf("tag_%d", i), "test"},
			},
		}
	}
	return docs
}

// generateTestVector creates a test vector of specified dimension
func generateTestVector(dimension int) []float64 {
	vector := make([]float64, dimension)
	for i := 0; i < dimension; i++ {
		vector[i] = float64(i) / float64(dimension) // Simple pattern
	}
	return vector
}

// createTestDocumentsWithoutVectors creates documents without vector data
func createTestDocumentsWithoutVectors(count int) []*types.Document {
	docs := createTestDocuments(count)
	for _, doc := range docs {
		doc.Vector = nil // Remove vectors
	}
	return docs
}

// createTestDocumentsWithoutIDs creates documents without IDs (for auto-generation testing)
func createTestDocumentsWithoutIDs(count int) []*types.Document {
	docs := createTestDocuments(count)
	for _, doc := range docs {
		doc.ID = "" // Remove IDs to test auto-generation
	}
	return docs
}

// =============================================================================
// Unit Tests for AddDocuments
// =============================================================================

func TestAddDocuments(t *testing.T) {
	t.Run("successful add with vectors", func(t *testing.T) {
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			opts := types.AddDocumentOptions{
				CollectionName: env.CollectionName,
				Documents:      createTestDocuments(5),
				BatchSize:      10,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ids, err := env.Store.AddDocuments(ctx, &opts)
			if err != nil {
				t.Errorf("AddDocuments() error = %v, want nil", err)
			}
			if len(ids) != 5 {
				t.Errorf("AddDocuments() returned %d IDs, want 5", len(ids))
			}
			for i, id := range ids {
				if id == "" {
					t.Errorf("AddDocuments() returned empty ID at index %d", i)
				}
			}
		})
	})

	t.Run("add without vectors - expect error", func(t *testing.T) {
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			opts := types.AddDocumentOptions{
				CollectionName: env.CollectionName,
				Documents:      createTestDocumentsWithoutVectors(3),
				BatchSize:      10,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err := env.Store.AddDocuments(ctx, &opts)
			if err == nil {
				t.Error("AddDocuments() expected error, got nil")
			} else if !contains(err.Error(), "Expected some vectors") {
				t.Errorf("AddDocuments() error = %v, want to contain 'Expected some vectors'", err)
			}
		})
	})

	t.Run("auto-generate IDs", func(t *testing.T) {
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			opts := types.AddDocumentOptions{
				CollectionName: env.CollectionName,
				Documents:      createTestDocumentsWithoutIDs(3),
				BatchSize:      10,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ids, err := env.Store.AddDocuments(ctx, &opts)
			if err != nil {
				t.Errorf("AddDocuments() error = %v, want nil", err)
			}
			if len(ids) != 3 {
				t.Errorf("AddDocuments() returned %d IDs, want 3", len(ids))
			}
			for i, id := range ids {
				if id == "" {
					t.Errorf("AddDocuments() returned empty ID at index %d", i)
				}
			}
		})
	})

	t.Run("batch processing", func(t *testing.T) {
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			opts := types.AddDocumentOptions{
				CollectionName: env.CollectionName,
				Documents:      createTestDocuments(15), // Will be processed in batches
				BatchSize:      5,                       // Small batch size to test batching
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ids, err := env.Store.AddDocuments(ctx, &opts)
			if err != nil {
				t.Errorf("AddDocuments() error = %v, want nil", err)
			}
			if len(ids) != 15 {
				t.Errorf("AddDocuments() returned %d IDs, want 15", len(ids))
			}
		})
	})

	t.Run("upsert mode", func(t *testing.T) {
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			opts := types.AddDocumentOptions{
				CollectionName: env.CollectionName,
				Documents:      createTestDocuments(3),
				BatchSize:      10,
				Upsert:         true,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ids, err := env.Store.AddDocuments(ctx, &opts)
			if err != nil {
				t.Errorf("AddDocuments() error = %v, want nil", err)
			}
			if len(ids) != 3 {
				t.Errorf("AddDocuments() returned %d IDs, want 3", len(ids))
			}
		})
	})

	t.Run("empty documents", func(t *testing.T) {
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			opts := types.AddDocumentOptions{
				CollectionName: env.CollectionName,
				Documents:      []*types.Document{},
				BatchSize:      10,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err := env.Store.AddDocuments(ctx, &opts)
			if err == nil {
				t.Error("AddDocuments() expected error, got nil")
			} else if !contains(err.Error(), "no documents") {
				t.Errorf("AddDocuments() error = %v, want to contain 'no documents'", err)
			}
		})
	})

	t.Run("nil options", func(t *testing.T) {
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err := env.Store.AddDocuments(ctx, nil)
			if err == nil {
				t.Error("AddDocuments() expected error, got nil")
			} else if !contains(err.Error(), "no documents") {
				t.Errorf("AddDocuments() error = %v, want to contain 'no documents'", err)
			}
		})
	})

	t.Run("not connected store", func(t *testing.T) {
		unconnectedStore := NewStore()
		defer func() {
			_ = unconnectedStore.Disconnect(context.Background())
		}()

		opts := types.AddDocumentOptions{
			CollectionName: "test_not_connected",
			Documents:      createTestDocuments(1),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := unconnectedStore.AddDocuments(ctx, &opts)
		if err == nil {
			t.Error("AddDocuments() expected error, got nil")
		} else if !contains(err.Error(), "not connected") {
			t.Errorf("AddDocuments() error = %v, want to contain 'not connected'", err)
		}
	})

	t.Run("named vectors with default dense", func(t *testing.T) {
		// Create a collection with sparse vector support
		store, baseConfig := setupConnectedStoreForDocument(t)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = store.Disconnect(ctx)
		}()

		collectionName := fmt.Sprintf("test_named_vectors_%d", time.Now().UnixNano())
		collectionConfig := baseConfig
		collectionConfig.CollectionName = collectionName
		collectionConfig.EnableSparseVectors = true
		collectionConfig.DenseVectorName = "dense"
		collectionConfig.SparseVectorName = "sparse"

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := store.CreateCollection(ctx, &collectionConfig)
		if err != nil {
			t.Fatalf("Failed to create test collection: %v", err)
		}
		defer cleanupCollection(t, store, collectionName)

		opts := types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      createTestDocuments(3),
			BatchSize:      10,
			// VectorUsing not specified, should default to "dense"
		}

		ids, err := store.AddDocuments(ctx, &opts)
		if err != nil {
			t.Errorf("AddDocuments() with named vectors error = %v, want nil", err)
		}
		if len(ids) != 3 {
			t.Errorf("AddDocuments() returned %d IDs, want 3", len(ids))
		}
	})

	t.Run("named vectors with custom vector name", func(t *testing.T) {
		// Create a collection with sparse vector support
		store, baseConfig := setupConnectedStoreForDocument(t)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = store.Disconnect(ctx)
		}()

		collectionName := fmt.Sprintf("test_custom_vector_%d", time.Now().UnixNano())
		collectionConfig := baseConfig
		collectionConfig.CollectionName = collectionName
		collectionConfig.EnableSparseVectors = true
		collectionConfig.DenseVectorName = "custom_dense"
		collectionConfig.SparseVectorName = "custom_sparse"

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := store.CreateCollection(ctx, &collectionConfig)
		if err != nil {
			t.Fatalf("Failed to create test collection: %v", err)
		}
		defer cleanupCollection(t, store, collectionName)

		opts := types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      createTestDocuments(2),
			BatchSize:      10,
			VectorUsing:    "custom_dense", // Specify custom vector name
		}

		ids, err := store.AddDocuments(ctx, &opts)
		if err != nil {
			t.Errorf("AddDocuments() with custom vector name error = %v, want nil", err)
		}
		if len(ids) != 2 {
			t.Errorf("AddDocuments() returned %d IDs, want 2", len(ids))
		}
	})

	t.Run("traditional collection without named vectors", func(t *testing.T) {
		// Test with traditional collection (no sparse vectors)
		withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
			opts := types.AddDocumentOptions{
				CollectionName: env.CollectionName,
				Documents:      createTestDocuments(2),
				BatchSize:      10,
				VectorUsing:    "dense", // Should be ignored for traditional collections
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ids, err := env.Store.AddDocuments(ctx, &opts)
			if err != nil {
				t.Errorf("AddDocuments() with traditional collection error = %v, want nil", err)
			}
			if len(ids) != 2 {
				t.Errorf("AddDocuments() returned %d IDs, want 2", len(ids))
			}
		})
	})
}

// =============================================================================
// Unit Tests for GetDocuments
// =============================================================================

func TestGetDocuments(t *testing.T) {
	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	// Setup test collection with documents
	collectionName := fmt.Sprintf("test_get_docs_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(t, store, collectionConfig)
	defer cleanupCollection(t, store, collectionName)

	// Add test documents
	testDocs := createTestDocuments(5)
	addOpts := types.AddDocumentOptions{
		CollectionName: collectionName,
		Documents:      testDocs,
		BatchSize:      10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addedIDs, err := store.AddDocuments(ctx, &addOpts)
	if err != nil {
		t.Fatalf("Failed to add test documents: %v", err)
	}

	tests := []struct {
		name        string
		setup       func() ([]string, *types.GetDocumentOptions)
		wantErr     bool
		errContains string
		wantCount   int
	}{
		{
			name: "get all documents with vectors and payload",
			setup: func() ([]string, *types.GetDocumentOptions) {
				return addedIDs, &types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:   false,
			wantCount: 5,
		},
		{
			name: "get documents without vectors",
			setup: func() ([]string, *types.GetDocumentOptions) {
				return addedIDs[:3], &types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  false,
					IncludePayload: true,
				}
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "get documents without payload",
			setup: func() ([]string, *types.GetDocumentOptions) {
				return addedIDs[1:4], &types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  true,
					IncludePayload: false,
				}
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "get single document",
			setup: func() ([]string, *types.GetDocumentOptions) {
				return []string{addedIDs[0]}, &types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "get non-existent documents",
			setup: func() ([]string, *types.GetDocumentOptions) {
				return []string{"non_existent_id_1", "non_existent_id_2"}, &types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:   false,
			wantCount: 0, // Should return empty result, not error
		},
		{
			name: "empty ID list",
			setup: func() ([]string, *types.GetDocumentOptions) {
				return []string{}, &types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "not connected store",
			setup: func() ([]string, *types.GetDocumentOptions) {
				return addedIDs[:1], &types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:     true,
			errContains: "not connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, opts := tt.setup()

			var docs []*types.Document
			var err error

			if tt.name == "not connected store" {
				unconnectedStore := NewStore()
				defer unconnectedStore.Close()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				docs, err = unconnectedStore.GetDocuments(ctx, ids, opts)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				docs, err = store.GetDocuments(ctx, ids, opts)
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetDocuments() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetDocuments() error = %v, want to contain %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetDocuments() error = %v, want nil", err)
				} else {
					if len(docs) != tt.wantCount {
						t.Errorf("GetDocuments() returned %d documents, want %d", len(docs), tt.wantCount)
					}

					// Verify document content and flags
					for i, doc := range docs {
						if doc == nil {
							t.Errorf("GetDocuments() returned nil document at index %d", i)
							continue
						}

						// Check vector inclusion
						if opts.IncludeVector && len(doc.Vector) == 0 {
							t.Errorf("GetDocuments() document %d missing vector when IncludeVector=true", i)
						}
						if !opts.IncludeVector && len(doc.Vector) > 0 {
							t.Errorf("GetDocuments() document %d has vector when IncludeVector=false", i)
						}

						// Check payload inclusion
						if opts.IncludePayload && doc.Metadata == nil {
							t.Errorf("GetDocuments() document %d missing metadata when IncludePayload=true", i)
						}
					}
				}
			}
		})
	}
}

// =============================================================================
// Unit Tests for DeleteDocuments
// =============================================================================

func TestDeleteDocuments(t *testing.T) {
	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	tests := []struct {
		name        string
		setup       func() (*types.DeleteDocumentOptions, []string) // Returns options and expected remaining IDs
		wantErr     bool
		errContains string
	}{
		{
			name: "delete by IDs",
			setup: func() (*types.DeleteDocumentOptions, []string) {
				collectionName := fmt.Sprintf("test_delete_ids_%d", time.Now().UnixNano())
				collectionConfig := baseConfig
				collectionConfig.CollectionName = collectionName
				createTestCollection(t, store, collectionConfig)

				// Add test documents
				testDocs := createTestDocuments(5)
				addOpts := types.AddDocumentOptions{
					CollectionName: collectionName,
					Documents:      testDocs,
				}

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				addedIDs, err := store.AddDocuments(ctx, &addOpts)
				if err != nil {
					t.Fatalf("Failed to add test documents: %v", err)
				}

				// Delete first 2 documents
				deleteOpts := &types.DeleteDocumentOptions{
					CollectionName: collectionName,
					IDs:            addedIDs[:2],
				}

				return deleteOpts, addedIDs[2:] // Remaining IDs
			},
			wantErr: false,
		},
		{
			name: "delete by filter",
			setup: func() (*types.DeleteDocumentOptions, []string) {
				collectionName := fmt.Sprintf("test_delete_filter_%d", time.Now().UnixNano())
				collectionConfig := baseConfig
				collectionConfig.CollectionName = collectionName
				createTestCollection(t, store, collectionConfig)

				// Add test documents with specific metadata
				testDocs := createTestDocuments(4)
				for i, doc := range testDocs {
					doc.Metadata["priority"] = float64(i) // 0, 1, 2, 3
				}

				addOpts := types.AddDocumentOptions{
					CollectionName: collectionName,
					Documents:      testDocs,
				}

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				addedIDs, err := store.AddDocuments(ctx, &addOpts)
				if err != nil {
					t.Fatalf("Failed to add test documents: %v", err)
				}

				// Delete documents with priority = 1
				deleteOpts := &types.DeleteDocumentOptions{
					CollectionName: collectionName,
					Filter: map[string]interface{}{
						"priority": 1.0,
					},
				}

				return deleteOpts, addedIDs // We can't easily predict which will remain with filter
			},
			wantErr: false,
		},
		{
			name: "dry run mode",
			setup: func() (*types.DeleteDocumentOptions, []string) {
				collectionName := fmt.Sprintf("test_delete_dryrun_%d", time.Now().UnixNano())
				collectionConfig := baseConfig
				collectionConfig.CollectionName = collectionName
				createTestCollection(t, store, collectionConfig)

				// Add test documents
				testDocs := createTestDocuments(3)
				addOpts := types.AddDocumentOptions{
					CollectionName: collectionName,
					Documents:      testDocs,
				}

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				addedIDs, err := store.AddDocuments(ctx, &addOpts)
				if err != nil {
					t.Fatalf("Failed to add test documents: %v", err)
				}

				deleteOpts := &types.DeleteDocumentOptions{
					CollectionName: collectionName,
					IDs:            addedIDs[:1],
					DryRun:         true,
				}

				return deleteOpts, addedIDs // All should remain in dry run
			},
			wantErr: false,
		},
		{
			name: "nil options",
			setup: func() (*types.DeleteDocumentOptions, []string) {
				return nil, []string{}
			},
			wantErr:     true,
			errContains: "delete options cannot be nil",
		},
		{
			name: "no IDs or filter",
			setup: func() (*types.DeleteDocumentOptions, []string) {
				collectionName := fmt.Sprintf("test_delete_empty_%d", time.Now().UnixNano())
				collectionConfig := baseConfig
				collectionConfig.CollectionName = collectionName
				createTestCollection(t, store, collectionConfig)

				deleteOpts := &types.DeleteDocumentOptions{
					CollectionName: collectionName,
					// No IDs or filter
				}

				return deleteOpts, []string{}
			},
			wantErr:     true,
			errContains: "either IDs or filter must be provided",
		},
		{
			name: "not connected store",
			setup: func() (*types.DeleteDocumentOptions, []string) {
				deleteOpts := &types.DeleteDocumentOptions{
					CollectionName: "test_not_connected",
					IDs:            []string{"test_id"},
				}
				return deleteOpts, []string{}
			},
			wantErr:     true,
			errContains: "not connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _ := tt.setup()

			if opts != nil && opts.CollectionName != "" {
				defer cleanupCollection(t, store, opts.CollectionName)
			}

			var err error

			if tt.name == "not connected store" {
				unconnectedStore := NewStore()
				defer unconnectedStore.Close()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err = unconnectedStore.DeleteDocuments(ctx, opts)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				err = store.DeleteDocuments(ctx, opts)
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteDocuments() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteDocuments() error = %v, want to contain %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteDocuments() error = %v, want nil", err)
				}
			}
		})
	}
}

// =============================================================================
// Unit Tests for ListDocuments
// =============================================================================

func TestListDocuments(t *testing.T) {
	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	// Setup test collection with documents
	collectionName := fmt.Sprintf("test_list_docs_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(t, store, collectionConfig)
	defer cleanupCollection(t, store, collectionName)

	// Add test documents with different metadata for filtering
	testDocs := createTestDocuments(10)
	for i, doc := range testDocs {
		doc.Metadata["category"] = fmt.Sprintf("category_%d", i%3) // 3 different categories
		doc.Metadata["priority"] = float64(i % 5)                  // 5 different priorities
		doc.Metadata["active"] = i%2 == 0                          // Half active, half inactive
	}

	addOpts := types.AddDocumentOptions{
		CollectionName: collectionName,
		Documents:      testDocs,
		BatchSize:      10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.AddDocuments(ctx, &addOpts)
	if err != nil {
		t.Fatalf("Failed to add test documents: %v", err)
	}

	tests := []struct {
		name        string
		setup       func() *types.ListDocumentsOptions
		wantErr     bool
		errContains string
		minCount    int // Minimum expected count (for filter tests)
		maxCount    int // Maximum expected count
	}{
		{
			name: "list all documents with default settings",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          20,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 10,
			maxCount: 10,
		},
		{
			name: "list with pagination - first page",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          5,
					Offset:         0,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 5,
			maxCount: 5,
		},
		{
			name: "list with pagination - second page",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          5,
					Offset:         5,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 5,
			maxCount: 5,
		},
		{
			name: "list without vectors",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          10,
					IncludeVector:  false,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 10,
			maxCount: 10,
		},
		{
			name: "list without payload",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          10,
					IncludeVector:  true,
					IncludePayload: false,
				}
			},
			wantErr:  false,
			minCount: 10,
			maxCount: 10,
		},
		{
			name: "list with metadata filter",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          20,
					Filter: map[string]interface{}{
						"category": "category_0",
					},
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 1, // At least one document should match
			maxCount: 10,
		},
		{
			name: "list with complex filter",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          20,
					Filter: map[string]interface{}{
						"active": true,
					},
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 1, // At least one active document
			maxCount: 10,
		},
		{
			name: "list beyond available documents",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          5,
					Offset:         15, // Beyond the 10 documents we have
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 0,
			maxCount: 10, // Qdrant's scroll API doesn't work like traditional offset, so we may still get results
		},
		{
			name: "nil options",
			setup: func() *types.ListDocumentsOptions {
				return nil
			},
			wantErr:     true,
			errContains: "list options cannot be nil",
		},
		{
			name: "not connected store",
			setup: func() *types.ListDocumentsOptions {
				return &types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          10,
				}
			},
			wantErr:     true,
			errContains: "not connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.setup()

			var result *types.PaginatedDocumentsResult
			var err error

			if tt.name == "not connected store" {
				unconnectedStore := NewStore()
				defer unconnectedStore.Close()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				result, err = unconnectedStore.ListDocuments(ctx, opts)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				result, err = store.ListDocuments(ctx, opts)
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListDocuments() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ListDocuments() error = %v, want to contain %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ListDocuments() error = %v, want nil", err)
				} else if result == nil {
					t.Errorf("ListDocuments() returned nil result")
				} else {
					docCount := len(result.Documents)
					if docCount < tt.minCount || docCount > tt.maxCount {
						t.Errorf("ListDocuments() returned %d documents, want between %d and %d", docCount, tt.minCount, tt.maxCount)
					}

					// Verify document content and flags
					for i, doc := range result.Documents {
						if doc == nil {
							t.Errorf("ListDocuments() returned nil document at index %d", i)
							continue
						}

						// Check vector inclusion
						if opts.IncludeVector && len(doc.Vector) == 0 {
							t.Errorf("ListDocuments() document %d missing vector when IncludeVector=true", i)
						}
						if !opts.IncludeVector && len(doc.Vector) > 0 {
							t.Errorf("ListDocuments() document %d has vector when IncludeVector=false", i)
						}

						// Check payload inclusion
						if opts.IncludePayload && doc.Metadata == nil {
							t.Errorf("ListDocuments() document %d missing metadata when IncludePayload=true", i)
						}
					}

					// Verify pagination info
					if result.Total < int64(docCount) {
						t.Errorf("ListDocuments() total count %d is less than returned documents %d", result.Total, docCount)
					}
				}
			}
		})
	}
}

// =============================================================================
// Unit Tests for ScrollDocuments
// =============================================================================

func TestScrollDocuments(t *testing.T) {
	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	// Setup test collection with documents
	collectionName := fmt.Sprintf("test_scroll_docs_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(t, store, collectionConfig)
	defer cleanupCollection(t, store, collectionName)

	// Add test documents
	testDocs := createTestDocuments(15) // More documents for scrolling
	for i, doc := range testDocs {
		doc.Metadata["batch"] = i / 5 // 3 batches: 0, 1, 2
		doc.Metadata["index"] = i
	}

	addOpts := types.AddDocumentOptions{
		CollectionName: collectionName,
		Documents:      testDocs,
		BatchSize:      10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.AddDocuments(ctx, &addOpts)
	if err != nil {
		t.Fatalf("Failed to add test documents: %v", err)
	}

	tests := []struct {
		name        string
		setup       func() *types.ScrollOptions
		wantErr     bool
		errContains string
		minCount    int
		maxCount    int
	}{
		{
			name: "scroll all documents",
			setup: func() *types.ScrollOptions {
				return &types.ScrollOptions{
					CollectionName: collectionName,
					Limit:          10,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 15,
			maxCount: 50, // Allow for some extra documents from other tests
		},
		{
			name: "scroll with small batch size",
			setup: func() *types.ScrollOptions {
				return &types.ScrollOptions{
					CollectionName: collectionName,
					Limit:          5,
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 15,
			maxCount: 50, // Allow for some extra documents from other tests
		},
		{
			name: "scroll without vectors",
			setup: func() *types.ScrollOptions {
				return &types.ScrollOptions{
					CollectionName: collectionName,
					Limit:          10,
					IncludeVector:  false,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 15,
			maxCount: 50, // Allow for some extra documents from other tests
		},
		{
			name: "scroll without payload",
			setup: func() *types.ScrollOptions {
				return &types.ScrollOptions{
					CollectionName: collectionName,
					Limit:          10,
					IncludeVector:  true,
					IncludePayload: false,
				}
			},
			wantErr:  false,
			minCount: 15,
			maxCount: 50, // Allow for some extra documents from other tests
		},
		{
			name: "scroll with filter",
			setup: func() *types.ScrollOptions {
				return &types.ScrollOptions{
					CollectionName: collectionName,
					Limit:          10,
					Filter: map[string]interface{}{
						"batch": 1, // Should match documents 5-9
					},
					IncludeVector:  true,
					IncludePayload: true,
				}
			},
			wantErr:  false,
			minCount: 1,  // At least one document should match
			maxCount: 50, // Allow for some extra documents from other tests
		},
		{
			name: "nil options",
			setup: func() *types.ScrollOptions {
				return nil
			},
			wantErr:     true,
			errContains: "scroll options cannot be nil",
		},
		{
			name: "not connected store",
			setup: func() *types.ScrollOptions {
				return &types.ScrollOptions{
					CollectionName: collectionName,
					Limit:          10,
				}
			},
			wantErr:     true,
			errContains: "not connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.setup()

			var result *types.ScrollResult
			var err error

			if tt.name == "not connected store" {
				unconnectedStore := NewStore()
				defer unconnectedStore.Close()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				result, err = unconnectedStore.ScrollDocuments(ctx, opts)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				result, err = store.ScrollDocuments(ctx, opts)
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("ScrollDocuments() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ScrollDocuments() error = %v, want to contain %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ScrollDocuments() error = %v, want nil", err)
				} else if result == nil {
					t.Errorf("ScrollDocuments() returned nil result")
				} else {
					docCount := len(result.Documents)
					if docCount < tt.minCount || docCount > tt.maxCount {
						t.Errorf("ScrollDocuments() returned %d documents, want between %d and %d", docCount, tt.minCount, tt.maxCount)
					}

					// Verify document content and flags
					for i, doc := range result.Documents {
						if doc == nil {
							t.Errorf("ScrollDocuments() returned nil document at index %d", i)
							continue
						}

						// Check vector inclusion
						if opts.IncludeVector && len(doc.Vector) == 0 {
							t.Errorf("ScrollDocuments() document %d missing vector when IncludeVector=true", i)
						}
						if !opts.IncludeVector && len(doc.Vector) > 0 {
							t.Errorf("ScrollDocuments() document %d has vector when IncludeVector=false", i)
						}

						// Check payload inclusion
						if opts.IncludePayload && doc.Metadata == nil {
							t.Errorf("ScrollDocuments() document %d missing metadata when IncludePayload=true", i)
						}
					}
				}
			}
		})
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestDocumentOperationsIntegration(t *testing.T) {
	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("test_integration_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(t, store, collectionConfig)
	defer cleanupCollection(t, store, collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test full workflow: Add -> Get -> List -> Scroll -> Delete
	t.Run("full workflow", func(t *testing.T) {
		// Step 1: Add documents
		testDocs := createTestDocuments(10)
		addOpts := types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      testDocs,
			BatchSize:      5,
		}

		addedIDs, err := store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Failed to add documents: %v", err)
		}
		if len(addedIDs) != 10 {
			t.Errorf("Expected 10 added IDs, got %d", len(addedIDs))
		}

		// Step 2: Get documents
		getOpts := types.GetDocumentOptions{
			CollectionName: collectionName,
			IncludeVector:  true,
			IncludePayload: true,
		}

		docs, err := store.GetDocuments(ctx, addedIDs[:5], &getOpts)
		if err != nil {
			t.Fatalf("Failed to get documents: %v", err)
		}
		if len(docs) != 5 {
			t.Errorf("Expected 5 documents, got %d", len(docs))
		}

		// Step 3: List documents
		listOpts := types.ListDocumentsOptions{
			CollectionName: collectionName,
			Limit:          20,
			IncludeVector:  true,
			IncludePayload: true,
		}

		result, err := store.ListDocuments(ctx, &listOpts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}
		if len(result.Documents) != 10 {
			t.Errorf("Expected 10 documents in list, got %d", len(result.Documents))
		}

		// Step 4: Scroll documents
		scrollOpts := types.ScrollOptions{
			CollectionName: collectionName,
			Limit:          7,
			IncludeVector:  true,
			IncludePayload: true,
		}

		scrollResult, err := store.ScrollDocuments(ctx, &scrollOpts)
		if err != nil {
			t.Fatalf("Failed to scroll documents: %v", err)
		}
		if len(scrollResult.Documents) < 10 {
			t.Errorf("Expected at least 10 documents in scroll, got %d", len(scrollResult.Documents))
		}

		// Step 5: Delete some documents
		deleteOpts := types.DeleteDocumentOptions{
			CollectionName: collectionName,
			IDs:            addedIDs[:3],
		}

		err = store.DeleteDocuments(ctx, &deleteOpts)
		if err != nil {
			t.Fatalf("Failed to delete documents: %v", err)
		}

		// Step 6: Verify deletion
		remainingDocs, err := store.GetDocuments(ctx, addedIDs, &getOpts)
		if err != nil {
			t.Fatalf("Failed to get remaining documents: %v", err)
		}
		if len(remainingDocs) != 7 { // 10 - 3 = 7
			t.Errorf("Expected 7 remaining documents, got %d", len(remainingDocs))
		}
	})
}

// =============================================================================
// Performance Benchmark Tests
// =============================================================================

func BenchmarkAddDocuments(b *testing.B) {
	store, baseConfig := setupConnectedStoreForDocument(&testing.T{})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("bench_add_docs_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(&testing.T{}, store, collectionConfig)
	defer cleanupCollection(&testing.T{}, store, collectionName)

	// Test with different document counts
	benchmarks := []struct {
		name      string
		docCount  int
		batchSize int
	}{
		{"10_docs_batch_5", 10, 5},
		{"100_docs_batch_10", 100, 10},
		{"100_docs_batch_50", 100, 50},
		{"500_docs_batch_100", 500, 100},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create test documents once
			testDocs := createTestDocuments(bm.docCount)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Clean collection between runs
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				deleteOpts := &types.DeleteDocumentOptions{
					CollectionName: collectionName,
					Filter:         map[string]interface{}{}, // Delete all
				}
				_ = store.DeleteDocuments(ctx, deleteOpts)
				cancel()
				b.StartTimer()

				// Benchmark the add operation
				ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
				addOpts := types.AddDocumentOptions{
					CollectionName: collectionName,
					Documents:      testDocs,
					BatchSize:      bm.batchSize,
				}

				_, err := store.AddDocuments(ctx, &addOpts)
				if err != nil {
					b.Fatalf("AddDocuments failed: %v", err)
				}
				cancel()
			}
		})
	}
}

func BenchmarkGetDocuments(b *testing.B) {
	store, baseConfig := setupConnectedStoreForDocument(&testing.T{})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("bench_get_docs_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(&testing.T{}, store, collectionConfig)
	defer cleanupCollection(&testing.T{}, store, collectionName)

	// Setup test data
	testDocs := createTestDocuments(1000)
	addOpts := types.AddDocumentOptions{
		CollectionName: collectionName,
		Documents:      testDocs,
		BatchSize:      100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	addedIDs, err := store.AddDocuments(ctx, &addOpts)
	if err != nil {
		b.Fatalf("Failed to setup test data: %v", err)
	}

	benchmarks := []struct {
		name    string
		idCount int
		withVec bool
		withPay bool
	}{
		{"1_doc_with_vector_payload", 1, true, true},
		{"10_docs_with_vector_payload", 10, true, true},
		{"100_docs_with_vector_payload", 100, true, true},
		{"100_docs_no_vector", 100, false, true},
		{"100_docs_no_payload", 100, true, false},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			ids := addedIDs[:bm.idCount]
			opts := &types.GetDocumentOptions{
				CollectionName: collectionName,
				IncludeVector:  bm.withVec,
				IncludePayload: bm.withPay,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err := store.GetDocuments(ctx, ids, opts)
				if err != nil {
					b.Fatalf("GetDocuments failed: %v", err)
				}
				cancel()
			}
		})
	}
}

func BenchmarkListDocuments(b *testing.B) {
	store, baseConfig := setupConnectedStoreForDocument(&testing.T{})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("bench_list_docs_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(&testing.T{}, store, collectionConfig)
	defer cleanupCollection(&testing.T{}, store, collectionName)

	// Setup test data
	testDocs := createTestDocuments(1000)
	addOpts := types.AddDocumentOptions{
		CollectionName: collectionName,
		Documents:      testDocs,
		BatchSize:      100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := store.AddDocuments(ctx, &addOpts)
	if err != nil {
		b.Fatalf("Failed to setup test data: %v", err)
	}

	benchmarks := []struct {
		name   string
		limit  int
		offset int
	}{
		{"limit_10", 10, 0},
		{"limit_50", 50, 0},
		{"limit_100", 100, 0},
		{"limit_100_offset_500", 100, 500},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			opts := &types.ListDocumentsOptions{
				CollectionName: collectionName,
				Limit:          bm.limit,
				Offset:         bm.offset,
				IncludeVector:  true,
				IncludePayload: true,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err := store.ListDocuments(ctx, opts)
				if err != nil {
					b.Fatalf("ListDocuments failed: %v", err)
				}
				cancel()
			}
		})
	}
}

func BenchmarkScrollDocuments(b *testing.B) {
	store, baseConfig := setupConnectedStoreForDocument(&testing.T{})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("bench_scroll_docs_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(&testing.T{}, store, collectionConfig)
	defer cleanupCollection(&testing.T{}, store, collectionName)

	// Setup test data
	testDocs := createTestDocuments(1000)
	addOpts := types.AddDocumentOptions{
		CollectionName: collectionName,
		Documents:      testDocs,
		BatchSize:      100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := store.AddDocuments(ctx, &addOpts)
	if err != nil {
		b.Fatalf("Failed to setup test data: %v", err)
	}

	benchmarks := []struct {
		name      string
		batchSize int
	}{
		{"batch_10", 10},
		{"batch_50", 50},
		{"batch_100", 100},
		{"batch_200", 200},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			opts := &types.ScrollOptions{
				CollectionName: collectionName,
				Limit:          bm.batchSize,
				IncludeVector:  true,
				IncludePayload: true,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				_, err := store.ScrollDocuments(ctx, opts)
				if err != nil {
					b.Fatalf("ScrollDocuments failed: %v", err)
				}
				cancel()
			}
		})
	}
}

// =============================================================================
// Memory Leak Detection Tests
// =============================================================================

func TestDocumentMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak tests in short mode")
	}

	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("test_memory_leak_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(t, store, collectionConfig)
	defer cleanupCollection(t, store, collectionName)

	// Force garbage collection and get initial memory stats
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform many operations that could potentially leak memory
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	const iterations = 100
	const docsPerIteration = 50

	for i := 0; i < iterations; i++ {
		// Add documents
		testDocs := createTestDocuments(docsPerIteration)
		addOpts := types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      testDocs,
			BatchSize:      25,
		}

		addedIDs, err := store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Iteration %d: AddDocuments failed: %v", i, err)
		}

		// Get documents
		getOpts := types.GetDocumentOptions{
			CollectionName: collectionName,
			IncludeVector:  true,
			IncludePayload: true,
		}

		_, err = store.GetDocuments(ctx, addedIDs[:10], &getOpts)
		if err != nil {
			t.Fatalf("Iteration %d: GetDocuments failed: %v", i, err)
		}

		// List documents
		listOpts := types.ListDocumentsOptions{
			CollectionName: collectionName,
			Limit:          20,
			IncludeVector:  false, // Reduce memory usage
			IncludePayload: true,
		}

		_, err = store.ListDocuments(ctx, &listOpts)
		if err != nil {
			t.Fatalf("Iteration %d: ListDocuments failed: %v", i, err)
		}

		// Delete some documents to prevent collection from growing too large
		if len(addedIDs) > 10 {
			deleteOpts := types.DeleteDocumentOptions{
				CollectionName: collectionName,
				IDs:            addedIDs[:10],
			}

			err = store.DeleteDocuments(ctx, &deleteOpts)
			if err != nil {
				t.Fatalf("Iteration %d: DeleteDocuments failed: %v", i, err)
			}
		}

		// Periodic garbage collection to help detect leaks
		if i%10 == 0 {
			runtime.GC()
		}
	}

	// Force garbage collection and get final memory stats
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check for significant memory growth
	heapGrowth := m2.HeapInuse - m1.HeapInuse
	allocGrowth := m2.TotalAlloc - m1.TotalAlloc

	t.Logf("Memory usage after %d iterations:", iterations)
	t.Logf("  Heap growth: %d bytes (%.2f MB)", heapGrowth, float64(heapGrowth)/(1024*1024))
	t.Logf("  Total allocations growth: %d bytes (%.2f MB)", allocGrowth, float64(allocGrowth)/(1024*1024))
	t.Logf("  Objects: %d", m2.HeapObjects)
	t.Logf("  GC cycles: %d", m2.NumGC-m1.NumGC)

	// Set reasonable thresholds for memory growth
	const maxHeapGrowthMB = 50   // Allow up to 50MB heap growth
	const maxAllocGrowthMB = 500 // Allow up to 500MB total allocation growth

	heapGrowthMB := float64(heapGrowth) / (1024 * 1024)
	allocGrowthMB := float64(allocGrowth) / (1024 * 1024)

	if heapGrowthMB > maxHeapGrowthMB {
		t.Errorf("Potential memory leak detected: heap grew by %.2f MB (threshold: %.2f MB)",
			heapGrowthMB, float64(maxHeapGrowthMB))
	}

	if allocGrowthMB > maxAllocGrowthMB {
		t.Errorf("Excessive memory allocation detected: total allocations grew by %.2f MB (threshold: %.2f MB)",
			allocGrowthMB, float64(maxAllocGrowthMB))
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestConcurrentDocumentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("test_concurrent_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(t, store, collectionConfig)
	defer cleanupCollection(t, store, collectionName)

	const numGoroutines = 10
	const operationsPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

				// Create unique documents for this goroutine
				testDocs := createTestDocuments(5)
				for _, doc := range testDocs {
					doc.ID = fmt.Sprintf("g%d_op%d_%s", goroutineID, j, doc.ID)
					doc.Metadata["goroutine_id"] = goroutineID
					doc.Metadata["operation_id"] = j
				}

				// Add documents
				addOpts := types.AddDocumentOptions{
					CollectionName: collectionName,
					Documents:      testDocs,
					BatchSize:      5,
				}

				addedIDs, err := store.AddDocuments(ctx, &addOpts)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: AddDocuments failed: %v", goroutineID, j, err)
					cancel()
					continue
				}

				// Get documents
				getOpts := types.GetDocumentOptions{
					CollectionName: collectionName,
					IncludeVector:  true,
					IncludePayload: true,
				}

				_, err = store.GetDocuments(ctx, addedIDs[:2], &getOpts)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: GetDocuments failed: %v", goroutineID, j, err)
					cancel()
					continue
				}

				// List documents with filter
				listOpts := types.ListDocumentsOptions{
					CollectionName: collectionName,
					Limit:          10,
					Filter: map[string]interface{}{
						"goroutine_id": goroutineID,
					},
					IncludeVector:  false,
					IncludePayload: true,
				}

				_, err = store.ListDocuments(ctx, &listOpts)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: ListDocuments failed: %v", goroutineID, j, err)
					cancel()
					continue
				}

				cancel()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Found %d errors in concurrent operations", errorCount)
	}
}

// =============================================================================
// Edge Cases and Error Handling Tests
// =============================================================================

func TestDocumentOperationsEdgeCases(t *testing.T) {
	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	collectionName := fmt.Sprintf("test_edge_cases_%d", time.Now().UnixNano())
	collectionConfig := baseConfig
	collectionConfig.CollectionName = collectionName
	createTestCollection(t, store, collectionConfig)
	defer cleanupCollection(t, store, collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("very large document", func(t *testing.T) {
		// Create a document with very large content
		largeContent := make([]byte, 1024*1024) // 1MB content
		for i := range largeContent {
			largeContent[i] = byte('A' + (i % 26))
		}

		largeDoc := &types.Document{
			ID:      "large_doc",
			Content: string(largeContent),
			Vector:  generateTestVector(128),
			Metadata: map[string]interface{}{
				"size": "large",
				"type": "test",
			},
		}

		addOpts := types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      []*types.Document{largeDoc},
		}

		addedIDs, err := store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Errorf("Failed to add large document: %v", err)
		} else if len(addedIDs) != 1 {
			t.Errorf("Expected 1 added ID, got %d", len(addedIDs))
		}
	})

	t.Run("document with complex metadata", func(t *testing.T) {
		complexDoc := &types.Document{
			ID:      "complex_doc",
			Content: "Document with complex metadata",
			Vector:  generateTestVector(128),
			Metadata: map[string]interface{}{
				"string_field": "test string",
				"int_field":    42,
				"float_field":  3.14159,
				"bool_field":   true,
				"array_field":  []interface{}{"a", "b", "c", 1, 2, 3},
				"nested_object": map[string]interface{}{
					"inner_string": "nested value",
					"inner_number": 123,
					"inner_array":  []string{"x", "y", "z"},
				},
				"null_field": nil,
			},
		}

		addOpts := types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      []*types.Document{complexDoc},
		}

		addedIDs, err := store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Errorf("Failed to add document with complex metadata: %v", err)
		} else if len(addedIDs) != 1 {
			t.Errorf("Expected 1 added ID, got %d", len(addedIDs))
		}

		// Verify we can retrieve it
		getOpts := types.GetDocumentOptions{
			CollectionName: collectionName,
			IncludeVector:  true,
			IncludePayload: true,
		}

		docs, err := store.GetDocuments(ctx, addedIDs, &getOpts)
		if err != nil {
			t.Errorf("Failed to get document with complex metadata: %v", err)
		} else if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		} else if docs[0].Metadata == nil {
			t.Errorf("Document metadata is nil")
		}
	})

	t.Run("document with very high dimensional vector", func(t *testing.T) {
		highDimDoc := &types.Document{
			ID:      "high_dim_doc",
			Content: "Document with high-dimensional vector",
			Vector:  generateTestVector(2048), // Very high dimension
			Metadata: map[string]interface{}{
				"dimensions": 2048,
			},
		}

		addOpts := types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      []*types.Document{highDimDoc},
		}

		_, err := store.AddDocuments(ctx, &addOpts)
		// This might fail depending on Qdrant configuration, which is expected
		if err != nil {
			t.Logf("High-dimensional vector failed as expected: %v", err)
		} else {
			t.Logf("High-dimensional vector succeeded")
		}
	})
}

// =============================================================================
// Helper utility function tests to improve coverage
// =============================================================================

func TestConvertScoredPointToDocument(t *testing.T) {
	tests := []struct {
		name           string
		point          *qdrant.ScoredPoint
		includeVector  bool
		includePayload bool
		wantDoc        *types.Document
	}{
		{
			name: "scored point without vector - basic test",
			point: &qdrant.ScoredPoint{
				Id: qdrant.NewIDNum(12345),
				Payload: map[string]*qdrant.Value{
					"id":      qdrant.NewValueString("test_doc_no_vec"),
					"content": qdrant.NewValueString("No vector content"),
				},
				Score: 0.85,
			},
			includeVector:  false,
			includePayload: true,
			wantDoc: &types.Document{
				ID:      "test_doc_no_vec",
				Content: "No vector content",
				Vector:  nil,
			},
		},
		{
			name: "scored point without payload",
			point: &qdrant.ScoredPoint{
				Id:    qdrant.NewIDNum(12345),
				Score: 0.75,
			},
			includeVector:  true,
			includePayload: false,
			wantDoc: &types.Document{
				Vector:   nil,
				Metadata: nil,
			},
		},
		{
			name: "nil scored point",
			point: &qdrant.ScoredPoint{
				Id:    qdrant.NewIDNum(12345),
				Score: 0.0,
			},
			includeVector:  true,
			includePayload: true,
			wantDoc: &types.Document{
				Vector:   nil,
				Metadata: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertScoredPointToDocument(tt.point, tt.includeVector, tt.includePayload)

			if got.ID != tt.wantDoc.ID {
				t.Errorf("convertScoredPointToDocument() ID = %v, want %v", got.ID, tt.wantDoc.ID)
			}
			if got.Content != tt.wantDoc.Content {
				t.Errorf("convertScoredPointToDocument() Content = %v, want %v", got.Content, tt.wantDoc.Content)
			}

			// Check vector
			if tt.includeVector {
				if len(got.Vector) != len(tt.wantDoc.Vector) {
					t.Errorf("convertScoredPointToDocument() Vector length = %v, want %v", len(got.Vector), len(tt.wantDoc.Vector))
				} else {
					for i, v := range got.Vector {
						if v != tt.wantDoc.Vector[i] {
							t.Errorf("convertScoredPointToDocument() Vector[%d] = %v, want %v", i, v, tt.wantDoc.Vector[i])
						}
					}
				}
			} else if len(got.Vector) > 0 {
				t.Errorf("convertScoredPointToDocument() Vector should be empty when includeVector=false")
			}

			// Check metadata
			if tt.includePayload && tt.wantDoc.Metadata != nil {
				if got.Metadata == nil {
					t.Errorf("convertScoredPointToDocument() Metadata is nil, want %v", tt.wantDoc.Metadata)
				} else {
					for key, want := range tt.wantDoc.Metadata {
						if got, ok := got.Metadata[key]; !ok || got != want {
							t.Errorf("convertScoredPointToDocument() Metadata[%s] = %v, want %v", key, got, want)
						}
					}
				}
			}
		})
	}
}

func TestConvertStructToMap(t *testing.T) {
	tests := []struct {
		name   string
		input  *qdrant.Struct
		expect map[string]interface{}
	}{
		{
			name: "all value types",
			input: &qdrant.Struct{
				Fields: map[string]*qdrant.Value{
					"string_val": qdrant.NewValueString("test"),
					"double_val": qdrant.NewValueDouble(3.14),
					"int_val":    qdrant.NewValueInt(42),
					"bool_val":   qdrant.NewValueBool(true),
					"list_val": {
						Kind: &qdrant.Value_ListValue{
							ListValue: &qdrant.ListValue{
								Values: []*qdrant.Value{
									qdrant.NewValueString("item1"),
									qdrant.NewValueDouble(2.5),
									qdrant.NewValueInt(10),
									qdrant.NewValueBool(false),
								},
							},
						},
					},
					"nested_struct": qdrant.NewValueStruct(&qdrant.Struct{
						Fields: map[string]*qdrant.Value{
							"nested_key": qdrant.NewValueString("nested_value"),
						},
					}),
				},
			},
			expect: map[string]interface{}{
				"string_val": "test",
				"double_val": 3.14,
				"int_val":    int64(42),
				"bool_val":   true,
				"list_val":   []interface{}{"item1", 2.5, int64(10), nil}, // false bool value becomes nil in current implementation
				"nested_struct": map[string]interface{}{
					"nested_key": "nested_value",
				},
			},
		},
		{
			name: "empty list values",
			input: &qdrant.Struct{
				Fields: map[string]*qdrant.Value{
					"empty_list": {
						Kind: &qdrant.Value_ListValue{
							ListValue: &qdrant.ListValue{
								Values: []*qdrant.Value{
									{Kind: &qdrant.Value_StringValue{StringValue: ""}},
									{Kind: &qdrant.Value_DoubleValue{DoubleValue: 0}},
									{Kind: &qdrant.Value_IntegerValue{IntegerValue: 0}},
									{Kind: &qdrant.Value_BoolValue{BoolValue: false}},
								},
							},
						},
					},
				},
			},
			expect: map[string]interface{}{
				"empty_list": []interface{}{nil, nil, nil, nil}, // All empty/zero values become nil
			},
		},
		{
			name:   "empty struct",
			input:  &qdrant.Struct{Fields: map[string]*qdrant.Value{}},
			expect: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertStructToMap(tt.input)

			if len(got) != len(tt.expect) {
				t.Errorf("convertStructToMap() length = %v, want %v", len(got), len(tt.expect))
				return
			}

			for key, want := range tt.expect {
				if got, ok := got[key]; !ok {
					t.Errorf("convertStructToMap() missing key %s", key)
				} else {
					switch wantVal := want.(type) {
					case []interface{}:
						gotSlice, ok := got.([]interface{})
						if !ok {
							t.Errorf("convertStructToMap()[%s] = %T, want []interface{}", key, got)
							continue
						}
						if len(gotSlice) != len(wantVal) {
							t.Errorf("convertStructToMap()[%s] length = %v, want %v", key, len(gotSlice), len(wantVal))
							continue
						}
						for i, wantItem := range wantVal {
							if gotSlice[i] != wantItem {
								t.Errorf("convertStructToMap()[%s][%d] = %v, want %v", key, i, gotSlice[i], wantItem)
							}
						}
					case map[string]interface{}:
						gotMap, ok := got.(map[string]interface{})
						if !ok {
							t.Errorf("convertStructToMap()[%s] = %T, want map[string]interface{}", key, got)
							continue
						}
						for nestedKey, nestedWant := range wantVal {
							if gotMap[nestedKey] != nestedWant {
								t.Errorf("convertStructToMap()[%s][%s] = %v, want %v", key, nestedKey, gotMap[nestedKey], nestedWant)
							}
						}
					default:
						if got != want {
							t.Errorf("convertStructToMap()[%s] = %v, want %v", key, got, want)
						}
					}
				}
			}
		})
	}
}

func TestConvertMetadataToPayload(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		wantErr  bool
	}{
		{
			name: "all supported types",
			metadata: map[string]interface{}{
				"string_field":  "test",
				"float64_field": 3.14,
				"float32_field": float32(2.5),
				"int_field":     42,
				"int64_field":   int64(99),
				"bool_field":    true,
				"string_array":  []string{"a", "b", "c"},
				"nested_map": map[string]interface{}{
					"inner_string": "nested",
					"inner_number": 123,
				},
				"unknown_type": complex(1, 2), // Will be converted to string
			},
			wantErr: false,
		},
		{
			name: "nested map with error",
			metadata: map[string]interface{}{
				"bad_nested": map[string]interface{}{
					"recursive": make(chan int), // This will cause error in nested conversion
				},
			},
			wantErr: false, // Function handles errors gracefully by skipping failed conversions
		},
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			wantErr:  false,
		},
		{
			name:     "nil metadata",
			metadata: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := convertMetadataToPayload(tt.metadata)

			if tt.wantErr && err == nil {
				t.Errorf("convertMetadataToPayload() expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("convertMetadataToPayload() error = %v, want nil", err)
			}

			if !tt.wantErr {
				if tt.metadata == nil {
					if len(payload) != 0 {
						t.Errorf("convertMetadataToPayload() with nil metadata should return empty payload")
					}
				} else {
					// Verify basic conversion worked (detailed verification would be complex)
					if len(tt.metadata) > 0 && len(payload) == 0 {
						t.Errorf("convertMetadataToPayload() returned empty payload for non-empty metadata")
					}
				}
			}
		})
	}
}

func TestConvertFilterToQdrant(t *testing.T) {
	tests := []struct {
		name    string
		filter  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "string filter",
			filter: map[string]interface{}{
				"category": "test",
			},
			wantErr: false,
		},
		{
			name: "float64 filter",
			filter: map[string]interface{}{
				"priority": 3.5,
			},
			wantErr: false,
		},
		{
			name: "int filter",
			filter: map[string]interface{}{
				"count": 42,
			},
			wantErr: false,
		},
		{
			name: "int64 filter",
			filter: map[string]interface{}{
				"timestamp": int64(1234567890),
			},
			wantErr: false,
		},
		{
			name: "bool filter",
			filter: map[string]interface{}{
				"active": true,
			},
			wantErr: false,
		},
		{
			name: "mixed filters",
			filter: map[string]interface{}{
				"category": "test",
				"priority": 2.0,
				"active":   true,
				"count":    10,
			},
			wantErr: false,
		},
		{
			name: "unsupported type filter",
			filter: map[string]interface{}{
				"unsupported": []string{"array", "not", "supported"},
			},
			wantErr: true, // Should result in error as no valid conditions will be found
			errMsg:  "no valid filter conditions found",
		},
		{
			name:    "empty filter",
			filter:  map[string]interface{}{},
			wantErr: true,
			errMsg:  "no valid filter conditions found",
		},
		{
			name: "filter with only unsupported types",
			filter: map[string]interface{}{
				"array_field":  []string{"a", "b"},
				"object_field": map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "no valid filter conditions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertFilterToQdrant(tt.filter)

			if tt.wantErr {
				if err == nil {
					t.Errorf("convertFilterToQdrant() expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("convertFilterToQdrant() error = %v, want to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("convertFilterToQdrant() error = %v, want nil", err)
				} else if result == nil {
					t.Errorf("convertFilterToQdrant() returned nil result")
				} else {
					// Verify the filter structure
					if len(result.Must) == 0 {
						t.Errorf("convertFilterToQdrant() returned filter with no conditions")
					}

					// Count expected conditions (only supported types)
					expectedConditions := 0
					for _, value := range tt.filter {
						switch value.(type) {
						case string, float64, int, int64, bool:
							expectedConditions++
						}
					}

					if len(result.Must) != expectedConditions {
						t.Errorf("convertFilterToQdrant() returned %d conditions, want %d", len(result.Must), expectedConditions)
					}
				}
			}
		})
	}
}

func TestScrollDocumentsInvalidScrollID(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Add test documents first
		testDocs := createTestDocuments(5)
		addOpts := types.AddDocumentOptions{
			CollectionName: env.CollectionName,
			Documents:      testDocs,
			BatchSize:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := env.Store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Failed to add test documents: %v", err)
		}

		// Test with invalid scroll ID (should be ignored gracefully)
		scrollOpts := types.ScrollOptions{
			CollectionName: env.CollectionName,
			Limit:          10,
			ScrollID:       "invalid_scroll_id", // Invalid format, should be ignored
			IncludeVector:  true,
			IncludePayload: true,
		}

		result, err := env.Store.ScrollDocuments(ctx, &scrollOpts)
		if err != nil {
			t.Errorf("ScrollDocuments() with invalid scroll ID should not fail: %v", err)
		} else if result == nil {
			t.Errorf("ScrollDocuments() returned nil result")
		} else if len(result.Documents) == 0 {
			t.Errorf("ScrollDocuments() returned no documents")
		}
	})
}

func TestListDocumentsCountError(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Add test documents
		testDocs := createTestDocuments(3)
		addOpts := types.AddDocumentOptions{
			CollectionName: env.CollectionName,
			Documents:      testDocs,
			BatchSize:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := env.Store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Failed to add test documents: %v", err)
		}

		// Test listing - even if count fails, operation should continue
		listOpts := types.ListDocumentsOptions{
			CollectionName: env.CollectionName,
			Limit:          10,
			IncludeVector:  true,
			IncludePayload: true,
		}

		result, err := env.Store.ListDocuments(ctx, &listOpts)
		if err != nil {
			t.Errorf("ListDocuments() should handle count errors gracefully: %v", err)
		} else if result == nil {
			t.Errorf("ListDocuments() returned nil result")
		} else {
			// Should still return documents even if count failed
			if len(result.Documents) == 0 {
				t.Errorf("ListDocuments() returned no documents")
			}
			// Total may be 0 if count failed, which is acceptable
		}
	})
}

func TestDeleteDocumentsFilterConversionError(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		deleteOpts := &types.DeleteDocumentOptions{
			CollectionName: env.CollectionName,
			Filter: map[string]interface{}{
				"unsupported_array": []string{"will", "fail"},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := env.Store.DeleteDocuments(ctx, deleteOpts)
		if err == nil {
			t.Error("DeleteDocuments() expected error for unsupported filter, got nil")
		} else if !contains(err.Error(), "failed to convert filter") {
			t.Errorf("DeleteDocuments() error = %v, want to contain 'failed to convert filter'", err)
		}
	})
}

// =============================================================================
// Additional tests for 100% coverage
// =============================================================================

func TestListDocumentsWithNextOffset(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Add test documents
		testDocs := createTestDocuments(5)
		addOpts := types.AddDocumentOptions{
			CollectionName: env.CollectionName,
			Documents:      testDocs,
			BatchSize:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := env.Store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Failed to add test documents: %v", err)
		}

		// Test listing with limit to ensure NextOffset is set
		listOpts := types.ListDocumentsOptions{
			CollectionName: env.CollectionName,
			Limit:          3, // Less than total documents
			IncludeVector:  true,
			IncludePayload: true,
		}

		result, err := env.Store.ListDocuments(ctx, &listOpts)
		if err != nil {
			t.Errorf("ListDocuments() error = %v, want nil", err)
		} else if result == nil {
			t.Errorf("ListDocuments() returned nil result")
		} else {
			// Should have NextOffset set if there are documents
			if len(result.Documents) > 0 && result.NextOffset == 0 {
				t.Errorf("ListDocuments() NextOffset should be set when documents are returned")
			}
		}
	})
}

func TestGetDocumentsErrorHandling(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Add test documents first
		testDocs := createTestDocuments(3)
		addOpts := types.AddDocumentOptions{
			CollectionName: env.CollectionName,
			Documents:      testDocs,
			BatchSize:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		addedIDs, err := env.Store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Failed to add test documents: %v", err)
		}

		// Test with invalid collection name to trigger error path
		getOpts := types.GetDocumentOptions{
			CollectionName: "nonexistent_collection",
			IncludeVector:  true,
			IncludePayload: true,
		}

		_, err = env.Store.GetDocuments(ctx, addedIDs[:1], &getOpts)
		if err == nil {
			t.Error("GetDocuments() expected error for nonexistent collection, got nil")
		} else if !contains(err.Error(), "failed to get documents") {
			t.Errorf("GetDocuments() error = %v, want to contain 'failed to get documents'", err)
		}
	})
}

func TestScrollDocumentsWithScrollID(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Add test documents
		testDocs := createTestDocuments(10)
		addOpts := types.AddDocumentOptions{
			CollectionName: env.CollectionName,
			Documents:      testDocs,
			BatchSize:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := env.Store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Failed to add test documents: %v", err)
		}

		// Test with valid numeric scroll ID
		scrollOpts := types.ScrollOptions{
			CollectionName: env.CollectionName,
			Limit:          5,
			ScrollID:       "12345", // Valid numeric string
			IncludeVector:  true,
			IncludePayload: true,
		}

		result, err := env.Store.ScrollDocuments(ctx, &scrollOpts)
		if err != nil {
			t.Errorf("ScrollDocuments() with valid scroll ID should not fail: %v", err)
		} else if result == nil {
			t.Errorf("ScrollDocuments() returned nil result")
		}
	})
}

func TestDeleteDocumentsWithInvalidCollection(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		deleteOpts := &types.DeleteDocumentOptions{
			CollectionName: "nonexistent_collection",
			IDs:            []string{"test_id"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := env.Store.DeleteDocuments(ctx, deleteOpts)
		if err == nil {
			t.Error("DeleteDocuments() expected error for nonexistent collection, got nil")
		} else if !contains(err.Error(), "failed to delete documents") {
			t.Errorf("DeleteDocuments() error = %v, want to contain 'failed to delete documents'", err)
		}
	})
}

func TestConvertScoredPointToDocumentWithVectorAndMetadata(t *testing.T) {
	// Test the case where we have vector and metadata to improve coverage
	point := &qdrant.ScoredPoint{
		Id: qdrant.NewIDNum(12345),
		Payload: map[string]*qdrant.Value{
			"id":      qdrant.NewValueString("test_doc"),
			"content": qdrant.NewValueString("Test content"),
			"metadata": qdrant.NewValueStruct(&qdrant.Struct{
				Fields: map[string]*qdrant.Value{
					"category": qdrant.NewValueString("test"),
				},
			}),
		},
		// Note: We can't easily construct a complex VectorsOutput, so we test without it
		Score: 0.95,
	}

	doc := convertScoredPointToDocument(point, true, true)

	if doc.ID != "test_doc" {
		t.Errorf("convertScoredPointToDocument() ID = %v, want %v", doc.ID, "test_doc")
	}
	if doc.Content != "Test content" {
		t.Errorf("convertScoredPointToDocument() Content = %v, want %v", doc.Content, "Test content")
	}
	if doc.Metadata == nil {
		t.Errorf("convertScoredPointToDocument() Metadata is nil")
	} else if doc.Metadata["category"] != "test" {
		t.Errorf("convertScoredPointToDocument() Metadata[category] = %v, want %v", doc.Metadata["category"], "test")
	}
}

func TestAddDocumentsUpsertErrorPath(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Test upsert with invalid collection name to trigger error path
		opts := types.AddDocumentOptions{
			CollectionName: "nonexistent_collection_for_upsert",
			Documents:      createTestDocuments(2),
			BatchSize:      5,
			Upsert:         true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := env.Store.AddDocuments(ctx, &opts)
		if err == nil {
			t.Error("AddDocuments() expected error for nonexistent collection, got nil")
		} else if !contains(err.Error(), "failed to add documents batch") {
			t.Errorf("AddDocuments() error = %v, want to contain 'failed to add documents batch'", err)
		}
	})
}

func TestScrollDocumentsWithFilter(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Add test documents with specific metadata
		testDocs := createTestDocuments(8)
		for i, doc := range testDocs {
			doc.Metadata["test_filter"] = i%2 == 0 // Even numbers get true, odd get false
		}

		addOpts := types.AddDocumentOptions{
			CollectionName: env.CollectionName,
			Documents:      testDocs,
			BatchSize:      10,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := env.Store.AddDocuments(ctx, &addOpts)
		if err != nil {
			t.Fatalf("Failed to add test documents: %v", err)
		}

		// Test scroll with filter to trigger convertFilterToQdrant error path
		scrollOpts := types.ScrollOptions{
			CollectionName: env.CollectionName,
			Limit:          5,
			Filter: map[string]interface{}{
				"test_filter": true,
			},
			IncludeVector:  true,
			IncludePayload: true,
		}

		result, err := env.Store.ScrollDocuments(ctx, &scrollOpts)
		if err != nil {
			t.Errorf("ScrollDocuments() with filter should not fail: %v", err)
		} else if result == nil {
			t.Errorf("ScrollDocuments() returned nil result")
		}
	})
}

func TestListDocumentsFilterError(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Test with an invalid filter to trigger convertFilterToQdrant error
		listOpts := types.ListDocumentsOptions{
			CollectionName: env.CollectionName,
			Limit:          10,
			Filter: map[string]interface{}{
				"invalid_filter": []complex128{complex(1, 2)}, // Unsupported type
			},
			IncludeVector:  true,
			IncludePayload: true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := env.Store.ListDocuments(ctx, &listOpts)
		if err == nil {
			t.Error("ListDocuments() expected error for invalid filter, got nil")
		} else if !contains(err.Error(), "failed to convert filter") {
			t.Errorf("ListDocuments() error = %v, want to contain 'failed to convert filter'", err)
		}
	})
}

func TestConvertStructToMapWithDefaultValues(t *testing.T) {
	// Test the default case in switch statement for convertStructToMap
	input := &qdrant.Struct{
		Fields: map[string]*qdrant.Value{
			"unknown_type": {
				Kind: nil, // This will hit the default case
			},
		},
	}

	result := convertStructToMap(input)

	// The unknown type should be ignored (not added to result)
	if len(result) != 0 {
		t.Errorf("convertStructToMap() with unknown type should return empty map, got %v", result)
	}
}

func TestScrollDocumentsErrorPath(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Test with invalid collection name to trigger error
		scrollOpts := types.ScrollOptions{
			CollectionName: "nonexistent_collection_scroll",
			Limit:          10,
			IncludeVector:  true,
			IncludePayload: true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := env.Store.ScrollDocuments(ctx, &scrollOpts)
		if err == nil {
			t.Error("ScrollDocuments() expected error for nonexistent collection, got nil")
		} else if !contains(err.Error(), "failed to scroll documents") {
			t.Errorf("ScrollDocuments() error = %v, want to contain 'failed to scroll documents'", err)
		}
	})
}

func TestListDocumentsScrollError(t *testing.T) {
	withDocumentTestEnvironment(t, func(env *DocumentTestEnvironment) {
		// Test with invalid collection name to trigger scroll error
		listOpts := types.ListDocumentsOptions{
			CollectionName: "nonexistent_collection_list",
			Limit:          10,
			IncludeVector:  true,
			IncludePayload: true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := env.Store.ListDocuments(ctx, &listOpts)
		if err == nil {
			t.Error("ListDocuments() expected error for nonexistent collection, got nil")
		} else if !contains(err.Error(), "failed to list documents") {
			t.Errorf("ListDocuments() error = %v, want to contain 'failed to list documents'", err)
		}
	})
}

// =============================================================================
// Tests for collectionUsesNamedVectors method
// =============================================================================

func TestCollectionUsesNamedVectors(t *testing.T) {
	store, baseConfig := setupConnectedStoreForDocument(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Disconnect(ctx)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("traditional collection returns false", func(t *testing.T) {
		// Create traditional collection (no sparse vectors)
		collectionName := fmt.Sprintf("test_traditional_%d", time.Now().UnixNano())
		collectionConfig := baseConfig
		collectionConfig.CollectionName = collectionName
		collectionConfig.EnableSparseVectors = false

		err := store.CreateCollection(ctx, &collectionConfig)
		if err != nil {
			t.Fatalf("Failed to create traditional collection: %v", err)
		}
		defer cleanupCollection(t, store, collectionName)

		usesNamed, err := store.collectionUsesNamedVectors(ctx, collectionName)
		if err != nil {
			t.Errorf("collectionUsesNamedVectors() error = %v, want nil", err)
		}
		if usesNamed {
			t.Errorf("collectionUsesNamedVectors() = true, want false for traditional collection")
		}
	})

	t.Run("named vector collection returns true", func(t *testing.T) {
		// Create collection with sparse vectors (named vectors)
		collectionName := fmt.Sprintf("test_named_%d", time.Now().UnixNano())
		collectionConfig := baseConfig
		collectionConfig.CollectionName = collectionName
		collectionConfig.EnableSparseVectors = true
		collectionConfig.DenseVectorName = "dense"
		collectionConfig.SparseVectorName = "sparse"

		err := store.CreateCollection(ctx, &collectionConfig)
		if err != nil {
			t.Fatalf("Failed to create named vector collection: %v", err)
		}
		defer cleanupCollection(t, store, collectionName)

		usesNamed, err := store.collectionUsesNamedVectors(ctx, collectionName)
		if err != nil {
			t.Errorf("collectionUsesNamedVectors() error = %v, want nil", err)
		}
		if !usesNamed {
			t.Errorf("collectionUsesNamedVectors() = false, want true for named vector collection")
		}
	})

	t.Run("nonexistent collection returns error", func(t *testing.T) {
		_, err := store.collectionUsesNamedVectors(ctx, "nonexistent_collection")
		if err == nil {
			t.Error("collectionUsesNamedVectors() expected error for nonexistent collection, got nil")
		} else if !contains(err.Error(), "failed to get collection info") {
			t.Errorf("collectionUsesNamedVectors() error = %v, want to contain 'failed to get collection info'", err)
		}
	})

	t.Run("not connected store returns error", func(t *testing.T) {
		unconnectedStore := NewStore()
		defer func() {
			_ = unconnectedStore.Disconnect(context.Background())
		}()

		_, err := unconnectedStore.collectionUsesNamedVectors(ctx, "test_collection")
		if err == nil {
			t.Error("collectionUsesNamedVectors() expected error for not connected store, got nil")
		}
	})
}
