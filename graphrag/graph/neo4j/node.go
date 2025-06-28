package neo4j

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// AddNodes adds nodes to the graph
func (s *Store) AddNodes(ctx context.Context, opts *types.AddNodesOptions) ([]string, error) {
	s.mu.RLock()
	connected := s.connected
	s.mu.RUnlock()

	if !connected {
		return nil, fmt.Errorf("store is not connected")
	}

	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	if opts.GraphName == "" {
		return nil, fmt.Errorf("graph name cannot be empty")
	}

	if len(opts.Nodes) == 0 {
		return []string{}, nil
	}

	// Validate graph name
	if !isValidGraphName(opts.GraphName) {
		return nil, fmt.Errorf("invalid graph name: %s (only alphanumeric, underscore, and dash allowed)", opts.GraphName)
	}

	// Set default batch size if not specified
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	// Set timeout context if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.Timeout)*time.Second)
		defer cancel()
	}

	// Ensure the graph exists (no lock needed for this check)
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		// CreateGraph will handle its own locking
		err = s.CreateGraph(ctx, opts.GraphName, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create graph: %w", err)
		}
	}

	// Process nodes in batches
	var allNodeIDs []string
	for i := 0; i < len(opts.Nodes); i += batchSize {
		end := i + batchSize
		if end > len(opts.Nodes) {
			end = len(opts.Nodes)
		}

		batch := opts.Nodes[i:end]
		nodeIDs, err := s.addNodesBatch(ctx, opts.GraphName, batch, opts.Upsert)
		if err != nil {
			return nil, fmt.Errorf("failed to add nodes batch %d-%d: %w", i, end-1, err)
		}
		allNodeIDs = append(allNodeIDs, nodeIDs...)
	}

	return allNodeIDs, nil
}

// addNodesBatch adds a batch of nodes to the graph
func (s *Store) addNodesBatch(ctx context.Context, graphName string, nodes []*types.GraphNode, upsert bool) ([]string, error) {
	if len(nodes) == 0 {
		return []string{}, nil
	}

	// Choose session configuration based on separate database mode
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = graphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// Prepare node data
	nodeData := make([]map[string]interface{}, len(nodes))
	nodeIDs := make([]string, len(nodes))

	for i, node := range nodes {
		if node.ID == "" {
			return nil, fmt.Errorf("node ID cannot be empty at index %d", i)
		}

		nodeIDs[i] = node.ID

		// Build node properties map
		properties := make(map[string]interface{})

		// Copy user-defined properties
		for k, v := range node.Properties {
			properties[k] = v
		}

		// Add metadata fields
		if node.EntityType != "" {
			properties["entity_type"] = node.EntityType
		}
		if node.Description != "" {
			properties["description"] = node.Description
		}
		if node.Confidence > 0 {
			properties["confidence"] = node.Confidence
		}
		if node.Importance > 0 {
			properties["importance"] = node.Importance
		}
		if len(node.Embedding) > 0 {
			properties["embedding"] = node.Embedding
		}
		if len(node.Embeddings) > 0 {
			properties["embeddings"] = node.Embeddings
		}

		// Add timestamps
		now := time.Now().UTC() // Use UTC to avoid timezone issues
		if node.CreatedAt.IsZero() {
			properties["created_at"] = now.Unix() // Store as Unix timestamp
		} else {
			properties["created_at"] = node.CreatedAt.UTC().Unix()
		}
		properties["updated_at"] = now.Unix() // Store as Unix timestamp

		// Add version
		if node.Version <= 0 {
			properties["version"] = 1
		} else {
			properties["version"] = node.Version
		}

		// Prepare labels
		labels := make([]string, len(node.Labels))
		copy(labels, node.Labels)

		// Add graph label if using label-based storage
		if !s.useSeparateDatabase {
			graphLabel := s.GetGraphLabel(graphName)
			labels = append(labels, graphLabel)
		}

		nodeData[i] = map[string]interface{}{
			"id":         node.ID,
			"labels":     labels,
			"properties": properties,
		}
	}

	// Execute nodes in transaction for better performance and consistency
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return s.executeBatchNodeOperation(ctx, tx, nodeData, upsert)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute nodes transaction: %w", err)
	}

	return nodeIDs, nil
}

// executeBatchNodeOperation executes batch node create/upsert operation using UNWIND
func (s *Store) executeBatchNodeOperation(ctx context.Context, tx neo4j.ManagedTransaction, nodeDataList []map[string]interface{}, upsert bool) (interface{}, error) {
	if len(nodeDataList) == 0 {
		return nil, nil
	}

	// Group nodes by their label combinations for efficient batching
	labelGroups := make(map[string][]map[string]interface{})

	for _, nodeData := range nodeDataList {
		labels := nodeData["labels"].([]string)
		// Create a key from sorted labels
		labelsKey := strings.Join(labels, "|")
		labelGroups[labelsKey] = append(labelGroups[labelsKey], nodeData)
	}

	// Process each label group with optimized batch queries
	for labelsKey, nodes := range labelGroups {
		if len(nodes) == 0 {
			continue
		}

		// Extract labels from the first node (all nodes in group have same labels)
		labels := nodes[0]["labels"].([]string)

		// Build the labels string for Cypher query
		labelsStr := ""
		if len(labels) > 0 {
			escapedLabels := make([]string, len(labels))
			for i, label := range labels {
				escapedLabels[i] = "`" + strings.ReplaceAll(label, "`", "``") + "`"
			}
			labelsStr = ":" + strings.Join(escapedLabels, ":")
		}

		// Prepare batch data
		batchData := make([]map[string]interface{}, len(nodes))
		for i, nodeData := range nodes {
			batchData[i] = map[string]interface{}{
				"id":         nodeData["id"],
				"properties": nodeData["properties"],
			}
		}

		// Build batch query using UNWIND
		var query string
		if upsert {
			// Use MERGE for upsert operation with UNWIND
			query = fmt.Sprintf(`
				UNWIND $batch AS row
				MERGE (n%s {id: row.id})
				SET n = row.properties
				RETURN n.id AS id
			`, labelsStr)
		} else {
			// Use CREATE for new nodes with UNWIND
			query = fmt.Sprintf(`
				UNWIND $batch AS row
				CREATE (n%s)
				SET n = row.properties
				RETURN n.id AS id
			`, labelsStr)
		}

		// Execute batch query
		parameters := map[string]interface{}{
			"batch": batchData,
		}

		_, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to execute batch for labels %s: %w", labelsKey, err)
		}
	}

	return nil, nil
}

// GetNodes retrieves nodes from the graph
func (s *Store) GetNodes(ctx context.Context, opts *types.GetNodesOptions) ([]*types.GraphNode, error) {
	// TODO: implement node retrieval
	return nil, nil
}

// DeleteNodes deletes nodes from the graph
func (s *Store) DeleteNodes(ctx context.Context, opts *types.DeleteNodesOptions) error {
	// TODO: implement node deletion
	return nil
}
