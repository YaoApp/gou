package neo4j

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
)

// GetStats returns statistics about the graph
func (s *Store) GetStats(ctx context.Context, graphName string) (*types.GraphStats, error) {
	return s.DescribeGraph(ctx, graphName)
}

// Optimize optimizes the graph storage
func (s *Store) Optimize(ctx context.Context, graphName string) error {
	return fmt.Errorf("Optimize is not implemented, it will be implemented in the future")
}
