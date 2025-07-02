package neo4j

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

const (
	// DefaultDatabase is the default database name for community edition
	DefaultDatabase = "neo4j"
	// DefaultGraphLabelPrefix is the default prefix for graph labels in label-based mode
	// Using double underscore to avoid conflicts with user-defined labels
	DefaultGraphLabelPrefix = "__Graph_"
	// DefaultGraphNamespaceProperty is the default property name for graph namespace
	// Using double underscore to avoid conflicts with user-defined properties
	DefaultGraphNamespaceProperty = "__graph_namespace"
)

// CreateGraph creates a new graph (database for enterprise, namespace/label for community)
func (s *Store) CreateGraph(ctx context.Context, graphName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return fmt.Errorf("store is not connected")
	}

	if graphName == "" {
		return fmt.Errorf("graph name cannot be empty")
	}

	// Validate graph name (no special characters except underscore)
	if !isValidGraphName(graphName) {
		return fmt.Errorf("invalid graph name: %s (only alphanumeric, underscore, and dash allowed)", graphName)
	}

	if s.useSeparateDatabase {
		// Use separate database for each graph (requires Enterprise Edition)
		return s.createSeparateDatabaseGraph(ctx, graphName)
	}
	// Use labels/namespaces in default database
	return s.createLabelBasedGraph(ctx, graphName)
}

// DropGraph drops a graph
func (s *Store) DropGraph(ctx context.Context, graphName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return fmt.Errorf("store is not connected")
	}

	if graphName == "" {
		return fmt.Errorf("graph name cannot be empty")
	}

	if s.useSeparateDatabase {
		// Drop the separate database
		return s.dropSeparateDatabaseGraph(ctx, graphName)
	}
	// Remove all nodes and relationships with the graph label/namespace
	return s.dropLabelBasedGraph(ctx, graphName)
}

// GraphExists checks if a graph exists
func (s *Store) GraphExists(ctx context.Context, graphName string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return false, fmt.Errorf("store is not connected")
	}

	if graphName == "" {
		return false, fmt.Errorf("graph name cannot be empty")
	}

	if s.useSeparateDatabase {
		// Check if separate database exists
		return s.separateDatabaseGraphExists(ctx, graphName)
	}
	// Check if any nodes exist with the graph label/namespace
	return s.labelBasedGraphExists(ctx, graphName)
}

// ListGraphs returns a list of available graphs
func (s *Store) ListGraphs(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("store is not connected")
	}

	if s.useSeparateDatabase {
		// List separate databases
		return s.listSeparateDatabaseGraphs(ctx)
	}
	// List unique graph labels/namespaces
	return s.listLabelBasedGraphs(ctx)
}

// DescribeGraph returns statistics about a graph
func (s *Store) DescribeGraph(ctx context.Context, graphName string) (*types.GraphStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("store is not connected")
	}

	if graphName == "" {
		return nil, fmt.Errorf("graph name cannot be empty")
	}

	if s.useSeparateDatabase {
		// Get separate database statistics
		return s.describeSeparateDatabaseGraph(ctx, graphName)
	}
	// Get statistics for nodes/relationships with the graph label/namespace
	return s.describeLabelBasedGraph(ctx, graphName)
}

// Separate database implementations (requires Enterprise Edition)

func (s *Store) createSeparateDatabaseGraph(ctx context.Context, graphName string) error {
	// Use critical operation semaphore to serialize database creation and avoid conflicts
	return executeCriticalOperation(ctx, func() error {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: "system", // Use system database for administrative operations
		})
		defer session.Close(ctx)

		// Create database
		query := fmt.Sprintf("CREATE DATABASE `%s`", graphName)
		_, err := session.Run(ctx, query, nil)
		if err != nil {
			// Check if database already exists
			if strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("graph '%s' already exists", graphName)
			}
			return fmt.Errorf("failed to create database '%s': %w", graphName, err)
		}

		return nil
	})
}

func (s *Store) dropSeparateDatabaseGraph(ctx context.Context, graphName string) error {
	// Don't allow dropping the default database
	if graphName == DefaultDatabase {
		return fmt.Errorf("cannot drop default database '%s'", DefaultDatabase)
	}

	// Use critical operation semaphore to serialize database drop and avoid conflicts
	return executeCriticalOperation(ctx, func() error {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: "system",
		})
		defer session.Close(ctx)

		// Drop database
		query := fmt.Sprintf("DROP DATABASE `%s`", graphName)
		_, err := session.Run(ctx, query, nil)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				return fmt.Errorf("graph '%s' does not exist", graphName)
			}
			return fmt.Errorf("failed to drop database '%s': %w", graphName, err)
		}

		return nil
	})
}

func (s *Store) separateDatabaseGraphExists(ctx context.Context, graphName string) (bool, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "system",
	})
	defer session.Close(ctx)

	query := "SHOW DATABASES YIELD name WHERE name = $name"
	result, err := session.Run(ctx, query, map[string]interface{}{
		"name": graphName,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	return result.Next(ctx), nil
}

func (s *Store) listSeparateDatabaseGraphs(ctx context.Context) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "system",
	})
	defer session.Close(ctx)

	query := "SHOW DATABASES YIELD name"
	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	var graphs []string
	for result.Next(ctx) {
		record := result.Record()
		if name, ok := record.Get("name"); ok {
			if nameStr, ok := name.(string); ok {
				graphs = append(graphs, nameStr)
			}
		}
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate database list: %w", err)
	}

	return graphs, nil
}

func (s *Store) describeSeparateDatabaseGraph(ctx context.Context, graphName string) (*types.GraphStats, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: graphName,
	})
	defer session.Close(ctx)

	// Get node count
	nodeResult, err := session.Run(ctx, "MATCH (n) RETURN count(n) as nodeCount", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get node count: %w", err)
	}

	var nodeCount int64
	if nodeResult.Next(ctx) {
		if count, ok := nodeResult.Record().Get("nodeCount"); ok {
			if countInt, ok := count.(int64); ok {
				nodeCount = countInt
			}
		}
	}

	// Get relationship count
	relResult, err := session.Run(ctx, "MATCH ()-[r]->() RETURN count(r) as relCount", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship count: %w", err)
	}

	var relCount int64
	if relResult.Next(ctx) {
		if count, ok := relResult.Record().Get("relCount"); ok {
			if countInt, ok := count.(int64); ok {
				relCount = countInt
			}
		}
	}

	return &types.GraphStats{
		TotalNodes:         nodeCount,
		TotalRelationships: relCount,
		ExtraStats: map[string]interface{}{
			"storage_type":  "separate_database",
			"database_name": graphName,
		},
	}, nil
}

// Label-based implementations (works with Community and Enterprise Edition)

func (s *Store) createLabelBasedGraph(ctx context.Context, graphName string) error {
	// Use critical operation semaphore to serialize constraint creation and avoid conflicts
	return executeCriticalOperation(ctx, func() error {
		// For label-based storage, we don't need to create anything explicitly
		// The graph label will be used when adding nodes/relationships
		// Just verify we can connect to the default database
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
		defer session.Close(ctx)

		// Create a constraint to ensure graph namespace uniqueness if it doesn't exist
		constraintQuery := fmt.Sprintf(
			"CREATE CONSTRAINT __graph_namespace_unique IF NOT EXISTS FOR (n:%s) REQUIRE n.%s IS UNIQUE",
			s.getGraphLabelPrefix()+graphName,
			s.getGraphNamespaceProperty(),
		)

		_, err := session.Run(ctx, constraintQuery, nil)
		if err != nil {
			// Ignore constraint errors as they might already exist
			// Just log and continue
		}

		return nil
	})
}

func (s *Store) dropLabelBasedGraph(ctx context.Context, graphName string) error {
	// Use critical operation semaphore to serialize graph deletion and constraint drop
	return executeCriticalOperation(ctx, func() error {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
		defer session.Close(ctx)

		graphLabel := s.getGraphLabelPrefix() + graphName

		// Delete all relationships first
		relQuery := fmt.Sprintf("MATCH (n:%s)-[r]-() DELETE r", graphLabel)
		_, err := session.Run(ctx, relQuery, nil)
		if err != nil {
			return fmt.Errorf("failed to delete relationships for graph '%s': %w", graphName, err)
		}

		// Delete all nodes
		nodeQuery := fmt.Sprintf("MATCH (n:%s) DELETE n", graphLabel)
		_, err = session.Run(ctx, nodeQuery, nil)
		if err != nil {
			return fmt.Errorf("failed to delete nodes for graph '%s': %w", graphName, err)
		}

		// Drop constraint if exists
		constraintQuery := "DROP CONSTRAINT __graph_namespace_unique IF EXISTS"
		_, _ = session.Run(ctx, constraintQuery, nil) // Ignore errors

		return nil
	})
}

func (s *Store) labelBasedGraphExists(ctx context.Context, graphName string) (bool, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	graphLabel := s.getGraphLabelPrefix() + graphName
	query := fmt.Sprintf("MATCH (n:%s) RETURN count(n) > 0 as exists LIMIT 1", graphLabel)

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check graph existence: %w", err)
	}

	if result.Next(ctx) {
		if exists, ok := result.Record().Get("exists"); ok {
			if existsBool, ok := exists.(bool); ok {
				return existsBool, nil
			}
		}
	}

	return false, nil
}

func (s *Store) listLabelBasedGraphs(ctx context.Context) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	// Get all labels that start with our graph prefix
	query := "CALL db.labels() YIELD label WHERE label STARTS WITH $prefix RETURN label"
	result, err := session.Run(ctx, query, map[string]interface{}{
		"prefix": s.getGraphLabelPrefix(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list graph labels: %w", err)
	}

	var graphs []string
	for result.Next(ctx) {
		record := result.Record()
		if label, ok := record.Get("label"); ok {
			if labelStr, ok := label.(string); ok {
				// Remove the prefix to get the graph name
				graphName := strings.TrimPrefix(labelStr, s.getGraphLabelPrefix())
				if graphName != labelStr { // Ensure it actually had the prefix
					graphs = append(graphs, graphName)
				}
			}
		}
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate graph labels: %w", err)
	}

	return graphs, nil
}

func (s *Store) describeLabelBasedGraph(ctx context.Context, graphName string) (*types.GraphStats, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	graphLabel := s.getGraphLabelPrefix() + graphName

	// Get node count
	nodeQuery := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as nodeCount", graphLabel)
	nodeResult, err := session.Run(ctx, nodeQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get node count: %w", err)
	}

	var nodeCount int64
	if nodeResult.Next(ctx) {
		if count, ok := nodeResult.Record().Get("nodeCount"); ok {
			if countInt, ok := count.(int64); ok {
				nodeCount = countInt
			}
		}
	}

	// Get relationship count
	relQuery := fmt.Sprintf("MATCH (n:%s)-[r]-() RETURN count(r) as relCount", graphLabel)
	relResult, err := session.Run(ctx, relQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship count: %w", err)
	}

	var relCount int64
	if relResult.Next(ctx) {
		if count, ok := relResult.Record().Get("relCount"); ok {
			if countInt, ok := count.(int64); ok {
				relCount = countInt
			}
		}
	}

	return &types.GraphStats{
		TotalNodes:         nodeCount,
		TotalRelationships: relCount,
		ExtraStats: map[string]interface{}{
			"storage_type":  "label_based",
			"database_name": DefaultDatabase,
			"__graph_label": graphLabel,
			"namespace":     graphName,
		},
	}, nil
}

// Helper functions

// isValidGraphName checks if a graph name is valid (alphanumeric, underscore, and dash only)
func isValidGraphName(name string) bool {
	if len(name) == 0 {
		return false
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}

	return true
}

// GetGraphLabel returns the label used for a graph in community edition
func (s *Store) GetGraphLabel(graphName string) string {
	return s.getGraphLabelPrefix() + graphName
}

// GetGraphDatabase returns the database name for a graph
func (s *Store) GetGraphDatabase(graphName string) string {
	if s.useSeparateDatabase {
		return graphName
	}
	return DefaultDatabase
}

// getGraphLabelPrefix returns the graph label prefix from config or default
func (s *Store) getGraphLabelPrefix() string {
	if s.config.DriverConfig != nil {
		if prefix, ok := s.config.DriverConfig["graph_label_prefix"].(string); ok && prefix != "" {
			return prefix
		}
	}
	return DefaultGraphLabelPrefix
}

// getGraphNamespaceProperty returns the graph namespace property from config or default
func (s *Store) getGraphNamespaceProperty() string {
	if s.config.DriverConfig != nil {
		if prop, ok := s.config.DriverConfig["graph_namespace_property"].(string); ok && prop != "" {
			return prop
		}
	}
	return DefaultGraphNamespaceProperty
}
