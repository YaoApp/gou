package graphrag

import (
	"context"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// Store key formats (without docID to reduce queries)
const (
	StoreKeyVote   = "segment_vote_%s"   // segment_vote_{segmentID}
	StoreKeyScore  = "segment_score_%s"  // segment_score_{segmentID}
	StoreKeyWeight = "segment_weight_%s" // segment_weight_{segmentID}
	StoreKeyOrigin = "origin_%s"         // origin_{docID}
)

// storeSegmentValue stores a value for a segment with the given key format
func (g *GraphRag) storeSegmentValue(segmentID string, keyFormat string, value interface{}) error {
	if g.Store == nil {
		return fmt.Errorf("store is not configured")
	}

	key := fmt.Sprintf(keyFormat, segmentID)
	err := g.Store.Set(key, value, 0)
	if err != nil {
		return fmt.Errorf("failed to store %s for segment %s: %w", keyFormat, segmentID, err)
	}

	g.Logger.Debugf("Stored %s for segment %s: %v", keyFormat, segmentID, value)
	return nil
}

// getSegmentValue retrieves a value for a segment with the given key format
func (g *GraphRag) getSegmentValue(segmentID string, keyFormat string) (interface{}, bool) {
	if g.Store == nil {
		return nil, false
	}

	key := fmt.Sprintf(keyFormat, segmentID)
	value, ok := g.Store.Get(key)
	if !ok {
		g.Logger.Debugf("Key %s not found for segment %s", keyFormat, segmentID)
		return nil, false
	}

	return value, true
}

// deleteSegmentValue deletes a value for a segment with the given key format
func (g *GraphRag) deleteSegmentValue(segmentID string, keyFormat string) error {
	if g.Store == nil {
		return nil // No error if store is not configured
	}

	key := fmt.Sprintf(keyFormat, segmentID)
	err := g.Store.Del(key)
	if err != nil {
		return fmt.Errorf("failed to delete %s for segment %s: %w", keyFormat, segmentID, err)
	}

	g.Logger.Debugf("Deleted %s for segment %s", keyFormat, segmentID)
	return nil
}

// updateSegmentMetadataInVectorBatch updates multiple segment metadata in vector database in batch
func (g *GraphRag) updateSegmentMetadataInVectorBatch(ctx context.Context, updates []segmentMetadataUpdate) error {
	if g.Vector == nil {
		return fmt.Errorf("vector database is not configured")
	}

	if len(updates) == 0 {
		return nil
	}

	// Group updates by collection
	collectionUpdates := make(map[string][]segmentMetadataUpdate)
	for _, update := range updates {
		_, graphName, err := g.getDocIDFromExistingSegments(ctx, []string{update.SegmentID})
		if err != nil {
			g.Logger.Warnf("Failed to get document info for segment %s: %v", update.SegmentID, err)
			continue
		}

		collectionIDs, err := utils.GetCollectionIDs(graphName)
		if err != nil {
			g.Logger.Warnf("Failed to get collection IDs for segment %s: %v", update.SegmentID, err)
			continue
		}

		collectionUpdates[collectionIDs.Vector] = append(collectionUpdates[collectionIDs.Vector], update)
	}

	// Process each collection
	for collectionName, colUpdates := range collectionUpdates {
		// Check if collection exists
		exists, err := g.Vector.CollectionExists(ctx, collectionName)
		if err != nil {
			g.Logger.Warnf("Failed to check collection existence %s: %v", collectionName, err)
			continue
		}
		if !exists {
			g.Logger.Warnf("Vector collection %s does not exist", collectionName)
			continue
		}

		// Get segment IDs for this collection
		segmentIDs := make([]string, 0, len(colUpdates))
		for _, update := range colUpdates {
			segmentIDs = append(segmentIDs, update.SegmentID)
		}

		// Get all segment documents
		getOpts := &types.GetDocumentOptions{
			CollectionName: collectionName,
			IncludeVector:  false,
			IncludePayload: true,
		}

		docs, err := g.Vector.GetDocuments(ctx, segmentIDs, getOpts)
		if err != nil {
			g.Logger.Warnf("Failed to get segment documents from collection %s: %v", collectionName, err)
			continue
		}

		// Create a map of segment ID to document
		docMap := make(map[string]*types.Document)
		for _, doc := range docs {
			if doc != nil {
				docMap[doc.ID] = doc
			}
		}

		// Update documents with new metadata
		var docsToUpdate []*types.Document
		for _, update := range colUpdates {
			if doc, exists := docMap[update.SegmentID]; exists {
				if doc.Metadata == nil {
					doc.Metadata = make(map[string]interface{})
				}
				doc.Metadata[update.MetadataKey] = update.Value
				docsToUpdate = append(docsToUpdate, doc)
			}
		}

		// Batch update documents
		if len(docsToUpdate) > 0 {
			addOpts := &types.AddDocumentOptions{
				CollectionName: collectionName,
				Documents:      docsToUpdate,
				Upsert:         true,
				BatchSize:      50,
			}

			_, err = g.Vector.AddDocuments(ctx, addOpts)
			if err != nil {
				g.Logger.Warnf("Failed to update segment metadata in vector store collection %s: %v", collectionName, err)
			} else {
				g.Logger.Debugf("Updated %d segment metadata in vector store collection %s", len(docsToUpdate), collectionName)
			}
		}
	}

	return nil
}

// segmentMetadataUpdate represents a metadata update for a segment
type segmentMetadataUpdate struct {
	SegmentID   string
	MetadataKey string
	Value       interface{}
}

// UpdateVote updates vote for segments
func (g *GraphRag) UpdateVote(ctx context.Context, segments []types.SegmentVote) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}

	// Strategy 1: Store not configured - use Vector DB metadata
	if g.Store == nil {
		// Update directly in Vector DB metadata
		var updates []segmentMetadataUpdate
		for _, segment := range segments {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segment.ID,
				MetadataKey: "vote",
				Value:       segment.Vote,
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, updates)
		if err != nil {
			return 0, fmt.Errorf("failed to update vote in vector store: %w", err)
		}

		return len(segments), nil
	}

	// Strategy 2: Store configured - concurrent update to Store and Vector DB
	var wg sync.WaitGroup
	var storeErr, vectorErr error
	updatedCount := 0

	// Update Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeUpdated := 0
		for _, segment := range segments {
			err := g.storeSegmentValue(segment.ID, StoreKeyVote, segment.Vote)
			if err != nil {
				g.Logger.Warnf("Failed to update vote for segment %s in Store: %v", segment.ID, err)
			} else {
				storeUpdated++
			}
		}
		if storeUpdated < len(segments) {
			storeErr = fmt.Errorf("failed to update some votes in Store: %d/%d updated", storeUpdated, len(segments))
		}
	}()

	// Update Vector DB concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		var updates []segmentMetadataUpdate
		for _, segment := range segments {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segment.ID,
				MetadataKey: "vote",
				Value:       segment.Vote,
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to update vote in vector store: %w", err)
		}
	}()

	wg.Wait()

	// Count successful updates (at least one storage succeeded)
	if storeErr == nil || vectorErr == nil {
		updatedCount = len(segments)
	}

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store update error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB update error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to update vote in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return updatedCount, nil
}

// UpdateScore updates score for segments
func (g *GraphRag) UpdateScore(ctx context.Context, segments []types.SegmentScore) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}

	// Strategy 1: Store not configured - use Vector DB metadata
	if g.Store == nil {
		// Update directly in Vector DB metadata
		var updates []segmentMetadataUpdate
		for _, segment := range segments {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segment.ID,
				MetadataKey: "score",
				Value:       segment.Score,
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, updates)
		if err != nil {
			return 0, fmt.Errorf("failed to update score in vector store: %w", err)
		}

		return len(segments), nil
	}

	// Strategy 2: Store configured - concurrent update to Store and Vector DB
	var wg sync.WaitGroup
	var storeErr, vectorErr error
	updatedCount := 0

	// Update Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeUpdated := 0
		for _, segment := range segments {
			err := g.storeSegmentValue(segment.ID, StoreKeyScore, segment.Score)
			if err != nil {
				g.Logger.Warnf("Failed to update score for segment %s in Store: %v", segment.ID, err)
			} else {
				storeUpdated++
			}
		}
		if storeUpdated < len(segments) {
			storeErr = fmt.Errorf("failed to update some scores in Store: %d/%d updated", storeUpdated, len(segments))
		}
	}()

	// Update Vector DB concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		var updates []segmentMetadataUpdate
		for _, segment := range segments {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segment.ID,
				MetadataKey: "score",
				Value:       segment.Score,
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to update score in vector store: %w", err)
		}
	}()

	wg.Wait()

	// Count successful updates (at least one storage succeeded)
	if storeErr == nil || vectorErr == nil {
		updatedCount = len(segments)
	}

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store update error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB update error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to update score in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return updatedCount, nil
}

// UpdateWeight updates weight for segments
func (g *GraphRag) UpdateWeight(ctx context.Context, segments []types.SegmentWeight) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}

	// Strategy 1: Store not configured - use Vector DB metadata
	if g.Store == nil {
		// Update directly in Vector DB metadata
		var updates []segmentMetadataUpdate
		for _, segment := range segments {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segment.ID,
				MetadataKey: "weight",
				Value:       segment.Weight,
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, updates)
		if err != nil {
			return 0, fmt.Errorf("failed to update weight in vector store: %w", err)
		}

		return len(segments), nil
	}

	// Strategy 2: Store configured - concurrent update to Store and Vector DB
	var wg sync.WaitGroup
	var storeErr, vectorErr error
	updatedCount := 0

	// Update Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeUpdated := 0
		for _, segment := range segments {
			err := g.storeSegmentValue(segment.ID, StoreKeyWeight, segment.Weight)
			if err != nil {
				g.Logger.Warnf("Failed to update weight for segment %s in Store: %v", segment.ID, err)
			} else {
				storeUpdated++
			}
		}
		if storeUpdated < len(segments) {
			storeErr = fmt.Errorf("failed to update some weights in Store: %d/%d updated", storeUpdated, len(segments))
		}
	}()

	// Update Vector DB concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		var updates []segmentMetadataUpdate
		for _, segment := range segments {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segment.ID,
				MetadataKey: "weight",
				Value:       segment.Weight,
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to update weight in vector store: %w", err)
		}
	}()

	wg.Wait()

	// Count successful updates (at least one storage succeeded)
	if storeErr == nil || vectorErr == nil {
		updatedCount = len(segments)
	}

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store update error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB update error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to update weight in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return updatedCount, nil
}
