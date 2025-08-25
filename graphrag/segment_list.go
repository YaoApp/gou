package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// ================================================================================================
// Query Operations - List and Scroll Segments
// ================================================================================================

// ListSegments lists segments of a document with pagination (deprecated)
func (g *GraphRag) ListSegments(ctx context.Context, docID string, options *types.ListSegmentsOptions) (*types.PaginatedSegmentsResult, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}

	g.Logger.Debugf("Listing segments for document: %s", docID)

	// Parse GraphName from docID
	graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
	if graphName == "" {
		graphName = "default"
	}

	// Set default options
	if options == nil {
		options = &types.ListSegmentsOptions{}
	}

	// Set default limit
	limit := options.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Query segment data from all configured databases with pagination
	segmentData, total, hasMore, nextOffset, err := g.querySegmentDataWithPagination(ctx, &segmentQueryOptions{
		GraphName:            graphName,
		DocID:                docID,
		QueryType:            "list",
		Limit:                limit,
		Offset:               options.Offset,
		Filter:               options.Filter,
		OrderBy:              options.OrderBy,
		Fields:               options.Fields,
		IncludeNodes:         options.IncludeNodes,
		IncludeRelationships: options.IncludeRelationships,
		IncludeMetadata:      options.IncludeMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query segment data: %w", err)
	}

	// Assemble segments from the queried data
	segments := g.assembleSegments(segmentData, graphName, docID)

	result := &types.PaginatedSegmentsResult{
		Segments:   segments,
		Total:      total,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	}

	g.Logger.Debugf("Successfully listed %d segments for document %s", len(segments), docID)
	return result, nil
}

// ScrollSegments scrolls through segments of a document with iterator-style pagination
func (g *GraphRag) ScrollSegments(ctx context.Context, docID string, options *types.ScrollSegmentsOptions) (*types.SegmentScrollResult, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}

	g.Logger.Debugf("Scrolling segments for document: %s", docID)

	// Parse GraphName from docID
	graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
	if graphName == "" {
		graphName = "default"
	}

	// Set default options
	if options == nil {
		options = &types.ScrollSegmentsOptions{}
	}

	// Set default limit
	limit := options.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Query segment data from all configured databases with scroll
	segmentData, scrollID, hasMore, err := g.querySegmentDataWithScroll(ctx, &segmentQueryOptions{
		GraphName:            graphName,
		DocID:                docID,
		QueryType:            "scroll",
		BatchSize:            limit,
		ScrollID:             options.ScrollID,
		Filter:               options.Filter,
		OrderBy:              options.OrderBy,
		Fields:               options.Fields,
		IncludeNodes:         options.IncludeNodes,
		IncludeRelationships: options.IncludeRelationships,
		IncludeMetadata:      options.IncludeMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query segment data: %w", err)
	}

	// Assemble segments from the queried data
	segments := g.assembleSegments(segmentData, graphName, docID)

	result := &types.SegmentScrollResult{
		Segments: segments,
		ScrollID: scrollID,
		HasMore:  hasMore,
	}

	g.Logger.Debugf("Successfully scrolled %d segments for document %s", len(segments), docID)
	return result, nil
}

// ================================================================================================
// Internal Helper Methods - Query System with Pagination
// ================================================================================================

// querySegmentDataWithPagination queries segment data with pagination support
func (g *GraphRag) querySegmentDataWithPagination(ctx context.Context, opts *segmentQueryOptions) (*segmentQueryResult, int64, bool, int, error) {
	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(opts.GraphName)
	if err != nil {
		return nil, 0, false, 0, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	result := &segmentQueryResult{
		StoreData: make(map[string]interface{}),
	}

	var totalCount int64 = 0
	var hasMore bool = false
	var nextOffset int = opts.Offset

	// Create error channel for concurrent operations
	errChan := make(chan error, 3)

	// Query vector database with pagination
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in vector pagination query: %v", r)
			}
		}()
		chunks, total, hasMorePages, nextOff, err := g.queryChunksFromVectorWithPagination(ctx, collectionIDs.Vector, opts)
		if err != nil {
			errChan <- fmt.Errorf("vector pagination query failed: %w", err)
			return
		}
		result.Chunks = chunks
		totalCount = total
		hasMore = hasMorePages
		nextOffset = nextOff
		errChan <- nil
	}()

	// Query graph database (if configured)
	if g.Graph != nil && g.Graph.IsConnected() && opts.IncludeNodes {
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
	if g.Store != nil && opts.IncludeMetadata {
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
			return nil, 0, false, 0, err
		}
	}

	return result, totalCount, hasMore, nextOffset, nil
}

// querySegmentDataWithScroll queries segment data with scroll support
func (g *GraphRag) querySegmentDataWithScroll(ctx context.Context, opts *segmentQueryOptions) (*segmentQueryResult, string, bool, error) {
	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(opts.GraphName)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	result := &segmentQueryResult{
		StoreData: make(map[string]interface{}),
	}

	var scrollID string = ""
	var hasMore bool = false

	// Create error channel for concurrent operations
	errChan := make(chan error, 3)

	// Query vector database with scroll
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in vector scroll query: %v", r)
			}
		}()
		chunks, nextScrollID, hasMorePages, err := g.queryChunksFromVectorWithScroll(ctx, collectionIDs.Vector, opts)
		if err != nil {
			errChan <- fmt.Errorf("vector scroll query failed: %w", err)
			return
		}
		result.Chunks = chunks
		scrollID = nextScrollID
		hasMore = hasMorePages
		errChan <- nil
	}()

	// Query graph database (if configured)
	if g.Graph != nil && g.Graph.IsConnected() && opts.IncludeNodes {
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
	if g.Store != nil && opts.IncludeMetadata {
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
			return nil, "", false, err
		}
	}

	return result, scrollID, hasMore, nil
}

// ================================================================================================
// Internal Helper Methods - Vector Database Queries
// ================================================================================================

// queryChunksFromVectorWithPagination queries chunks from vector database with pagination
func (g *GraphRag) queryChunksFromVectorWithPagination(ctx context.Context, collectionName string, opts *segmentQueryOptions) ([]*types.Document, int64, bool, int, error) {
	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, 0, false, 0, fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Vector collection %s does not exist, returning empty chunks", collectionName)
		return []*types.Document{}, 0, false, 0, nil
	}

	// Build filter for documents
	filter := map[string]interface{}{
		"doc_id":        opts.DocID,
		"document_type": "chunk",
	}

	// Add user-provided filter if any
	if opts.Filter != nil {
		for k, v := range opts.Filter {
			filter[k] = v
		}
	}

	// Set limit, using default if not provided
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	// List documents with pagination
	listOpts := &types.ListDocumentsOptions{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          limit,
		Offset:         opts.Offset,
		OrderBy:        opts.OrderBy,
		Fields:         opts.Fields,
		IncludeVector:  false,
		IncludePayload: true,
	}

	result, err := g.Vector.ListDocuments(ctx, listOpts)
	if err != nil {
		return nil, 0, false, 0, fmt.Errorf("failed to list documents with pagination: %w", err)
	}

	// Filter out nil chunks
	var validChunks []*types.Document
	for _, chunk := range result.Documents {
		if chunk != nil {
			validChunks = append(validChunks, chunk)
		}
	}

	// Calculate next offset
	nextOffset := opts.Offset + len(validChunks)
	if nextOffset >= int(result.Total) {
		nextOffset = int(result.Total)
	}

	return validChunks, result.Total, result.HasMore, nextOffset, nil
}

// queryChunksFromVectorWithScroll queries chunks from vector database with scroll
func (g *GraphRag) queryChunksFromVectorWithScroll(ctx context.Context, collectionName string, opts *segmentQueryOptions) ([]*types.Document, string, bool, error) {
	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		g.Logger.Infof("Vector collection %s does not exist, returning empty chunks", collectionName)
		return []*types.Document{}, "", false, nil
	}

	// Build filter for documents
	filter := map[string]interface{}{
		"doc_id":        opts.DocID,
		"document_type": "chunk",
	}

	// Add user-provided filter if any
	if opts.Filter != nil {
		for k, v := range opts.Filter {
			filter[k] = v
		}
	}

	// Set limit, using default if not provided
	limit := opts.BatchSize
	if limit <= 0 {
		limit = 100
	}

	// Scroll documents
	scrollOpts := &types.ScrollOptions{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          limit,
		ScrollID:       opts.ScrollID,
		OrderBy:        opts.OrderBy,
		Fields:         opts.Fields,
		IncludeVector:  false,
		IncludePayload: true,
	}

	result, err := g.Vector.ScrollDocuments(ctx, scrollOpts)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to scroll documents: %w", err)
	}

	// Filter out nil chunks
	var validChunks []*types.Document
	for _, chunk := range result.Documents {
		if chunk != nil {
			validChunks = append(validChunks, chunk)
		}
	}

	return validChunks, result.ScrollID, result.HasMore, nil
}
