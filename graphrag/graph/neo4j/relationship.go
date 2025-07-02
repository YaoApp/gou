package neo4j

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// AddRelationships adds relationships to the graph
func (s *Store) AddRelationships(ctx context.Context, opts *types.AddRelationshipsOptions) ([]string, error) {
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

	if len(opts.Relationships) == 0 {
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

	// Ensure the graph exists
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		err = s.CreateGraph(ctx, opts.GraphName)
		if err != nil {
			return nil, fmt.Errorf("failed to create graph: %w", err)
		}
	}

	// Process relationships in batches
	var allRelIDs []string
	for i := 0; i < len(opts.Relationships); i += batchSize {
		end := i + batchSize
		if end > len(opts.Relationships) {
			end = len(opts.Relationships)
		}

		batch := opts.Relationships[i:end]
		relIDs, err := s.addRelationshipsBatch(ctx, opts.GraphName, batch, opts.Upsert, opts.CreateNodes)
		if err != nil {
			return nil, fmt.Errorf("failed to add relationships batch %d-%d: %w", i, end-1, err)
		}
		allRelIDs = append(allRelIDs, relIDs...)
	}

	return allRelIDs, nil
}

// addRelationshipsBatch adds a batch of relationships to the graph
func (s *Store) addRelationshipsBatch(ctx context.Context, graphName string, relationships []*types.GraphRelationship, upsert, createNodes bool) ([]string, error) {
	if len(relationships) == 0 {
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

	// Prepare relationship data
	relData := make([]map[string]interface{}, len(relationships))
	relIDs := make([]string, len(relationships))

	for i, rel := range relationships {
		if rel.ID == "" {
			// Generate ID if not provided
			rel.ID = fmt.Sprintf("%s_%s_%s", rel.StartNode, rel.Type, rel.EndNode)
		}

		if rel.StartNode == "" || rel.EndNode == "" {
			return nil, fmt.Errorf("start node and end node cannot be empty for relationship at index %d", i)
		}

		if rel.Type == "" {
			return nil, fmt.Errorf("relationship type cannot be empty at index %d", i)
		}

		relIDs[i] = rel.ID

		// Build relationship properties map
		properties := make(map[string]interface{})

		// Set the relationship ID as a property
		properties["id"] = rel.ID

		// Copy user-defined properties
		for k, v := range rel.Properties {
			properties[k] = v
		}

		// Add metadata fields
		if rel.Description != "" {
			properties["description"] = rel.Description
		}
		if rel.Confidence > 0 {
			properties["confidence"] = rel.Confidence
		}
		if rel.Weight > 0 {
			properties["weight"] = rel.Weight
		}
		if len(rel.Embedding) > 0 {
			properties["embedding"] = rel.Embedding
		}
		if len(rel.Embeddings) > 0 {
			properties["embeddings"] = rel.Embeddings
		}

		// Add timestamps
		now := time.Now().UTC()
		if rel.CreatedAt.IsZero() {
			properties["created_at"] = now.Unix()
		} else {
			properties["created_at"] = rel.CreatedAt.UTC().Unix()
		}
		properties["updated_at"] = now.Unix()

		// Add version
		if rel.Version <= 0 {
			properties["version"] = 1
		} else {
			properties["version"] = rel.Version
		}

		relData[i] = map[string]interface{}{
			"id":         rel.ID,
			"type":       rel.Type,
			"start_node": rel.StartNode,
			"end_node":   rel.EndNode,
			"properties": properties,
		}
	}

	// Execute relationships in transaction
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return s.executeBatchRelationshipOperation(ctx, tx, relData, upsert, createNodes, graphName)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute relationships transaction: %w", err)
	}

	return relIDs, nil
}

// executeBatchRelationshipOperation executes batch relationship create/upsert operation
func (s *Store) executeBatchRelationshipOperation(ctx context.Context, tx neo4j.ManagedTransaction, relDataList []map[string]interface{}, upsert, createNodes bool, graphName string) (interface{}, error) {
	if len(relDataList) == 0 {
		return nil, nil
	}

	// Prepare batch data
	batchData := make([]map[string]interface{}, len(relDataList))
	for i, relData := range relDataList {
		batchData[i] = map[string]interface{}{
			"id":         relData["id"],
			"type":       relData["type"],
			"start_node": relData["start_node"],
			"end_node":   relData["end_node"],
			"properties": relData["properties"],
		}
	}

	// Build match clauses for nodes
	var startNodeClause, endNodeClause string
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(graphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		startNodeClause = fmt.Sprintf("(start:%s {id: row.start_node})", escapedGraphLabel)
		endNodeClause = fmt.Sprintf("(end:%s {id: row.end_node})", escapedGraphLabel)
	} else {
		startNodeClause = "(start {id: row.start_node})"
		endNodeClause = "(end {id: row.end_node})"
	}

	var query string
	if createNodes {
		// Create nodes if they don't exist and then create relationship
		if !s.useSeparateDatabase {
			graphLabel := s.GetGraphLabel(graphName)
			escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"

			if upsert {
				query = fmt.Sprintf(`
					UNWIND $batch AS row
					MERGE (start:%s {id: row.start_node})
					MERGE (end:%s {id: row.end_node})
					MERGE (start)-[r:%s {id: row.id}]->(end)
					SET r = row.properties
					RETURN r.id AS id
				`, escapedGraphLabel, escapedGraphLabel, "`"+strings.ReplaceAll("RELATIONSHIP", "`", "``")+"`")
			} else {
				query = fmt.Sprintf(`
					UNWIND $batch AS row
					MERGE (start:%s {id: row.start_node})
					MERGE (end:%s {id: row.end_node})
					CREATE (start)-[r:%s]->(end)
					SET r = row.properties
					RETURN r.id AS id
				`, escapedGraphLabel, escapedGraphLabel, "`"+strings.ReplaceAll("RELATIONSHIP", "`", "``")+"`")
			}
		} else {
			if upsert {
				query = `
					UNWIND $batch AS row
					MERGE (start {id: row.start_node})
					MERGE (end {id: row.end_node})
					MERGE (start)-[r:GRAPH_RELATIONSHIP {id: row.id}]->(end)
					SET r = row.properties, r.type = row.type
					RETURN r.id AS id
				`
			} else {
				query = `
					UNWIND $batch AS row
					MERGE (start {id: row.start_node})
					MERGE (end {id: row.end_node})
					CREATE (start)-[r:GRAPH_RELATIONSHIP]->(end)
					SET r = row.properties, r.type = row.type
					RETURN r.id AS id
				`
			}
		}
	} else {
		// Require nodes to exist
		if upsert {
			query = fmt.Sprintf(`
				UNWIND $batch AS row
				MATCH %s
				MATCH %s
				MERGE (start)-[r:GRAPH_RELATIONSHIP {id: row.id}]->(end)
				SET r = row.properties, r.type = row.type
				RETURN r.id AS id
			`, startNodeClause, endNodeClause)
		} else {
			query = fmt.Sprintf(`
				UNWIND $batch AS row
				MATCH %s
				MATCH %s
				CREATE (start)-[r:GRAPH_RELATIONSHIP]->(end)
				SET r = row.properties, r.type = row.type
				RETURN r.id AS id
			`, startNodeClause, endNodeClause)
		}
	}

	// Execute batch query
	parameters := map[string]interface{}{
		"batch": batchData,
	}

	_, err := tx.Run(ctx, query, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to execute relationship batch: %w", err)
	}

	return nil, nil
}

// GetRelationships retrieves relationships from the graph
func (s *Store) GetRelationships(ctx context.Context, opts *types.GetRelationshipsOptions) ([]*types.GraphRelationship, error) {
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
		return []*types.GraphRelationship{}, nil
	}

	// Set default limit if not specified
	limit := opts.Limit
	if limit <= 0 {
		limit = 1000 // Default limit to prevent massive queries
	}

	return s.getRelationshipsFromGraph(ctx, opts, limit)
}

// getRelationshipsFromGraph retrieves relationships from the graph based on the options
func (s *Store) getRelationshipsFromGraph(ctx context.Context, opts *types.GetRelationshipsOptions, limit int) ([]*types.GraphRelationship, error) {
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
	query, parameters := s.buildGetRelationshipsQuery(opts, limit)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		records, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, err
		}

		var relationships []*types.GraphRelationship
		for records.Next(ctx) {
			record := records.Record()
			rel, err := s.parseRelationshipFromRecord(record, opts)
			if err != nil {
				return nil, err
			}
			relationships = append(relationships, rel)
		}

		if err = records.Err(); err != nil {
			return nil, err
		}

		return relationships, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute get relationships query: %w", err)
	}

	return result.([]*types.GraphRelationship), nil
}

// buildGetRelationshipsQuery builds the Cypher query for retrieving relationships
func (s *Store) buildGetRelationshipsQuery(opts *types.GetRelationshipsOptions, limit int) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause for nodes and relationships
	var matchClause string
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause = fmt.Sprintf("MATCH (start:%s)-[r]->(end:%s)", escapedGraphLabel, escapedGraphLabel)
	} else {
		matchClause = "MATCH (start)-[r:GRAPH_RELATIONSHIP]->(end)"
	}

	// Add relationship type filters
	if len(opts.Types) > 0 {
		typeFilters := make([]string, len(opts.Types))
		for i, relType := range opts.Types {
			escapedType := "`" + strings.ReplaceAll(relType, "`", "``") + "`"
			typeFilters[i] = ":" + escapedType
		}
		matchClause = strings.Replace(matchClause, "-[r]->", fmt.Sprintf("-[r%s]->", strings.Join(typeFilters, "|")), 1)
	}

	queryParts = append(queryParts, matchClause)

	// Add WHERE conditions
	var whereConditions []string

	// Filter by relationship IDs
	if len(opts.IDs) > 0 {
		whereConditions = append(whereConditions, "r.id IN $ids")
		parameters["ids"] = opts.IDs
	}

	// Filter by connected node IDs
	if len(opts.NodeIDs) > 0 {
		switch strings.ToUpper(opts.Direction) {
		case "IN", "INCOMING":
			whereConditions = append(whereConditions, "end.id IN $node_ids")
		case "OUT", "OUTGOING":
			whereConditions = append(whereConditions, "start.id IN $node_ids")
		default: // "BOTH" or empty
			whereConditions = append(whereConditions, "(start.id IN $node_ids OR end.id IN $node_ids)")
		}
		parameters["node_ids"] = opts.NodeIDs
	}

	// Filter by properties
	if len(opts.Filter) > 0 {
		for key, value := range opts.Filter {
			paramKey := "filter_" + strings.ReplaceAll(key, ".", "_")
			whereConditions = append(whereConditions, fmt.Sprintf("r.%s = $%s", key, paramKey))
			parameters[paramKey] = value
		}
	}

	if len(whereConditions) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))
	}

	// Add RETURN clause
	returnClause := "RETURN r, start.id AS start_id, end.id AS end_id"
	queryParts = append(queryParts, returnClause)

	// Add LIMIT
	queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", limit))

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// parseRelationshipFromRecord parses a Neo4j record into a GraphRelationship
func (s *Store) parseRelationshipFromRecord(record *neo4j.Record, opts *types.GetRelationshipsOptions) (*types.GraphRelationship, error) {
	relValue, ok := record.Get("r")
	if !ok {
		return nil, fmt.Errorf("relationship not found in record")
	}

	neo4jRel, ok := relValue.(neo4j.Relationship)
	if !ok {
		return nil, fmt.Errorf("invalid relationship type in record")
	}

	// Get start and end node IDs
	startID, ok := record.Get("start_id")
	if !ok {
		return nil, fmt.Errorf("start node ID not found in record")
	}
	endID, ok := record.Get("end_id")
	if !ok {
		return nil, fmt.Errorf("end node ID not found in record")
	}

	// Parse basic relationship information
	rel := &types.GraphRelationship{
		StartNode:  startID.(string),
		EndNode:    endID.(string),
		Properties: make(map[string]interface{}),
	}

	// Get relationship type from properties or fallback to Neo4j type
	if relType, exists := neo4jRel.Props["type"]; exists {
		if typeStr, ok := relType.(string); ok {
			rel.Type = typeStr
		}
	} else {
		rel.Type = neo4jRel.Type
	}

	// Get relationship ID
	if id, exists := neo4jRel.Props["id"]; exists {
		if idStr, ok := id.(string); ok {
			rel.ID = idStr
		}
	}
	if rel.ID == "" {
		rel.ID = fmt.Sprintf("rel_%d", neo4jRel.Id)
	}

	// Parse properties
	for key, value := range neo4jRel.Props {
		// Handle special fields
		switch key {
		case "id":
			// Already handled above
		case "type":
			// Already handled above
		case "description":
			if str, ok := value.(string); ok {
				rel.Description = str
			}
		case "confidence":
			if num, ok := value.(float64); ok {
				rel.Confidence = num
			}
		case "weight":
			if num, ok := value.(float64); ok {
				rel.Weight = num
			}
		case "embedding":
			if slice, ok := value.([]interface{}); ok {
				embedding := make([]float64, len(slice))
				for i, v := range slice {
					if f, ok := v.(float64); ok {
						embedding[i] = f
					}
				}
				rel.Embedding = embedding
			}
		case "created_at":
			if timestamp, ok := value.(int64); ok {
				rel.CreatedAt = time.Unix(timestamp, 0).UTC()
			}
		case "updated_at":
			if timestamp, ok := value.(int64); ok {
				rel.UpdatedAt = time.Unix(timestamp, 0).UTC()
			}
		case "version":
			if version, ok := value.(int64); ok {
				rel.Version = int(version)
			}
		default:
			// Include property if requested
			if opts.IncludeProperties {
				// Filter fields if specified
				if len(opts.Fields) == 0 || containsString(opts.Fields, key) {
					rel.Properties[key] = value
				}
			}
		}
	}

	// Include metadata if requested
	if !opts.IncludeMetadata {
		// Clear metadata fields for cleaner output
		rel.CreatedAt = time.Time{}
		rel.UpdatedAt = time.Time{}
		rel.Version = 0
	}

	return rel, nil
}

// DeleteRelationships deletes relationships from the graph
func (s *Store) DeleteRelationships(ctx context.Context, opts *types.DeleteRelationshipsOptions) error {
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

	// Both IDs and Filter cannot be empty (prevents accidental deletion of all relationships)
	if len(opts.IDs) == 0 && len(opts.Filter) == 0 {
		return fmt.Errorf("either IDs or Filter must be specified to prevent accidental deletion of all relationships")
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
		return s.performDeleteRelationshipsDryRun(ctx, opts)
	}

	return s.performDeleteRelationships(ctx, opts, batchSize)
}

// performDeleteRelationshipsDryRun performs a dry run of relationship deletion
func (s *Store) performDeleteRelationshipsDryRun(ctx context.Context, opts *types.DeleteRelationshipsOptions) error {
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	query, parameters := s.buildDeleteRelationshipsCountQuery(opts)

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
	return fmt.Errorf("dry run: would delete %d relationships", count)
}

// performDeleteRelationships performs the actual relationship deletion
func (s *Store) performDeleteRelationships(ctx context.Context, opts *types.DeleteRelationshipsOptions, batchSize int) error {
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
		return s.deleteRelationshipsByIDs(ctx, session, opts, batchSize)
	}

	// If deleting by filter, use single query
	if len(opts.Filter) > 0 {
		return s.deleteRelationshipsByFilter(ctx, session, opts)
	}

	return fmt.Errorf("no IDs or filters specified for deletion")
}

// deleteRelationshipsByIDs deletes relationships by their IDs in batches
func (s *Store) deleteRelationshipsByIDs(ctx context.Context, session neo4j.SessionWithContext, opts *types.DeleteRelationshipsOptions, batchSize int) error {
	for i := 0; i < len(opts.IDs); i += batchSize {
		end := i + batchSize
		if end > len(opts.IDs) {
			end = len(opts.IDs)
		}

		batchIDs := opts.IDs[i:end]
		query, parameters := s.buildDeleteRelationshipsByIDsQuery(opts, batchIDs)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			_, err := tx.Run(ctx, query, parameters)
			return nil, err
		})

		if err != nil {
			return fmt.Errorf("failed to delete relationships batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// deleteRelationshipsByFilter deletes relationships by filter criteria
func (s *Store) deleteRelationshipsByFilter(ctx context.Context, session neo4j.SessionWithContext, opts *types.DeleteRelationshipsOptions) error {
	query, parameters := s.buildDeleteRelationshipsByFilterQuery(opts)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, query, parameters)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to delete relationships by filter: %w", err)
	}

	return nil
}

// buildDeleteRelationshipsCountQuery builds a query to count relationships that would be deleted
func (s *Store) buildDeleteRelationshipsCountQuery(opts *types.DeleteRelationshipsOptions) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause
	var matchClause string
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause = fmt.Sprintf("MATCH (start:%s)-[r]->(end:%s)", escapedGraphLabel, escapedGraphLabel)
	} else {
		matchClause = "MATCH (start)-[r]->(end)"
	}

	queryParts = append(queryParts, matchClause)

	// Add WHERE conditions
	whereConditions := s.buildDeleteRelationshipsWhereConditions(opts, parameters)
	if len(whereConditions) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))
	}

	// Add RETURN count
	queryParts = append(queryParts, "RETURN count(r)")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildDeleteRelationshipsByIDsQuery builds query for deleting relationships by IDs
func (s *Store) buildDeleteRelationshipsByIDsQuery(opts *types.DeleteRelationshipsOptions, ids []string) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause
	var matchClause string
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause = fmt.Sprintf("MATCH (start:%s)-[r]->(end:%s)", escapedGraphLabel, escapedGraphLabel)
	} else {
		matchClause = "MATCH (start)-[r]->(end)"
	}

	queryParts = append(queryParts, matchClause)

	// Add WHERE clause for IDs
	queryParts = append(queryParts, "WHERE r.id IN $ids")
	parameters["ids"] = ids

	// Add DELETE clause
	queryParts = append(queryParts, "DELETE r")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildDeleteRelationshipsByFilterQuery builds query for deleting relationships by filter
func (s *Store) buildDeleteRelationshipsByFilterQuery(opts *types.DeleteRelationshipsOptions) (string, map[string]interface{}) {
	var queryParts []string
	parameters := make(map[string]interface{})

	// Build MATCH clause
	var matchClause string
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		matchClause = fmt.Sprintf("MATCH (start:%s)-[r]->(end:%s)", escapedGraphLabel, escapedGraphLabel)
	} else {
		matchClause = "MATCH (start)-[r]->(end)"
	}

	queryParts = append(queryParts, matchClause)

	// Add WHERE conditions
	whereConditions := s.buildDeleteRelationshipsWhereConditions(opts, parameters)
	if len(whereConditions) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))
	}

	// Add DELETE clause
	queryParts = append(queryParts, "DELETE r")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildDeleteRelationshipsWhereConditions builds WHERE conditions for delete queries
func (s *Store) buildDeleteRelationshipsWhereConditions(opts *types.DeleteRelationshipsOptions, parameters map[string]interface{}) []string {
	var whereConditions []string

	// Filter by IDs
	if len(opts.IDs) > 0 {
		whereConditions = append(whereConditions, "r.id IN $ids")
		parameters["ids"] = opts.IDs
	}

	// Filter by properties
	if len(opts.Filter) > 0 {
		for key, value := range opts.Filter {
			paramKey := "filter_" + strings.ReplaceAll(key, ".", "_")
			whereConditions = append(whereConditions, fmt.Sprintf("r.%s = $%s", key, paramKey))
			parameters[paramKey] = value
		}
	}

	return whereConditions
}
