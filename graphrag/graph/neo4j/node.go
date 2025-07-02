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
		err = s.CreateGraph(ctx, opts.GraphName)
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

		// Set the node ID as a property
		properties["id"] = node.ID

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

	// Validate graph name
	if !isValidGraphName(opts.GraphName) {
		return nil, fmt.Errorf("invalid graph name: %s (only alphanumeric, underscore, and dash allowed)", opts.GraphName)
	}

	// Check if graph exists
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return []*types.GraphNode{}, nil // Return empty result for non-existent graph
	}

	// Set default limit if not specified
	limit := opts.Limit
	if limit <= 0 {
		limit = 1000 // Default limit to prevent massive queries
	}

	return s.getNodesFromGraph(ctx, opts, limit)
}

// getNodesFromGraph retrieves nodes from the graph based on the options
func (s *Store) getNodesFromGraph(ctx context.Context, opts *types.GetNodesOptions, limit int) ([]*types.GraphNode, error) {
	// Choose session configuration based on separate database mode
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// Build query based on filtering options
	query, parameters := s.buildGetNodesQuery(opts, limit)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		records, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, err
		}

		var nodes []*types.GraphNode
		for records.Next(ctx) {
			record := records.Record()
			node, err := s.parseNodeFromRecord(record, opts)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		if err = records.Err(); err != nil {
			return nil, err
		}

		return nodes, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute get nodes query: %w", err)
	}

	return result.([]*types.GraphNode), nil
}

// buildGetNodesQuery builds the Cypher query for retrieving nodes
func (s *Store) buildGetNodesQuery(opts *types.GetNodesOptions, limit int) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause
	matchClause := "MATCH (n"

	// Add label filters
	if len(opts.Labels) > 0 {
		for i, label := range opts.Labels {
			escapedLabel := "`" + strings.ReplaceAll(label, "`", "``") + "`"
			matchClause += ":" + escapedLabel
			parameters[fmt.Sprintf("label_%d", i)] = label
		}
	}

	// Add graph label for label-based storage
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause += ":" + escapedGraphLabel
	}

	matchClause += ")"
	queryParts = append(queryParts, matchClause)

	// Add WHERE conditions
	var whereConditions []string

	// Filter by IDs
	if len(opts.IDs) > 0 {
		whereConditions = append(whereConditions, "n.id IN $ids")
		parameters["ids"] = opts.IDs
	}

	// Filter by properties
	if len(opts.Filter) > 0 {
		for key, value := range opts.Filter {
			paramKey := "filter_" + strings.ReplaceAll(key, ".", "_")
			whereConditions = append(whereConditions, fmt.Sprintf("n.%s = $%s", key, paramKey))
			parameters[paramKey] = value
		}
	}

	if len(whereConditions) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))
	}

	// Add RETURN clause
	returnClause := "RETURN n"
	queryParts = append(queryParts, returnClause)

	// Add LIMIT
	queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", limit))

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// parseNodeFromRecord parses a Neo4j record into a GraphNode
func (s *Store) parseNodeFromRecord(record *neo4j.Record, opts *types.GetNodesOptions) (*types.GraphNode, error) {
	nodeValue, ok := record.Get("n")
	if !ok {
		return nil, fmt.Errorf("node not found in record")
	}

	neo4jNode, ok := nodeValue.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("invalid node type in record")
	}

	// Parse basic node information
	node := &types.GraphNode{
		Properties: make(map[string]interface{}),
	}

	// Get node ID - try multiple ways to get it
	if id, exists := neo4jNode.Props["id"]; exists {
		if idStr, ok := id.(string); ok {
			node.ID = idStr
		}
	}
	// If ID is still empty, this might be an issue with node creation
	// For debugging, let's also check if the node has an ID property at all
	if node.ID == "" {
		// Log available properties for debugging
		// In production, we might want to use the Neo4j internal ID as fallback
		node.ID = fmt.Sprintf("node_%d", neo4jNode.Id)
	}

	// Get labels (filter out graph label for label-based storage)
	labels := neo4jNode.Labels
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		var filteredLabels []string
		for _, label := range labels {
			if label != graphLabel {
				filteredLabels = append(filteredLabels, label)
			}
		}
		labels = filteredLabels
	}
	node.Labels = labels

	// Parse properties
	for key, value := range neo4jNode.Props {
		// Handle special fields
		switch key {
		case "id":
			// Already handled above
		case "entity_type":
			if str, ok := value.(string); ok {
				node.EntityType = str
			}
		case "description":
			if str, ok := value.(string); ok {
				node.Description = str
			}
		case "confidence":
			if num, ok := value.(float64); ok {
				node.Confidence = num
			}
		case "importance":
			if num, ok := value.(float64); ok {
				node.Importance = num
			}
		case "embedding":
			if slice, ok := value.([]interface{}); ok {
				embedding := make([]float64, len(slice))
				for i, v := range slice {
					if f, ok := v.(float64); ok {
						embedding[i] = f
					}
				}
				node.Embedding = embedding
			}
		case "created_at":
			if timestamp, ok := value.(int64); ok {
				node.CreatedAt = time.Unix(timestamp, 0).UTC()
			}
		case "updated_at":
			if timestamp, ok := value.(int64); ok {
				node.UpdatedAt = time.Unix(timestamp, 0).UTC()
			}
		case "version":
			if version, ok := value.(int64); ok {
				node.Version = int(version)
			}
		default:
			// Include property if requested
			if opts.IncludeProperties {
				// Filter fields if specified
				if len(opts.Fields) == 0 || containsString(opts.Fields, key) {
					node.Properties[key] = value
				}
			}
		}
	}

	// Include metadata if requested
	if !opts.IncludeMetadata {
		// Clear metadata fields for cleaner output
		node.CreatedAt = time.Time{}
		node.UpdatedAt = time.Time{}
		node.Version = 0
	}

	return node, nil
}

// DeleteNodes deletes nodes from the graph
func (s *Store) DeleteNodes(ctx context.Context, opts *types.DeleteNodesOptions) error {
	s.mu.RLock()
	connected := s.connected
	s.mu.RUnlock()

	if !connected {
		return fmt.Errorf("store is not connected")
	}

	if opts == nil {
		return fmt.Errorf("options cannot be nil")
	}

	if opts.GraphName == "" {
		return fmt.Errorf("graph name cannot be empty")
	}

	// Validate graph name
	if !isValidGraphName(opts.GraphName) {
		return fmt.Errorf("invalid graph name: %s (only alphanumeric, underscore, and dash allowed)", opts.GraphName)
	}

	// Both IDs and Filter cannot be empty (prevents accidental deletion of all nodes)
	// This validation should happen before checking graph existence
	if len(opts.IDs) == 0 && len(opts.Filter) == 0 {
		return fmt.Errorf("either IDs or Filter must be specified to prevent accidental deletion of all nodes")
	}

	// Check if graph exists
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return nil // No error for deleting from non-existent graph
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

	if opts.DryRun {
		return s.performDeleteNodesDryRun(ctx, opts)
	}

	return s.performDeleteNodes(ctx, opts, batchSize)
}

// performDeleteNodesDryRun performs a dry run of node deletion to show what would be deleted
func (s *Store) performDeleteNodesDryRun(ctx context.Context, opts *types.DeleteNodesOptions) error {
	// Build a query to count what would be deleted
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	query, parameters := s.buildDeleteNodesCountQuery(opts)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, err
		}

		if record.Next(ctx) {
			return record.Record().Values[0], nil
		}
		return 0, nil
	})

	if err != nil {
		return fmt.Errorf("failed to execute dry run query: %w", err)
	}

	count := result.(int64)
	return fmt.Errorf("dry run: would delete %d nodes", count)
}

// performDeleteNodes performs the actual node deletion
func (s *Store) performDeleteNodes(ctx context.Context, opts *types.DeleteNodesOptions, batchSize int) error {
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// If deleting by IDs, process in batches
	if len(opts.IDs) > 0 {
		return s.deleteNodesByIDs(ctx, session, opts, batchSize)
	}

	// If deleting by filter, use single query (be careful with large datasets)
	if len(opts.Filter) > 0 {
		return s.deleteNodesByFilter(ctx, session, opts)
	}

	// This should not happen due to earlier validation, but just in case
	return fmt.Errorf("no IDs or filters specified for deletion")
}

// deleteNodesByIDs deletes nodes by their IDs in batches
func (s *Store) deleteNodesByIDs(ctx context.Context, session neo4j.SessionWithContext, opts *types.DeleteNodesOptions, batchSize int) error {
	for i := 0; i < len(opts.IDs); i += batchSize {
		end := i + batchSize
		if end > len(opts.IDs) {
			end = len(opts.IDs)
		}

		batchIDs := opts.IDs[i:end]
		query, parameters := s.buildDeleteNodesByIDsQuery(opts, batchIDs)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			_, err := tx.Run(ctx, query, parameters)
			return nil, err
		})

		if err != nil {
			return fmt.Errorf("failed to delete nodes batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// deleteNodesByFilter deletes nodes by filter criteria
func (s *Store) deleteNodesByFilter(ctx context.Context, session neo4j.SessionWithContext, opts *types.DeleteNodesOptions) error {
	query, parameters := s.buildDeleteNodesByFilterQuery(opts)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, query, parameters)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to delete nodes by filter: %w", err)
	}

	return nil
}

// buildDeleteNodesCountQuery builds a query to count nodes that would be deleted
func (s *Store) buildDeleteNodesCountQuery(opts *types.DeleteNodesOptions) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause
	matchClause := "MATCH (n"

	// Add graph label for label-based storage
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause += ":" + escapedGraphLabel
	}

	matchClause += ")"
	queryParts = append(queryParts, matchClause)

	// Add WHERE conditions
	whereConditions := s.buildDeleteWhereConditions(opts, parameters)
	if len(whereConditions) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))
	}

	// Add RETURN count
	queryParts = append(queryParts, "RETURN count(n)")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildDeleteNodesByIDsQuery builds query for deleting nodes by IDs
func (s *Store) buildDeleteNodesByIDsQuery(opts *types.DeleteNodesOptions, ids []string) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause
	matchClause := "MATCH (n"

	// Add graph label for label-based storage
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause += ":" + escapedGraphLabel
	}

	matchClause += ")"
	queryParts = append(queryParts, matchClause)

	// Add WHERE clause for IDs
	queryParts = append(queryParts, "WHERE n.id IN $ids")
	parameters["ids"] = ids

	// Optionally delete relationships first
	if opts.DeleteRels {
		queryParts = append(queryParts, "DETACH DELETE n")
	} else {
		queryParts = append(queryParts, "DELETE n")
	}

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildDeleteNodesByFilterQuery builds query for deleting nodes by filter
func (s *Store) buildDeleteNodesByFilterQuery(opts *types.DeleteNodesOptions) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause
	matchClause := "MATCH (n"

	// Add graph label for label-based storage
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause += ":" + escapedGraphLabel
	}

	matchClause += ")"
	queryParts = append(queryParts, matchClause)

	// Add WHERE conditions
	whereConditions := s.buildDeleteWhereConditions(opts, parameters)
	if len(whereConditions) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))
	}

	// Optionally delete relationships first
	if opts.DeleteRels {
		queryParts = append(queryParts, "DETACH DELETE n")
	} else {
		queryParts = append(queryParts, "DELETE n")
	}

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildDeleteWhereConditions builds WHERE conditions for delete queries
func (s *Store) buildDeleteWhereConditions(opts *types.DeleteNodesOptions, parameters map[string]interface{}) []string {
	var whereConditions []string

	// Filter by IDs
	if len(opts.IDs) > 0 {
		whereConditions = append(whereConditions, "n.id IN $ids")
		parameters["ids"] = opts.IDs
	}

	// Filter by properties
	if len(opts.Filter) > 0 {
		for key, value := range opts.Filter {
			paramKey := "filter_" + strings.ReplaceAll(key, ".", "_")
			whereConditions = append(whereConditions, fmt.Sprintf("n.%s = $%s", key, paramKey))
			parameters[paramKey] = value
		}
	}

	return whereConditions
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
