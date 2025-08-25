package graphrag

import (
	"context"
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
		entity := types.GraphNode{
			ID:         node.ID,
			Labels:     node.Labels,
			Properties: node.Properties,
		}
		entities = append(entities, entity)
		g.Logger.Debugf("Converted entity: %+v", entity)
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
	}

	// Query relationships that contain this segmentID in their source_chunks
	queryOpts := &types.GraphQueryOptions{
		GraphName: graphName,
		QueryType: "cypher",
		Query:     "MATCH ()-[r]->() WHERE $segmentID IN r.source_chunks RETURN r",
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
		g.Logger.Debugf("Relationship %d: ID=%s, Type=%s, StartNode=%s, EndNode=%s, Properties=%+v",
			i, rel.ID, rel.Type, rel.StartNode, rel.EndNode, rel.Properties)
	}

	// Convert result relationships to GraphRelationships
	var relationships []types.GraphRelationship
	for _, rel := range result.Relationships {
		relationship := types.GraphRelationship{
			ID:         rel.ID,
			Type:       rel.Type,
			StartNode:  rel.StartNode,
			EndNode:    rel.EndNode,
			Properties: rel.Properties,
		}
		relationships = append(relationships, relationship)
		g.Logger.Debugf("Converted relationship: %+v", relationship)
	}

	g.Logger.Debugf("Successfully retrieved %d relationships for segment %s from graph database", len(relationships), segmentID)
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

	result := extractionResults[0]

	// Set SourceChunks for all extracted entities and relationships
	for i := range result.Nodes {
		if result.Nodes[i].SourceChunks == nil {
			result.Nodes[i].SourceChunks = []string{}
		}
		result.Nodes[i].SourceChunks = append(result.Nodes[i].SourceChunks, segmentID)
		g.Logger.Debugf("Set SourceChunks for entity %s: %v", result.Nodes[i].ID, result.Nodes[i].SourceChunks)
	}

	for i := range result.Relationships {
		if result.Relationships[i].SourceChunks == nil {
			result.Relationships[i].SourceChunks = []string{}
		}
		result.Relationships[i].SourceChunks = append(result.Relationships[i].SourceChunks, segmentID)
		g.Logger.Debugf("Set SourceChunks for relationship %s: %v", result.Relationships[i].ID, result.Relationships[i].SourceChunks)
	}

	// Get collection IDs for graph store
	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	// Store entities to graph store with deduplication
	var actualEntityIDs []string
	var entityDeduplicationResults map[string]*types.EntityDeduplicationResult
	if len(result.Nodes) > 0 {
		// Convert Node to types.Node for storage
		entities := make([]types.Node, len(result.Nodes))
		copy(entities, result.Nodes)

		var oldEntityDeduplicationResults map[string]*EntityDeduplicationResult
		actualEntityIDs, oldEntityDeduplicationResults, err = g.storeEntitiesToGraphStore(ctx, entities, collectionIDs.Graph, docID)
		if err != nil {
			return nil, fmt.Errorf("failed to store entities to graph store: %w", err)
		}

		// Convert old type to new type
		entityDeduplicationResults = make(map[string]*types.EntityDeduplicationResult)
		for k, v := range oldEntityDeduplicationResults {
			entityDeduplicationResults[k] = &types.EntityDeduplicationResult{
				NormalizedID: v.NormalizedID,
				DocIDs:       v.DocIDs,
				IsUpdate:     v.IsUpdate,
			}
		}
	}

	// Store relationships to graph store with deduplication
	var actualRelationshipIDs []string
	var relationshipDeduplicationResults map[string]*types.RelationshipDeduplicationResult
	if len(result.Relationships) > 0 {
		// Convert Relationship to types.Relationship for storage
		relationships := make([]types.Relationship, len(result.Relationships))
		copy(relationships, result.Relationships)

		var oldRelationshipDeduplicationResults map[string]*RelationshipDeduplicationResult
		actualRelationshipIDs, oldRelationshipDeduplicationResults, err = g.storeRelationshipsToGraphStore(ctx, relationships, collectionIDs.Graph, docID)
		if err != nil {
			return nil, fmt.Errorf("failed to store relationships to graph store: %w", err)
		}

		// Convert old type to new type
		relationshipDeduplicationResults = make(map[string]*types.RelationshipDeduplicationResult)
		for k, v := range oldRelationshipDeduplicationResults {
			relationshipDeduplicationResults[k] = &types.RelationshipDeduplicationResult{
				NormalizedID: v.NormalizedID,
				DocIDs:       v.DocIDs,
				IsUpdate:     v.IsUpdate,
			}
		}
	}

	// Update the segment in vector database with new extracted entities and relationships
	err = g.updateSegmentWithExtraction(ctx, docID, segmentID, result, collectionIDs.Vector)
	if err != nil {
		g.Logger.Warnf("Failed to update segment in vector database: %v", err)
	}

	// Create extraction result
	extractionResult := &types.SegmentExtractionResult{
		DocID:                            docID,
		SegmentID:                        segmentID,
		Text:                             segment.Text,
		ExtractedEntities:                result.Nodes,
		ExtractedRelationships:           result.Relationships,
		ActualEntityIDs:                  actualEntityIDs,
		ActualRelationshipIDs:            actualRelationshipIDs,
		EntityDeduplicationResults:       entityDeduplicationResults,
		RelationshipDeduplicationResults: relationshipDeduplicationResults,
		ExtractionModel:                  result.Model,
		EntitiesCount:                    len(result.Nodes),
		RelationshipsCount:               len(result.Relationships),
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
