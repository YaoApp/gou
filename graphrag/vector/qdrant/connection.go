package qdrant

import (
	"context"
	"fmt"
	"strconv"

	"github.com/qdrant/go-client/qdrant"
	"github.com/yaoapp/gou/graphrag/types"
)

// Connect establishes connection to Qdrant server
func (s *Store) Connect(ctx context.Context, storeConfig ...types.VectorStoreConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	// Extract connection parameters from ExtraParams
	host := "localhost"
	port := 6334

	// If storeConfig is provided, use it, otherwise use the default config
	config := s.config
	if len(storeConfig) > 0 {
		config = storeConfig[0]
		s.config = config
	}

	if config.ExtraParams != nil {
		if h, ok := config.ExtraParams["host"].(string); ok && h != "" {
			host = h
		}
		if p, ok := config.ExtraParams["port"].(string); ok && p != "" {
			if portInt, err := strconv.Atoi(p); err == nil {
				port = portInt
			}
		}
		if p, ok := config.ExtraParams["port"].(int); ok {
			port = p
		}
	}

	// Create Qdrant client configuration
	clientConfig := &qdrant.Config{
		Host: host,
		Port: port,
	}

	// Add API key if provided
	if config.ExtraParams != nil {
		if apiKey, ok := config.ExtraParams["api_key"].(string); ok && apiKey != "" {
			clientConfig.APIKey = apiKey
		}
	}

	// Create high-level Qdrant client
	client, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	// Test connection with health check
	_, err = client.HealthCheck(ctx)
	if err != nil {
		client.Close()
		return fmt.Errorf("health check failed: %w", err)
	}

	s.config = config
	s.client = client
	s.connected = true

	return nil
}

// Disconnect closes the connection to Qdrant server
func (s *Store) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	if s.client != nil {
		if err := s.client.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	s.client = nil
	s.conn = nil
	s.connected = false

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

// tryConnect tries to connect to Qdrant server
func (s *Store) tryConnect(ctx context.Context) error {

	if s.connected {
		return nil
	}

	err := s.Connect(ctx, s.config)
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant server: %w", err)
	}
	return nil
}
