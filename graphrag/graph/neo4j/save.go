package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// SaveExtractionResults saves the extraction results to the graph database
// It handles entity deduplication and relationship updates
// Returns the actual entities and relationships that were saved
// Collects all errors and returns them after processing all results
func (s *Store) SaveExtractionResults(ctx context.Context, graphName string, results []*types.ExtractionResult) (*types.SaveExtractionResultsResponse, error) {
	if len(results) == 0 {
		return &types.SaveExtractionResultsResponse{}, nil
	}

	if graphName == "" {
		graphName = "default" // Use default graph name if not specified
	}

	var allSavedEntities []types.GraphNode
	var allSavedRelationships []types.GraphRelationship
	var allErrors []error

	// Process each extraction result, collecting all errors
	for i, result := range results {
		savedEntities, savedRelationships, err := s.saveExtractionResult(ctx, graphName, result)
		if err != nil {
			// Collect error but continue processing other results
			allErrors = append(allErrors, fmt.Errorf("failed to save extraction result %d: %w", i, err))
			continue
		}

		// Collect successfully saved entities and relationships
		allSavedEntities = append(allSavedEntities, savedEntities...)
		allSavedRelationships = append(allSavedRelationships, savedRelationships...)
	}

	response := &types.SaveExtractionResultsResponse{
		SavedEntities:      allSavedEntities,
		SavedRelationships: allSavedRelationships,
		EntitiesCount:      len(allSavedEntities),
		RelationshipsCount: len(allSavedRelationships),
		ProcessedCount:     len(results),
	}

	// Return accumulated errors if any occurred
	if len(allErrors) > 0 {
		// Combine all errors into a single error message
		var errorMessages []string
		for _, err := range allErrors {
			errorMessages = append(errorMessages, err.Error())
		}
		combinedError := fmt.Errorf("encountered %d errors during save: %s", len(allErrors), strings.Join(errorMessages, "; "))
		return response, combinedError
	}

	return response, nil
}

// saveExtractionResult processes a single extraction result
// Returns the actual entities and relationships that were saved
func (s *Store) saveExtractionResult(ctx context.Context, graphName string, result *types.ExtractionResult) ([]types.GraphNode, []types.GraphRelationship, error) {
	if result == nil {
		return nil, nil, nil
	}

	// Step 1: Process entities with deduplication
	entityIDMap, savedEntities, err := s.processEntitiesWithDeduplication(ctx, graphName, result.Nodes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process entities: %w", err)
	}

	// Step 2: Update relationships with deduplicated entity IDs
	updatedRelationships, err := s.updateRelationshipsWithDeduplicatedIDs(result.Relationships, entityIDMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update relationships: %w", err)
	}

	// Step 3: Save relationships to the graph database
	var savedRelationships []types.GraphRelationship
	if len(updatedRelationships) > 0 {
		err = s.saveRelationships(ctx, graphName, updatedRelationships)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to save relationships: %w", err)
		}
		// Convert to types.GraphRelationship for response
		for _, rel := range updatedRelationships {
			savedRelationships = append(savedRelationships, *rel)
		}
	}

	return savedEntities, savedRelationships, nil
}

// processEntitiesWithDeduplication processes entities with deduplication logic
// Returns a map of original ID -> final ID and the actual saved entities
func (s *Store) processEntitiesWithDeduplication(ctx context.Context, graphName string, nodes []types.Node) (map[string]string, []types.GraphNode, error) {
	entityIDMap := make(map[string]string)
	var savedEntities []types.GraphNode

	if len(nodes) == 0 {
		return entityIDMap, savedEntities, nil
	}

	var entitiesToCreate []*types.GraphNode
	var entityIDsToRetrieve []string

	for _, node := range nodes {
		// Search for existing entity by name and type
		existingEntity, err := s.findExistingEntity(ctx, graphName, node.Name, node.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to search for existing entity %s: %w", node.Name, err)
		}

		if existingEntity != nil {
			// Entity exists - map to existing ID and update docs/chunks
			entityIDMap[node.ID] = existingEntity.ID
			entityIDsToRetrieve = append(entityIDsToRetrieve, existingEntity.ID)

			// Update existing entity with new source documents and chunks
			err = s.updateEntitySources(ctx, graphName, existingEntity.ID, node.SourceDocuments, node.SourceChunks)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to update entity sources for %s: %w", existingEntity.ID, err)
			}
		} else {
			// Entity doesn't exist - prepare for creation
			graphNode := s.convertNodeToGraphNode(node)
			entitiesToCreate = append(entitiesToCreate, graphNode)
			entityIDMap[node.ID] = node.ID // Keep original ID for new entities
			entityIDsToRetrieve = append(entityIDsToRetrieve, node.ID)
		}
	}

	// Create new entities in batch
	if len(entitiesToCreate) > 0 {
		err := s.createEntitiesInBatch(ctx, graphName, entitiesToCreate)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create new entities: %w", err)
		}
	}

	// Retrieve all final entities (both existing updated ones and new ones)
	if len(entityIDsToRetrieve) > 0 {
		opts := &types.GetNodesOptions{
			GraphName:         graphName,
			IDs:               entityIDsToRetrieve,
			IncludeProperties: true,
			IncludeMetadata:   true,
		}

		retrievedEntities, err := s.GetNodes(ctx, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retrieve saved entities: %w", err)
		}

		// Convert to []types.GraphNode
		for _, entity := range retrievedEntities {
			savedEntities = append(savedEntities, *entity)
		}
	}

	return entityIDMap, savedEntities, nil
}

// findExistingEntity searches for an existing entity by name and type
func (s *Store) findExistingEntity(ctx context.Context, graphName, name, entityType string) (*types.GraphNode, error) {
	// Build filter to search by name and entity_type
	filter := map[string]interface{}{
		"name":        name,
		"entity_type": entityType,
	}

	opts := &types.GetNodesOptions{
		GraphName:         graphName,
		Filter:            filter,
		IncludeProperties: true,
		IncludeMetadata:   true,
		Limit:             1,
	}

	nodes, err := s.GetNodes(ctx, opts)
	if err != nil {
		return nil, err
	}

	if len(nodes) > 0 {
		return nodes[0], nil
	}

	return nil, nil
}

// updateEntitySources updates an existing entity with new source documents and chunks
func (s *Store) updateEntitySources(ctx context.Context, graphName, entityID string, newDocs, newChunks []string) error {
	// First get the existing entity to retrieve current sources
	opts := &types.GetNodesOptions{
		GraphName:         graphName,
		IDs:               []string{entityID},
		IncludeProperties: true,
		Limit:             1,
	}

	existingNodes, err := s.GetNodes(ctx, opts)
	if err != nil {
		return err
	}

	if len(existingNodes) == 0 {
		return fmt.Errorf("entity %s not found", entityID)
	}

	existingNode := existingNodes[0]

	// Merge source documents
	existingDocs := s.getStringSliceFromProperty(existingNode.Properties, "source_documents")
	mergedDocs := s.mergeStringSlices(existingDocs, newDocs)

	// Merge source chunks
	existingChunks := s.getStringSliceFromProperty(existingNode.Properties, "source_chunks")
	mergedChunks := s.mergeStringSlices(existingChunks, newChunks)

	// Update the entity with merged sources
	updatedNode := &types.GraphNode{
		ID:          entityID,
		Labels:      existingNode.Labels,
		Properties:  existingNode.Properties,
		EntityType:  existingNode.EntityType,
		Description: existingNode.Description,
		Confidence:  existingNode.Confidence,
		Importance:  existingNode.Importance,
		Embedding:   existingNode.Embedding,
		Embeddings:  existingNode.Embeddings,
		CreatedAt:   existingNode.CreatedAt,
		UpdatedAt:   time.Now().UTC(),
		Version:     existingNode.Version + 1,
	}

	// Update properties with merged sources
	if updatedNode.Properties == nil {
		updatedNode.Properties = make(map[string]interface{})
	}
	updatedNode.Properties["source_documents"] = mergedDocs
	updatedNode.Properties["source_chunks"] = mergedChunks

	// Use upsert to update the existing entity
	addOpts := &types.AddNodesOptions{
		GraphName: graphName,
		Nodes:     []*types.GraphNode{updatedNode},
		Upsert:    true,
		BatchSize: 1,
	}

	_, err = s.AddNodes(ctx, addOpts)
	return err
}

// convertNodeToGraphNode converts a types.Node to types.GraphNode
func (s *Store) convertNodeToGraphNode(node types.Node) *types.GraphNode {
	properties := make(map[string]interface{})

	// Copy user-defined properties with type validation
	for k, v := range node.Properties {
		// Only include properties that Neo4j can handle
		if s.isValidNeo4jPropertyValue(v) {
			properties[k] = v
		} else {
			// Convert complex types to strings or skip them
			if str, ok := s.convertToString(v); ok {
				properties[k] = str
				// Property converted successfully
			}
			// If conversion fails, we skip this property to avoid Neo4j errors
		}
	}

	// Add GraphRAG-specific properties
	properties["name"] = node.Name
	if len(node.SourceDocuments) > 0 {
		properties["source_documents"] = node.SourceDocuments
	}
	if len(node.SourceChunks) > 0 {
		properties["source_chunks"] = node.SourceChunks
	}

	return &types.GraphNode{
		ID:          node.ID,
		Labels:      append([]string{node.Type}, node.Labels...),
		Properties:  properties,
		EntityType:  node.Type,
		Description: node.Description,
		Confidence:  node.Confidence,
		Embedding:   node.EmbeddingVector,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Version:     1,
	}
}

// isValidNeo4jPropertyValue checks if a value is a valid Neo4j property type
// Neo4j supports: primitives (bool, int*, uint*, float*, string) and their arrays
// Neo4j does NOT support: null, nested maps, complex objects
func (s *Store) isValidNeo4jPropertyValue(value interface{}) bool {
	if value == nil {
		return false // Neo4j doesn't support null values as properties
	}

	switch v := value.(type) {
	// Primitive types supported by Neo4j
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
		return true
	// Primitive array types supported by Neo4j
	case []bool, []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []float32, []float64, []string:
		return true
	// Generic slice - check if all elements are primitives
	case []interface{}:
		if len(v) == 0 {
			return true // Empty slice is valid
		}
		// All elements must be primitive types (no nested structures)
		for _, elem := range v {
			if !s.isPrimitiveType(elem) {
				return false
			}
		}
		return true
	// Complex types not supported by Neo4j
	case map[string]interface{}, []map[string]interface{}:
		return false
	default:
		return false
	}
}

// isPrimitiveType checks if a value is a Neo4j primitive type (no arrays)
func (s *Store) isPrimitiveType(value interface{}) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
		return true
	default:
		return false
	}
}

// convertToString attempts to convert a value to a string representation
func (s *Store) convertToString(value interface{}) (string, bool) {
	if value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	case map[string]interface{}:
		// Convert map to JSON string for storage
		if jsonStr, err := json.Marshal(v); err == nil {
			return string(jsonStr), true
		}
		return "", false
	case []interface{}:
		// Convert slice to JSON string for storage
		if jsonStr, err := json.Marshal(v); err == nil {
			return string(jsonStr), true
		}
		return "", false
	default:
		// Try to convert to string using fmt.Sprintf
		return fmt.Sprintf("%v", v), true
	}
}

// createEntitiesInBatch creates multiple entities in a batch operation
func (s *Store) createEntitiesInBatch(ctx context.Context, graphName string, entities []*types.GraphNode) error {
	opts := &types.AddNodesOptions{
		GraphName: graphName,
		Nodes:     entities,
		Upsert:    false, // These are new entities
		BatchSize: 100,
	}

	_, err := s.AddNodes(ctx, opts)
	return err
}

// updateRelationshipsWithDeduplicatedIDs updates relationship node references with deduplicated entity IDs
func (s *Store) updateRelationshipsWithDeduplicatedIDs(relationships []types.Relationship, entityIDMap map[string]string) ([]*types.GraphRelationship, error) {
	var updatedRelationships []*types.GraphRelationship

	for _, rel := range relationships {
		// Map start and end node IDs to deduplicated IDs
		startNodeID := rel.StartNode
		if mappedID, exists := entityIDMap[startNodeID]; exists {
			startNodeID = mappedID
		}

		endNodeID := rel.EndNode
		if mappedID, exists := entityIDMap[endNodeID]; exists {
			endNodeID = mappedID
		}

		// Convert to GraphRelationship
		graphRel := s.convertRelationshipToGraphRelationship(rel, startNodeID, endNodeID)
		updatedRelationships = append(updatedRelationships, graphRel)
	}

	return updatedRelationships, nil
}

// convertRelationshipToGraphRelationship converts a types.Relationship to types.GraphRelationship
func (s *Store) convertRelationshipToGraphRelationship(rel types.Relationship, startNodeID, endNodeID string) *types.GraphRelationship {
	properties := make(map[string]interface{})

	// Copy user-defined properties with type validation
	for k, v := range rel.Properties {
		// Only include properties that Neo4j can handle
		if s.isValidNeo4jPropertyValue(v) {
			properties[k] = v
		} else {
			// Convert complex types to strings or skip them
			if str, ok := s.convertToString(v); ok {
				properties[k] = str
				// Property converted successfully
			}
			// If conversion fails, we skip this property to avoid Neo4j errors
		}
	}

	// Add GraphRAG-specific properties
	if len(rel.SourceDocuments) > 0 {
		properties["source_documents"] = rel.SourceDocuments
	}
	if len(rel.SourceChunks) > 0 {
		properties["source_chunks"] = rel.SourceChunks
	}

	return &types.GraphRelationship{
		ID:          rel.ID,
		Type:        rel.Type,
		StartNode:   startNodeID,
		EndNode:     endNodeID,
		Properties:  properties,
		Description: rel.Description,
		Confidence:  rel.Confidence,
		Weight:      rel.Weight,
		Embedding:   rel.EmbeddingVector,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Version:     1,
	}
}

// saveRelationships saves relationships to the graph database
func (s *Store) saveRelationships(ctx context.Context, graphName string, relationships []*types.GraphRelationship) error {

	opts := &types.AddRelationshipsOptions{
		GraphName:     graphName,
		Relationships: relationships,
		Upsert:        true,  // Allow updating existing relationships
		CreateNodes:   false, // Nodes should already exist
		BatchSize:     100,
	}

	_, err := s.AddRelationships(ctx, opts)
	return err
}

// Helper functions

// getStringSliceFromProperty extracts a string slice from properties map
func (s *Store) getStringSliceFromProperty(properties map[string]interface{}, key string) []string {
	if properties == nil {
		return nil
	}

	value, exists := properties[key]
	if !exists {
		return nil
	}

	// Handle different possible types
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// mergeStringSlices merges two string slices, removing duplicates
func (s *Store) mergeStringSlices(slice1, slice2 []string) []string {
	seen := make(map[string]bool)
	var result []string

	// Add items from first slice
	for _, item := range slice1 {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	// Add items from second slice
	for _, item := range slice2 {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
