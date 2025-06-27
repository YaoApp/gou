package neo4j

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// AddNodes adds nodes to the graph
func (s *Store) AddNodes(ctx context.Context, opts *types.AddNodesOptions) ([]string, error) {
	// TODO: implement node addition
	return nil, nil
}

// GetNodes retrieves nodes from the graph
func (s *Store) GetNodes(ctx context.Context, opts *types.GetNodesOptions) ([]*types.GraphNode, error) {
	// TODO: implement node retrieval
	return nil, nil
}

// DeleteNodes deletes nodes from the graph
func (s *Store) DeleteNodes(ctx context.Context, opts *types.DeleteNodesOptions) error {
	// TODO: implement node deletion
	return nil
}
