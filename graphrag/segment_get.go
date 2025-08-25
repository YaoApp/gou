package graphrag

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// ================================================================================================
// Query Operations - Get Segments
// ================================================================================================

// GetSegment gets a single segment by ID
func (g *GraphRag) GetSegment(ctx context.Context, docID string, segmentID string) (*types.Segment, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}

	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Getting segment by ID: %s for document: %s", segmentID, docID)

	// Get segments using GetSegments
	segments, err := g.GetSegments(ctx, docID, []string{segmentID})
	if err != nil {
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("segment %s not found", segmentID)
	}

	return &segments[0], nil
}

// GetSegmentParents gets parent tree of a given segment
func (g *GraphRag) GetSegmentParents(ctx context.Context, docID string, segmentID string) (*types.SegmentTree, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}

	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Getting parent tree for segment ID: %s in document: %s", segmentID, docID)

	// Get the target segment first
	segment, err := g.GetSegment(ctx, docID, segmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}

	// Build the tree structure starting from the target segment
	tree, err := g.buildSegmentTree(ctx, docID, segment)
	if err != nil {
		return nil, fmt.Errorf("failed to build segment tree: %w", err)
	}

	g.Logger.Debugf("Successfully built parent tree for segment %s", segmentID)
	return tree, nil
}

// GetSegments gets segments by IDs
func (g *GraphRag) GetSegments(ctx context.Context, docID string, segmentIDs []string) ([]types.Segment, error) {
	if len(segmentIDs) == 0 {
		return []types.Segment{}, nil
	}

	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}

	g.Logger.Debugf("Getting %d segments by IDs for document %s", len(segmentIDs), docID)

	// Extract GraphName from docID
	graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
	if graphName == "" {
		graphName = "default"
	}

	// Query segment data from all configured databases
	segmentData, err := g.querySegmentData(ctx, &segmentQueryOptions{
		GraphName:  graphName,
		DocID:      docID,
		SegmentIDs: segmentIDs,
		QueryType:  "by_ids",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query segment data: %w", err)
	}

	// Assemble segments from the queried data
	segments := g.assembleSegments(segmentData, graphName, docID)

	g.Logger.Debugf("Successfully retrieved %d segments", len(segments))
	return segments, nil
}

// ================================================================================================
// Internal Helper Methods - Tree Building
// ================================================================================================

// buildSegmentTree recursively builds a tree structure of segment parents using metadata
func (g *GraphRag) buildSegmentTree(ctx context.Context, docID string, segment *types.Segment) (*types.SegmentTree, error) {
	// Extract depth from segment metadata
	segmentDepth := g.extractDepthFromSegment(segment)

	// Create the tree node for the current segment
	tree := &types.SegmentTree{
		Segment: segment,
		Parent:  nil,
		Depth:   segmentDepth,
	}

	// Extract parent ID from segment metadata (only one parent in document hierarchy)
	parentID := g.extractParentIDFromSegment(segment)
	if parentID == "" {
		// No parent found, this is a root node
		return tree, nil
	}

	// Get the parent segment
	parentSegment, err := g.GetSegment(ctx, docID, parentID)
	if err != nil {
		g.Logger.Warnf("Failed to get parent segment %s: %v", parentID, err)
		// Return the tree without parent rather than failing completely
		return tree, nil
	}

	// Build parent tree recursively
	parentTree, err := g.buildSegmentTree(ctx, docID, parentSegment)
	if err != nil {
		g.Logger.Warnf("Failed to build parent tree for segment %s: %v", parentSegment.ID, err)
		return tree, nil
	}

	// Set the parent
	tree.Parent = parentTree

	return tree, nil
}

// extractParentIDFromSegment extracts the parent ID from segment metadata
func (g *GraphRag) extractParentIDFromSegment(segment *types.Segment) string {
	if segment == nil {
		return ""
	}

	// Strategy 1: Check segment's Parents field (take the first one, should only be one)
	if len(segment.Parents) > 0 {
		return segment.Parents[0]
	}

	// Strategy 2: Check chunk_details in metadata for parent_id
	if segment.Metadata != nil {
		if chunkDetails, ok := segment.Metadata["chunk_details"].(map[string]interface{}); ok {
			if parentID := types.SafeExtractString(chunkDetails["parent_id"], ""); parentID != "" {
				return parentID
			}
		}

		// Strategy 3: Check direct parent_id in metadata
		if parentID := types.SafeExtractString(segment.Metadata["parent_id"], ""); parentID != "" {
			return parentID
		}
	}

	return ""
}

// extractDepthFromSegment extracts depth value from segment metadata
func (g *GraphRag) extractDepthFromSegment(segment *types.Segment) int {
	if segment == nil {
		return 0
	}

	// Strategy 1: Check chunk_details in metadata for depth
	if segment.Metadata != nil {
		if chunkDetails, ok := segment.Metadata["chunk_details"].(map[string]interface{}); ok {
			if depth := types.SafeExtractInt(chunkDetails["depth"], 0); depth > 0 {
				return depth
			}
		}

		// Strategy 2: Check direct depth in metadata
		if depth := types.SafeExtractInt(segment.Metadata["depth"], 0); depth > 0 {
			return depth
		}
	}

	// Strategy 3: Fallback - calculate from parent chain length if available
	// This is a fallback when depth is not stored in metadata
	if len(segment.Parents) > 0 {
		// If we have parents, we're at least depth 1 (not root)
		// This is a rough estimation, the actual depth should be in metadata
		return len(segment.Parents)
	}

	// Default: assume root level (depth 1) if no depth information found
	return 1
}

// ================================================================================================
// Internal Helper Methods - Segment Assembly
// ================================================================================================

// assembleSegments assembles segments from the queried data
func (g *GraphRag) assembleSegments(data *segmentQueryResult, graphName string, docID string) []types.Segment {
	var segments []types.Segment

	// Create segments from chunks
	for _, chunk := range data.Chunks {
		if chunk == nil {
			continue
		}

		segment := types.Segment{
			CollectionID:  graphName,
			DocumentID:    docID,
			ID:            chunk.ID,
			Text:          chunk.Content,
			Metadata:      chunk.Metadata,
			Nodes:         []types.GraphNode{},
			Relationships: []types.GraphRelationship{},
			Parents:       []string{},
			Children:      []string{},
			Version:       1,
			Weight:        0.0,
			Score:         0.0,
			Positive:      0,
			Negative:      0,
			Hit:           0,
		}

		// Set timestamps from metadata if available
		if createdAt, ok := chunk.Metadata["created_at"]; ok {
			if createdAtStr, ok := createdAt.(string); ok {
				if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
					segment.CreatedAt = t
				}
			}
		}
		if updatedAt, ok := chunk.Metadata["updated_at"]; ok {
			if updatedAtStr, ok := updatedAt.(string); ok {
				if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
					segment.UpdatedAt = t
				}
			}
		}

		// Add nodes that are related to this segment
		for _, node := range data.Nodes {
			if g.isNodeRelatedToSegment(node, chunk.ID) {
				segment.Nodes = append(segment.Nodes, node)
			}
		}

		// Add relationships that are related to this segment
		for _, rel := range data.Relationships {
			if g.isRelationshipRelatedToSegment(rel, chunk.ID) {
				segment.Relationships = append(segment.Relationships, rel)
			}
		}

		// Extract weight/score/vote from chunk metadata (Vector DB data) using safe extraction
		if chunk.Metadata != nil {
			segment.Weight = types.SafeExtractFloat64(chunk.Metadata["weight"], segment.Weight)
			segment.Score = types.SafeExtractFloat64(chunk.Metadata["score"], segment.Score)
			segment.Positive = types.SafeExtractInt(chunk.Metadata["positive"], segment.Positive)
			segment.Negative = types.SafeExtractInt(chunk.Metadata["negative"], segment.Negative)
			segment.Hit = types.SafeExtractInt(chunk.Metadata["hit"], segment.Hit)

			// Extract score dimensions if available
			if scoreDimensions, ok := chunk.Metadata["score_dimensions"]; ok {
				if dimensionsMap, ok := scoreDimensions.(map[string]interface{}); ok {
					segment.ScoreDimensions = make(map[string]float64)
					for key, value := range dimensionsMap {
						segment.ScoreDimensions[key] = types.SafeExtractFloat64(value, 0.0)
					}
				}
			}

			// Remove these fields from metadata since they're now in external fields
			delete(segment.Metadata, "weight")
			delete(segment.Metadata, "score")
			delete(segment.Metadata, "score_dimensions")
			delete(segment.Metadata, "positive")
			delete(segment.Metadata, "negative")
			delete(segment.Metadata, "hit")
			// Note: vote is not stored in Vector DB - it's a list stored in Store only
		}

		// Also add metadata from store (for backward compatibility if Store is configured)
		if segmentData, ok := data.StoreData[chunk.ID]; ok {
			if segmentMap, ok := segmentData.(map[string]interface{}); ok {
				// Store data takes precedence over Vector DB data if both exist, using safe extraction
				segment.Weight = types.SafeExtractFloat64(segmentMap["weight"], segment.Weight)
				segment.Score = types.SafeExtractFloat64(segmentMap["score"], segment.Score)
				segment.Positive = types.SafeExtractInt(segmentMap["positive"], segment.Positive)
				segment.Negative = types.SafeExtractInt(segmentMap["negative"], segment.Negative)
				segment.Hit = types.SafeExtractInt(segmentMap["hit"], segment.Hit)

				// Extract score dimensions from store if available
				if scoreDimensions, ok := segmentMap["score_dimensions"]; ok {
					if dimensionsMap, ok := scoreDimensions.(map[string]interface{}); ok {
						segment.ScoreDimensions = make(map[string]float64)
						for key, value := range dimensionsMap {
							segment.ScoreDimensions[key] = types.SafeExtractFloat64(value, 0.0)
						}
					}
				}

				// Ensure these fields are not duplicated in metadata
				delete(segment.Metadata, "weight")
				delete(segment.Metadata, "score")
				delete(segment.Metadata, "score_dimensions")
				delete(segment.Metadata, "positive")
				delete(segment.Metadata, "negative")
				delete(segment.Metadata, "hit")
				// Note: vote is not a single value - it's a list stored in Store only
			}
		}

		segments = append(segments, segment)
	}

	return segments
}

// isNodeRelatedToSegment checks if a node is related to a segment
func (g *GraphRag) isNodeRelatedToSegment(node types.GraphNode, segmentID string) bool {
	// Check if the segment ID is in the node's source_chunks
	if sourceChunks, ok := node.Properties["source_chunks"]; ok {
		switch sourceChunksValue := sourceChunks.(type) {
		case []interface{}:
			for _, chunk := range sourceChunksValue {
				if chunkStr, ok := chunk.(string); ok && chunkStr == segmentID {
					return true
				}
			}
		case []string:
			for _, chunk := range sourceChunksValue {
				if chunk == segmentID {
					return true
				}
			}
		}
	}
	return false
}

// isRelationshipRelatedToSegment checks if a relationship is related to a segment
func (g *GraphRag) isRelationshipRelatedToSegment(rel types.GraphRelationship, segmentID string) bool {
	// Check if the segment ID is in the relationship's source_chunks
	if sourceChunks, ok := rel.Properties["source_chunks"]; ok {
		switch sourceChunksValue := sourceChunks.(type) {
		case []interface{}:
			for _, chunk := range sourceChunksValue {
				if chunkStr, ok := chunk.(string); ok && chunkStr == segmentID {
					return true
				}
			}
		case []string:
			for _, chunk := range sourceChunksValue {
				if chunk == segmentID {
					return true
				}
			}
		}
	}
	return false
}
