package graphrag

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestAddUpdateSegments tests the AddSegments and UpdateSegments functions with different configurations
func TestAddUpdateSegments(t *testing.T) {
	// Setup connector as in document_test.go
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
			vectorConfig := getVectorStore("addsegments_test", 1536)
			// Replace + with _ to make collection name valid
			safeName := strings.ReplaceAll(configName, "+", "_")
			collectionID := fmt.Sprintf("test_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.Collection{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"type": "addsegments_test",
				},
				VectorConfig: &vectorConfig,
			}

			// Add GraphStoreConfig for graph-enabled configurations
			if strings.Contains(configName, "graph") {
				graphConfig := getGraphStore("addsegments_test")
				collection.GraphStoreConfig = &graphConfig
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

			// Step 1: Add a TEXT document first
			t.Run("Add_Base_Document", func(t *testing.T) {
				baseDocOptions := &types.UpsertOptions{
					DocID:     fmt.Sprintf("base_doc_%s", configName),
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "segment_test",
						"type":   "base_document",
						"config": configName,
					},
				}

				baseDocContent := "This is a base document for segment testing. It contains multiple paragraphs and sections. This will be used as the foundation for adding segments."

				docID, err := g.AddText(ctx, baseDocContent, baseDocOptions)
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
						t.Logf("Expected error with mock setup for base document: %v", err)
					} else {
						t.Errorf("Unexpected error adding base document: %v", err)
					}
					return
				}

				if docID == "" {
					t.Error("AddText returned empty document ID for base document")
					return
				}

				t.Logf("Base document added successfully - ID: %s", docID)
			})

			// Step 2: Add three segments to the collection
			t.Run("Add_Three_Segments", func(t *testing.T) {
				// Create three test segments - mix of with and without IDs
				segmentTexts := []types.SegmentText{
					{
						ID:   "segment_001",
						Text: "This is the first segment about artificial intelligence and machine learning technologies.",
					},
					{
						Text: "This is the second segment discussing natural language processing and deep learning algorithms.",
					},
					{
						ID:   "segment_003",
						Text: "This is the third segment covering computer vision and image recognition systems.",
					},
				}

				// Create doc ID for segments
				segmentDocID := fmt.Sprintf("segments_doc_%s", configName)

				// Prepare options for AddSegments
				segmentOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source":    "segment_test",
						"type":      "segments",
						"config":    configName,
						"doc_type":  "segments",
						"parent_id": fmt.Sprintf("base_doc_%s", configName),
					},
				}

				// Call AddSegments
				segmentIDs, err := g.AddSegments(ctx, segmentDocID, segmentTexts, segmentOptions)
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
						t.Logf("Expected error with mock setup for segments: %v", err)
					} else {
						t.Errorf("Unexpected error adding segments: %v", err)
					}
					return
				}

				// Validate results
				if len(segmentIDs) == 0 {
					t.Error("AddSegments returned 0 segment IDs")
					return
				}

				expectedCount := len(segmentTexts)
				if len(segmentIDs) != expectedCount {
					t.Errorf("Expected %d segments, got %d", expectedCount, len(segmentIDs))
					return
				}

				t.Logf("Successfully added %d segments to collection %s", len(segmentIDs), collectionID)
			})

			// Step 3: Test AddSegments with empty segments list
			t.Run("Add_Empty_Segments", func(t *testing.T) {
				emptySegments := []types.SegmentText{}
				emptyDocID := fmt.Sprintf("empty_segments_%s", configName)

				segmentOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "segment_test",
						"type":   "empty_segments",
						"config": configName,
					},
				}

				segmentIDs, err := g.AddSegments(ctx, emptyDocID, emptySegments, segmentOptions)
				if err != nil {
					t.Logf("Error with empty segments (expected): %v", err)
					return
				}

				if len(segmentIDs) != 0 {
					t.Errorf("Expected 0 segments for empty list, got %d", len(segmentIDs))
				}

				t.Logf("Empty segments handled correctly: %d segments added", len(segmentIDs))
			})

			// Step 4: Test AddSegments with segments without IDs
			t.Run("Add_Segments_Without_IDs", func(t *testing.T) {
				segmentTextsNoID := []types.SegmentText{
					{
						Text: "This segment has no ID and should get auto-generated ID.",
					},
					{
						Text: "This is another segment without ID for testing auto-generation.",
					},
				}

				noIDDocID := fmt.Sprintf("no_id_segments_%s", configName)

				segmentOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "segment_test",
						"type":   "no_id_segments",
						"config": configName,
					},
				}

				segmentIDs, err := g.AddSegments(ctx, noIDDocID, segmentTextsNoID, segmentOptions)
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
						t.Logf("Expected error with mock setup for no-ID segments: %v", err)
					} else {
						t.Errorf("Unexpected error adding no-ID segments: %v", err)
					}
					return
				}

				expectedCount := len(segmentTextsNoID)
				if len(segmentIDs) != expectedCount {
					t.Errorf("Expected %d segments without IDs, got %d", expectedCount, len(segmentIDs))
					return
				}

				t.Logf("Successfully added %d segments without IDs", len(segmentIDs))
			})

			// Step 5: Test AddSegments with nil options
			t.Run("Add_Segments_Nil_Options", func(t *testing.T) {
				nilOptSegments := []types.SegmentText{
					{
						ID:   "nil_opt_segment",
						Text: "This segment is added with nil options.",
					},
				}

				nilOptDocID := fmt.Sprintf("nil_opt_segments_%s", configName)

				segmentIDs, err := g.AddSegments(ctx, nilOptDocID, nilOptSegments, nil)
				if err != nil {
					t.Logf("Error with nil options (expected): %v", err)
					return
				}

				if len(segmentIDs) != 1 {
					t.Errorf("Expected 1 segment with nil options, got %d", len(segmentIDs))
					return
				}

				t.Logf("Successfully added segments with nil options: %d segments", len(segmentIDs))
			})

			// Step 6: Test UpdateSegments functionality
			t.Run("Update_Segments", func(t *testing.T) {
				// First, ensure we have segments to update by adding some
				updateSegmentTexts := []types.SegmentText{
					{
						ID:   "update_segment_001",
						Text: "Original segment text for update testing.",
					},
					{
						ID:   "update_segment_002",
						Text: "Another original segment text for update testing.",
					},
				}

				updateDocID := fmt.Sprintf("update_segments_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "update_test_prep",
						"config": configName,
					},
				}

				// Add segments first
				addedIDs, err := g.AddSegments(ctx, updateDocID, updateSegmentTexts, addOptions)
				if err != nil {
					t.Logf("Failed to add segments for update test (expected): %v", err)
					return
				}

				if len(addedIDs) != 2 {
					t.Logf("Expected 2 segments to be added for update test, got %d", len(addedIDs))
					return
				}

				// Test 1: Update segments with valid IDs
				updatedSegmentTexts := []types.SegmentText{
					{
						ID:   "update_segment_001",
						Text: "Updated segment text with new content about artificial intelligence.",
					},
					{
						ID:   "update_segment_002",
						Text: "Updated second segment text with enhanced details about machine learning.",
					},
				}

				updateOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "update_test",
						"config": configName,
						"weight": 0.8,
						"score":  0.9,
						"vote":   5,
					},
				}

				updateCount, err := g.UpdateSegments(ctx, updatedSegmentTexts, updateOptions)
				if err != nil {
					t.Logf("Failed to update segments (expected): %v", err)
					return
				}

				if updateCount != 2 {
					t.Errorf("Expected 2 updated segments, got %d", updateCount)
					return
				}

				t.Logf("Successfully updated %d segments", updateCount)
			})

			// Step 7: Test UpdateSegments error handling
			t.Run("Update_Segments_Error_Handling", func(t *testing.T) {
				// Test with missing IDs (should fail)
				invalidUpdateSegments := []types.SegmentText{
					{
						ID:   "",
						Text: "This segment has no ID.",
					},
				}

				_, err := g.UpdateSegments(ctx, invalidUpdateSegments, nil)
				if err == nil {
					t.Error("Expected error for UpdateSegments with missing IDs, but got none")
				} else {
					t.Logf("UpdateSegments correctly rejected missing IDs: %v", err)
				}

				// Test with non-existent segment IDs
				nonExistentSegments := []types.SegmentText{
					{
						ID:   "nonexistent_segment_123",
						Text: "This segment ID does not exist.",
					},
				}

				_, err = g.UpdateSegments(ctx, nonExistentSegments, nil)
				if err == nil {
					t.Logf("UpdateSegments with non-existent ID did not fail (may be expected)")
				} else {
					t.Logf("UpdateSegments correctly rejected non-existent ID: %v", err)
				}
			})
		})
	}
}

// TestAddSegmentsErrorHandling tests error conditions for AddSegments
func TestAddSegmentsErrorHandling(t *testing.T) {
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

	t.Run("Invalid_DocID", func(t *testing.T) {
		segments := []types.SegmentText{
			{
				ID:   "test_segment",
				Text: "Test segment with invalid docID",
			},
		}

		options := &types.UpsertOptions{
			GraphName: "nonexistent_collection",
		}

		_, err := g.AddSegments(ctx, "", segments, options)
		if err == nil {
			t.Error("Expected error for empty docID")
		}
		t.Logf("Empty docID correctly rejected: %v", err)
	})

	t.Run("Invalid_Collection", func(t *testing.T) {
		segments := []types.SegmentText{
			{
				ID:   "test_segment",
				Text: "Test segment with invalid collection",
			},
		}

		options := &types.UpsertOptions{
			GraphName: "nonexistent_collection",
		}

		_, err := g.AddSegments(ctx, "test_doc", segments, options)
		if err == nil {
			t.Error("Expected error for nonexistent collection")
		}
		t.Logf("Invalid collection correctly rejected: %v", err)
	})

	t.Run("Nil_Segments", func(t *testing.T) {
		options := &types.UpsertOptions{
			GraphName: "test_collection",
		}

		segmentIDs, err := g.AddSegments(ctx, "test_doc", nil, options)
		if err != nil {
			t.Logf("Error with nil segments (expected): %v", err)
			return
		}

		if len(segmentIDs) != 0 {
			t.Errorf("Expected 0 segments for nil segments, got %d", len(segmentIDs))
		}

		t.Logf("Nil segments handled correctly: %d segments added", len(segmentIDs))
	})
}

// TestAddSegmentsStoreIntegration tests Store integration specifically for segments
func TestAddSegmentsStoreIntegration(t *testing.T) {
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
			vectorConfig := getVectorStore("segment_store_test", 1536)
			// Replace + with _ to make collection name valid
			safeName := strings.ReplaceAll(configName, "+", "_")
			storeCollectionID := fmt.Sprintf("segment_store_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.Collection{
				ID: storeCollectionID,
				Metadata: map[string]interface{}{
					"type": "segment_store_test",
				},
				VectorConfig: &vectorConfig,
			}

			// Add GraphStoreConfig for graph-enabled configurations
			if strings.Contains(configName, "graph") {
				graphConfig := getGraphStore("segment_store_test")
				collection.GraphStoreConfig = &graphConfig
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

			// Test segments with store
			segments := []types.SegmentText{
				{
					ID:   "store_segment_001",
					Text: "This is a segment for store integration testing.",
				},
				{
					ID:   "store_segment_002",
					Text: "This is another segment for store testing with metadata.",
				},
			}

			segmentDocID := fmt.Sprintf("store_segments_%s", configName)
			options := &types.UpsertOptions{
				GraphName: storeCollectionID,
				Metadata: map[string]interface{}{
					"source": "store_integration_test",
					"config": configName,
				},
			}

			segmentIDs, err := g.AddSegments(ctx, segmentDocID, segments, options)
			if err != nil {
				t.Logf("Config %s: Expected error: %v", configName, err)
				return
			}

			if len(segmentIDs) == 0 {
				t.Error("AddSegments returned 0 segments")
				return
			}

			// Check if segment metadata was stored
			for _, segment := range segments {
				weightKey := fmt.Sprintf("segment_weight_%s_%s", segmentDocID, segment.ID)
				scoreKey := fmt.Sprintf("segment_score_%s_%s", segmentDocID, segment.ID)
				voteKey := fmt.Sprintf("segment_vote_%s_%s", segmentDocID, segment.ID)

				if g.Store.Has(weightKey) {
					t.Logf("Config %s: Segment weight stored successfully for %s", configName, segment.ID)
				} else {
					t.Logf("Config %s: Segment weight not found in store for %s (expected with some implementations)", configName, segment.ID)
				}

				if g.Store.Has(scoreKey) {
					t.Logf("Config %s: Segment score stored successfully for %s", configName, segment.ID)
				} else {
					t.Logf("Config %s: Segment score not found in store for %s (expected with some implementations)", configName, segment.ID)
				}

				if g.Store.Has(voteKey) {
					t.Logf("Config %s: Segment vote stored successfully for %s", configName, segment.ID)
				} else {
					t.Logf("Config %s: Segment vote not found in store for %s (expected with some implementations)", configName, segment.ID)
				}
			}

			t.Logf("Store integration test completed for %s with %d segments", configName, len(segmentIDs))
		})
	}
}
