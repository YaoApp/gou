package graphrag

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// Vote votes for segments
func (g *GraphRag) Vote(ctx context.Context, segments []types.SegmentVote, callback ...types.VoteProgress) ([]types.SegmentVote, error) {
	// TODO: Implement Vote
	return nil, nil
}
