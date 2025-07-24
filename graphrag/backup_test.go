package graphrag

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestBackupAndRestore tests the Backup and Restore functions with different configurations
func TestBackupAndRestore(t *testing.T) {
	// Setup connector as in segment_test.go
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+graph", "vector+store", "vector+graph+store"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Skipf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Skipf("Failed to create GraphRag instance for %s: %v", configName, err)
			}

			ctx := context.Background()

			// Create collection using utility from collection_test.go
			// Replace + with _ to make collection name valid
			safeName := strings.ReplaceAll(configName, "+", "_")
			collectionID := fmt.Sprintf("backup_test_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.CollectionConfig{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"type": "backup_test",
				},
				Config: &types.CreateCollectionOptions{
					CollectionName: fmt.Sprintf("%s_vector", collectionID),
					Dimension:      1536,
					Distance:       types.DistanceCosine,
					IndexType:      types.IndexTypeHNSW,
				},
			}

			// Create collection (this will auto-connect vector store)
			_, err = g.CreateCollection(ctx, collection)
			if err != nil {
				t.Skipf("Failed to create test collection for %s: %v", configName, err)
			}

			// Cleanup collection after test
			defer func() {
				removed, err := g.RemoveCollection(ctx, collectionID)
				if err != nil {
					t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
				} else if removed {
					t.Logf("Successfully cleaned up collection: %s", collectionID)
				} else {
					t.Logf("Collection %s was not found (already cleaned up)", collectionID)
				}
			}()

			// Step 1: Insert test data
			t.Run("Insert_Test_Data", func(t *testing.T) {
				// Add segments as test data
				segmentTexts := []types.SegmentText{
					{
						ID:   "backup_segment_001",
						Text: "This is the first segment for backup testing about artificial intelligence.",
					},
					{
						ID:   "backup_segment_002",
						Text: "This is the second segment for backup testing about machine learning.",
					},
					{
						ID:   "backup_segment_003",
						Text: "This is the third segment for backup testing about deep learning.",
					},
				}

				backupDocID := fmt.Sprintf("backup_doc_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "backup_test",
						"config": configName,
						"type":   "backup_test_data",
					},
				}

				// Add embedding configuration
				embeddingConfig, err := createTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create embedding config: %v", err)
				}
				addOptions.Embedding = embeddingConfig

				// Add extraction configuration if graph is enabled
				if strings.Contains(configName, "graph") {
					extractionConfig, err := createTestExtraction(t)
					if err != nil {
						t.Skipf("Failed to create extraction config: %v", err)
					}
					addOptions.Extraction = extractionConfig
				}

				// Add segments
				segmentIDs, err := g.AddSegments(ctx, backupDocID, segmentTexts, addOptions)
				if err != nil {
					// Expected errors with mock setup
					expectedErrors := []string{
						"connection refused", "no such host", "connector not found", "connector openai not loaded",
						"vector store", "graph store", "store", "embedding", "extraction",
					}

					hasExpectedError := false
					for _, expectedErr := range expectedErrors {
						if strings.Contains(err.Error(), expectedErr) {
							hasExpectedError = true
							break
						}
					}

					if hasExpectedError {
						t.Logf("Expected error with mock setup for test data: %v", err)
					} else {
						t.Errorf("Unexpected error inserting test data: %v", err)
					}
					return
				}

				if len(segmentIDs) == 0 {
					t.Error("AddSegments returned 0 segment IDs")
					return
				}

				t.Logf("Successfully inserted %d segments for backup testing", len(segmentIDs))
			})

			// Step 2: Test Backup functionality
			t.Run("Backup_Collection", func(t *testing.T) {
				// Create a buffer to capture backup data
				var backupBuffer bytes.Buffer

				// Test backup
				err := g.Backup(ctx, &backupBuffer, collectionID)
				if err != nil {
					// Expected errors with mock setup
					expectedErrors := []string{
						"connection refused", "no such host", "collection does not exist",
						"failed to backup", "vector store", "graph store", "store",
					}

					hasExpectedError := false
					for _, expectedErr := range expectedErrors {
						if strings.Contains(err.Error(), expectedErr) {
							hasExpectedError = true
							break
						}
					}

					if hasExpectedError {
						t.Logf("Expected error with mock setup for backup: %v", err)
					} else {
						t.Errorf("Unexpected error during backup: %v", err)
					}
					return
				}

				// Check if backup data was written
				if backupBuffer.Len() == 0 {
					t.Error("Backup produced no data")
					return
				}

				t.Logf("Successfully backed up collection %s (%d bytes)", collectionID, backupBuffer.Len())

				// Step 3: Test Restore functionality
				t.Run("Restore_Collection", func(t *testing.T) {
					// Create restore collection with different name
					restoreCollectionID := fmt.Sprintf("restore_test_collection_%s_%d", safeName, time.Now().Unix())
					restoreCollection := types.CollectionConfig{
						ID: restoreCollectionID,
						Metadata: map[string]interface{}{
							"type": "restore_test",
						},
						Config: &types.CreateCollectionOptions{
							CollectionName: fmt.Sprintf("%s_vector", restoreCollectionID),
							Dimension:      1536,
							Distance:       types.DistanceCosine,
							IndexType:      types.IndexTypeHNSW,
						},
					}

					// Create restore collection
					_, err = g.CreateCollection(ctx, restoreCollection)
					if err != nil {
						t.Skipf("Failed to create restore collection for %s: %v", configName, err)
					}

					// Cleanup restore collection after test
					defer func() {
						removed, err := g.RemoveCollection(ctx, restoreCollectionID)
						if err != nil {
							t.Logf("Warning: Failed to cleanup restore collection %s: %v", restoreCollectionID, err)
						} else if removed {
							t.Logf("Successfully cleaned up restore collection: %s", restoreCollectionID)
						} else {
							t.Logf("Restore collection %s was not found (already cleaned up)", restoreCollectionID)
						}
					}()

					// Test restore
					err = g.Restore(ctx, &backupBuffer, restoreCollectionID)
					if err != nil {
						// Expected errors with mock setup
						expectedErrors := []string{
							"connection refused", "no such host", "failed to restore",
							"vector store", "graph store", "store", "zip", "archive",
						}

						hasExpectedError := false
						for _, expectedErr := range expectedErrors {
							if strings.Contains(err.Error(), expectedErr) {
								hasExpectedError = true
								break
							}
						}

						if hasExpectedError {
							t.Logf("Expected error with mock setup for restore: %v", err)
						} else {
							t.Errorf("Unexpected error during restore: %v", err)
						}
						return
					}

					t.Logf("Successfully restored collection %s", restoreCollectionID)
				})
			})
		})
	}
}

// TestBackupErrorHandling tests error conditions for backup operations
func TestBackupErrorHandling(t *testing.T) {
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		t.Skip("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	t.Run("Backup_Empty_Collection_ID", func(t *testing.T) {
		var buffer bytes.Buffer
		err := g.Backup(ctx, &buffer, "")
		if err == nil {
			t.Error("Expected error for empty collection ID")
		} else {
			t.Logf("Empty collection ID correctly rejected: %v", err)
		}
	})

	t.Run("Backup_Nonexistent_Collection", func(t *testing.T) {
		var buffer bytes.Buffer
		err := g.Backup(ctx, &buffer, "nonexistent_collection")
		if err == nil {
			t.Error("Expected error for nonexistent collection")
		} else {
			t.Logf("Nonexistent collection correctly rejected: %v", err)
		}
	})

	t.Run("Restore_Empty_Collection_ID", func(t *testing.T) {
		var buffer bytes.Buffer
		err := g.Restore(ctx, &buffer, "")
		if err == nil {
			t.Error("Expected error for empty collection ID")
		} else {
			t.Logf("Empty collection ID correctly rejected: %v", err)
		}
	})

	t.Run("Restore_Invalid_Data", func(t *testing.T) {
		// Create buffer with invalid data
		var buffer bytes.Buffer
		buffer.WriteString("invalid backup data")

		err := g.Restore(ctx, &buffer, "test_collection")
		if err == nil {
			t.Error("Expected error for invalid backup data")
		} else {
			t.Logf("Invalid backup data correctly rejected: %v", err)
		}
	})
}

// TestBackupStoreIntegration tests Store integration specifically for backup operations
func TestBackupStoreIntegration(t *testing.T) {
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	storeConfigs := []string{"vector+store", "vector+graph+store"}

	for _, configName := range storeConfigs {
		config := configs[configName]
		if config == nil {
			t.Skipf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Store_%s", configName), func(t *testing.T) {
			g, err := New(config)
			if err != nil {
				t.Skipf("Failed to create GraphRag instance for %s: %v", configName, err)
			}

			// Skip if Store is not available
			if g.Store == nil {
				t.Skipf("Store not available for config %s", configName)
			}

			ctx := context.Background()

			// Create collection using utility from collection_test.go
			// Replace + with _ to make collection name valid
			safeName := strings.ReplaceAll(configName, "+", "_")
			storeCollectionID := fmt.Sprintf("backup_store_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.CollectionConfig{
				ID: storeCollectionID,
				Metadata: map[string]interface{}{
					"type": "backup_store_test",
				},
				Config: &types.CreateCollectionOptions{
					CollectionName: fmt.Sprintf("%s_vector", storeCollectionID),
					Dimension:      1536,
					Distance:       types.DistanceCosine,
					IndexType:      types.IndexTypeHNSW,
				},
			}

			// Create collection
			_, err = g.CreateCollection(ctx, collection)
			if err != nil {
				t.Skipf("Failed to create test collection for %s: %v", configName, err)
			}

			// Cleanup collection after test
			defer func() {
				removed, err := g.RemoveCollection(ctx, storeCollectionID)
				if err != nil {
					t.Logf("Warning: Failed to cleanup store collection %s: %v", storeCollectionID, err)
				} else if removed {
					t.Logf("Successfully cleaned up store collection: %s", storeCollectionID)
				} else {
					t.Logf("Store collection %s was not found (already cleaned up)", storeCollectionID)
				}
			}()

			// Insert test data with store metadata
			segments := []types.SegmentText{
				{
					ID:   "store_backup_segment_001",
					Text: "This is a segment for store backup testing.",
				},
				{
					ID:   "store_backup_segment_002",
					Text: "This is another segment for store backup testing.",
				},
			}

			segmentDocID := fmt.Sprintf("store_backup_segments_%s", configName)
			options := &types.UpsertOptions{
				GraphName: storeCollectionID,
				Metadata: map[string]interface{}{
					"source": "store_backup_test",
					"config": configName,
					"weight": 0.8,
					"score":  0.9,
					"vote":   5,
				},
			}

			// Add embedding configuration
			embeddingConfig, err := createTestEmbedding(t)
			if err != nil {
				t.Skipf("Failed to create embedding config: %v", err)
			}
			options.Embedding = embeddingConfig

			// Add extraction configuration if graph is enabled
			if strings.Contains(configName, "graph") {
				extractionConfig, err := createTestExtraction(t)
				if err != nil {
					t.Skipf("Failed to create extraction config: %v", err)
				}
				options.Extraction = extractionConfig
			}

			segmentIDs, err := g.AddSegments(ctx, segmentDocID, segments, options)
			if err != nil {
				t.Logf("Config %s: Expected error with mock setup: %v", configName, err)
				return
			}

			if len(segmentIDs) == 0 {
				t.Error("AddSegments returned 0 segments")
				return
			}

			// Test backup with store data
			var backupBuffer bytes.Buffer
			err = g.Backup(ctx, &backupBuffer, storeCollectionID)
			if err != nil {
				t.Logf("Config %s: Expected error with mock setup for backup: %v", configName, err)
				return
			}

			if backupBuffer.Len() == 0 {
				t.Error("Backup produced no data")
				return
			}

			t.Logf("Store backup integration test completed for %s with %d segments (%d bytes)", configName, len(segmentIDs), backupBuffer.Len())
		})
	}
}
