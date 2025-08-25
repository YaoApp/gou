package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// ================================================================================================
// CRUD Operations - Remove Segments
// ================================================================================================

// RemoveSegments removes segments by IDs
func (g *GraphRag) RemoveSegments(ctx context.Context, docID string, segmentIDs []string) (int, error) {
	if len(segmentIDs) == 0 {
		return 0, nil
	}

	if docID == "" {
		return 0, fmt.Errorf("docID cannot be empty")
	}

	g.Logger.Infof("Starting to remove %d segments for document %s", len(segmentIDs), docID)

	// Extract GraphName from docID
	graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
	if graphName == "" {
		graphName = "default"
	}

	// Step 2: Get collection IDs for this graph
	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection IDs for graph %s: %w", graphName, err)
	}

	// Step 3: Remove segments from vector store
	if g.Vector != nil {
		err := g.removeSegmentsFromVectorStore(ctx, collectionIDs.Vector, segmentIDs)
		if err != nil {
			g.Logger.Warnf("Failed to remove segments from vector store %s: %v", collectionIDs.Vector, err)
		} else {
			g.Logger.Infof("Removed segments from vector store: %s", collectionIDs.Vector)
		}
	}

	// Step 4: Update entities and relationships in graph store
	if g.Graph != nil && g.Graph.IsConnected() {
		err := g.removeEntitiesAndRelationshipsForSegments(ctx, collectionIDs.Graph, segmentIDs)
		if err != nil {
			g.Logger.Warnf("Failed to update entities/relationships in graph store %s: %v", collectionIDs.Graph, err)
		} else {
			g.Logger.Infof("Updated entities/relationships in graph store: %s", collectionIDs.Graph)
		}
	}

	// Step 5: Remove segment metadata from Store
	if g.Store != nil {
		g.removeSegmentsFromStore(ctx, docID, segmentIDs)
	}

	removedCount := len(segmentIDs)
	g.Logger.Infof("Successfully removed %d segments", removedCount)
	return removedCount, nil
}

// RemoveSegmentsByDocID removes all segments of a document
func (g *GraphRag) RemoveSegmentsByDocID(ctx context.Context, docID string) (int, error) {
	if docID == "" {
		return 0, fmt.Errorf("docID cannot be empty")
	}

	g.Logger.Infof("Starting to remove all segments for document: %s", docID)

	// Step 1: Parse GraphName from docID
	graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
	if graphName == "" {
		graphName = "default"
	}

	// Step 2: Get collection IDs for this graph
	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection IDs for graph %s: %w", graphName, err)
	}

	// Step 3: Count segments before deletion (for return value)
	segmentCount := 0
	if g.Vector != nil {
		exists, err := g.Vector.CollectionExists(ctx, collectionIDs.Vector)
		if err == nil && exists {
			// Get count of segments for this document
			listOpts := &types.ListDocumentsOptions{
				CollectionName: collectionIDs.Vector,
				Filter: map[string]interface{}{
					"doc_id":        docID,
					"document_type": "chunk",
				},
				Limit:          1000,
				IncludeVector:  false,
				IncludePayload: false,
			}
			result, err := g.Vector.ListDocuments(ctx, listOpts)
			if err == nil {
				segmentCount = len(result.Documents)
			}
		}
	}

	// Step 4: Remove segments from vector store by doc_id filter
	if g.Vector != nil {
		err := g.removeSegmentsByDocIDFromVectorStore(ctx, collectionIDs.Vector, docID)
		if err != nil {
			g.Logger.Warnf("Failed to remove segments from vector store %s: %v", collectionIDs.Vector, err)
		} else {
			g.Logger.Infof("Removed segments from vector store: %s", collectionIDs.Vector)
		}
	}

	// Step 5: Process entities and relationships in graph store (same as document deletion)
	if g.Graph != nil && g.Graph.IsConnected() {
		err := g.removeDocsFromGraphStore(ctx, collectionIDs.Graph, []string{docID})
		if err != nil {
			g.Logger.Warnf("Failed to process entities/relationships in graph store %s: %v", collectionIDs.Graph, err)
		} else {
			g.Logger.Infof("Processed entities/relationships in graph store: %s", collectionIDs.Graph)
		}
	}

	// Step 6: Remove all segment metadata from Store for this docID
	if g.Store != nil {
		g.removeAllSegmentMetadataFromStore(ctx, docID)
	}

	g.Logger.Infof("Successfully removed %d segments for document: %s", segmentCount, docID)
	return segmentCount, nil
}

// ================================================================================================
// Internal Helper Methods - Vector Store Operations
// ================================================================================================

// removeSegmentsFromVectorStore removes segments from vector database
func (g *GraphRag) removeSegmentsFromVectorStore(ctx context.Context, collectionName string, segmentIDs []string) error {
	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Vector collection %s does not exist, skipping vector deletion", collectionName)
		return nil
	}

	// Delete segments (chunks) by IDs
	err = g.Vector.DeleteDocuments(ctx, &types.DeleteDocumentOptions{
		CollectionName: collectionName,
		IDs:            segmentIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to delete segments: %w", err)
	}

	g.Logger.Debugf("Deleted %d segments from vector store", len(segmentIDs))
	return nil
}

// removeSegmentsByDocIDFromVectorStore removes all segments for a document from vector store using filter
func (g *GraphRag) removeSegmentsByDocIDFromVectorStore(ctx context.Context, collectionName string, docID string) error {
	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Vector collection %s does not exist, skipping vector deletion", collectionName)
		return nil
	}

	// Delete all chunks (segments) for this document
	chunksFilter := map[string]interface{}{
		"doc_id":        docID,
		"document_type": "chunk",
	}

	err = g.Vector.DeleteDocuments(ctx, &types.DeleteDocumentOptions{
		CollectionName: collectionName,
		Filter:         chunksFilter,
	})
	if err != nil {
		return fmt.Errorf("failed to delete segments for document %s: %w", docID, err)
	}

	g.Logger.Debugf("Deleted all segments for document %s from vector store", docID)
	return nil
}

// findSegmentsByDocID finds all segment IDs for a given document from vector store
func (g *GraphRag) findSegmentsByDocID(ctx context.Context, collectionName string, docID string) ([]string, error) {
	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Vector collection %s does not exist, no segments to find", collectionName)
		return []string{}, nil
	}

	// Use ListDocuments to find all chunks (segments) for this document
	// We filter by doc_id and document_type = "chunk"
	listOpts := &types.ListDocumentsOptions{
		CollectionName: collectionName,
		Filter: map[string]interface{}{
			"doc_id":        docID,
			"document_type": "chunk",
		},
		Limit:          1000, // Set a reasonable limit to avoid too much data
		IncludeVector:  false,
		IncludePayload: false, // We only need the IDs
	}

	result, err := g.Vector.ListDocuments(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list segments: %w", err)
	}

	// Extract segment IDs from the result
	segmentIDs := make([]string, 0, len(result.Documents))
	for _, doc := range result.Documents {
		if doc.ID != "" {
			segmentIDs = append(segmentIDs, doc.ID)
		}
	}

	g.Logger.Debugf("Found %d segments for document %s", len(segmentIDs), docID)
	return segmentIDs, nil
}

// ================================================================================================
// Internal Helper Methods - Store Operations
// ================================================================================================

// removeSegmentsFromStore removes segment metadata from Store
func (g *GraphRag) removeSegmentsFromStore(ctx context.Context, docID string, segmentIDs []string) {
	for _, segmentID := range segmentIDs {
		// Delete Weight
		err := g.deleteSegmentValue(docID, segmentID, StoreKeyWeight)
		if err != nil {
			g.Logger.Warnf("Failed to delete weight for segment %s: %v", segmentID, err)
		}

		// Delete Score
		err = g.deleteSegmentValue(docID, segmentID, StoreKeyScore)
		if err != nil {
			g.Logger.Warnf("Failed to delete score for segment %s: %v", segmentID, err)
		}

		// Delete Vote
		err = g.deleteSegmentValue(docID, segmentID, StoreKeyVote)
		if err != nil {
			g.Logger.Warnf("Failed to delete vote for segment %s: %v", segmentID, err)
		}
	}
}

// removeAllSegmentMetadataFromStore removes all segment metadata for a document from Store
func (g *GraphRag) removeAllSegmentMetadataFromStore(ctx context.Context, docID string) {
	if g.Store == nil {
		return
	}

	g.Logger.Debugf("Attempting to remove segment metadata for document %s", docID)

	// Parse CollectionID from docID to find the right collection
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
	}

	// Get collection IDs for this collection
	collectionIDs, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		g.Logger.Warnf("Failed to get collection IDs for document %s: %v", docID, err)
		return
	}

	// Find all segment IDs for this document from vector store
	segmentIDs, err := g.findSegmentsByDocID(ctx, collectionIDs.Vector, docID)
	if err != nil {
		g.Logger.Warnf("Failed to find segments for document %s: %v", docID, err)
		return
	}

	if len(segmentIDs) == 0 {
		g.Logger.Debugf("No segments found for document %s, no Store cleanup needed", docID)
		return
	}

	// Delete Store metadata for each segment using the helper function
	removedCount := 0
	for _, segmentID := range segmentIDs {
		// Delete Weight
		err := g.deleteSegmentValue(docID, segmentID, StoreKeyWeight)
		if err != nil {
			g.Logger.Warnf("Failed to delete weight for segment %s: %v", segmentID, err)
		} else {
			removedCount++
		}

		// Delete Score
		err = g.deleteSegmentValue(docID, segmentID, StoreKeyScore)
		if err != nil {
			g.Logger.Warnf("Failed to delete score for segment %s: %v", segmentID, err)
		} else {
			removedCount++
		}

		// Delete Vote
		err = g.deleteSegmentValue(docID, segmentID, StoreKeyVote)
		if err != nil {
			g.Logger.Warnf("Failed to delete vote for segment %s: %v", segmentID, err)
		} else {
			removedCount++
		}
	}

	g.Logger.Infof("Segment metadata cleanup completed for document %s: removed %d Store entries for %d segments", docID, removedCount, len(segmentIDs))
}

// ================================================================================================
// Internal Helper Methods - Graph Operations
// ================================================================================================

// removeEntitiesAndRelationshipsForSegments removes entities and relationships associated with specific segments
func (g *GraphRag) removeEntitiesAndRelationshipsForSegments(ctx context.Context, graphName string, segmentIDs []string) error {
	if len(segmentIDs) == 0 {
		return nil
	}

	// Check if graph exists
	exists, err := g.Graph.GraphExists(ctx, graphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Graph %s does not exist, skipping entity/relationship removal", graphName)
		return nil
	}

	// Remove entities and relationships that were extracted from these specific chunks
	// We need to query entities and relationships that have these chunk IDs in their source_chunks

	// Remove entities associated with these segments
	for _, segmentID := range segmentIDs {
		// Query entities that have this segment in their source_chunks
		queryOpts := &types.GraphQueryOptions{
			GraphName: graphName,
			QueryType: "cypher",
			Query:     "MATCH (n) WHERE $segmentID IN n.source_chunks RETURN n",
			Parameters: map[string]interface{}{
				"segmentID": segmentID,
			},
		}

		result, err := g.Graph.Query(ctx, queryOpts)
		if err != nil {
			g.Logger.Warnf("Failed to query entities for segment %s: %v", segmentID, err)
			continue
		}

		// Remove segmentID from source_chunks, delete entity if source_chunks becomes empty
		for _, node := range result.Nodes {
			sourceChunksInterface, exists := node.Properties["source_chunks"]
			if !exists {
				continue
			}

			var newSourceChunks []string
			switch sourceChunksValue := sourceChunksInterface.(type) {
			case []interface{}:
				for _, chunk := range sourceChunksValue {
					if chunkStr, ok := chunk.(string); ok && chunkStr != segmentID {
						newSourceChunks = append(newSourceChunks, chunkStr)
					}
				}
			case []string:
				for _, chunk := range sourceChunksValue {
					if chunk != segmentID {
						newSourceChunks = append(newSourceChunks, chunk)
					}
				}
			}

			// If no more source chunks, delete the entity
			if len(newSourceChunks) == 0 {
				err := g.Graph.DeleteNodes(ctx, &types.DeleteNodesOptions{
					GraphName:  graphName,
					IDs:        []string{node.ID},
					DeleteRels: true,
				})
				if err != nil {
					g.Logger.Warnf("Failed to delete entity %s: %v", node.ID, err)
				}
			} else {
				// Update entity's source_chunks
				node.Properties["source_chunks"] = newSourceChunks
				graphNode := &types.GraphNode{
					ID:         node.ID,
					Labels:     node.Labels,
					Properties: node.Properties,
				}

				_, err := g.Graph.AddNodes(ctx, &types.AddNodesOptions{
					GraphName: graphName,
					Nodes:     []*types.GraphNode{graphNode},
					Upsert:    true,
				})
				if err != nil {
					g.Logger.Warnf("Failed to update entity %s source_chunks: %v", node.ID, err)
				}
			}
		}
	}

	// Remove relationships associated with these segments
	for _, segmentID := range segmentIDs {
		// Query relationships that have this segment in their source_chunks
		queryOpts := &types.GraphQueryOptions{
			GraphName: graphName,
			QueryType: "cypher",
			Query:     "MATCH ()-[r]->() WHERE $segmentID IN r.source_chunks RETURN r",
			Parameters: map[string]interface{}{
				"segmentID": segmentID,
			},
		}

		result, err := g.Graph.Query(ctx, queryOpts)
		if err != nil {
			g.Logger.Warnf("Failed to query relationships for segment %s: %v", segmentID, err)
			continue
		}

		// Remove segmentID from source_chunks, delete relationship if source_chunks becomes empty
		for _, rel := range result.Relationships {
			sourceChunksInterface, exists := rel.Properties["source_chunks"]
			if !exists {
				continue
			}

			var newSourceChunks []string
			switch sourceChunksValue := sourceChunksInterface.(type) {
			case []interface{}:
				for _, chunk := range sourceChunksValue {
					if chunkStr, ok := chunk.(string); ok && chunkStr != segmentID {
						newSourceChunks = append(newSourceChunks, chunkStr)
					}
				}
			case []string:
				for _, chunk := range sourceChunksValue {
					if chunk != segmentID {
						newSourceChunks = append(newSourceChunks, chunk)
					}
				}
			}

			// If no more source chunks, delete the relationship
			if len(newSourceChunks) == 0 {
				err := g.Graph.DeleteRelationships(ctx, &types.DeleteRelationshipsOptions{
					GraphName: graphName,
					IDs:       []string{rel.ID},
				})
				if err != nil {
					g.Logger.Warnf("Failed to delete relationship %s: %v", rel.ID, err)
				}
			} else {
				// Update relationship's source_chunks
				rel.Properties["source_chunks"] = newSourceChunks
				graphRel := &types.GraphRelationship{
					ID:         rel.ID,
					Type:       rel.Type,
					StartNode:  rel.StartNode,
					EndNode:    rel.EndNode,
					Properties: rel.Properties,
				}

				_, err := g.Graph.AddRelationships(ctx, &types.AddRelationshipsOptions{
					GraphName:     graphName,
					Relationships: []*types.GraphRelationship{graphRel},
					Upsert:        true,
				})
				if err != nil {
					g.Logger.Warnf("Failed to update relationship %s source_chunks: %v", rel.ID, err)
				}
			}
		}
	}

	return nil
}
