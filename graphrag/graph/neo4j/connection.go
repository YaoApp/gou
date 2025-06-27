package neo4j

import (
	"context"

	"github.com/yaoapp/gou/graphrag/types"
)

// Connect establishes connection to Neo4j server
func (s *Store) Connect(ctx context.Context, config types.GraphStoreConfig) error {
	// TODO: implement Neo4j connection
	return nil
}

// Disconnect closes the connection to Neo4j server
func (s *Store) Disconnect(ctx context.Context) error {
	// TODO: implement Neo4j disconnection
	return nil
}

// IsConnected returns whether the store is connected
func (s *Store) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// Close closes the connection and cleans up resources
func (s *Store) Close() error {
	return s.Disconnect(context.Background())
}
