package graphrag

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yaoapp/gou/graphrag/types"
)

// StoreKeyScore key format for score storage
const StoreKeyScore = "doc:%s:segment:score:%s" // doc:{docID}:segment:score:{segmentID}

// StoreKeyScoreDimensions key format for score dimensions storage
const StoreKeyScoreDimensions = "doc:%s:segment:score:dimensions:%s" // doc:{docID}:segment:score:dimensions:{segmentID}

// UpdateScores updates score for segments
func (g *GraphRag) UpdateScores(ctx context.Context, docID string, segments []types.SegmentScore, options ...types.UpdateScoreOptions) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}

	// Compute score if compute is provided
	if len(options) > 0 && options[0].Compute != nil {
		var computeErr error
		for i, segment := range segments {
			score, dimensions, err := options[0].Compute.Compute(ctx, docID, segment.ID, options[0].Progress)
			if err != nil {
				g.Logger.Errorf("Failed to compute score for segment %s: %v", segment.ID, err)
				computeErr = errors.Join(computeErr, err)
				continue
			}
			segments[i].Score = score
			segments[i].Dimensions = dimensions
		}

		if computeErr != nil {
			return 0, fmt.Errorf("failed to compute score for segments: %w", computeErr)
		}
	}

	// Strategy 1: Store not configured - use Vector DB metadata
	if g.Store == nil {
		// Update directly in Vector DB metadata
		var updates []segmentMetadataUpdate
		for _, segment := range segments {
			updates = append(updates,
				segmentMetadataUpdate{
					SegmentID:   segment.ID,
					MetadataKey: "score",
					Value:       segment.Score,
				},
				segmentMetadataUpdate{
					SegmentID:   segment.ID,
					MetadataKey: "score_dimensions",
					Value:       segment.Dimensions,
				},
			)
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
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
			// Update score
			err := g.storeSegmentValue(docID, segment.ID, StoreKeyScore, segment.Score)
			if err != nil {
				g.Logger.Warnf("Failed to update score for segment %s in Store: %v", segment.ID, err)
				continue
			}

			// Update score dimensions
			err = g.storeSegmentValue(docID, segment.ID, StoreKeyScoreDimensions, segment.Dimensions)
			if err != nil {
				g.Logger.Warnf("Failed to update score dimensions for segment %s in Store: %v", segment.ID, err)
				continue
			}
			storeUpdated++
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
			updates = append(updates,
				segmentMetadataUpdate{
					SegmentID:   segment.ID,
					MetadataKey: "score",
					Value:       segment.Score,
				},
				segmentMetadataUpdate{
					SegmentID:   segment.ID,
					MetadataKey: "score_dimensions",
					Value:       segment.Dimensions,
				},
			)
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
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
