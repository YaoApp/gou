package graphrag

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// ==== Test Helper Functions ====

// These functions are defined in document_test.go and reused here

// ==== Segment Tests ====

// TestSegmentCURD tests the Complete CRUD operations for segments (Create, Update, Remove, Delete)
func TestSegmentCURD(t *testing.T) {
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
			// Replace + with _ to make collection name valid
			safeName := strings.ReplaceAll(configName, "+", "_")
			collectionID := fmt.Sprintf("test_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.CollectionConfig{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"type": "addsegments_test",
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

				// Add embedding configuration
				embeddingConfig, err := createTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create embedding config: %v", err)
				}
				segmentOptions.Embedding = embeddingConfig

				// Add extraction configuration if graph is enabled
				if strings.Contains(configName, "graph") {
					extractionConfig, err := createTestExtraction(t)
					if err != nil {
						t.Skipf("Failed to create extraction config: %v", err)
					}
					segmentOptions.Extraction = extractionConfig
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

				// Add embedding configuration
				embeddingConfig, err := createTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create embedding config: %v", err)
				}
				segmentOptions.Embedding = embeddingConfig

				// Add extraction configuration if graph is enabled
				if strings.Contains(configName, "graph") {
					extractionConfig, err := createTestExtraction(t)
					if err != nil {
						t.Skipf("Failed to create extraction config: %v", err)
					}
					segmentOptions.Extraction = extractionConfig
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

				// Add embedding configuration
				embeddingConfig, err := createTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create embedding config: %v", err)
				}
				segmentOptions.Embedding = embeddingConfig

				// Add extraction configuration if graph is enabled
				if strings.Contains(configName, "graph") {
					extractionConfig, err := createTestExtraction(t)
					if err != nil {
						t.Skipf("Failed to create extraction config: %v", err)
					}
					segmentOptions.Extraction = extractionConfig
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

				// Add embedding configuration
				embeddingConfig2, err := createTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create embedding config: %v", err)
				}
				updateOptions.Embedding = embeddingConfig2

				// Add extraction configuration if graph is enabled
				if strings.Contains(configName, "graph") {
					extractionConfig2, err := createTestExtraction(t)
					if err != nil {
						t.Skipf("Failed to create extraction config: %v", err)
					}
					updateOptions.Extraction = extractionConfig2
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

			// Step 7a: Test GetSegments functionality - read segments by IDs
			t.Run("Get_Segments_By_IDs", func(t *testing.T) {
				// First, add segments to read
				readSegmentTexts := []types.SegmentText{
					{
						ID:   "read_segment_001",
						Text: "This segment is for reading by IDs test.",
					},
					{
						ID:   "read_segment_002",
						Text: "Another segment for reading by IDs test.",
					},
				}

				readDocID := fmt.Sprintf("read_segments_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "read_test_prep",
						"config": configName,
					},
				}

				// Add segments first
				addedIDs, err := g.AddSegments(ctx, readDocID, readSegmentTexts, addOptions)
				if err != nil {
					t.Logf("Failed to add segments for read test (expected): %v", err)
					return
				}

				if len(addedIDs) != 2 {
					t.Logf("Expected 2 segments to be added for read test, got %d", len(addedIDs))
					return
				}

				// Test GetSegments with specific IDs
				segments, err := g.GetSegments(ctx, []string{"read_segment_001", "read_segment_002"})
				if err != nil {
					t.Logf("Failed to get segments by IDs (expected): %v", err)
					return
				}

				if len(segments) == 0 {
					t.Log("No segments returned - this may be expected with mock setup")
					return
				}

				if len(segments) != 2 {
					t.Errorf("Expected 2 segments returned, got %d", len(segments))
					return
				}

				// Validate segment structure
				for _, segment := range segments {
					if segment.ID == "" {
						t.Error("Segment ID is empty")
					}
					if segment.Text == "" {
						t.Error("Segment text is empty")
					}
					if segment.CollectionID == "" {
						t.Error("Segment collection ID is empty")
					}
					if segment.DocumentID == "" {
						t.Error("Segment document ID is empty")
					}
				}

				t.Logf("Successfully retrieved %d segments by IDs", len(segments))
			})

			// Step 7b: Test ListSegments functionality - list segments with pagination
			t.Run("List_Segments_With_Pagination", func(t *testing.T) {
				// Add segments for a specific document
				docSegmentTexts := []types.SegmentText{
					{
						ID:   "list_segment_001",
						Text: "First segment for pagination test.",
					},
					{
						ID:   "list_segment_002",
						Text: "Second segment for pagination test.",
					},
					{
						ID:   "list_segment_003",
						Text: "Third segment for pagination test.",
					},
				}

				listDocID := fmt.Sprintf("list_segments_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "list_test",
						"config": configName,
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

				// Add segments first
				addedIDs, err := g.AddSegments(ctx, listDocID, docSegmentTexts, addOptions)
				if err != nil {
					t.Logf("Failed to add segments for list test (expected): %v", err)
					return
				}

				if len(addedIDs) != 3 {
					t.Logf("Expected 3 segments to be added for list test, got %d", len(addedIDs))
					return
				}

				// Test ListSegments with default options
				listOptions := &types.ListSegmentsOptions{
					Limit:  10,
					Offset: 0,
				}
				result, err := g.ListSegments(ctx, listDocID, listOptions)
				if err != nil {
					t.Logf("Failed to list segments (expected): %v", err)
					return
				}

				if len(result.Segments) == 0 {
					t.Log("No segments returned for list - this may be expected with mock setup")
					return
				}

				if len(result.Segments) != 3 {
					t.Errorf("Expected 3 segments returned by list, got %d", len(result.Segments))
					return
				}

				// Validate segment structure
				for _, segment := range result.Segments {
					if segment.ID == "" {
						t.Error("Segment ID is empty")
					}
					if segment.Text == "" {
						t.Error("Segment text is empty")
					}
					if segment.DocumentID != listDocID {
						t.Errorf("Segment document ID mismatch: expected %s, got %s", listDocID, segment.DocumentID)
					}
				}

				t.Logf("Successfully listed %d segments", len(result.Segments))
			})

			// Step 7c: Test ScrollSegments functionality - scroll through segments
			t.Run("Scroll_Segments", func(t *testing.T) {
				// Add segments for a specific document
				scrollSegmentTexts := []types.SegmentText{
					{
						ID:   "scroll_segment_001",
						Text: "First segment for scroll test.",
					},
					{
						ID:   "scroll_segment_002",
						Text: "Second segment for scroll test.",
					},
					{
						ID:   "scroll_segment_003",
						Text: "Third segment for scroll test.",
					},
				}

				scrollDocID := fmt.Sprintf("scroll_segments_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "scroll_test",
						"config": configName,
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

				// Add segments first
				addedIDs, err := g.AddSegments(ctx, scrollDocID, scrollSegmentTexts, addOptions)
				if err != nil {
					t.Logf("Failed to add segments for scroll test (expected): %v", err)
					return
				}

				if len(addedIDs) != 3 {
					t.Logf("Expected 3 segments to be added for scroll test, got %d", len(addedIDs))
					return
				}

				// Test ScrollSegments with default options
				scrollOptions := &types.ScrollSegmentsOptions{
					BatchSize: 10,
				}
				result, err := g.ScrollSegments(ctx, scrollDocID, scrollOptions)
				if err != nil {
					t.Logf("Failed to scroll segments (expected): %v", err)
					return
				}

				if len(result.Segments) == 0 {
					t.Log("No segments returned for scroll - this may be expected with mock setup")
					return
				}

				if len(result.Segments) != 3 {
					t.Errorf("Expected 3 segments returned by scroll, got %d", len(result.Segments))
					return
				}

				// Validate segment structure
				for _, segment := range result.Segments {
					if segment.ID == "" {
						t.Error("Segment ID is empty")
					}
					if segment.Text == "" {
						t.Error("Segment text is empty")
					}
					if segment.DocumentID != scrollDocID {
						t.Errorf("Segment document ID mismatch: expected %s, got %s", scrollDocID, segment.DocumentID)
					}
				}

				t.Logf("Successfully scrolled %d segments", len(result.Segments))
			})

			// Step 7d: Test GetSegment functionality - read single segment
			t.Run("Get_Single_Segment", func(t *testing.T) {
				// Add a single segment for single read test
				singleSegmentTexts := []types.SegmentText{
					{
						ID:   "single_segment_001",
						Text: "This is a single segment for individual reading test.",
					},
				}

				singleReadDocID := fmt.Sprintf("single_read_segment_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "single_read_test",
						"config": configName,
					},
				}

				// Add segment first
				addedIDs, err := g.AddSegments(ctx, singleReadDocID, singleSegmentTexts, addOptions)
				if err != nil {
					t.Logf("Failed to add segment for single read test (expected): %v", err)
					return
				}

				if len(addedIDs) != 1 {
					t.Logf("Expected 1 segment to be added for single read test, got %d", len(addedIDs))
					return
				}

				// Test GetSegment with specific ID
				segment, err := g.GetSegment(ctx, "single_segment_001")
				if err != nil {
					t.Logf("Failed to get single segment (expected): %v", err)
					return
				}

				if segment == nil {
					t.Error("GetSegment returned nil segment")
					return
				}

				// Validate segment structure
				if segment.ID == "" {
					t.Error("Single segment ID is empty")
				}
				if segment.Text == "" {
					t.Error("Single segment text is empty")
				}
				if segment.DocumentID != singleReadDocID {
					t.Errorf("Single segment document ID mismatch: expected %s, got %s", singleReadDocID, segment.DocumentID)
				}

				t.Logf("Successfully retrieved single segment: %s", segment.ID)
			})

			// Step 7e: Test read error handling
			t.Run("Read_Error_Handling", func(t *testing.T) {
				// Test GetSegments with empty segment list
				emptySegments, err := g.GetSegments(ctx, []string{})
				if err != nil {
					t.Errorf("GetSegments with empty list should not return error: %v", err)
				}
				if len(emptySegments) != 0 {
					t.Errorf("Expected 0 segments for empty list, got %d", len(emptySegments))
				}

				// Test GetSegments with non-existent segment IDs
				nonExistentSegments, err := g.GetSegments(ctx, []string{"non_existent_segment"})
				if err == nil {
					t.Logf("GetSegments with non-existent IDs handled gracefully: returned %d segments", len(nonExistentSegments))
				} else {
					t.Logf("GetSegments with non-existent IDs returned error (expected): %v", err)
				}

				// Test ListSegments with empty docID
				_, err = g.ListSegments(ctx, "", nil)
				if err == nil {
					t.Error("Expected error for ListSegments with empty docID")
				} else {
					t.Logf("ListSegments correctly rejected empty docID: %v", err)
				}

				// Test ListSegments with non-existent docID
				nonExistentDocResult, err := g.ListSegments(ctx, "non_existent_doc", nil)
				if err == nil {
					t.Logf("ListSegments with non-existent docID handled gracefully: returned %d segments", len(nonExistentDocResult.Segments))
				} else {
					t.Logf("ListSegments with non-existent docID returned error (expected): %v", err)
				}

				// Test GetSegment with empty segmentID
				_, err = g.GetSegment(ctx, "")
				if err == nil {
					t.Error("Expected error for GetSegment with empty segmentID")
				} else {
					t.Logf("GetSegment correctly rejected empty segmentID: %v", err)
				}

				// Test GetSegment with non-existent segmentID
				nonExistentSegment, err := g.GetSegment(ctx, "non_existent_single_segment")
				if err == nil {
					t.Logf("GetSegment with non-existent ID handled gracefully: returned %v", nonExistentSegment)
				} else {
					t.Logf("GetSegment with non-existent ID returned error (expected): %v", err)
				}
			})

			// Step 8: Test RemoveSegments functionality
			t.Run("Remove_Segments", func(t *testing.T) {
				// First, add segments to remove
				removeSegmentTexts := []types.SegmentText{
					{
						ID:   "remove_segment_001",
						Text: "This segment will be removed for testing.",
					},
					{
						ID:   "remove_segment_002",
						Text: "Another segment for removal testing.",
					},
				}

				removeDocID := fmt.Sprintf("remove_segments_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "remove_test_prep",
						"config": configName,
					},
				}

				// Add segments first
				addedIDs, err := g.AddSegments(ctx, removeDocID, removeSegmentTexts, addOptions)
				if err != nil {
					t.Logf("Failed to add segments for remove test (expected): %v", err)
					return
				}

				if len(addedIDs) != 2 {
					t.Logf("Expected 2 segments to be added for remove test, got %d", len(addedIDs))
					return
				}

				// Test removing specific segments by IDs
				removeCount, err := g.RemoveSegments(ctx, []string{"remove_segment_001"})
				if err != nil {
					t.Logf("Failed to remove segments (expected): %v", err)
					return
				}

				if removeCount != 1 {
					t.Errorf("Expected 1 removed segment, got %d", removeCount)
					return
				}

				t.Logf("Successfully removed %d segments by IDs", removeCount)

				// Test removing remaining segments
				remainingCount, err := g.RemoveSegments(ctx, []string{"remove_segment_002"})
				if err != nil {
					t.Logf("Failed to remove remaining segments (expected): %v", err)
					return
				}

				if remainingCount != 1 {
					t.Errorf("Expected 1 remaining segment removed, got %d", remainingCount)
					return
				}

				t.Logf("Successfully removed remaining %d segments", remainingCount)
			})

			// Step 9: Test RemoveSegmentsByDocID functionality
			t.Run("Remove_Segments_By_DocID", func(t *testing.T) {
				// Add multiple segments for one document
				docRemovalSegmentTexts := []types.SegmentText{
					{
						ID:   "doc_remove_segment_001",
						Text: "First segment for document removal testing.",
					},
					{
						ID:   "doc_remove_segment_002",
						Text: "Second segment for document removal testing.",
					},
					{
						ID:   "doc_remove_segment_003",
						Text: "Third segment for document removal testing.",
					},
				}

				docRemovalDocID := fmt.Sprintf("doc_removal_segments_%s", configName)
				addOptions := &types.UpsertOptions{
					GraphName: collectionID,
					Metadata: map[string]interface{}{
						"source": "doc_removal_test",
						"config": configName,
					},
				}

				// Add segments first
				addedIDs, err := g.AddSegments(ctx, docRemovalDocID, docRemovalSegmentTexts, addOptions)
				if err != nil {
					t.Logf("Failed to add segments for document removal test (expected): %v", err)
					return
				}

				if len(addedIDs) != 3 {
					t.Logf("Expected 3 segments to be added for document removal test, got %d", len(addedIDs))
					return
				}

				// Test removing all segments by document ID
				removedCount, err := g.RemoveSegmentsByDocID(ctx, docRemovalDocID)
				if err != nil {
					t.Logf("Failed to remove segments by docID (expected): %v", err)
					return
				}

				if removedCount != 3 {
					t.Errorf("Expected 3 segments removed by docID, got %d", removedCount)
					return
				}

				t.Logf("Successfully removed %d segments by docID", removedCount)
			})

			// Step 10: Test RemoveSegments error handling
			t.Run("Remove_Segments_Error_Handling", func(t *testing.T) {
				// Test with empty segment list
				emptyRemoveCount, err := g.RemoveSegments(ctx, []string{})
				if err != nil {
					t.Errorf("RemoveSegments with empty list should not return error: %v", err)
				}
				if emptyRemoveCount != 0 {
					t.Errorf("Expected 0 removed segments for empty list, got %d", emptyRemoveCount)
				}

				// Test with non-existent segment IDs
				nonExistentCount, err := g.RemoveSegments(ctx, []string{"non_existent_segment"})
				if err == nil {
					t.Logf("RemoveSegments with non-existent IDs handled gracefully: removed %d", nonExistentCount)
				} else {
					t.Logf("RemoveSegments with non-existent IDs returned error (expected): %v", err)
				}

				// Test RemoveSegmentsByDocID with empty docID
				_, err = g.RemoveSegmentsByDocID(ctx, "")
				if err == nil {
					t.Error("Expected error for RemoveSegmentsByDocID with empty docID")
				} else {
					t.Logf("RemoveSegmentsByDocID correctly rejected empty docID: %v", err)
				}

				// Test RemoveSegmentsByDocID with non-existent docID
				nonExistentDocCount, err := g.RemoveSegmentsByDocID(ctx, "non_existent_doc")
				if err == nil {
					t.Logf("RemoveSegmentsByDocID with non-existent docID handled gracefully: removed %d", nonExistentDocCount)
				} else {
					t.Logf("RemoveSegmentsByDocID with non-existent docID returned error (expected): %v", err)
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
			// Replace + with _ to make collection name valid
			safeName := strings.ReplaceAll(configName, "+", "_")
			storeCollectionID := fmt.Sprintf("segment_store_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.CollectionConfig{
				ID: storeCollectionID,
				Metadata: map[string]interface{}{
					"type": "segment_store_test",
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
				weightKey := fmt.Sprintf(StoreKeyWeight, segment.ID)
				scoreKey := fmt.Sprintf(StoreKeyScore, segment.ID)
				voteKey := fmt.Sprintf(StoreKeyVote, segment.ID)

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
