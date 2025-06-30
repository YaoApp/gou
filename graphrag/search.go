package graphrag

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// Search searches for segments
func (g *GraphRag) Search(ctx context.Context, options *types.QueryOptions, callback ...types.SearcherProgress) ([]types.Segment, error) {
	// TODO: Implement Search
	return nil, nil
}

// MultiSearch multi-searches for segments
func (g *GraphRag) MultiSearch(ctx context.Context, options []types.QueryOptions, callback ...types.SearcherProgress) (map[string][]types.Segment, error) {
	// TODO: Implement MultiSearch
	return nil, nil
}
