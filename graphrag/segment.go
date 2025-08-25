package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// ================================================================================================
// Exported Utility Functions - Segment Text to Chunk Conversion
// ================================================================================================

// convertSegmentTextsToChunks converts SegmentTexts to Chunks for processing
func (g *GraphRag) convertSegmentTextsToChunks(segmentTexts []types.SegmentText, docID string) ([]*types.Chunk, error) {
	chunks := make([]*types.Chunk, 0, len(segmentTexts))

	for i, segmentText := range segmentTexts {
		chunkID := segmentText.ID
		if chunkID == "" {
			// Generate UUID for chunk ID
			chunkID = utils.GenChunkID()
		}

		// Create default chunk with all fields initialized
		chunk := &types.Chunk{
			ID:        chunkID,
			Text:      segmentText.Text,
			Type:      types.ChunkingTypeText,
			ParentID:  "",
			Depth:     1,
			Leaf:      true,
			Root:      true,
			Index:     i,
			Status:    types.ChunkingStatusCompleted,
			Parents:   nil,
			TextPos:   nil,
			MediaPos:  nil,
			Extracted: nil,
			Metadata:  nil,
		}

		// Merge metadata from existing segment and new segment
		var mergedMetadata map[string]interface{}

		// Step 1: Try to get existing segment metadata if chunkID is provided
		if chunkID != "" && segmentText.ID != "" {
			existingSegment, err := g.GetSegment(context.Background(), docID, chunkID)
			if err == nil && existingSegment != nil {
				// Step 2: Restore external fields back to metadata to preserve them
				existingMetadataWithFields := make(map[string]interface{})
				// Copy existing metadata
				if existingSegment.Metadata != nil {
					for k, v := range existingSegment.Metadata {
						existingMetadataWithFields[k] = v
					}
				}

				// Add external fields back to metadata so they can be preserved
				existingMetadataWithFields["weight"] = existingSegment.Weight
				existingMetadataWithFields["score"] = existingSegment.Score
				existingMetadataWithFields["positive"] = existingSegment.Positive
				existingMetadataWithFields["negative"] = existingSegment.Negative
				existingMetadataWithFields["hit"] = existingSegment.Hit

				// Step 3: Merge metadata - existing as base, new segmentText.Metadata takes precedence
				mergedMetadata = types.MergeMetadata(existingMetadataWithFields, segmentText.Metadata)

				// Step 4: Extract and set chunk fields from merged metadata
				g.applyMetadataToChunk(chunk, mergedMetadata)
			} else {
				// No existing segment found, use only new metadata
				g.applyMetadataToChunk(chunk, segmentText.Metadata)
			}
		} else {
			// No existing segment to merge with, use only new metadata
			g.applyMetadataToChunk(chunk, segmentText.Metadata)
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// applyMetadataToChunk applies metadata to chunk fields
func (g *GraphRag) applyMetadataToChunk(chunk *types.Chunk, metadata map[string]interface{}) {
	if metadata == nil {
		return
	}

	// Add metadata to chunk
	chunk.Metadata = metadata

	// Extract chunk structure information from metadata
	if chunkDetails, ok := metadata["chunk_details"].(map[string]interface{}); ok {
		chunk.Depth = types.SafeExtractInt(chunkDetails["depth"], chunk.Depth)
		chunk.Index = types.SafeExtractInt(chunkDetails["index"], chunk.Index)
		chunk.Leaf = types.SafeExtractBool(chunkDetails["is_leaf"], chunk.Leaf)
		chunk.Root = types.SafeExtractBool(chunkDetails["is_root"], chunk.Root)
		if parentID := types.SafeExtractString(chunkDetails["parent_id"], ""); parentID != "" {
			chunk.ParentID = parentID
		}
	}

	// Convert and set position information
	chunk.TextPos = types.MetadataToTextPosition(metadata)
	chunk.MediaPos = types.MetadataToMediaPosition(metadata)
	chunk.Extracted = types.MetadataToExtractionResult(metadata)

	// Set chunk type and status from metadata
	if chunkType := types.MetadataToChunkingType(metadata); chunkType != "" {
		chunk.Type = chunkType
	}
	if status := types.MetadataToChunkingStatus(metadata); status != "" {
		chunk.Status = status
	}
}

// ================================================================================================
// Exported Utility Functions - Store Operations
// ================================================================================================

// storeSegmentMetadataToStore stores segment metadata (Weight, Score, Vote) to Store and/or Vector DB
func (g *GraphRag) storeSegmentMetadataToStore(ctx context.Context, docID string, chunks []*types.Chunk, storeCollectionName string) error {
	// Strategy 1: Store not configured - metadata already stored in Vector DB during storeAllDocumentsToVectorStore
	if g.Store == nil {
		g.Logger.Debugf("Store not configured, segment metadata already stored in Vector DB metadata")
		return nil
	}

	// Strategy 2: Store configured - concurrent storage to Store and Vector DB
	// Store metadata is stored during storeAllDocumentsToVectorStore
	// Here we only need to store to Store (Vector DB storage is handled elsewhere)
	for _, chunk := range chunks {
		segmentID := chunk.ID

		// Store default Weight
		err := g.storeSegmentValue(docID, segmentID, StoreKeyWeight, 0.0)
		if err != nil {
			g.Logger.Warnf("Failed to store weight for segment %s: %v", segmentID, err)
		}

		// Store default Score
		err = g.storeSegmentValue(docID, segmentID, StoreKeyScore, 0.0)
		if err != nil {
			g.Logger.Warnf("Failed to store score for segment %s: %v", segmentID, err)
		}

		// Note: Vote is not stored as a single value - it's managed as a list via UpdateVotes function
	}

	return nil
}

// updateSegmentMetadataInStore updates segment metadata (Weight, Score) in Store from user metadata
// Note: Vote is handled separately via UpdateVotes function as it's a list, not a single value
func (g *GraphRag) updateSegmentMetadataInStore(ctx context.Context, docID string, segmentTexts []types.SegmentText, userMetadata map[string]interface{}, storeCollectionName string) error {
	if g.Store == nil {
		return nil
	}

	// Check if user metadata contains weight or score
	weight, hasWeight := userMetadata["weight"]
	score, hasScore := userMetadata["score"]

	if !hasWeight && !hasScore {
		// No relevant metadata to update
		return nil
	}

	// Update metadata for each segment
	for _, segmentText := range segmentTexts {
		segmentID := segmentText.ID

		// Update Weight if provided
		if hasWeight {
			err := g.storeSegmentValue(docID, segmentID, StoreKeyWeight, weight)
			if err != nil {
				g.Logger.Warnf("Failed to update weight for segment %s: %v", segmentID, err)
			}
		}

		// Update Score if provided
		if hasScore {
			err := g.storeSegmentValue(docID, segmentID, StoreKeyScore, score)
			if err != nil {
				g.Logger.Warnf("Failed to update score for segment %s: %v", segmentID, err)
			}
		}
	}

	return nil
}

// ================================================================================================
// Internal Types and Structures - Query System
// ================================================================================================

// segmentQueryOptions represents options for querying segment data
type segmentQueryOptions struct {
	GraphName  string   // Graph name for collection IDs
	DocID      string   // Document ID
	SegmentIDs []string // Segment IDs (for specific ID queries)
	QueryType  string   // "by_ids", "by_doc_id", "list", "scroll"

	// Pagination options
	Limit     int    // Limit for pagination
	Offset    int    // Offset for pagination
	BatchSize int    // Batch size for scroll
	ScrollID  string // Scroll ID for scroll continuation

	// Filter and ordering options
	Filter  map[string]interface{} // Metadata filter
	OrderBy []string               // Fields to order by
	Fields  []string               // Specific fields to retrieve

	// Include options
	IncludeNodes         bool // Whether to include graph nodes
	IncludeRelationships bool // Whether to include graph relationships
	IncludeMetadata      bool // Whether to include segment metadata
}

// segmentQueryResult represents the result of querying segment data from multiple databases
type segmentQueryResult struct {
	Chunks        []*types.Document         // Chunks from vector database
	Nodes         []types.GraphNode         // Nodes from graph database
	Relationships []types.GraphRelationship // Relationships from graph database
	StoreData     map[string]interface{}    // Metadata from KV store
}

// ================================================================================================
// Exported Utility Functions - Query System
// ================================================================================================

// querySegmentData queries segment data from vector, graph, and store databases concurrently
func (g *GraphRag) querySegmentData(ctx context.Context, opts *segmentQueryOptions) (*segmentQueryResult, error) {
	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(opts.GraphName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	result := &segmentQueryResult{
		StoreData: make(map[string]interface{}),
	}

	// Create error channel for concurrent operations
	errChan := make(chan error, 3)

	// Query vector database
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in vector query: %v", r)
			}
		}()
		chunks, err := g.queryChunksFromVector(ctx, collectionIDs.Vector, opts)
		if err != nil {
			errChan <- fmt.Errorf("vector query failed: %w", err)
			return
		}
		result.Chunks = chunks
		errChan <- nil
	}()

	// Query graph database (if configured)
	if g.Graph != nil && g.Graph.IsConnected() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("panic in graph query: %v", r)
				}
			}()
			nodes, relationships, err := g.queryNodesAndRelationshipsFromGraph(ctx, collectionIDs.Graph, opts)
			if err != nil {
				errChan <- fmt.Errorf("graph query failed: %w", err)
				return
			}
			result.Nodes = nodes
			result.Relationships = relationships
			errChan <- nil
		}()
	} else {
		// Send nil error if graph is not configured
		go func() {
			errChan <- nil
		}()
	}

	// Query KV store (if configured)
	if g.Store != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("panic in store query: %v", r)
				}
			}()
			storeData, err := g.queryMetadataFromStore(ctx, opts)
			if err != nil {
				errChan <- fmt.Errorf("store query failed: %w", err)
				return
			}
			result.StoreData = storeData
			errChan <- nil
		}()
	} else {
		// Send nil error if store is not configured
		go func() {
			errChan <- nil
		}()
	}

	// Wait for all queries to complete
	for i := 0; i < 3; i++ {
		if err := <-errChan; err != nil {
			return nil, err
		}
	}

	return result, nil
}

// queryChunksFromVector queries chunks from vector database
func (g *GraphRag) queryChunksFromVector(ctx context.Context, collectionName string, opts *segmentQueryOptions) ([]*types.Document, error) {
	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Vector collection %s does not exist, returning empty chunks", collectionName)
		return []*types.Document{}, nil
	}

	var chunks []*types.Document

	switch opts.QueryType {
	case "by_ids":
		// Query by specific segment IDs
		getOpts := &types.GetDocumentOptions{
			CollectionName: collectionName,
			IncludeVector:  false,
			IncludePayload: true,
		}
		chunks, err = g.Vector.GetDocuments(ctx, opts.SegmentIDs, getOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to get documents by IDs: %w", err)
		}

	case "by_doc_id":
		// Query by document ID
		listOpts := &types.ListDocumentsOptions{
			CollectionName: collectionName,
			Filter: map[string]interface{}{
				"doc_id":        opts.DocID,
				"document_type": "chunk",
			},
			Limit:          1000,
			IncludeVector:  false,
			IncludePayload: true,
		}
		result, err := g.Vector.ListDocuments(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list documents by doc_id: %w", err)
		}
		chunks = result.Documents

	default:
		return nil, fmt.Errorf("unknown query type: %s", opts.QueryType)
	}

	// Filter out nil chunks
	var validChunks []*types.Document
	for _, chunk := range chunks {
		if chunk != nil {
			validChunks = append(validChunks, chunk)
		}
	}

	return validChunks, nil
}

// queryNodesAndRelationshipsFromGraph queries nodes and relationships from graph database
func (g *GraphRag) queryNodesAndRelationshipsFromGraph(ctx context.Context, graphName string, opts *segmentQueryOptions) ([]types.GraphNode, []types.GraphRelationship, error) {
	// Check if graph exists
	exists, err := g.Graph.GraphExists(ctx, graphName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Graph %s does not exist, returning empty nodes and relationships", graphName)
		return []types.GraphNode{}, []types.GraphRelationship{}, nil
	}

	var nodes []types.GraphNode
	var relationships []types.GraphRelationship

	switch opts.QueryType {
	case "by_ids":
		// Query entities and relationships that have these segment IDs in their source_chunks
		for _, segmentID := range opts.SegmentIDs {
			// Query nodes
			nodeQueryOpts := &types.GraphQueryOptions{
				GraphName: graphName,
				QueryType: "cypher",
				Query:     "MATCH (n) WHERE $segmentID IN n.source_chunks RETURN n",
				Parameters: map[string]interface{}{
					"segmentID": segmentID,
				},
			}
			nodeResult, err := g.Graph.Query(ctx, nodeQueryOpts)
			if err != nil {
				g.Logger.Warnf("Failed to query nodes for segment %s: %v", segmentID, err)
				continue
			}
			for _, node := range nodeResult.Nodes {
				nodes = append(nodes, types.GraphNode{
					ID:         node.ID,
					Labels:     node.Labels,
					Properties: node.Properties,
				})
			}

			// Query relationships
			relQueryOpts := &types.GraphQueryOptions{
				GraphName: graphName,
				QueryType: "cypher",
				Query:     "MATCH ()-[r]->() WHERE $segmentID IN r.source_chunks RETURN r",
				Parameters: map[string]interface{}{
					"segmentID": segmentID,
				},
			}
			relResult, err := g.Graph.Query(ctx, relQueryOpts)
			if err != nil {
				g.Logger.Warnf("Failed to query relationships for segment %s: %v", segmentID, err)
				continue
			}
			for _, rel := range relResult.Relationships {
				relationships = append(relationships, types.GraphRelationship{
					ID:         rel.ID,
					Type:       rel.Type,
					StartNode:  rel.StartNode,
					EndNode:    rel.EndNode,
					Properties: rel.Properties,
				})
			}
		}

	case "by_doc_id", "list", "scroll":
		// Query all entities and relationships for this document
		nodeQueryOpts := &types.GraphQueryOptions{
			GraphName: graphName,
			QueryType: "cypher",
			Query:     "MATCH (n) WHERE n.doc_id = $docID RETURN n",
			Parameters: map[string]interface{}{
				"docID": opts.DocID,
			},
		}
		nodeResult, err := g.Graph.Query(ctx, nodeQueryOpts)
		if err != nil {
			g.Logger.Warnf("Failed to query nodes for document %s: %v", opts.DocID, err)
		} else {
			for _, node := range nodeResult.Nodes {
				nodes = append(nodes, types.GraphNode{
					ID:         node.ID,
					Labels:     node.Labels,
					Properties: node.Properties,
				})
			}
		}

		// Query relationships
		relQueryOpts := &types.GraphQueryOptions{
			GraphName: graphName,
			QueryType: "cypher",
			Query:     "MATCH ()-[r]->() WHERE r.doc_id = $docID RETURN r",
			Parameters: map[string]interface{}{
				"docID": opts.DocID,
			},
		}
		relResult, err := g.Graph.Query(ctx, relQueryOpts)
		if err != nil {
			g.Logger.Warnf("Failed to query relationships for document %s: %v", opts.DocID, err)
		} else {
			for _, rel := range relResult.Relationships {
				relationships = append(relationships, types.GraphRelationship{
					ID:         rel.ID,
					Type:       rel.Type,
					StartNode:  rel.StartNode,
					EndNode:    rel.EndNode,
					Properties: rel.Properties,
				})
			}
		}

	default:
		return nil, nil, fmt.Errorf("unknown query type: %s", opts.QueryType)
	}

	return nodes, relationships, nil
}

// queryMetadataFromStore queries metadata from Vector DB first, then Store if needed
func (g *GraphRag) queryMetadataFromStore(ctx context.Context, opts *segmentQueryOptions) (map[string]interface{}, error) {
	storeData := make(map[string]interface{})

	var segmentIDs []string
	switch opts.QueryType {
	case "by_ids":
		segmentIDs = opts.SegmentIDs
	case "by_doc_id", "list", "scroll":
		// For by_doc_id, list, and scroll queries, we need to find segments first from vector database
		// This is a limitation - we could optimize this by querying vector first
		// For now, we'll skip store data for these query types
		g.Logger.Debugf("Skipping store data query for %s type", opts.QueryType)
		return storeData, nil
	default:
		return storeData, fmt.Errorf("unknown query type: %s", opts.QueryType)
	}

	// Strategy 1: Try to get metadata from Vector DB first
	vectorData, err := g.queryMetadataFromVector(ctx, opts)
	if err != nil {
		g.Logger.Warnf("Failed to query metadata from Vector DB: %v", err)
	}

	// Strategy 2: Query from Store if needed (when Store is configured)
	var storeDataFromStore map[string]interface{}
	if g.Store != nil {
		storeDataFromStore = g.queryMetadataFromStoreOnly(opts.DocID, segmentIDs)
	}

	// Merge data: prioritize Vector DB data, fallback to Store data
	for _, segmentID := range segmentIDs {
		segmentData := make(map[string]interface{})

		// First check Vector DB data
		if vectorSegmentData, ok := vectorData[segmentID]; ok {
			if vectorMap, ok := vectorSegmentData.(map[string]interface{}); ok {
				// Copy from Vector DB
				for k, v := range vectorMap {
					segmentData[k] = v
				}
			}
		}

		// Then check Store data for missing fields (if Store is configured)
		if g.Store != nil {
			if storeSegmentData, ok := storeDataFromStore[segmentID]; ok {
				if storeMap, ok := storeSegmentData.(map[string]interface{}); ok {
					// Only add from Store if not already in Vector DB data
					for k, v := range storeMap {
						if _, exists := segmentData[k]; !exists {
							segmentData[k] = v
						}
					}
				}
			}
		}

		// Only add to result if we have data
		if len(segmentData) > 0 {
			storeData[segmentID] = segmentData
		}
	}

	return storeData, nil
}

// queryMetadataFromVector queries metadata from Vector DB
func (g *GraphRag) queryMetadataFromVector(ctx context.Context, opts *segmentQueryOptions) (map[string]interface{}, error) {
	vectorData := make(map[string]interface{})

	if g.Vector == nil {
		return vectorData, nil
	}

	var segmentIDs []string
	switch opts.QueryType {
	case "by_ids":
		segmentIDs = opts.SegmentIDs
	default:
		return vectorData, nil
	}

	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(opts.GraphName)
	if err != nil {
		return vectorData, err
	}

	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionIDs.Vector)
	if err != nil {
		return vectorData, err
	}
	if !exists {
		return vectorData, nil
	}

	// Get segment documents
	getOpts := &types.GetDocumentOptions{
		CollectionName: collectionIDs.Vector,
		IncludeVector:  false,
		IncludePayload: true,
	}

	docs, err := g.Vector.GetDocuments(ctx, segmentIDs, getOpts)
	if err != nil {
		return vectorData, err
	}

	// Extract metadata from documents
	for _, doc := range docs {
		if doc != nil && doc.Metadata != nil {
			segmentData := make(map[string]interface{})

			// Extract Score, Weight, Positive, Negative, Hit from metadata
			if score, ok := doc.Metadata["score"]; ok {
				segmentData["score"] = score
			}
			if weight, ok := doc.Metadata["weight"]; ok {
				segmentData["weight"] = weight
			}
			if positive, ok := doc.Metadata["positive"]; ok {
				segmentData["positive"] = positive
			}
			if negative, ok := doc.Metadata["negative"]; ok {
				segmentData["negative"] = negative
			}
			if hit, ok := doc.Metadata["hit"]; ok {
				segmentData["hit"] = hit
			}
			// Note: vote is not stored in Vector DB - it's a list stored in Store only

			// If Store is not configured, also extract Origin from metadata
			if g.Store == nil {
				if origin, ok := doc.Metadata["origin"]; ok {
					segmentData["origin"] = origin
				}
			}

			vectorData[doc.ID] = segmentData
		}
	}

	return vectorData, nil
}

// queryMetadataFromStoreOnly queries metadata from Store only
func (g *GraphRag) queryMetadataFromStoreOnly(docID string, segmentIDs []string) map[string]interface{} {
	storeData := make(map[string]interface{})

	if g.Store == nil {
		return storeData
	}

	// Query metadata for each segment
	for _, segmentID := range segmentIDs {
		segmentData := make(map[string]interface{})

		// Query Weight
		weight, ok := g.getSegmentValue(docID, segmentID, StoreKeyWeight)
		if ok {
			segmentData["weight"] = weight
		}

		// Query Score
		score, ok := g.getSegmentValue(docID, segmentID, StoreKeyScore)
		if ok {
			segmentData["score"] = score
		}

		// Query Score Dimensions
		scoreDimensions, ok := g.getSegmentValue(docID, segmentID, StoreKeyScoreDimensions)
		if ok {
			segmentData["score_dimensions"] = scoreDimensions
		}

		// Note: vote is a list (StoreKeyVote), not a single value, so we don't query it here
		// We only query the statistical counters: positive, negative, hit

		// Query Positive vote count
		positive, ok := g.getSegmentValue(docID, segmentID, StoreKeyVotePositive)
		if ok {
			segmentData["positive"] = positive
		}

		// Query Negative vote count
		negative, ok := g.getSegmentValue(docID, segmentID, StoreKeyVoteNegative)
		if ok {
			segmentData["negative"] = negative
		}

		// Query Hit count
		hit, ok := g.getSegmentValue(docID, segmentID, StoreKeyHitCount)
		if ok {
			segmentData["hit"] = hit
		}

		if len(segmentData) > 0 {
			storeData[segmentID] = segmentData
		}
	}

	return storeData
}
