package graphrag

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// AddSegments adds segments to a collection manually
func (g *GraphRag) AddSegments(ctx context.Context, id string, segmentTexts []types.SegmentText, options *types.UpsertOptions) (int, error) {
	// TODO: Implement AddSegments
	return 0, nil
}

// UpdateSegments updates segments manually
func (g *GraphRag) UpdateSegments(ctx context.Context, segmentTexts []types.SegmentText, options *types.UpsertOptions) (int, error) {
	// TODO: Implement UpdateSegments
	return 0, nil
}

// RemoveSegments removes segments by IDs
func (g *GraphRag) RemoveSegments(ctx context.Context, segmentIDs []string) (int, error) {
	// TODO: Implement RemoveSegments
	return 0, nil
}

// GetSegments gets all segments of a collection
func (g *GraphRag) GetSegments(ctx context.Context, id string) ([]types.Segment, error) {
	// TODO: Implement GetSegments
	return nil, nil
}

// GetSegment gets a single segment by ID
func (g *GraphRag) GetSegment(ctx context.Context, segmentID string) (*types.Segment, error) {
	// TODO: Implement GetSegment
	return nil, nil
}
