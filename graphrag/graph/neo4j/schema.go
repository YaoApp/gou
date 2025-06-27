package neo4j

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// GetSchema returns the schema of the graph
func (s *Store) GetSchema(ctx context.Context, graphName string) (*types.DynamicGraphSchema, error) {
	// TODO: implement schema retrieval
	return nil, nil
}

// CreateIndex creates an index on the graph
func (s *Store) CreateIndex(ctx context.Context, opts *types.CreateIndexOptions) error {
	// TODO: implement index creation
	return nil
}

// DropIndex drops an index from the graph
func (s *Store) DropIndex(ctx context.Context, opts *types.DropIndexOptions) error {
	// TODO: implement index deletion
	return nil
}
