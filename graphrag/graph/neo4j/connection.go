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

	// Read separate database configuration from config
	s.useSeparateDatabase = false // Default to using labels/namespaces
	if config.DriverConfig != nil {
		if useSeparateDB, ok := config.DriverConfig["use_separate_database"].(bool); ok {
			s.useSeparateDatabase = useSeparateDB
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

	// Detect if this is Neo4j Enterprise Edition
	s.isEnterpriseEdition, err = s.detectEnterpriseEdition(ctx, driver)
	if err != nil {
		driver.Close(ctx)
		return fmt.Errorf("failed to detect Neo4j edition: %w", err)
	}

	// Validate configuration: if separate database is required but not enterprise edition
	if s.useSeparateDatabase && !s.isEnterpriseEdition {
		driver.Close(ctx)
		return fmt.Errorf("separate database storage requires Neo4j Enterprise Edition, but connected to Community Edition. Please use Neo4j Enterprise Edition or set use_separate_database to false")
	}

	s.config = config
	s.driver = driver
	s.connected = true

	return nil
}

// detectEnterpriseEdition detects if the connected Neo4j instance is Enterprise Edition
func (s *Store) detectEnterpriseEdition(ctx context.Context, driver neo4j.DriverWithContext) (bool, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "system",
	})
	defer session.Close(ctx)

	// Try to run an enterprise-only command (SHOW DATABASES)
	// Community edition will return an error
	result, err := session.Run(ctx, "SHOW DATABASES YIELD name LIMIT 1", nil)
	if err != nil {
		// If error contains "Unsupported administration command", it's community edition
		if contains(err.Error(), "Unsupported administration command") ||
			contains(err.Error(), "Unknown procedure") ||
			contains(err.Error(), "There is no procedure") {
			return false, nil
		}
		// Other errors are actual connection/permission issues
		return false, fmt.Errorf("failed to detect edition: %w", err)
	}

	// If we can execute SHOW DATABASES, it's enterprise edition
	_ = result.Next(ctx) // Consume one record if available
	return true, nil
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0
}

func indexOfSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
	s.useSeparateDatabase = false
	s.isEnterpriseEdition = false
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
