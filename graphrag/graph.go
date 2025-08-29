package graphrag

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// ================================================================================================
// Segment Graph Management - Get Operations
// ================================================================================================

// GetSegmentGraph gets the graph information (entities and relationships) for a specific segment
func (g *GraphRag) GetSegmentGraph(ctx context.Context, docID string, segmentID string) (*types.SegmentGraph, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}
	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Getting graph information for segment %s in document %s", segmentID, docID)

	// Get entities and relationships directly from graph database
	entities, err := g.GetSegmentEntities(ctx, docID, segmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment entities: %w", err)
	}

	relationships, err := g.GetSegmentRelationships(ctx, docID, segmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment relationships: %w", err)
	}

	// Create segment graph response with minimal information
	segmentGraph := &types.SegmentGraph{
		DocID:         docID,
		SegmentID:     segmentID,
		Entities:      entities,
		Relationships: relationships,
	}

	g.Logger.Debugf("Successfully retrieved graph information for segment %s: %d entities, %d relationships",
		segmentID, len(segmentGraph.Entities), len(segmentGraph.Relationships))

	return segmentGraph, nil
}

// GetSegmentEntities gets the entities for a specific segment
func (g *GraphRag) GetSegmentEntities(ctx context.Context, docID string, segmentID string) ([]types.GraphNode, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}
	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Getting entities for segment %s in document %s", segmentID, docID)

	// Query directly from graph database
	if g.Graph == nil {
		g.Logger.Debugf("Graph database not configured, returning empty entities")
		return []types.GraphNode{}, nil
	}

	// Parse collection ID from docID and get the actual graph name
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
	}

	// Get the actual graph name using GetCollectionIDs
	ids, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		g.Logger.Warnf("Failed to generate collection IDs for %s: %v", collectionID, err)
		return []types.GraphNode{}, nil
	}
	graphName := ids.Graph
	g.Logger.Debugf("Using collection ID: %s, graph name: %s", collectionID, graphName)

	// Check if graph exists
	exists, err := g.Graph.GraphExists(ctx, graphName)
	if err != nil {
		g.Logger.Warnf("Failed to check graph existence: %v", err)
		return []types.GraphNode{}, nil
	}
	if !exists {
		g.Logger.Debugf("Graph %s does not exist, returning empty entities", graphName)
		return []types.GraphNode{}, nil
	}

	// Debug: Check total nodes in graph
	debugQueryOpts := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      "MATCH (n) RETURN count(n) as total_nodes",
		Parameters: map[string]interface{}{},
	}
	debugResult, err := g.Graph.Query(ctx, debugQueryOpts)
	if err != nil {
		g.Logger.Warnf("Failed to execute debug query: %v", err)
	} else {
		g.Logger.Debugf("Total nodes in graph %s: %+v", graphName, debugResult.Records)
	}

	// Debug: Check sample node properties
	debugQueryOpts2 := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      "MATCH (n) RETURN keys(n) as node_properties LIMIT 3",
		Parameters: map[string]interface{}{},
	}
	debugResult2, err := g.Graph.Query(ctx, debugQueryOpts2)
	if err != nil {
		g.Logger.Warnf("Failed to execute debug query 2: %v", err)
	} else {
		g.Logger.Debugf("Sample node properties in graph: %+v", debugResult2.Records)
	}

	// Debug: Check nodes with source_chunks property
	debugQueryOpts3 := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      "MATCH (n) WHERE n.source_chunks IS NOT NULL RETURN n.source_chunks LIMIT 5",
		Parameters: map[string]interface{}{},
	}
	debugResult3, err := g.Graph.Query(ctx, debugQueryOpts3)
	if err != nil {
		g.Logger.Warnf("Failed to execute debug query 3: %v", err)
	} else {
		g.Logger.Debugf("Sample source_chunks in graph: %+v", debugResult3.Records)
	}

	// Query entities that contain this segmentID in their source_chunks
	queryOpts := &types.GraphQueryOptions{
		GraphName: graphName,
		QueryType: "cypher",
		Query:     "MATCH (n) WHERE $segmentID IN n.source_chunks RETURN n",
		Parameters: map[string]interface{}{
			"segmentID": segmentID,
		},
	}

	g.Logger.Debugf("Executing graph query: %s with segmentID: %s", queryOpts.Query, segmentID)
	result, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		g.Logger.Warnf("Failed to query entities from graph database: %v", err)
		return []types.GraphNode{}, nil
	}

	g.Logger.Debugf("Graph query returned %d nodes", len(result.Nodes))

	// Print detailed query results for debugging
	g.Logger.Debugf("Raw query result: %+v", result)
	for i, node := range result.Nodes {
		g.Logger.Debugf("Node %d: ID=%s, Labels=%v, Properties=%+v", i, node.ID, node.Labels, node.Properties)
	}

	// Convert result nodes to GraphNodes
	var entities []types.GraphNode
	for _, node := range result.Nodes {
		// Extract logical ID from properties (not Neo4j internal ID)
		var logicalID string
		if id, ok := node.Properties["id"].(string); ok {
			logicalID = id
		} else {
			// Fallback to Neo4j internal ID if logical ID not found
			logicalID = node.ID
			g.Logger.Warnf("No logical ID found in properties for node %s, using internal ID", node.ID)
		}

		entity := types.GraphNode{
			ID:         logicalID, // Use logical ID, not Neo4j internal ID
			Labels:     node.Labels,
			Properties: node.Properties,
		}

		// Extract other fields from properties if available
		if entityType, ok := node.Properties["entity_type"].(string); ok {
			entity.EntityType = entityType
		}
		if description, ok := node.Properties["description"].(string); ok {
			entity.Description = description
		}
		if confidence, ok := node.Properties["confidence"].(float64); ok {
			entity.Confidence = confidence
		}

		entities = append(entities, entity)
		g.Logger.Debugf("Converted entity with logical ID %s: %+v", logicalID, entity)
	}

	g.Logger.Debugf("Successfully retrieved %d entities for segment %s from graph database", len(entities), segmentID)
	return entities, nil
}

// GetSegmentRelationships gets the relationships for a specific segment
func (g *GraphRag) GetSegmentRelationships(ctx context.Context, docID string, segmentID string) ([]types.GraphRelationship, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}
	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Getting relationships for segment %s in document %s", segmentID, docID)

	// Query directly from graph database
	if g.Graph == nil {
		g.Logger.Debugf("Graph database not configured, returning empty relationships")
		return []types.GraphRelationship{}, nil
	}

	// Parse collection ID from docID and get the actual graph name
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
	}

	// Get the actual graph name using GetCollectionIDs
	ids, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		g.Logger.Warnf("Failed to generate collection IDs for %s: %v", collectionID, err)
		return []types.GraphRelationship{}, nil
	}
	graphName := ids.Graph
	g.Logger.Debugf("Using collection ID: %s, graph name: %s", collectionID, graphName)

	// Check if graph exists
	exists, err := g.Graph.GraphExists(ctx, graphName)
	if err != nil {
		g.Logger.Warnf("Failed to check graph existence: %v", err)
		return []types.GraphRelationship{}, nil
	}
	if !exists {
		g.Logger.Debugf("Graph %s does not exist, returning empty relationships", graphName)
		return []types.GraphRelationship{}, nil
	}

	// Debug: Check total relationships in graph
	debugQueryOpts := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      "MATCH ()-[r]->() RETURN count(r) as total_relationships",
		Parameters: map[string]interface{}{},
	}
	debugResult, err := g.Graph.Query(ctx, debugQueryOpts)
	if err != nil {
		g.Logger.Warnf("Failed to execute debug relationship query: %v", err)
	} else {
		g.Logger.Debugf("Total relationships in graph %s: %+v", graphName, debugResult.Records)
	}

	// Debug: Check sample relationship properties
	debugQueryOpts2 := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      "MATCH ()-[r]->() RETURN keys(r) as rel_properties LIMIT 3",
		Parameters: map[string]interface{}{},
	}
	debugResult2, err := g.Graph.Query(ctx, debugQueryOpts2)
	if err != nil {
		g.Logger.Warnf("Failed to execute debug relationship query 2: %v", err)
	} else {
		g.Logger.Debugf("Sample relationship properties in graph: %+v", debugResult2.Records)
	}

	// Debug: Check relationships with source_chunks property
	debugQueryOpts3 := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      "MATCH ()-[r]->() WHERE r.source_chunks IS NOT NULL RETURN r.source_chunks LIMIT 5",
		Parameters: map[string]interface{}{},
	}
	debugResult3, err := g.Graph.Query(ctx, debugQueryOpts3)
	if err != nil {
		g.Logger.Warnf("Failed to execute debug relationship query 3: %v", err)
	} else {
		g.Logger.Debugf("Sample relationship source_chunks in graph: %+v", debugResult3.Records)

		// Debug: Check for specific relationship between stargate_project and openai
		debugQueryOpts4 := &types.GraphQueryOptions{
			GraphName:  graphName,
			QueryType:  "cypher",
			Query:      "MATCH (a {id: 'stargate_project'})-[r]-(b {id: 'openai'}) RETURN r, r.source_chunks",
			Parameters: map[string]interface{}{},
		}
		debugResult4, err := g.Graph.Query(ctx, debugQueryOpts4)
		if err != nil {
			g.Logger.Warnf("Failed to execute debug relationship query 4: %v", err)
		} else {
			g.Logger.Debugf("Stargate-OpenAI relationships in graph: %+v", debugResult4.Records)
		}
	}

	// Query relationships that contain this segmentID in their source_chunks
	// Also return the connected nodes to get their logical IDs
	// Use DISTINCT to avoid duplicates and query both directions
	queryOpts := &types.GraphQueryOptions{
		GraphName: graphName,
		QueryType: "cypher",
		Query:     "MATCH (start)-[r]-(end) WHERE $segmentID IN r.source_chunks RETURN DISTINCT r, start.id as start_logical_id, end.id as end_logical_id",
		Parameters: map[string]interface{}{
			"segmentID": segmentID,
		},
	}

	g.Logger.Debugf("Executing graph query: %s with segmentID: %s", queryOpts.Query, segmentID)
	result, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		g.Logger.Warnf("Failed to query relationships from graph database: %v", err)
		return []types.GraphRelationship{}, nil
	}

	g.Logger.Debugf("Graph query returned %d relationships", len(result.Relationships))

	// Print detailed query results for debugging
	g.Logger.Debugf("Raw relationship query result: %+v", result)
	for i, rel := range result.Relationships {
		// Extract logical ID for debugging
		var logicalID string
		if id, ok := rel.Properties["id"].(string); ok {
			logicalID = id
		} else {
			logicalID = rel.ID
		}
		g.Logger.Debugf("Relationship %d: LogicalID=%s, Neo4jID=%s, Type=%s, StartNode=%s, EndNode=%s, Properties=%+v",
			i, logicalID, rel.ID, rel.Type, rel.StartNode, rel.EndNode, rel.Properties)
	}

	// Convert result relationships to GraphRelationships
	var relationships []types.GraphRelationship
	seenRelationships := make(map[string]bool) // Track seen relationship IDs to avoid duplicates

	for i, rel := range result.Relationships {
		// Extract logical ID from properties (not Neo4j internal ID)
		var logicalID string
		if id, ok := rel.Properties["id"].(string); ok {
			logicalID = id
		} else {
			// Fallback to Neo4j internal ID if logical ID not found
			logicalID = rel.ID
			g.Logger.Warnf("No logical ID found in properties for relationship %s, using internal ID", rel.ID)
		}

		// Skip if we've already seen this relationship (avoid duplicates)
		if seenRelationships[logicalID] {
			g.Logger.Debugf("Skipping duplicate relationship: %s", logicalID)
			continue
		}
		seenRelationships[logicalID] = true

		// Extract logical node IDs from query results
		var startLogicalID, endLogicalID string
		if i < len(result.Records) {
			if record, ok := result.Records[i].(map[string]interface{}); ok {
				if startID, ok := record["start_logical_id"].(string); ok {
					startLogicalID = startID
				} else {
					startLogicalID = rel.StartNode // Fallback to internal ID
					g.Logger.Warnf("No start logical ID found for relationship %s, using internal ID %s", logicalID, rel.StartNode)
				}
				if endID, ok := record["end_logical_id"].(string); ok {
					endLogicalID = endID
				} else {
					endLogicalID = rel.EndNode // Fallback to internal ID
					g.Logger.Warnf("No end logical ID found for relationship %s, using internal ID %s", logicalID, rel.EndNode)
				}
			} else {
				// Fallback if record is not a map
				startLogicalID = rel.StartNode
				endLogicalID = rel.EndNode
				g.Logger.Warnf("Record is not a map for relationship %s, using internal node IDs", logicalID)
			}
		} else {
			// Fallback if records don't match
			startLogicalID = rel.StartNode
			endLogicalID = rel.EndNode
			g.Logger.Warnf("No matching record for relationship %s, using internal node IDs", logicalID)
		}

		// Extract business relationship type from properties (not Neo4j label)
		var businessType string
		if relType, ok := rel.Properties["type"].(string); ok {
			businessType = relType
		} else {
			// Fallback to Neo4j relationship label if business type not found
			businessType = rel.Type
			g.Logger.Warnf("No business type found in properties for relationship %s, using Neo4j label %s", logicalID, rel.Type)
		}

		relationship := types.GraphRelationship{
			ID:         logicalID,      // Use logical ID, not Neo4j internal ID
			Type:       businessType,   // Use business type from properties, not Neo4j label
			StartNode:  startLogicalID, // Use logical node ID
			EndNode:    endLogicalID,   // Use logical node ID
			Properties: rel.Properties,
		}

		// Extract other fields from properties if available
		if description, ok := rel.Properties["description"].(string); ok {
			relationship.Description = description
		}
		if confidence, ok := rel.Properties["confidence"].(float64); ok {
			relationship.Confidence = confidence
		}
		if weight, ok := rel.Properties["weight"].(float64); ok {
			relationship.Weight = weight
		}

		relationships = append(relationships, relationship)
		g.Logger.Debugf("Converted relationship with logical ID %s: start=%s, end=%s", logicalID, startLogicalID, endLogicalID)
	}

	g.Logger.Debugf("Successfully retrieved %d relationships for segment %s from graph database", len(relationships), segmentID)
	return relationships, nil
}

// GetSegmentRelationshipsByEntities gets all relationships connected to entities in this segment
// This method finds all relationships that involve entities from the segment, regardless of where the relationship was originally extracted
func (g *GraphRag) GetSegmentRelationshipsByEntities(ctx context.Context, docID string, segmentID string) ([]types.GraphRelationship, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}
	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Getting relationships by entities for segment %s in document %s", segmentID, docID)

	// Query directly from graph database
	if g.Graph == nil {
		g.Logger.Debugf("Graph database not configured, returning empty relationships")
		return []types.GraphRelationship{}, nil
	}

	// Parse collection ID from docID and get the actual graph name
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
	}

	// Get the actual graph name using GetCollectionIDs
	ids, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		g.Logger.Warnf("Failed to generate collection IDs for %s: %v", collectionID, err)
		return []types.GraphRelationship{}, nil
	}
	graphName := ids.Graph
	g.Logger.Debugf("Using collection ID: %s, graph name: %s", collectionID, graphName)

	// Check if graph exists
	exists, err := g.Graph.GraphExists(ctx, graphName)
	if err != nil {
		g.Logger.Warnf("Failed to check graph existence: %v", err)
		return []types.GraphRelationship{}, nil
	}
	if !exists {
		g.Logger.Debugf("Graph %s does not exist, returning empty relationships", graphName)
		return []types.GraphRelationship{}, nil
	}

	// First, get all entities in this segment
	entities, err := g.GetSegmentEntities(ctx, docID, segmentID)
	if err != nil {
		g.Logger.Warnf("Failed to get entities for segment: %v", err)
		return []types.GraphRelationship{}, nil
	}

	if len(entities) == 0 {
		g.Logger.Debugf("No entities found for segment %s, returning empty relationships", segmentID)
		return []types.GraphRelationship{}, nil
	}

	// Build list of entity IDs
	entityIDs := make([]string, len(entities))
	for i, entity := range entities {
		entityIDs[i] = entity.ID
	}
	g.Logger.Debugf("Found %d entities in segment: %v", len(entityIDs), entityIDs)

	// Query all relationships that involve any of these entities
	// Use DISTINCT to avoid duplicates and query both directions
	queryOpts := &types.GraphQueryOptions{
		GraphName: graphName,
		QueryType: "cypher",
		Query:     "MATCH (start)-[r]-(end) WHERE start.id IN $entityIDs OR end.id IN $entityIDs RETURN DISTINCT r, start.id as start_logical_id, end.id as end_logical_id",
		Parameters: map[string]interface{}{
			"entityIDs": entityIDs,
		},
	}

	g.Logger.Debugf("Executing entity-based relationship query: %s with entityIDs: %v", queryOpts.Query, entityIDs)
	result, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		g.Logger.Warnf("Failed to query relationships by entities from graph database: %v", err)
		return []types.GraphRelationship{}, nil
	}

	g.Logger.Debugf("Entity-based relationship query returned %d relationships", len(result.Relationships))

	// Convert result relationships to GraphRelationships (reuse the same logic)
	var relationships []types.GraphRelationship
	seenRelationships := make(map[string]bool) // Track seen relationship IDs to avoid duplicates

	for i, rel := range result.Relationships {
		// Extract logical ID from properties (not Neo4j internal ID)
		var logicalID string
		if id, ok := rel.Properties["id"].(string); ok {
			logicalID = id
		} else {
			// Fallback to Neo4j internal ID if logical ID not found
			logicalID = rel.ID
			g.Logger.Warnf("No logical ID found in properties for relationship %s, using internal ID", rel.ID)
		}

		// Skip if we've already seen this relationship (avoid duplicates)
		if seenRelationships[logicalID] {
			g.Logger.Debugf("Skipping duplicate relationship: %s", logicalID)
			continue
		}
		seenRelationships[logicalID] = true

		// Extract logical node IDs from query results
		var startLogicalID, endLogicalID string
		if i < len(result.Records) {
			if record, ok := result.Records[i].(map[string]interface{}); ok {
				if startID, ok := record["start_logical_id"].(string); ok {
					startLogicalID = startID
				} else {
					startLogicalID = rel.StartNode // Fallback to internal ID
					g.Logger.Warnf("No start logical ID found for relationship %s, using internal ID %s", logicalID, rel.StartNode)
				}
				if endID, ok := record["end_logical_id"].(string); ok {
					endLogicalID = endID
				} else {
					endLogicalID = rel.EndNode // Fallback to internal ID
					g.Logger.Warnf("No end logical ID found for relationship %s, using internal ID %s", logicalID, rel.EndNode)
				}
			} else {
				// Fallback if record is not a map
				startLogicalID = rel.StartNode
				endLogicalID = rel.EndNode
				g.Logger.Warnf("Record is not a map for relationship %s, using internal node IDs", logicalID)
			}
		} else {
			// Fallback if records don't match
			startLogicalID = rel.StartNode
			endLogicalID = rel.EndNode
			g.Logger.Warnf("No matching record for relationship %s, using internal node IDs", logicalID)
		}

		// Extract business relationship type from properties (not Neo4j label)
		var businessType string
		if relType, ok := rel.Properties["type"].(string); ok {
			businessType = relType
		} else {
			// Fallback to Neo4j relationship label if business type not found
			businessType = rel.Type
			g.Logger.Warnf("No business type found in properties for relationship %s, using Neo4j label %s", logicalID, rel.Type)
		}

		relationship := types.GraphRelationship{
			ID:         logicalID,      // Use logical ID, not Neo4j internal ID
			Type:       businessType,   // Use business type from properties, not Neo4j label
			StartNode:  startLogicalID, // Use logical node ID
			EndNode:    endLogicalID,   // Use logical node ID
			Properties: rel.Properties,
		}

		// Extract other fields from properties if available
		if description, ok := rel.Properties["description"].(string); ok {
			relationship.Description = description
		}
		if confidence, ok := rel.Properties["confidence"].(float64); ok {
			relationship.Confidence = confidence
		}
		if weight, ok := rel.Properties["weight"].(float64); ok {
			relationship.Weight = weight
		}

		relationships = append(relationships, relationship)
		g.Logger.Debugf("Converted entity-based relationship with logical ID %s: start=%s, end=%s", logicalID, startLogicalID, endLogicalID)
	}

	g.Logger.Debugf("Successfully retrieved %d entity-based relationships for segment %s from graph database", len(relationships), segmentID)
	return relationships, nil
}

// ================================================================================================
// Segment Graph Management - Extract Operations
// ================================================================================================

// ExtractSegmentGraph re-extracts entities and relationships for a specific segment
func (g *GraphRag) ExtractSegmentGraph(ctx context.Context, docID string, segmentID string, options *types.ExtractionOptions) (*types.SegmentExtractionResult, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}
	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Re-extracting graph for segment %s in document %s", segmentID, docID)

	// Validate that graph store is configured
	if g.Graph == nil {
		return nil, fmt.Errorf("graph store is not configured")
	}

	// Get the segment first to validate it exists and get its text
	segment, err := g.GetSegment(ctx, docID, segmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}

	// Parse collection ID from docID and get the actual graph name
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
	}

	// Get the actual graph name using GetCollectionIDs
	ids, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate collection IDs for %s: %w", collectionID, err)
	}
	graphName := ids.Graph

	// Set default extraction options if not provided
	if options == nil {
		options = &types.ExtractionOptions{}
	}

	// Use default extractor if not specified
	if options.Use == nil {
		extractor, err := DetectExtractor("")
		if err != nil {
			return nil, fmt.Errorf("failed to detect extractor: %w", err)
		}
		options.Use = extractor
	}

	// Create callback for extraction progress
	callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
		g.Logger.Debugf("Extraction progress for segment %s: %s (%d/%d) - %s",
			segmentID, status, payload.Current, payload.Total, payload.Message)
	}

	// Extract entities and relationships from segment text
	extractionResults, err := options.Use.ExtractDocuments(ctx, []string{segment.Text}, callback)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities and relationships: %w", err)
	}

	if len(extractionResults) == 0 {
		return nil, fmt.Errorf("no extraction results returned")
	}

	// Set source information for all extracted entities and relationships before saving
	g.setSourceInformation(extractionResults, docID, segmentID)

	// Save the extraction results to the graph database
	saveResponse, err := g.Graph.SaveExtractionResults(ctx, graphName, extractionResults)
	if err != nil {
		return nil, fmt.Errorf("failed to save extraction results: %w", err)
	}

	// --- DEGUG Print Extraction Results ---
	raw, _ := json.Marshal(extractionResults)
	fmt.Printf("Extraction Results: ---\n %s\n ---\n", string(raw))

	result := extractionResults[0]

	// Update the segment in vector database with new extracted entities and relationships
	// Reuse the ids from earlier call to avoid duplicate GetCollectionIDs
	err = g.updateSegmentWithExtraction(ctx, docID, segmentID, result, ids.Vector)
	if err != nil {
		g.Logger.Warnf("Failed to update segment in vector database: %v", err)
	}

	// Create extraction result with only statistical information
	extractionResult := &types.SegmentExtractionResult{
		DocID:              docID,
		SegmentID:          segmentID,
		ExtractionModel:    result.Model,
		EntitiesCount:      saveResponse.EntitiesCount,      // Actual count from database
		RelationshipsCount: saveResponse.RelationshipsCount, // Actual count from database
	}

	g.Logger.Infof("Successfully re-extracted graph for segment %s: %d entities, %d relationships",
		segmentID, len(result.Nodes), len(result.Relationships))

	return extractionResult, nil
}

// ================================================================================================
// Internal Helper Methods
// ================================================================================================

// updateSegmentWithExtraction updates a segment in vector database with new extraction results
func (g *GraphRag) updateSegmentWithExtraction(ctx context.Context, docID string, segmentID string, extraction *types.ExtractionResult, vectorCollectionName string) error {
	if g.Vector == nil {
		return nil // No vector database configured, skip update
	}

	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, vectorCollectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Vector collection %s does not exist, skipping segment update", vectorCollectionName)
		return nil
	}

	// Get the current segment document from vector database
	listOpts := &types.ListDocumentsOptions{
		CollectionName: vectorCollectionName,
		Filter: map[string]interface{}{
			"doc_id":           docID,
			"chunk_details.id": segmentID,
			"document_type":    "chunk",
		},
		Limit:          1,
		IncludeVector:  false,
		IncludePayload: true,
	}

	searchResult, err := g.Vector.ListDocuments(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to search for segment document: %w", err)
	}

	if len(searchResult.Documents) == 0 {
		g.Logger.Warnf("Segment %s not found in vector database, skipping update", segmentID)
		return nil
	}

	document := searchResult.Documents[0]

	// Update the chunk_details in metadata with new extraction results
	if document.Metadata["chunk_details"] != nil {
		if chunkDetails, ok := document.Metadata["chunk_details"].(map[string]interface{}); ok {
			// Collect entity information
			var entityList []map[string]interface{}
			for _, node := range extraction.Nodes {
				entityInfo := map[string]interface{}{
					"id":          node.ID,
					"name":        node.Name,
					"type":        node.Type,
					"description": node.Description,
					"confidence":  node.Confidence,
				}
				entityList = append(entityList, entityInfo)
			}

			// Collect relationship information
			var relationshipList []map[string]interface{}
			for _, rel := range extraction.Relationships {
				relationshipInfo := map[string]interface{}{
					"id":          rel.ID,
					"type":        rel.Type,
					"start_node":  rel.StartNode,
					"end_node":    rel.EndNode,
					"description": rel.Description,
					"confidence":  rel.Confidence,
					"weight":      rel.Weight,
				}
				relationshipList = append(relationshipList, relationshipInfo)
			}

			chunkDetails["entities"] = entityList
			chunkDetails["relationships"] = relationshipList
			chunkDetails["extraction_model"] = extraction.Model

			document.Metadata["chunk_details"] = chunkDetails
		}
	}

	// Update the document in vector database
	updateOpts := &types.AddDocumentOptions{
		CollectionName: vectorCollectionName,
		Documents:      []*types.Document{document},
		Upsert:         true,
		BatchSize:      1,
	}

	_, err = g.Vector.AddDocuments(ctx, updateOpts)
	if err != nil {
		return fmt.Errorf("failed to update segment document: %w", err)
	}

	g.Logger.Debugf("Successfully updated segment %s in vector database", segmentID)
	return nil
}

// setSourceInformation sets source documents and chunks for all extracted entities and relationships
// This ensures proper tracking of where each entity/relationship was extracted from
func (g *GraphRag) setSourceInformation(extractionResults []*types.ExtractionResult, docID, segmentID string) {
	for _, result := range extractionResults {
		// Set source information for all entities
		for i := range result.Nodes {
			node := &result.Nodes[i]

			// Set SourceDocuments with deduplication
			if node.SourceDocuments == nil {
				node.SourceDocuments = []string{}
			}
			if !g.containsString(node.SourceDocuments, docID) {
				node.SourceDocuments = append(node.SourceDocuments, docID)
			}

			// Set SourceChunks with deduplication
			if node.SourceChunks == nil {
				node.SourceChunks = []string{}
			}
			if !g.containsString(node.SourceChunks, segmentID) {
				node.SourceChunks = append(node.SourceChunks, segmentID)
			}

			g.Logger.Debugf("Set source info for entity %s: docs=%v, chunks=%v",
				node.ID, node.SourceDocuments, node.SourceChunks)
		}

		// Set source information for all relationships
		for i := range result.Relationships {
			relationship := &result.Relationships[i]

			// Set SourceDocuments with deduplication
			if relationship.SourceDocuments == nil {
				relationship.SourceDocuments = []string{}
			}
			if !g.containsString(relationship.SourceDocuments, docID) {
				relationship.SourceDocuments = append(relationship.SourceDocuments, docID)
			}

			// Set SourceChunks with deduplication
			if relationship.SourceChunks == nil {
				relationship.SourceChunks = []string{}
			}
			if !g.containsString(relationship.SourceChunks, segmentID) {
				relationship.SourceChunks = append(relationship.SourceChunks, segmentID)
			}

			g.Logger.Debugf("Set source info for relationship %s: docs=%v, chunks=%v",
				relationship.ID, relationship.SourceDocuments, relationship.SourceChunks)
		}
	}
}

// containsString checks if a string slice contains a specific string
func (g *GraphRag) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
