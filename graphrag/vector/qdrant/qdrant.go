package qdrant

import (
	"sync"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"

	"github.com/yaoapp/gou/graphrag/types"
)

// Store implements the VectorStore interface for Qdrant
type Store struct {
	config    types.VectorStoreConfig
	client    *qdrant.Client
	conn      *grpc.ClientConn
	connected bool
	mu        sync.RWMutex
}

// NewStore creates a new Qdrant vector store instance
func NewStore() *Store {
	return &Store{}
}

// GetClient returns the underlying Qdrant client
func (s *Store) GetClient() *qdrant.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// GetConfig returns the current configuration
func (s *Store) GetConfig() types.VectorStoreConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}
