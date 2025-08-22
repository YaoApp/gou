package graphrag

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
)

// StoreKeyHit key format for hit storage (List)
const StoreKeyHit = "segment_hits_%s" // segment_hits_{segmentID}

// StoreKeyHitCount key format for hit count storage
const StoreKeyHitCount = "segment_hit_count_%s" // segment_hit_count_{segmentID}

// UpdateHit updates hit for segments
func (g *GraphRag) UpdateHit(ctx context.Context, docID string, segments []types.SegmentHit, options ...types.UpdateHitOptions) (int, error) {
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
	var updates []segmentMetadataUpdate
	for _, segment := range segments {
		updates = append(updates, segmentMetadataUpdate{
			SegmentID:   segment.ID,
			MetadataKey: "hit",
			Value:       segment.HitID, // Store HitID as hit metadata
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
	var wg sync.WaitGroup
	var storeErr, vectorErr error
	updatedCount := 0

	// Update Store concurrently (using List for hits and counters for statistics)
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeUpdated := 0
		for _, segment := range segments {
			// Convert segment hit to map for Store operations
			hitMap, err := segmentHitToMap(segment)
			if err != nil {
				g.Logger.Warnf("Failed to convert hit to map for segment %s: %v", segment.ID, err)
				continue
			}

			// Add hit to list
			err = g.Store.Push(fmt.Sprintf(StoreKeyHit, segment.ID), hitMap)
			if err != nil {
				g.Logger.Warnf("Failed to add hit for segment %s to Store list: %v", segment.ID, err)
				continue
			}

			// Update hit count
			hitCountKey := fmt.Sprintf(StoreKeyHitCount, segment.ID)
			count, ok := g.Store.Get(hitCountKey)
			if !ok {
				count = 0
			}
			if countInt, ok := count.(int); ok {
				err = g.Store.Set(hitCountKey, countInt+1, 0)
			} else {
				err = g.Store.Set(hitCountKey, 1, 0)
			}
			if err != nil {
				g.Logger.Warnf("Failed to increment hit count for segment %s: %v", segment.ID, err)
			}

			storeUpdated++
		}
		if storeUpdated < len(segments) {
			storeErr = fmt.Errorf("failed to update some hits in Store: %d/%d updated", storeUpdated, len(segments))
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
				MetadataKey: "hit",
				Value:       segment.HitID, // Store HitID as hit metadata
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to update hit in vector store: %w", err)
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
		return 0, fmt.Errorf("failed to update hit in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return updatedCount, nil
}

// RemoveHit removes a single hit by HitID and updates statistics
func (g *GraphRag) RemoveHit(ctx context.Context, docID string, segmentID string, hitID string) error {
	if g.Store == nil {
		return fmt.Errorf("store is not configured, cannot remove hit")
	}

	var wg sync.WaitGroup
	var storeErr, vectorErr error

	// Remove from Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get all hits for the segment
		hitKey := fmt.Sprintf(StoreKeyHit, segmentID)
		hits, err := g.Store.ArrayAll(hitKey)
		if err != nil {
			storeErr = fmt.Errorf("failed to get hits from Store: %w", err)
			return
		}

		// Find and remove the hit with matching HitID
		var removedHit *types.SegmentHit
		var removedHitMap map[string]interface{}
		for _, h := range hits {
			hit, err := mapToSegmentHit(h)
			if err != nil {
				g.Logger.Warnf("Failed to convert stored hit to struct: %v", err)
				continue
			}
			if hit.HitID == hitID {
				removedHit = &hit
				// Convert back to map for Pull operation
				removedHitMap, err = segmentHitToMap(hit)
				if err != nil {
					storeErr = fmt.Errorf("failed to convert hit to map for removal: %w", err)
					return
				}
				break
			}
		}

		if removedHit == nil {
			storeErr = fmt.Errorf("hit with ID %s not found for segment %s", hitID, segmentID)
			return
		}

		// Remove hit from list using the map representation
		err = g.Store.Pull(hitKey, removedHitMap)
		if err != nil {
			storeErr = fmt.Errorf("failed to remove hit from Store list: %w", err)
			return
		}

		// Update hit count
		hitCountKey := fmt.Sprintf(StoreKeyHitCount, segmentID)
		count, ok := g.Store.Get(hitCountKey)
		if ok {
			if countInt, ok := count.(int); ok && countInt > 0 {
				err = g.Store.Set(hitCountKey, countInt-1, 0)
				if err != nil {
					g.Logger.Warnf("Failed to decrement hit count for segment %s: %v", segmentID, err)
				}
			}
		}
	}()

	// Update Vector DB metadata concurrently (remove hit metadata)
	wg.Add(1)
	go func() {
		defer wg.Done()

		updates := []segmentMetadataUpdate{
			{
				SegmentID:   segmentID,
				MetadataKey: "hit",
				Value:       nil, // Remove hit metadata
			},
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to remove hit in vector store: %w", err)
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
		return fmt.Errorf("failed to remove hit in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return nil
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

	return g.listHitsForSegment(ctx, options.SegmentID, options)
}

// listHitsForSegment lists hits for a specific segment
func (g *GraphRag) listHitsForSegment(ctx context.Context, segmentID string, options *types.ScrollHitsOptions) (*types.HitScrollResult, error) {
	hitKey := fmt.Sprintf(StoreKeyHit, segmentID)

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
