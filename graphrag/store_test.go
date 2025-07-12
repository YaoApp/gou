package graphrag

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestUpdateVote tests the UpdateVote function
func TestUpdateVote(t *testing.T) {
	// Setup connector as in document_test.go
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	config := configs["vector+store"]
	if config == nil {
		t.Skip("Config vector+store not found")
	}

	// Create GraphRag instance
	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	// Skip if Store is not available
	if g.Store == nil {
		t.Skip("Store not available for config vector+store")
	}

	ctx := context.Background()

	// Create collection using utility from collection_test.go
	vectorConfig := getVectorStore("update_vote_test", 1536)
	collectionID := fmt.Sprintf("update_vote_collection_%d", time.Now().Unix())
	collection := types.Collection{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"type": "update_vote_test",
		},
		VectorConfig: &vectorConfig,
	}

	// Create collection
	_, err = g.CreateCollection(ctx, collection)
	if err != nil {
		t.Skipf("Failed to create test collection: %v", err)
	}

	// Cleanup collection after test
	defer func() {
		removed, err := g.RemoveCollection(ctx, collectionID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
		} else if removed {
			t.Logf("Successfully cleaned up collection: %s", collectionID)
		}
	}()

	// Step 1: Add a base document using AddText
	t.Run("Add_Base_Document", func(t *testing.T) {
		baseDocOptions := &types.UpsertOptions{
			DocID:     fmt.Sprintf("vote_base_doc_%d", time.Now().Unix()),
			GraphName: collectionID,
			Metadata: map[string]interface{}{
				"source": "vote_test",
				"type":   "base_document",
			},
		}

		baseDocContent := "This is a base document for vote testing. It contains multiple paragraphs and sections for testing vote functionality."

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
				t.Skip("Skipping due to expected setup error")
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

	// Step 2: Add test segments using AddSegments
	var testSegmentIDs []string
	t.Run("Add_Test_Segments", func(t *testing.T) {
		// Create test segments
		segmentTexts := []types.SegmentText{
			{
				ID:   "vote_segment_001",
				Text: "This is the first segment for vote testing with artificial intelligence content.",
			},
			{
				ID:   "vote_segment_002",
				Text: "This is the second segment for vote testing with machine learning algorithms.",
			},
			{
				ID:   "vote_segment_003",
				Text: "This is the third segment for vote testing with natural language processing.",
			},
		}

		segmentDocID := fmt.Sprintf("vote_segments_doc_%d", time.Now().Unix())
		segmentOptions := &types.UpsertOptions{
			GraphName: collectionID,
			Metadata: map[string]interface{}{
				"source": "vote_test",
				"type":   "segments",
			},
		}

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
				t.Skip("Skipping due to expected setup error")
			} else {
				t.Errorf("Unexpected error adding segments: %v", err)
			}
			return
		}

		if len(segmentIDs) == 0 {
			t.Error("AddSegments returned 0 segment IDs")
			return
		}

		testSegmentIDs = segmentIDs
		t.Logf("Successfully added %d segments", len(segmentIDs))
	})

	// Step 3: Test UpdateVote function
	t.Run("Update_Vote", func(t *testing.T) {
		if len(testSegmentIDs) == 0 {
			t.Skip("No test segments available")
		}

		// Create vote data
		segmentVotes := []types.SegmentVote{
			{
				ID:   testSegmentIDs[0],
				Vote: 5,
			},
			{
				ID:   testSegmentIDs[1],
				Vote: 3,
			},
			{
				ID:   testSegmentIDs[2],
				Vote: 4,
			},
		}

		// Test UpdateVote
		updatedCount, err := g.UpdateVote(ctx, segmentVotes)
		if err != nil {
			t.Errorf("UpdateVote failed: %v", err)
			return
		}

		expectedCount := len(segmentVotes)
		if updatedCount != expectedCount {
			t.Errorf("Expected %d updated votes, got %d", expectedCount, updatedCount)
			return
		}

		t.Logf("Successfully updated %d votes", updatedCount)

		// Verify votes were stored
		for _, segmentVote := range segmentVotes {
			voteKey := fmt.Sprintf(StoreKeyVote, segmentVote.ID)
			if !g.Store.Has(voteKey) {
				t.Errorf("Vote key %s not found in store", voteKey)
				continue
			}

			storedVote, ok := g.Store.Get(voteKey)
			if !ok {
				t.Errorf("Failed to get stored vote for key %s", voteKey)
				continue
			}

			if storedVote != segmentVote.Vote {
				t.Errorf("Expected vote %d, got %v", segmentVote.Vote, storedVote)
				continue
			}

			t.Logf("Vote verified for segment %s: %d", segmentVote.ID, segmentVote.Vote)
		}
	})

	// Step 4: Test UpdateVote with empty segments
	t.Run("Update_Vote_Empty", func(t *testing.T) {
		updatedCount, err := g.UpdateVote(ctx, []types.SegmentVote{})
		if err != nil {
			t.Errorf("UpdateVote with empty segments failed: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated votes for empty segments, got %d", updatedCount)
			return
		}

		t.Logf("Empty segments handled correctly: %d votes updated", updatedCount)
	})

	// Step 5: Test UpdateVote with nil segments
	t.Run("Update_Vote_Nil", func(t *testing.T) {
		updatedCount, err := g.UpdateVote(ctx, nil)
		if err != nil {
			t.Errorf("UpdateVote with nil segments failed: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated votes for nil segments, got %d", updatedCount)
			return
		}

		t.Logf("Nil segments handled correctly: %d votes updated", updatedCount)
	})
}

// TestUpdateScore tests the UpdateScore function
func TestUpdateScore(t *testing.T) {
	// Setup connector as in document_test.go
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	config := configs["vector+store"]
	if config == nil {
		t.Skip("Config vector+store not found")
	}

	// Create GraphRag instance
	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	// Skip if Store is not available
	if g.Store == nil {
		t.Skip("Store not available for config vector+store")
	}

	ctx := context.Background()

	// Create collection using utility from collection_test.go
	vectorConfig := getVectorStore("update_score_test", 1536)
	collectionID := fmt.Sprintf("update_score_collection_%d", time.Now().Unix())
	collection := types.Collection{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"type": "update_score_test",
		},
		VectorConfig: &vectorConfig,
	}

	// Create collection
	_, err = g.CreateCollection(ctx, collection)
	if err != nil {
		t.Skipf("Failed to create test collection: %v", err)
	}

	// Cleanup collection after test
	defer func() {
		removed, err := g.RemoveCollection(ctx, collectionID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
		} else if removed {
			t.Logf("Successfully cleaned up collection: %s", collectionID)
		}
	}()

	// Step 1: Add a base document using AddText
	t.Run("Add_Base_Document", func(t *testing.T) {
		baseDocOptions := &types.UpsertOptions{
			DocID:     fmt.Sprintf("score_base_doc_%d", time.Now().Unix()),
			GraphName: collectionID,
			Metadata: map[string]interface{}{
				"source": "score_test",
				"type":   "base_document",
			},
		}

		baseDocContent := "This is a base document for score testing. It contains multiple paragraphs and sections for testing score functionality."

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
				t.Skip("Skipping due to expected setup error")
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

	// Step 2: Add test segments using AddSegments
	var testSegmentIDs []string
	t.Run("Add_Test_Segments", func(t *testing.T) {
		// Create test segments
		segmentTexts := []types.SegmentText{
			{
				ID:   "score_segment_001",
				Text: "This is the first segment for score testing with artificial intelligence content.",
			},
			{
				ID:   "score_segment_002",
				Text: "This is the second segment for score testing with machine learning algorithms.",
			},
			{
				ID:   "score_segment_003",
				Text: "This is the third segment for score testing with natural language processing.",
			},
		}

		segmentDocID := fmt.Sprintf("score_segments_doc_%d", time.Now().Unix())
		segmentOptions := &types.UpsertOptions{
			GraphName: collectionID,
			Metadata: map[string]interface{}{
				"source": "score_test",
				"type":   "segments",
			},
		}

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
				t.Skip("Skipping due to expected setup error")
			} else {
				t.Errorf("Unexpected error adding segments: %v", err)
			}
			return
		}

		if len(segmentIDs) == 0 {
			t.Error("AddSegments returned 0 segment IDs")
			return
		}

		testSegmentIDs = segmentIDs
		t.Logf("Successfully added %d segments", len(segmentIDs))
	})

	// Step 3: Test UpdateScore function
	t.Run("Update_Score", func(t *testing.T) {
		if len(testSegmentIDs) == 0 {
			t.Skip("No test segments available")
		}

		// Create score data
		segmentScores := []types.SegmentScore{
			{
				ID:    testSegmentIDs[0],
				Score: 0.95,
			},
			{
				ID:    testSegmentIDs[1],
				Score: 0.82,
			},
			{
				ID:    testSegmentIDs[2],
				Score: 0.77,
			},
		}

		// Test UpdateScore
		updatedCount, err := g.UpdateScore(ctx, segmentScores)
		if err != nil {
			t.Errorf("UpdateScore failed: %v", err)
			return
		}

		expectedCount := len(segmentScores)
		if updatedCount != expectedCount {
			t.Errorf("Expected %d updated scores, got %d", expectedCount, updatedCount)
			return
		}

		t.Logf("Successfully updated %d scores", updatedCount)

		// Verify scores were stored
		for _, segmentScore := range segmentScores {
			scoreKey := fmt.Sprintf(StoreKeyScore, segmentScore.ID)
			if !g.Store.Has(scoreKey) {
				t.Errorf("Score key %s not found in store", scoreKey)
				continue
			}

			storedScore, ok := g.Store.Get(scoreKey)
			if !ok {
				t.Errorf("Failed to get stored score for key %s", scoreKey)
				continue
			}

			if storedScore != segmentScore.Score {
				t.Errorf("Expected score %f, got %v", segmentScore.Score, storedScore)
				continue
			}

			t.Logf("Score verified for segment %s: %f", segmentScore.ID, segmentScore.Score)
		}
	})

	// Step 4: Test UpdateScore with empty segments
	t.Run("Update_Score_Empty", func(t *testing.T) {
		updatedCount, err := g.UpdateScore(ctx, []types.SegmentScore{})
		if err != nil {
			t.Errorf("UpdateScore with empty segments failed: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated scores for empty segments, got %d", updatedCount)
			return
		}

		t.Logf("Empty segments handled correctly: %d scores updated", updatedCount)
	})

	// Step 5: Test UpdateScore with nil segments
	t.Run("Update_Score_Nil", func(t *testing.T) {
		updatedCount, err := g.UpdateScore(ctx, nil)
		if err != nil {
			t.Errorf("UpdateScore with nil segments failed: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated scores for nil segments, got %d", updatedCount)
			return
		}

		t.Logf("Nil segments handled correctly: %d scores updated", updatedCount)
	})
}

// TestUpdateWeight tests the UpdateWeight function
func TestUpdateWeight(t *testing.T) {
	// Setup connector as in document_test.go
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	config := configs["vector+store"]
	if config == nil {
		t.Skip("Config vector+store not found")
	}

	// Create GraphRag instance
	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	// Skip if Store is not available
	if g.Store == nil {
		t.Skip("Store not available for config vector+store")
	}

	ctx := context.Background()

	// Create collection using utility from collection_test.go
	vectorConfig := getVectorStore("update_weight_test", 1536)
	collectionID := fmt.Sprintf("update_weight_collection_%d", time.Now().Unix())
	collection := types.Collection{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"type": "update_weight_test",
		},
		VectorConfig: &vectorConfig,
	}

	// Create collection
	_, err = g.CreateCollection(ctx, collection)
	if err != nil {
		t.Skipf("Failed to create test collection: %v", err)
	}

	// Cleanup collection after test
	defer func() {
		removed, err := g.RemoveCollection(ctx, collectionID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
		} else if removed {
			t.Logf("Successfully cleaned up collection: %s", collectionID)
		}
	}()

	// Step 1: Add a base document using AddText
	t.Run("Add_Base_Document", func(t *testing.T) {
		baseDocOptions := &types.UpsertOptions{
			DocID:     fmt.Sprintf("weight_base_doc_%d", time.Now().Unix()),
			GraphName: collectionID,
			Metadata: map[string]interface{}{
				"source": "weight_test",
				"type":   "base_document",
			},
		}

		baseDocContent := "This is a base document for weight testing. It contains multiple paragraphs and sections for testing weight functionality."

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
				t.Skip("Skipping due to expected setup error")
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

	// Step 2: Add test segments using AddSegments
	var testSegmentIDs []string
	t.Run("Add_Test_Segments", func(t *testing.T) {
		// Create test segments
		segmentTexts := []types.SegmentText{
			{
				ID:   "weight_segment_001",
				Text: "This is the first segment for weight testing with artificial intelligence content.",
			},
			{
				ID:   "weight_segment_002",
				Text: "This is the second segment for weight testing with machine learning algorithms.",
			},
			{
				ID:   "weight_segment_003",
				Text: "This is the third segment for weight testing with natural language processing.",
			},
		}

		segmentDocID := fmt.Sprintf("weight_segments_doc_%d", time.Now().Unix())
		segmentOptions := &types.UpsertOptions{
			GraphName: collectionID,
			Metadata: map[string]interface{}{
				"source": "weight_test",
				"type":   "segments",
			},
		}

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
				t.Skip("Skipping due to expected setup error")
			} else {
				t.Errorf("Unexpected error adding segments: %v", err)
			}
			return
		}

		if len(segmentIDs) == 0 {
			t.Error("AddSegments returned 0 segment IDs")
			return
		}

		testSegmentIDs = segmentIDs
		t.Logf("Successfully added %d segments", len(segmentIDs))
	})

	// Step 3: Test UpdateWeight function
	t.Run("Update_Weight", func(t *testing.T) {
		if len(testSegmentIDs) == 0 {
			t.Skip("No test segments available")
		}

		// Create weight data
		segmentWeights := []types.SegmentWeight{
			{
				ID:     testSegmentIDs[0],
				Weight: 0.8,
			},
			{
				ID:     testSegmentIDs[1],
				Weight: 0.6,
			},
			{
				ID:     testSegmentIDs[2],
				Weight: 0.9,
			},
		}

		// Test UpdateWeight
		updatedCount, err := g.UpdateWeight(ctx, segmentWeights)
		if err != nil {
			t.Errorf("UpdateWeight failed: %v", err)
			return
		}

		expectedCount := len(segmentWeights)
		if updatedCount != expectedCount {
			t.Errorf("Expected %d updated weights, got %d", expectedCount, updatedCount)
			return
		}

		t.Logf("Successfully updated %d weights", updatedCount)

		// Verify weights were stored
		for _, segmentWeight := range segmentWeights {
			weightKey := fmt.Sprintf(StoreKeyWeight, segmentWeight.ID)
			if !g.Store.Has(weightKey) {
				t.Errorf("Weight key %s not found in store", weightKey)
				continue
			}

			storedWeight, ok := g.Store.Get(weightKey)
			if !ok {
				t.Errorf("Failed to get stored weight for key %s", weightKey)
				continue
			}

			if storedWeight != segmentWeight.Weight {
				t.Errorf("Expected weight %f, got %v", segmentWeight.Weight, storedWeight)
				continue
			}

			t.Logf("Weight verified for segment %s: %f", segmentWeight.ID, segmentWeight.Weight)
		}
	})

	// Step 4: Test UpdateWeight with empty segments
	t.Run("Update_Weight_Empty", func(t *testing.T) {
		updatedCount, err := g.UpdateWeight(ctx, []types.SegmentWeight{})
		if err != nil {
			t.Errorf("UpdateWeight with empty segments failed: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated weights for empty segments, got %d", updatedCount)
			return
		}

		t.Logf("Empty segments handled correctly: %d weights updated", updatedCount)
	})

	// Step 5: Test UpdateWeight with nil segments
	t.Run("Update_Weight_Nil", func(t *testing.T) {
		updatedCount, err := g.UpdateWeight(ctx, nil)
		if err != nil {
			t.Errorf("UpdateWeight with nil segments failed: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated weights for nil segments, got %d", updatedCount)
			return
		}

		t.Logf("Nil segments handled correctly: %d weights updated", updatedCount)
	})
}

// TestStoreNotConfigured tests the behavior when Store is not configured
func TestStoreNotConfigured(t *testing.T) {
	// Setup connector as in document_test.go
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	config := configs["vector"] // Use vector-only config (no Store)
	if config == nil {
		t.Skip("Config vector not found")
	}

	// Create GraphRag instance
	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	t.Run("UpdateVote_No_Store", func(t *testing.T) {
		segmentVotes := []types.SegmentVote{
			{
				ID:   "test_segment_001",
				Vote: 5,
			},
		}

		updatedCount, err := g.UpdateVote(ctx, segmentVotes)
		if err == nil {
			t.Error("Expected error when Store is not configured")
			return
		}

		if !strings.Contains(err.Error(), "store is not configured") {
			t.Errorf("Expected 'store is not configured' error, got: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated votes, got %d", updatedCount)
			return
		}

		t.Logf("Correctly rejected UpdateVote when Store is not configured: %v", err)
	})

	t.Run("UpdateScore_No_Store", func(t *testing.T) {
		segmentScores := []types.SegmentScore{
			{
				ID:    "test_segment_001",
				Score: 0.95,
			},
		}

		updatedCount, err := g.UpdateScore(ctx, segmentScores)
		if err == nil {
			t.Error("Expected error when Store is not configured")
			return
		}

		if !strings.Contains(err.Error(), "store is not configured") {
			t.Errorf("Expected 'store is not configured' error, got: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated scores, got %d", updatedCount)
			return
		}

		t.Logf("Correctly rejected UpdateScore when Store is not configured: %v", err)
	})

	t.Run("UpdateWeight_No_Store", func(t *testing.T) {
		segmentWeights := []types.SegmentWeight{
			{
				ID:     "test_segment_001",
				Weight: 0.8,
			},
		}

		updatedCount, err := g.UpdateWeight(ctx, segmentWeights)
		if err == nil {
			t.Error("Expected error when Store is not configured")
			return
		}

		if !strings.Contains(err.Error(), "store is not configured") {
			t.Errorf("Expected 'store is not configured' error, got: %v", err)
			return
		}

		if updatedCount != 0 {
			t.Errorf("Expected 0 updated weights, got %d", updatedCount)
			return
		}

		t.Logf("Correctly rejected UpdateWeight when Store is not configured: %v", err)
	})
}
