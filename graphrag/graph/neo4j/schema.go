package neo4j

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// GetSchema returns the schema of the graph
func (s *Store) GetSchema(ctx context.Context, graphName string) (*types.DynamicGraphSchema, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("store is not connected")
	}

	if graphName == "" {
		return nil, fmt.Errorf("graph name cannot be empty")
	}

	if s.useSeparateDatabase {
		return s.getSeparateDatabaseSchema(ctx, graphName)
	}
	return s.getLabelBasedSchema(ctx, graphName)
}

// CreateIndex creates an index on the graph
func (s *Store) CreateIndex(ctx context.Context, opts *types.CreateIndexOptions) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("store is not connected")
	}

	if opts == nil {
		return fmt.Errorf("create index options cannot be nil")
	}

	if opts.GraphName == "" {
		return fmt.Errorf("graph name cannot be empty")
	}

	if len(opts.Properties) == 0 {
		return fmt.Errorf("properties cannot be empty")
	}

	if s.useSeparateDatabase {
		return s.createSeparateDatabaseIndex(ctx, opts)
	}
	return s.createLabelBasedIndex(ctx, opts)
}

// DropIndex drops an index from the graph
func (s *Store) DropIndex(ctx context.Context, opts *types.DropIndexOptions) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("store is not connected")
	}

	if opts == nil {
		return fmt.Errorf("drop index options cannot be nil")
	}

	if opts.GraphName == "" {
		return fmt.Errorf("graph name cannot be empty")
	}

	if opts.Name == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	if s.useSeparateDatabase {
		return s.dropSeparateDatabaseIndex(ctx, opts)
	}
	return s.dropLabelBasedIndex(ctx, opts)
}

// Separate database implementations (Enterprise Edition)

func (s *Store) getSeparateDatabaseSchema(ctx context.Context, graphName string) (*types.DynamicGraphSchema, error) {
	// Check if graph/database exists
	exists, err := s.separateDatabaseGraphExists(ctx, graphName)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("graph '%s' does not exist", graphName)
	}

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: graphName,
	})
	defer session.Close(ctx)

	schema := &types.DynamicGraphSchema{
		NodeProperties: make(map[string][]types.PropertyInfo),
		RelProperties:  make(map[string][]types.PropertyInfo),
		Statistics:     &types.GraphSchemaStats{},
	}

	// Get node labels
	nodeLabels, err := s.getNodeLabels(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get node labels: %w", err)
	}
	schema.NodeLabels = nodeLabels

	// Get relationship types
	relTypes, err := s.getRelationshipTypes(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship types: %w", err)
	}
	schema.RelationshipTypes = relTypes

	// Get node properties for each label
	for _, label := range nodeLabels {
		props, err := s.getNodeProperties(ctx, session, label)
		if err != nil {
			return nil, fmt.Errorf("failed to get properties for node label '%s': %w", label, err)
		}
		schema.NodeProperties[label] = props
	}

	// Get relationship properties for each type
	for _, relType := range relTypes {
		props, err := s.getRelationshipProperties(ctx, session, relType)
		if err != nil {
			return nil, fmt.Errorf("failed to get properties for relationship type '%s': %w", relType, err)
		}
		schema.RelProperties[relType] = props
	}

	// Get constraints
	constraints, err := s.getConstraints(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints: %w", err)
	}
	schema.Constraints = constraints

	// Get indexes
	indexes, err := s.getIndexes(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	schema.Indexes = indexes

	// Get statistics
	stats, err := s.getSchemaStatistics(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema statistics: %w", err)
	}
	schema.Statistics = stats

	return schema, nil
}

func (s *Store) createSeparateDatabaseIndex(ctx context.Context, opts *types.CreateIndexOptions) error {
	// Check if graph/database exists
	exists, err := s.separateDatabaseGraphExists(ctx, opts.GraphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("graph '%s' does not exist", opts.GraphName)
	}

	// Use critical operation semaphore to serialize index creation and avoid deadlocks
	return executeCriticalOperation(ctx, func() error {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: opts.GraphName,
		})
		defer session.Close(ctx)

		return s.executeCreateIndex(ctx, session, opts)
	})
}

func (s *Store) dropSeparateDatabaseIndex(ctx context.Context, opts *types.DropIndexOptions) error {
	// Check if graph/database exists
	exists, err := s.separateDatabaseGraphExists(ctx, opts.GraphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		if opts.IfExists {
			return nil // Silently succeed if index doesn't exist
		}
		return fmt.Errorf("graph '%s' does not exist", opts.GraphName)
	}

	// Use critical operation semaphore to serialize index drop and avoid deadlocks
	return executeCriticalOperation(ctx, func() error {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: opts.GraphName,
		})
		defer session.Close(ctx)

		return s.executeDropIndex(ctx, session, opts)
	})
}

// Label-based implementations (Community Edition)

func (s *Store) getLabelBasedSchema(ctx context.Context, graphName string) (*types.DynamicGraphSchema, error) {
	// Check if graph exists
	exists, err := s.labelBasedGraphExists(ctx, graphName)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("graph '%s' does not exist", graphName)
	}

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: DefaultDatabase,
	})
	defer session.Close(ctx)

	graphLabel := s.getGraphLabelPrefix() + graphName

	schema := &types.DynamicGraphSchema{
		NodeProperties: make(map[string][]types.PropertyInfo),
		RelProperties:  make(map[string][]types.PropertyInfo),
		Statistics:     &types.GraphSchemaStats{},
	}

	// Get node labels (filtered by graph label)
	nodeLabels, err := s.getFilteredNodeLabels(ctx, session, graphLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to get node labels: %w", err)
	}
	schema.NodeLabels = nodeLabels

	// Get relationship types (filtered by graph)
	relTypes, err := s.getFilteredRelationshipTypes(ctx, session, graphLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship types: %w", err)
	}
	schema.RelationshipTypes = relTypes

	// Get node properties for each label (filtered by graph)
	for _, label := range nodeLabels {
		props, err := s.getFilteredNodeProperties(ctx, session, graphLabel, label)
		if err != nil {
			return nil, fmt.Errorf("failed to get properties for node label '%s': %w", label, err)
		}
		schema.NodeProperties[label] = props
	}

	// Get relationship properties for each type (filtered by graph)
	for _, relType := range relTypes {
		props, err := s.getFilteredRelationshipProperties(ctx, session, graphLabel, relType)
		if err != nil {
			return nil, fmt.Errorf("failed to get properties for relationship type '%s': %w", relType, err)
		}
		schema.RelProperties[relType] = props
	}

	// Get constraints (filtered by graph)
	constraints, err := s.getFilteredConstraints(ctx, session, graphLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints: %w", err)
	}
	schema.Constraints = constraints

	// Get indexes (filtered by graph)
	indexes, err := s.getFilteredIndexes(ctx, session, graphLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	schema.Indexes = indexes

	// Get statistics (filtered by graph)
	stats, err := s.getFilteredSchemaStatistics(ctx, session, graphLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema statistics: %w", err)
	}
	schema.Statistics = stats

	return schema, nil
}

func (s *Store) createLabelBasedIndex(ctx context.Context, opts *types.CreateIndexOptions) error {
	// Check if graph exists
	exists, err := s.labelBasedGraphExists(ctx, opts.GraphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("graph '%s' does not exist", opts.GraphName)
	}

	// Use critical operation semaphore to serialize index creation and avoid deadlocks
	return executeCriticalOperation(ctx, func() error {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
		defer session.Close(ctx)

		// For label-based graphs, we need to prefix labels with graph-specific prefix
		modifiedOpts := *opts
		if opts.Target == "NODE" {
			modifiedOpts.Labels = s.prefixLabelsWithGraph(opts.Labels, opts.GraphName)
		}

		return s.executeCreateIndex(ctx, session, &modifiedOpts)
	})
}

func (s *Store) dropLabelBasedIndex(ctx context.Context, opts *types.DropIndexOptions) error {
	// Check if graph exists
	exists, err := s.labelBasedGraphExists(ctx, opts.GraphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		if opts.IfExists {
			return nil // Silently succeed if graph doesn't exist
		}
		return fmt.Errorf("graph '%s' does not exist", opts.GraphName)
	}

	// Use critical operation semaphore to serialize index drop and avoid deadlocks
	return executeCriticalOperation(ctx, func() error {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
		defer session.Close(ctx)

		return s.executeDropIndex(ctx, session, opts)
	})
}

// Helper methods for schema retrieval

func (s *Store) getNodeLabels(ctx context.Context, session neo4j.SessionWithContext) ([]string, error) {
	result, err := session.Run(ctx, "CALL db.labels() YIELD label RETURN label ORDER BY label", nil)
	if err != nil {
		return nil, err
	}

	var labels []string
	for result.Next(ctx) {
		if label, ok := result.Record().Get("label"); ok {
			if labelStr, ok := label.(string); ok {
				labels = append(labels, labelStr)
			}
		}
	}

	return labels, result.Err()
}

func (s *Store) getRelationshipTypes(ctx context.Context, session neo4j.SessionWithContext) ([]string, error) {
	result, err := session.Run(ctx, "CALL db.relationshipTypes() YIELD relationshipType RETURN relationshipType ORDER BY relationshipType", nil)
	if err != nil {
		return nil, err
	}

	var types []string
	for result.Next(ctx) {
		if relType, ok := result.Record().Get("relationshipType"); ok {
			if relTypeStr, ok := relType.(string); ok {
				types = append(types, relTypeStr)
			}
		}
	}

	return types, result.Err()
}

func (s *Store) getNodeProperties(ctx context.Context, session neo4j.SessionWithContext, label string) ([]types.PropertyInfo, error) {
	query := "MATCH (n:" + label + ") UNWIND keys(n) AS key RETURN DISTINCT key, apoc.meta.cypher.type(n[key]) AS type, count(*) AS count ORDER BY key"

	// Fallback query if APOC is not available
	fallbackQuery := "MATCH (n:" + label + ") UNWIND keys(n) AS key WITH key, collect(DISTINCT n[key]) AS values RETURN key, 'mixed' AS type, size(values) AS count ORDER BY key"

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		// Try fallback query
		result, err = session.Run(ctx, fallbackQuery, nil)
		if err != nil {
			return nil, err
		}
	}

	var properties []types.PropertyInfo
	for result.Next(ctx) {
		record := result.Record()
		key, _ := record.Get("key")
		propType, _ := record.Get("type")
		count, _ := record.Get("count")

		prop := types.PropertyInfo{
			Name:     key.(string),
			Type:     propType.(string),
			Nullable: true, // Neo4j properties are generally nullable
		}

		if countInt, ok := count.(int64); ok {
			prop.Count = countInt
		}

		properties = append(properties, prop)
	}

	return properties, result.Err()
}

func (s *Store) getRelationshipProperties(ctx context.Context, session neo4j.SessionWithContext, relType string) ([]types.PropertyInfo, error) {
	query := "MATCH ()-[r:" + relType + "]-() UNWIND keys(r) AS key RETURN DISTINCT key, apoc.meta.cypher.type(r[key]) AS type, count(*) AS count ORDER BY key"

	// Fallback query if APOC is not available
	fallbackQuery := "MATCH ()-[r:" + relType + "]-() UNWIND keys(r) AS key WITH key, collect(DISTINCT r[key]) AS values RETURN key, 'mixed' AS type, size(values) AS count ORDER BY key"

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		// Try fallback query
		result, err = session.Run(ctx, fallbackQuery, nil)
		if err != nil {
			return nil, err
		}
	}

	var properties []types.PropertyInfo
	for result.Next(ctx) {
		record := result.Record()
		key, _ := record.Get("key")
		propType, _ := record.Get("type")
		count, _ := record.Get("count")

		prop := types.PropertyInfo{
			Name:     key.(string),
			Type:     propType.(string),
			Nullable: true,
		}

		if countInt, ok := count.(int64); ok {
			prop.Count = countInt
		}

		properties = append(properties, prop)
	}

	return properties, result.Err()
}

func (s *Store) getConstraints(ctx context.Context, session neo4j.SessionWithContext) ([]types.SchemaConstraint, error) {
	result, err := session.Run(ctx, "SHOW CONSTRAINTS YIELD name, type, entityType, labelsOrTypes, properties", nil)
	if err != nil {
		return nil, err
	}

	var constraints []types.SchemaConstraint
	for result.Next(ctx) {
		record := result.Record()

		constraintType, _ := record.Get("type")
		labelsOrTypes, _ := record.Get("labelsOrTypes")
		properties, _ := record.Get("properties")

		constraint := types.SchemaConstraint{
			Type: constraintType.(string),
		}

		if labels, ok := labelsOrTypes.([]interface{}); ok && len(labels) > 0 {
			if label, ok := labels[0].(string); ok {
				constraint.Label = label
			}
		}

		if props, ok := properties.([]interface{}); ok {
			for _, prop := range props {
				if propStr, ok := prop.(string); ok {
					constraint.Properties = append(constraint.Properties, propStr)
				}
			}
		}

		constraints = append(constraints, constraint)
	}

	return constraints, result.Err()
}

func (s *Store) getIndexes(ctx context.Context, session neo4j.SessionWithContext) ([]types.SchemaIndex, error) {
	result, err := session.Run(ctx, "SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties", nil)
	if err != nil {
		return nil, err
	}

	var indexes []types.SchemaIndex
	for result.Next(ctx) {
		record := result.Record()

		indexType, _ := record.Get("type")
		labelsOrTypes, _ := record.Get("labelsOrTypes")
		properties, _ := record.Get("properties")

		index := types.SchemaIndex{
			Type: indexType.(string),
		}

		if labels, ok := labelsOrTypes.([]interface{}); ok && len(labels) > 0 {
			if label, ok := labels[0].(string); ok {
				index.Label = label
			}
		}

		if props, ok := properties.([]interface{}); ok {
			for _, prop := range props {
				if propStr, ok := prop.(string); ok {
					index.Properties = append(index.Properties, propStr)
				}
			}
		}

		indexes = append(indexes, index)
	}

	return indexes, result.Err()
}

func (s *Store) getSchemaStatistics(ctx context.Context, session neo4j.SessionWithContext) (*types.GraphSchemaStats, error) {
	stats := &types.GraphSchemaStats{
		NodeCounts: make(map[string]int64),
		RelCounts:  make(map[string]int64),
	}

	// Get total node count
	nodeResult, err := session.Run(ctx, "MATCH (n) RETURN count(n) as total", nil)
	if err != nil {
		return nil, err
	}
	if nodeResult.Next(ctx) {
		if total, ok := nodeResult.Record().Get("total"); ok {
			if totalInt, ok := total.(int64); ok {
				stats.TotalNodes = totalInt
			}
		}
	}

	// Get total relationship count
	relResult, err := session.Run(ctx, "MATCH ()-[r]-() RETURN count(r) as total", nil)
	if err != nil {
		return nil, err
	}
	if relResult.Next(ctx) {
		if total, ok := relResult.Record().Get("total"); ok {
			if totalInt, ok := total.(int64); ok {
				stats.TotalRelationships = totalInt
			}
		}
	}

	return stats, nil
}

// Filtered methods for label-based graphs

func (s *Store) getFilteredNodeLabels(ctx context.Context, session neo4j.SessionWithContext, graphLabel string) ([]string, error) {
	query := fmt.Sprintf("MATCH (n:%s) UNWIND labels(n) AS label WITH DISTINCT label WHERE label <> $graphLabel RETURN label ORDER BY label", graphLabel)
	result, err := session.Run(ctx, query, map[string]interface{}{
		"graphLabel": graphLabel,
	})
	if err != nil {
		return nil, err
	}

	var labels []string
	for result.Next(ctx) {
		if label, ok := result.Record().Get("label"); ok {
			if labelStr, ok := label.(string); ok {
				labels = append(labels, labelStr)
			}
		}
	}

	return labels, result.Err()
}

func (s *Store) getFilteredRelationshipTypes(ctx context.Context, session neo4j.SessionWithContext, graphLabel string) ([]string, error) {
	query := fmt.Sprintf("MATCH (n:%s)-[r]-() RETURN DISTINCT type(r) AS relType ORDER BY relType", graphLabel)
	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var types []string
	for result.Next(ctx) {
		if relType, ok := result.Record().Get("relType"); ok {
			if relTypeStr, ok := relType.(string); ok {
				types = append(types, relTypeStr)
			}
		}
	}

	return types, result.Err()
}

func (s *Store) getFilteredNodeProperties(ctx context.Context, session neo4j.SessionWithContext, graphLabel, label string) ([]types.PropertyInfo, error) {
	query := fmt.Sprintf("MATCH (n:%s:%s) UNWIND keys(n) AS key RETURN DISTINCT key, 'mixed' AS type, count(*) AS count ORDER BY key", graphLabel, label)
	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var properties []types.PropertyInfo
	for result.Next(ctx) {
		record := result.Record()
		key, _ := record.Get("key")
		propType, _ := record.Get("type")
		count, _ := record.Get("count")

		// Skip graph namespace properties
		if keyStr, ok := key.(string); ok && keyStr == s.getGraphNamespaceProperty() {
			continue
		}

		prop := types.PropertyInfo{
			Name:     key.(string),
			Type:     propType.(string),
			Nullable: true,
		}

		if countInt, ok := count.(int64); ok {
			prop.Count = countInt
		}

		properties = append(properties, prop)
	}

	return properties, result.Err()
}

func (s *Store) getFilteredRelationshipProperties(ctx context.Context, session neo4j.SessionWithContext, graphLabel, relType string) ([]types.PropertyInfo, error) {
	query := fmt.Sprintf("MATCH (n:%s)-[r:%s]-() UNWIND keys(r) AS key RETURN DISTINCT key, 'mixed' AS type, count(*) AS count ORDER BY key", graphLabel, relType)
	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var properties []types.PropertyInfo
	for result.Next(ctx) {
		record := result.Record()
		key, _ := record.Get("key")
		propType, _ := record.Get("type")
		count, _ := record.Get("count")

		prop := types.PropertyInfo{
			Name:     key.(string),
			Type:     propType.(string),
			Nullable: true,
		}

		if countInt, ok := count.(int64); ok {
			prop.Count = countInt
		}

		properties = append(properties, prop)
	}

	return properties, result.Err()
}

func (s *Store) getFilteredConstraints(ctx context.Context, session neo4j.SessionWithContext, graphLabel string) ([]types.SchemaConstraint, error) {
	// For label-based graphs, we need to filter constraints that involve our graph labels
	result, err := session.Run(ctx, "SHOW CONSTRAINTS YIELD name, type, entityType, labelsOrTypes, properties", nil)
	if err != nil {
		return nil, err
	}

	var constraints []types.SchemaConstraint
	for result.Next(ctx) {
		record := result.Record()

		constraintType, _ := record.Get("type")
		labelsOrTypes, _ := record.Get("labelsOrTypes")
		properties, _ := record.Get("properties")

		// Check if this constraint is related to our graph
		isRelated := false
		var label string
		if labels, ok := labelsOrTypes.([]interface{}); ok && len(labels) > 0 {
			if labelStr, ok := labels[0].(string); ok {
				label = labelStr
				if labelStr == graphLabel || strings.HasPrefix(labelStr, s.getGraphLabelPrefix()) {
					isRelated = true
				}
			}
		}

		if !isRelated {
			continue
		}

		constraint := types.SchemaConstraint{
			Type:  constraintType.(string),
			Label: label,
		}

		if props, ok := properties.([]interface{}); ok {
			for _, prop := range props {
				if propStr, ok := prop.(string); ok {
					constraint.Properties = append(constraint.Properties, propStr)
				}
			}
		}

		constraints = append(constraints, constraint)
	}

	return constraints, result.Err()
}

func (s *Store) getFilteredIndexes(ctx context.Context, session neo4j.SessionWithContext, graphLabel string) ([]types.SchemaIndex, error) {
	// For label-based graphs, we need to filter indexes that involve our graph labels
	result, err := session.Run(ctx, "SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties", nil)
	if err != nil {
		return nil, err
	}

	var indexes []types.SchemaIndex
	for result.Next(ctx) {
		record := result.Record()

		indexType, _ := record.Get("type")
		labelsOrTypes, _ := record.Get("labelsOrTypes")
		properties, _ := record.Get("properties")

		// Check if this index is related to our graph
		isRelated := false
		var label string
		if labels, ok := labelsOrTypes.([]interface{}); ok && len(labels) > 0 {
			if labelStr, ok := labels[0].(string); ok {
				label = labelStr
				if labelStr == graphLabel || strings.HasPrefix(labelStr, s.getGraphLabelPrefix()) {
					isRelated = true
				}
			}
		}

		if !isRelated {
			continue
		}

		index := types.SchemaIndex{
			Type:  indexType.(string),
			Label: label,
		}

		if props, ok := properties.([]interface{}); ok {
			for _, prop := range props {
				if propStr, ok := prop.(string); ok {
					index.Properties = append(index.Properties, propStr)
				}
			}
		}

		indexes = append(indexes, index)
	}

	return indexes, result.Err()
}

func (s *Store) getFilteredSchemaStatistics(ctx context.Context, session neo4j.SessionWithContext, graphLabel string) (*types.GraphSchemaStats, error) {
	stats := &types.GraphSchemaStats{
		NodeCounts: make(map[string]int64),
		RelCounts:  make(map[string]int64),
	}

	// Get total node count for this graph
	nodeQuery := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as total", graphLabel)
	nodeResult, err := session.Run(ctx, nodeQuery, nil)
	if err != nil {
		return nil, err
	}
	if nodeResult.Next(ctx) {
		if total, ok := nodeResult.Record().Get("total"); ok {
			if totalInt, ok := total.(int64); ok {
				stats.TotalNodes = totalInt
			}
		}
	}

	// Get total relationship count for this graph
	relQuery := fmt.Sprintf("MATCH (n:%s)-[r]-() RETURN count(r) as total", graphLabel)
	relResult, err := session.Run(ctx, relQuery, nil)
	if err != nil {
		return nil, err
	}
	if relResult.Next(ctx) {
		if total, ok := relResult.Record().Get("total"); ok {
			if totalInt, ok := total.(int64); ok {
				stats.TotalRelationships = totalInt
			}
		}
	}

	return stats, nil
}

// Index operation helper methods

func (s *Store) executeCreateIndex(ctx context.Context, session neo4j.SessionWithContext, opts *types.CreateIndexOptions) error {
	// Build index name if not provided
	indexName := opts.Name
	if indexName == "" {
		indexName = s.generateIndexName(opts)
	}

	var query string
	params := make(map[string]interface{})

	// Validate index type
	indexType := strings.ToUpper(opts.IndexType)
	if indexType == "" {
		indexType = "BTREE" // Default index type
	}

	if opts.Target == "NODE" {
		if len(opts.Labels) == 0 {
			return fmt.Errorf("labels are required for node indexes")
		}

		label := opts.Labels[0] // Use first label

		switch indexType {
		case "BTREE", "RANGE":
			// Build properties list with node prefix
			nodeProps := make([]string, len(opts.Properties))
			for i, prop := range opts.Properties {
				nodeProps[i] = "n." + prop
			}
			propsStr := strings.Join(nodeProps, ", ")
			query = fmt.Sprintf("CREATE INDEX %s FOR (n:%s) ON (%s)", indexName, label, propsStr)
		case "FULLTEXT":
			// Build properties list with node prefix for FULLTEXT
			nodeProps := make([]string, len(opts.Properties))
			for i, prop := range opts.Properties {
				nodeProps[i] = "n." + prop
			}
			propsStr := strings.Join(nodeProps, ", ")
			query = fmt.Sprintf("CREATE FULLTEXT INDEX %s FOR (n:%s) ON EACH [%s]", indexName, label, propsStr)
		case "VECTOR":
			if len(opts.Properties) != 1 {
				return fmt.Errorf("vector indexes require exactly one property")
			}
			// Vector index configuration
			dimension := 1536      // Default dimension
			similarity := "cosine" // Default similarity
			if opts.Config != nil {
				if d, ok := opts.Config["dimension"].(int); ok {
					dimension = d
				}
				if s, ok := opts.Config["similarity"].(string); ok {
					similarity = s
				}
			}
			query = fmt.Sprintf("CREATE VECTOR INDEX %s FOR (n:%s) ON (n.%s) OPTIONS {indexConfig: {`vector.dimensions`: %d, `vector.similarity_function`: '%s'}}",
				indexName, label, opts.Properties[0], dimension, similarity)
		default:
			return fmt.Errorf("unsupported index type: %s", indexType)
		}
	} else if opts.Target == "RELATIONSHIP" {
		if len(opts.Labels) == 0 {
			return fmt.Errorf("relationship types are required for relationship indexes")
		}

		relType := opts.Labels[0] // Use first relationship type

		switch indexType {
		case "BTREE", "RANGE":
			// Build properties list with relationship prefix
			relProps := make([]string, len(opts.Properties))
			for i, prop := range opts.Properties {
				relProps[i] = "r." + prop
			}
			propsStr := strings.Join(relProps, ", ")
			query = fmt.Sprintf("CREATE INDEX %s FOR ()-[r:%s]-() ON (%s)", indexName, relType, propsStr)
		case "FULLTEXT":
			// Build properties list with relationship prefix for FULLTEXT
			relProps := make([]string, len(opts.Properties))
			for i, prop := range opts.Properties {
				relProps[i] = "r." + prop
			}
			propsStr := strings.Join(relProps, ", ")
			query = fmt.Sprintf("CREATE FULLTEXT INDEX %s FOR ()-[r:%s]-() ON EACH [%s]", indexName, relType, propsStr)
		default:
			return fmt.Errorf("unsupported relationship index type: %s", indexType)
		}
	} else {
		return fmt.Errorf("invalid target: %s (must be NODE or RELATIONSHIP)", opts.Target)
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") ||
			strings.Contains(err.Error(), "An equivalent index already exists") ||
			strings.Contains(err.Error(), "There already exists an index") {
			if opts.IfNotExists {
				return nil // Silently succeed if IfNotExists is true
			}
			return fmt.Errorf("index '%s' already exists", indexName)
		}
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func (s *Store) executeDropIndex(ctx context.Context, session neo4j.SessionWithContext, opts *types.DropIndexOptions) error {
	query := fmt.Sprintf("DROP INDEX %s", opts.Name)
	if opts.IfExists {
		query += " IF EXISTS"
	}

	_, err := session.Run(ctx, query, nil)
	if err != nil {
		if opts.IfExists && strings.Contains(err.Error(), "does not exist") {
			return nil // Silently succeed
		}
		return fmt.Errorf("failed to drop index '%s': %w", opts.Name, err)
	}

	return nil
}

func (s *Store) generateIndexName(opts *types.CreateIndexOptions) string {
	parts := []string{"idx", opts.GraphName}

	if opts.Target == "NODE" && len(opts.Labels) > 0 {
		parts = append(parts, "node", opts.Labels[0])
	} else if opts.Target == "RELATIONSHIP" && len(opts.Labels) > 0 {
		parts = append(parts, "rel", opts.Labels[0])
	}

	parts = append(parts, strings.Join(opts.Properties, "_"))

	if opts.IndexType != "" {
		parts = append(parts, strings.ToLower(opts.IndexType))
	}

	return strings.Join(parts, "_")
}

func (s *Store) prefixLabelsWithGraph(labels []string, graphName string) []string {
	graphPrefix := s.getGraphLabelPrefix() + graphName
	prefixed := make([]string, len(labels))

	for i, label := range labels {
		// If the label is already the graph label, use it as is
		if label == graphPrefix {
			prefixed[i] = label
		} else {
			// For user labels, we should use them as compound labels along with graph label
			// In Neo4j, nodes can have multiple labels
			prefixed[i] = label
		}
	}

	return prefixed
}
