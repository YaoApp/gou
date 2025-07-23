package neo4j

import (
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// Store implements the GraphStore interface for Neo4j
type Store struct {
	config              types.GraphStoreConfig
	driver              neo4j.DriverWithContext
	connected           bool
	useSeparateDatabase bool // Whether to use separate databases for each graph (requires Enterprise Edition)
	isEnterpriseEdition bool // Whether the connected Neo4j instance is Enterprise Edition
	mu                  sync.RWMutex
}

// NewStore creates a new Neo4j graph store instance
func NewStore() *Store {
	return &Store{}
}

// NewStoreWithConfig creates a new Neo4j graph store instance with a configuration
func NewStoreWithConfig(config types.GraphStoreConfig) *Store {
	return &Store{config: config}
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

// UseSeparateDatabase returns whether to use separate databases for each graph
func (s *Store) UseSeparateDatabase() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.useSeparateDatabase
}

// SetUseSeparateDatabase sets the separate database flag
func (s *Store) SetUseSeparateDatabase(useSeparateDatabase bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.useSeparateDatabase = useSeparateDatabase
}

// IsEnterpriseEdition returns whether the connected Neo4j instance is Enterprise Edition
func (s *Store) IsEnterpriseEdition() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isEnterpriseEdition
}

// SetIsEnterpriseEdition sets the enterprise edition flag
func (s *Store) SetIsEnterpriseEdition(isEnterprise bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isEnterpriseEdition = isEnterprise
}
