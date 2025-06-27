package neo4j

import (
	"sync"

	"github.com/yaoapp/gou/graphrag/types"
)

// Store implements the GraphStore interface for Neo4j
type Store struct {
	config    types.GraphStoreConfig
	driver    interface{} // TODO: replace with actual Neo4j driver
	connected bool
	mu        sync.RWMutex
}

// NewStore creates a new Neo4j graph store instance
func NewStore() *Store {
	return &Store{}
}

// GetDriver returns the underlying Neo4j driver
func (s *Store) GetDriver() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.driver
}

// GetConfig returns the current configuration
func (s *Store) GetConfig() types.GraphStoreConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}
