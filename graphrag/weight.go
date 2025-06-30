package graphrag

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// Weight weights for segments
func (g *GraphRag) Weight(ctx context.Context, segments []types.SegmentWeight, callback ...types.WeightProgress) ([]types.SegmentWeight, error) {
	// TODO: Implement Weight
	return nil, nil
}
