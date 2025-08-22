package graphrag

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/graphrag/types"
)

// StoreKeyWeight key format for weight storage
const StoreKeyWeight = "doc:%s:segment:weight:%s" // doc:{docID}:segment:weight:{segmentID}

// UpdateWeights updates weight for segments
func (g *GraphRag) UpdateWeights(ctx context.Context, docID string, segments []types.SegmentWeight, options ...types.UpdateWeightOptions) (int, error) {
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

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			return 0, fmt.Errorf("failed to update weight in vector store: %w", err)
		}

		return len(segments), nil
	}

	// Strategy 2: Store configured - concurrent update to Store and Vector DB
	var wg sync.WaitGroup
	var storeErr, vectorErr error
	updatedCount := 0

	// Compute weight if compute is provided
	if len(options) > 0 && options[0].Compute != nil {
		var computeErr error
		for i, segment := range segments {
			weight, err := options[0].Compute.Compute(ctx, docID, segment.ID, options[0].Progress)
			if err != nil {
				g.Logger.Errorf("Failed to compute weight for segment %s: %v", segment.ID, err)
				computeErr = errors.Join(computeErr, err)
				continue
			}
			segments[i].Weight = weight
		}
	}

	// Update Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeUpdated := 0
		for _, segment := range segments {
			err := g.storeSegmentValue(docID, segment.ID, StoreKeyWeight, segment.Weight)
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

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
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
