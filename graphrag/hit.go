package graphrag

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// StoreKeyHit key format for hit storage (List)
const StoreKeyHit = "doc:%s:segment:hits:%s" // doc:{docID}:segment:hits:{segmentID}

// StoreKeyHitCount key format for hit count storage
const StoreKeyHitCount = "doc:%s:segment:hit:count:%s" // doc:{docID}:segment:hit:count:{segmentID}

// UpdateHits updates hit for segments
func (g *GraphRag) UpdateHits(ctx context.Context, docID string, segments []types.SegmentHit, options ...types.UpdateHitOptions) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}

	// Apply reaction from options if SegmentReaction is not provided in segments
	if len(options) > 0 && options[0].Reaction != nil {
		for i := range segments {
			if segments[i].SegmentReaction == nil {
				segments[i].SegmentReaction = options[0].Reaction
			}
		}
	}

	// Generate HitIDs for segments that don't have them
	for i := range segments {
		if segments[i].HitID == "" {
			segments[i].HitID = uuid.New().String()
		}
	}

	// Strategy 1: Store not configured - use Vector DB metadata only
	if g.Store == nil {
		return g.updateHitInVectorOnly(ctx, docID, segments)
	}

	// Strategy 2: Store configured - concurrent update to Store and Vector DB
	return g.updateHitInStoreAndVector(ctx, docID, segments)
}

// updateHitInVectorOnly updates hits in Vector DB metadata only
func (g *GraphRag) updateHitInVectorOnly(ctx context.Context, docID string, segments []types.SegmentHit) (int, error) {
	// Group segments by ID to count hits per segment
	hitCounts := make(map[string]int)
	for _, segment := range segments {
		hitCounts[segment.ID]++
	}

	var updates []segmentMetadataUpdate
	for segmentID, count := range hitCounts {
		// Get current hit count from vector metadata
		currentCount := 0
		if g.Vector != nil {
			// Extract graphName from docID
			graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
			if graphName == "" {
				graphName = "default"
			}
			collectionIDs, err := utils.GetCollectionIDs(graphName)
			if err == nil {
				// Try to get current hit count from vector metadata
				if metadata, err := g.Vector.GetMetadata(ctx, collectionIDs.Vector, segmentID); err == nil {
					if hitValue, ok := metadata["hit"]; ok {
						currentCount = types.SafeExtractInt(hitValue, 0)
					}
				}
			}
		}

		updates = append(updates, segmentMetadataUpdate{
			SegmentID:   segmentID,
			MetadataKey: "hit",
			Value:       currentCount + count, // Store hit count as metadata
		})
	}

	err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
	if err != nil {
		return 0, fmt.Errorf("failed to update hit in vector store: %w", err)
	}

	return len(segments), nil
}

// updateHitInStoreAndVector updates hits in both Store (as List) and Vector DB
func (g *GraphRag) updateHitInStoreAndVector(ctx context.Context, docID string, segments []types.SegmentHit) (int, error) {
	var storeErr, vectorErr error
	updatedCount := 0

	// Group segments by ID for processing
	segmentsByID := make(map[string][]types.SegmentHit)
	for _, segment := range segments {
		segmentsByID[segment.ID] = append(segmentsByID[segment.ID], segment)
	}

	// Process each unique segment ID
	finalHitCounts := make(map[string]int)
	storeUpdated := 0

	for segmentID, segmentHits := range segmentsByID {
		hitKey := fmt.Sprintf(StoreKeyHit, docID, segmentID)
		hitCountKey := fmt.Sprintf(StoreKeyHitCount, docID, segmentID)

		// Add all hits to the list first
		for _, segment := range segmentHits {
			hitMap, err := segmentHitToMap(segment)
			if err != nil {
				g.Logger.Warnf("Failed to convert hit to map for segment %s: %v", segment.ID, err)
				continue
			}

			err = g.Store.Push(hitKey, hitMap)
			if err != nil {
				g.Logger.Warnf("Failed to add hit for segment %s to Store list: %v", segment.ID, err)
				continue
			}
			storeUpdated++
		}

		// Get the accurate count directly from the list length
		actualCount := g.Store.ArrayLen(hitKey)

		// Update the count cache to match actual records
		err := g.Store.Set(hitCountKey, actualCount, 0)
		if err != nil {
			g.Logger.Warnf("Failed to update hit count for segment %s: %v", segmentID, err)
		}

		// Store the final count for Vector DB update
		finalHitCounts[segmentID] = actualCount
	}

	if storeUpdated < len(segments) {
		storeErr = fmt.Errorf("failed to update some hits in Store: %d/%d updated", storeUpdated, len(segments))
	}

	// Step 2: Update Vector DB with the accurate counts from Store
	if len(finalHitCounts) > 0 {
		var updates []segmentMetadataUpdate
		for segmentID, actualCount := range finalHitCounts {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segmentID,
				MetadataKey: "hit",
				Value:       actualCount, // Use the accurate count from ArrayLen
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to update hit in vector store: %w", err)
		}
	}

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
		return 0, fmt.Errorf("failed to update hit in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return updatedCount, nil
}

// RemoveHits removes multiple hits by HitID and updates statistics
func (g *GraphRag) RemoveHits(ctx context.Context, docID string, hits []types.HitRemoval) (int, error) {
	if len(hits) == 0 {
		return 0, nil
	}

	if g.Store == nil {
		return 0, fmt.Errorf("store is not configured, cannot remove hits")
	}

	var wg sync.WaitGroup
	var storeErr, vectorErr error
	removedCount := 0

	// Group hits by segment ID for efficient processing
	hitsBySegment := make(map[string][]types.HitRemoval)
	for _, hit := range hits {
		hitsBySegment[hit.SegmentID] = append(hitsBySegment[hit.SegmentID], hit)
	}

	// Remove from Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeRemoved := 0

		for segmentID, segmentHits := range hitsBySegment {
			// Get all hits for the segment
			hitKey := fmt.Sprintf(StoreKeyHit, docID, segmentID)
			allHits, err := g.Store.ArrayAll(hitKey)
			if err != nil {
				g.Logger.Warnf("Failed to get hits from Store for segment %s: %v", segmentID, err)
				continue
			}

			// Create a map of HitID to remove for efficient lookup
			hitsToRemove := make(map[string]bool)
			for _, h := range segmentHits {
				hitsToRemove[h.HitID] = true
			}

			// Find hits to remove
			var removedHits []types.SegmentHit
			var hitsToKeep []interface{}

			for _, h := range allHits {
				hit, err := mapToSegmentHit(h)
				if err != nil {
					g.Logger.Warnf("Failed to convert stored hit to struct: %v", err)
					hitsToKeep = append(hitsToKeep, h) // Keep invalid hits
					continue
				}

				if hitsToRemove[hit.HitID] {
					// This hit should be removed
					removedHits = append(removedHits, hit)
				} else {
					// Keep this hit
					hitsToKeep = append(hitsToKeep, h)
				}
			}

			// Update the hit list
			if len(removedHits) > 0 {
				// Clear the list and re-add remaining hits
				g.Store.Del(hitKey)
				if len(hitsToKeep) > 0 {
					err = g.Store.Push(hitKey, hitsToKeep...)
				}
				if err != nil {
					g.Logger.Warnf("Failed to update hit list for segment %s: %v", segmentID, err)
					continue
				}

				// Update hit count to match actual records after removal
				hitCountKey := fmt.Sprintf(StoreKeyHitCount, docID, segmentID)
				hitKey := fmt.Sprintf(StoreKeyHit, docID, segmentID)
				actualCount := g.Store.ArrayLen(hitKey)

				if actualCount == 0 {
					g.Store.Del(hitCountKey)
				} else {
					g.Store.Set(hitCountKey, actualCount, 0)
				}

				storeRemoved += len(removedHits)
			}
		}

		if storeRemoved < len(hits) {
			storeErr = fmt.Errorf("failed to remove some hits in Store: %d/%d removed", storeRemoved, len(hits))
		}
	}()

	// Update Vector DB metadata concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Group hits by segment ID
		segmentIDs := make(map[string]bool)
		for _, hit := range hits {
			segmentIDs[hit.SegmentID] = true
		}

		var updates []segmentMetadataUpdate
		for segmentID := range segmentIDs {
			// Get actual count from Store after removal
			hitKey := fmt.Sprintf(StoreKeyHit, docID, segmentID)
			actualCount := g.Store.ArrayLen(hitKey)

			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segmentID,
				MetadataKey: "hit",
				Value:       actualCount, // Use actual count from Store
			})
		}

		if len(updates) > 0 {
			err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
			if err != nil {
				vectorErr = fmt.Errorf("failed to update hits in vector store: %w", err)
			}
		}
	}()

	wg.Wait()

	// Count successful removals (at least one storage succeeded)
	if storeErr == nil || vectorErr == nil {
		removedCount = len(hits)
	}

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store remove error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB remove error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to remove hits in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return removedCount, nil
}

// RemoveHitsBySegmentID removes all hits for a segment and clears statistics
func (g *GraphRag) RemoveHitsBySegmentID(ctx context.Context, docID string, segmentID string) (int, error) {
	if g.Store == nil {
		return 0, fmt.Errorf("store is not configured, cannot remove hits")
	}

	var wg sync.WaitGroup
	var storeErr, vectorErr error
	removedCount := 0

	// Remove from Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get all hits for the segment
		hitKey := fmt.Sprintf(StoreKeyHit, docID, segmentID)
		allHits, err := g.Store.ArrayAll(hitKey)
		if err != nil {
			g.Logger.Warnf("Failed to get hits from Store for segment %s: %v", segmentID, err)
			storeErr = fmt.Errorf("failed to get hits from Store: %w", err)
			return
		}

		// Count hits before removal
		removedCount = len(allHits)

		// Clear all hits for the segment
		g.Store.Del(hitKey)

		// Clear hit count
		hitCountKey := fmt.Sprintf(StoreKeyHitCount, docID, segmentID)
		g.Store.Del(hitCountKey)
	}()

	// Update Vector DB metadata concurrently (clear hit count)
	wg.Add(1)
	go func() {
		defer wg.Done()

		updates := []segmentMetadataUpdate{
			{
				SegmentID:   segmentID,
				MetadataKey: "hit",
				Value:       0, // Clear hit count to 0
			},
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to remove hits in vector store: %w", err)
		}
	}()

	wg.Wait()

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store remove error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB remove error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to remove hits in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return removedCount, nil
}

// ScrollHits scrolls hits for a document with pagination support
func (g *GraphRag) ScrollHits(ctx context.Context, docID string, options *types.ScrollHitsOptions) (*types.HitScrollResult, error) {
	if g.Store == nil {
		return nil, fmt.Errorf("store is not configured, cannot list hits")
	}

	if options == nil {
		options = &types.ScrollHitsOptions{}
	}

	// Set default limit
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}

	// SegmentID is required for listing hits
	if options.SegmentID == "" {
		return nil, fmt.Errorf("segment_id is required for listing hits")
	}

	return g.listHitsForSegment(ctx, docID, options.SegmentID, options)
}

// listHitsForSegment lists hits for a specific segment
func (g *GraphRag) listHitsForSegment(ctx context.Context, docID string, segmentID string, options *types.ScrollHitsOptions) (*types.HitScrollResult, error) {
	hitKey := fmt.Sprintf(StoreKeyHit, docID, segmentID)

	// Get all hits for the segment
	allHits, err := g.Store.ArrayAll(hitKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get hits from Store: %w", err)
	}

	// Convert to SegmentHit slice and apply filters
	var hits []types.SegmentHit
	for _, h := range allHits {
		hit, err := mapToSegmentHit(h)
		if err != nil {
			g.Logger.Warnf("Failed to convert stored hit to struct: %v", err)
			continue
		}

		// Apply filters
		if !g.matchesHitFilters(hit, options) {
			continue
		}

		hits = append(hits, hit)
	}

	return g.paginateHits(hits, options)
}

// matchesHitFilters checks if a hit matches the filter criteria
func (g *GraphRag) matchesHitFilters(hit types.SegmentHit, options *types.ScrollHitsOptions) bool {
	// Filter by reaction source
	if options.Source != "" && hit.SegmentReaction != nil && hit.SegmentReaction.Source != options.Source {
		return false
	}

	// Filter by reaction scenario
	if options.Scenario != "" && hit.SegmentReaction != nil && hit.SegmentReaction.Scenario != options.Scenario {
		return false
	}

	return true
}

// paginateHits handles pagination of hit results
func (g *GraphRag) paginateHits(hits []types.SegmentHit, options *types.ScrollHitsOptions) (*types.HitScrollResult, error) {
	result := &types.HitScrollResult{
		Total: len(hits),
	}

	// Find start index based on cursor
	startIndex := 0
	if options.Cursor != "" {
		// Find the hit with the cursor HitID
		for i, hit := range hits {
			if hit.HitID == options.Cursor {
				startIndex = i + 1
				break
			}
		}
	}

	// Calculate end index
	endIndex := startIndex + options.Limit
	if endIndex > len(hits) {
		endIndex = len(hits)
	}

	// Extract the page of hits
	if startIndex < len(hits) {
		result.Hits = hits[startIndex:endIndex]
	}

	// Set HasMore and NextCursor
	if endIndex < len(hits) {
		result.HasMore = true
		if len(result.Hits) > 0 {
			result.NextCursor = result.Hits[len(result.Hits)-1].HitID
		}
	}

	return result, nil
}

// GetHit gets a single hit by ID
func (g *GraphRag) GetHit(ctx context.Context, docID string, segmentID string, hitID string) (*types.SegmentHit, error) {
	if g.Store == nil {
		return nil, fmt.Errorf("store is not configured, cannot get hit")
	}

	// Get all hits for the segment
	hitKey := fmt.Sprintf(StoreKeyHit, docID, segmentID)
	allHits, err := g.Store.ArrayAll(hitKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get hits from Store: %w", err)
	}

	// Find the specific hit by hitID
	for _, h := range allHits {
		hit, err := mapToSegmentHit(h)
		if err != nil {
			g.Logger.Warnf("Failed to convert stored hit to struct: %v", err)
			continue
		}

		if hit.HitID == hitID {
			return &hit, nil
		}
	}

	return nil, fmt.Errorf("hit not found")
}
