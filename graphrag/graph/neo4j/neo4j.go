package neo4j

import (
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// Store implements the GraphStore interface for Neo4j
type Store struct {
	config     types.GraphStoreConfig
	driver     neo4j.DriverWithContext
	connected  bool
	enterprise bool // Whether this is Neo4j Enterprise Edition
	mu         sync.RWMutex
}

// NewStore creates a new Neo4j graph store instance
func NewStore() *Store {
	return &Store{}
}

// GetDriver returns the underlying Neo4j driver
func (s *Store) GetDriver() neo4j.DriverWithContext {
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

// Enterprise returns whether this is Neo4j Enterprise Edition
func (s *Store) Enterprise() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enterprise
}

// SetEnterprise sets the enterprise flag
func (s *Store) SetEnterprise(enterprise bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enterprise = enterprise
}
