package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/utils"
)

// SegmentCount returns the number of segments for a given document
func (g *GraphRag) SegmentCount(ctx context.Context, docID string) (int, error) {
	if docID == "" {
		return 0, fmt.Errorf("document ID is required")
	}

	g.Logger.Debugf("Getting segment count for document: %s", docID)

	// Get count from vector store
	if g.Vector != nil {
		count, err := g.getSegmentCountFromVector(ctx, docID)
		if err == nil {
			g.Logger.Debugf("Got segment count from vector store: %d", count)
			return count, nil
		}
		g.Logger.Warnf("Failed to get segment count from vector store: %v", err)
	}

	return 0, fmt.Errorf("failed to get segment count for document %s: vector store not available", docID)
}

// getSegmentCountFromVector gets segment count from vector database
func (g *GraphRag) getSegmentCountFromVector(ctx context.Context, docID string) (int, error) {
	// Extract collection ID from document ID using utils
	graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
	if graphName == "" {
		graphName = "default" // Backward compatibility
	}

	// Get collection IDs for the graph
	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	vectorCollection := collectionIDs.Vector

	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, vectorCollection)
	if err != nil {
		return 0, fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Debugf("Vector collection %s does not exist", vectorCollection)
		return 0, nil
	}

	// Count documents with matching doc_id and document_type = "chunk"
	filter := map[string]interface{}{
		"doc_id":        docID,
		"document_type": "chunk",
	}

	g.Logger.Debugf("Counting segments with filter: %+v in collection: %s", filter, vectorCollection)

	count, err := g.Vector.Count(ctx, vectorCollection, filter)
	if err != nil {
		g.Logger.Errorf("Failed to count segments: %v", err)
		return 0, fmt.Errorf("failed to count segments in vector store: %w", err)
	}

	g.Logger.Debugf("Found %d segments for document %s", count, docID)
	return int(count), nil
}
