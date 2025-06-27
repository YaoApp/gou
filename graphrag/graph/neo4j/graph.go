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
	// GraphLabelPrefix is the prefix for graph labels in community edition
	GraphLabelPrefix = "Graph_"
	// GraphNamespaceProperty is the property name for graph namespace
	GraphNamespaceProperty = "graph_namespace"
)

// CreateGraph creates a new graph (database for enterprise, namespace/label for community)
func (s *Store) CreateGraph(ctx context.Context, graphName string, _ *types.GraphConfig) error {
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
		return fmt.Errorf("invalid graph name: %s (only alphanumeric and underscore allowed)", graphName)
	}

	if s.enterprise {
		// Enterprise edition: create a separate database
		return s.createEnterpriseGraph(ctx, graphName)
	}
	// Community edition: use namespace or labels
	return s.createCommunityGraph(ctx, graphName)
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

	if s.enterprise {
		// Enterprise edition: drop the database
		return s.dropEnterpriseGraph(ctx, graphName)
	}
	// Community edition: remove all nodes and relationships with the graph namespace/label
	return s.dropCommunityGraph(ctx, graphName)
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

	if s.enterprise {
		// Enterprise edition: check if database exists
		return s.enterpriseGraphExists(ctx, graphName)
	}
	// Community edition: check if any nodes exist with the graph namespace/label
	return s.communityGraphExists(ctx, graphName)
}

// ListGraphs returns a list of available graphs
func (s *Store) ListGraphs(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("store is not connected")
	}

	if s.enterprise {
		// Enterprise edition: list databases
		return s.listEnterpriseGraphs(ctx)
	}
	// Community edition: list unique graph namespaces/labels
	return s.listCommunityGraphs(ctx)
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

	if s.enterprise {
		// Enterprise edition: get database statistics
		return s.describeEnterpriseGraph(ctx, graphName)
	}
	// Community edition: get statistics for nodes/relationships with the graph namespace/label
	return s.describeCommunityGraph(ctx, graphName)
}

// Enterprise edition implementations

func (s *Store) createEnterpriseGraph(ctx context.Context, graphName string) error {
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
}

func (s *Store) dropEnterpriseGraph(ctx context.Context, graphName string) error {
	// Don't allow dropping the default database
	if graphName == DefaultDatabase {
		return fmt.Errorf("cannot drop default database '%s'", DefaultDatabase)
	}

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
}

func (s *Store) enterpriseGraphExists(ctx context.Context, graphName string) (bool, error) {
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

func (s *Store) listEnterpriseGraphs(ctx context.Context) ([]string, error) {
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

func (s *Store) describeEnterpriseGraph(ctx context.Context, graphName string) (*types.GraphStats, error) {
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
			"database_type": "enterprise",
			"database_name": graphName,
		},
	}, nil
}

// Community edition implementations

func (s *Store) createCommunityGraph(ctx context.Context, graphName string) error {
	// For community edition, we don't need to create anything explicitly
	// The graph namespace/label will be used when adding nodes/relationships
	// Just verify we can connect to the default database
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	// Create a constraint to ensure graph namespace uniqueness if it doesn't exist
	constraintQuery := fmt.Sprintf(
		"CREATE CONSTRAINT graph_namespace_unique IF NOT EXISTS FOR (n:%s) REQUIRE n.%s IS UNIQUE",
		GraphLabelPrefix+graphName,
		GraphNamespaceProperty,
	)

	_, err := session.Run(ctx, constraintQuery, nil)
	if err != nil {
		// Ignore constraint errors as they might already exist
		// Just log and continue
	}

	return nil
}

func (s *Store) dropCommunityGraph(ctx context.Context, graphName string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	graphLabel := GraphLabelPrefix + graphName

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
	constraintQuery := "DROP CONSTRAINT graph_namespace_unique IF EXISTS"
	_, _ = session.Run(ctx, constraintQuery, nil) // Ignore errors

	return nil
}

func (s *Store) communityGraphExists(ctx context.Context, graphName string) (bool, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	graphLabel := GraphLabelPrefix + graphName
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

func (s *Store) listCommunityGraphs(ctx context.Context) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	// Get all labels that start with our graph prefix
	query := "CALL db.labels() YIELD label WHERE label STARTS WITH $prefix RETURN label"
	result, err := session.Run(ctx, query, map[string]interface{}{
		"prefix": GraphLabelPrefix,
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
				graphName := strings.TrimPrefix(labelStr, GraphLabelPrefix)
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

func (s *Store) describeCommunityGraph(ctx context.Context, graphName string) (*types.GraphStats, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	graphLabel := GraphLabelPrefix + graphName

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
			"database_type": "community",
			"database_name": DefaultDatabase,
			"graph_label":   graphLabel,
			"namespace":     graphName,
		},
	}, nil
}

// Helper functions

// isValidGraphName checks if a graph name is valid (alphanumeric and underscore only)
func isValidGraphName(name string) bool {
	if len(name) == 0 {
		return false
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return true
}

// GetGraphLabel returns the label used for a graph in community edition
func (s *Store) GetGraphLabel(graphName string) string {
	return GraphLabelPrefix + graphName
}

// GetGraphDatabase returns the database name for a graph
func (s *Store) GetGraphDatabase(graphName string) string {
	if s.enterprise {
		return graphName
	}
	return DefaultDatabase
}
