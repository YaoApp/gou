package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// Connect establishes connection to Neo4j server
func (s *Store) Connect(ctx context.Context, config types.GraphStoreConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	// Validate required configuration
	if config.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}

	// Extract connection parameters from config
	username := "neo4j"
	password := ""

	if config.DriverConfig != nil {
		if u, ok := config.DriverConfig["username"].(string); ok && u != "" {
			username = u
		}
		if p, ok := config.DriverConfig["password"].(string); ok {
			password = p
		}
	}

	if password == "" {
		return fmt.Errorf("password is required")
	}

	// Set enterprise edition from config
	s.enterprise = false // Default to community edition
	if config.DriverConfig != nil {
		if enterprise, ok := config.DriverConfig["enterprise"].(bool); ok {
			s.enterprise = enterprise
		}
	}

	// Create Neo4j driver
	auth := neo4j.BasicAuth(username, password, "")
	driver, err := neo4j.NewDriverWithContext(config.DatabaseURL, auth)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Test connection with a simple query
	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		driver.Close(ctx)
		return fmt.Errorf("connection test failed: %w", err)
	}

	s.config = config
	s.driver = driver
	s.connected = true

	return nil
}

// Disconnect closes the connection to Neo4j server
func (s *Store) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	// Close Neo4j driver
	if s.driver != nil {
		if err := s.driver.Close(ctx); err != nil {
			return fmt.Errorf("failed to close Neo4j driver: %w", err)
		}
	}

	s.connected = false
	s.enterprise = false
	s.config = types.GraphStoreConfig{}
	s.driver = nil

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
