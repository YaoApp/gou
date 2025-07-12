package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
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

// UpdateVote updates vote for segments
func (g *GraphRag) UpdateVote(ctx context.Context, segments []types.SegmentVote) (int, error) {
	if g.Store == nil {
		return 0, fmt.Errorf("store is not configured")
	}

	if len(segments) == 0 {
		return 0, nil
	}

	// Update vote for each segment
	updatedCount := 0
	for _, segment := range segments {
		err := g.storeSegmentValue(segment.ID, StoreKeyVote, segment.Vote)
		if err != nil {
			g.Logger.Warnf("Failed to update vote for segment %s: %v", segment.ID, err)
		} else {
			updatedCount++
		}
	}

	return updatedCount, nil
}

// UpdateScore updates score for segments
func (g *GraphRag) UpdateScore(ctx context.Context, segments []types.SegmentScore) (int, error) {
	if g.Store == nil {
		return 0, fmt.Errorf("store is not configured")
	}

	if len(segments) == 0 {
		return 0, nil
	}

	// Update score for each segment
	updatedCount := 0
	for _, segment := range segments {
		err := g.storeSegmentValue(segment.ID, StoreKeyScore, segment.Score)
		if err != nil {
			g.Logger.Warnf("Failed to update score for segment %s: %v", segment.ID, err)
		} else {
			updatedCount++
		}
	}

	return updatedCount, nil
}

// UpdateWeight updates weight for segments
func (g *GraphRag) UpdateWeight(ctx context.Context, segments []types.SegmentWeight) (int, error) {
	if g.Store == nil {
		return 0, fmt.Errorf("store is not configured")
	}

	if len(segments) == 0 {
		return 0, nil
	}

	// Update weight for each segment
	updatedCount := 0
	for _, segment := range segments {
		err := g.storeSegmentValue(segment.ID, StoreKeyWeight, segment.Weight)
		if err != nil {
			g.Logger.Warnf("Failed to update weight for segment %s: %v", segment.ID, err)
		} else {
			updatedCount++
		}
	}

	return updatedCount, nil
}
