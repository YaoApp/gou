package graphrag

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// Score scores for segments
func (g *GraphRag) Score(ctx context.Context, segments []types.SegmentScore, callback ...types.ScoreProgress) ([]types.SegmentScore, error) {
	// TODO: Implement Score
	return nil, nil
}
