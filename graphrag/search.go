package graphrag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// defaultSearchK is the default number of results to return
const defaultSearchK = 10

// Search searches for segments based on query options
// It performs vector similarity search and optionally enriches results with graph data
func (g *GraphRag) Search(ctx context.Context, options *types.QueryOptions, callback ...types.SearcherProgress) ([]types.Segment, error) {
	if options == nil {
		return nil, fmt.Errorf("query options cannot be nil")
	}

	// Validate required fields
	if options.Query == "" && len(options.History) == 0 {
		return nil, fmt.Errorf("either query or history is required")
	}

	// Report starting status
	g.reportSearchProgress(callback, types.SearchStatusPending, "Starting search...", 0)

	// Step 1: Determine the collection to search
	collectionID := options.CollectionID
	if collectionID == "" {
		// Try to extract from DocumentID if provided
		if options.DocumentID != "" {
			collectionID, _ = utils.ExtractCollectionIDFromDocID(options.DocumentID)
		}
		if collectionID == "" {
			return nil, fmt.Errorf("collection ID is required (either via CollectionID or DocumentID)")
		}
	}

	// Get collection IDs (vector, graph, store)
	collectionIDs, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	// Step 2: Get query text (from Query or last message in History)
	queryText := options.Query
	if queryText == "" && len(options.History) > 0 {
		// Use the last user message from history as query
		for i := len(options.History) - 1; i >= 0; i-- {
			if options.History[i].Role == "user" {
				queryText = options.History[i].Content
				break
			}
		}
	}
	if queryText == "" {
		return nil, fmt.Errorf("no query text found in query or history")
	}

	g.Logger.Debugf("Searching with query: %s in collection: %s", queryText, collectionID)
	g.reportSearchProgress(callback, types.SearchStatusPending, "Generating embedding...", 10)

	// Step 3: Generate embedding for the query
	if options.Embedding == nil {
		return nil, fmt.Errorf("embedding function is required for search")
	}

	embeddingResult, err := options.Embedding.EmbedQuery(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	queryVector := embeddingResult.Embedding
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("embedding returned empty vector")
	}

	g.reportSearchProgress(callback, types.SearchStatusPending, "Searching vector database...", 30)

	// Step 4: Search in vector database
	searchOpts := &types.SearchOptions{
		CollectionName:  collectionIDs.Vector,
		QueryVector:     queryVector,
		K:               defaultSearchK,
		Filter:          options.Filter,
		IncludeMetadata: true,
		IncludeContent:  true,
	}

	// Add document filter if DocumentID is specified
	if options.DocumentID != "" {
		if searchOpts.Filter == nil {
			searchOpts.Filter = make(map[string]interface{})
		}
		searchOpts.Filter["doc_id"] = options.DocumentID
	}

	searchResult, err := g.Vector.SearchSimilar(ctx, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	if len(searchResult.Documents) == 0 {
		g.Logger.Debugf("No documents found in vector search")
		g.reportSearchProgress(callback, types.SearchStatusSuccess, "Search completed, no results found", 100)
		return []types.Segment{}, nil
	}

	g.Logger.Debugf("Found %d documents in vector search", len(searchResult.Documents))
	g.reportSearchProgress(callback, types.SearchStatusPending, "Converting results to segments...", 60)

	// Step 5: Convert search results to segments
	segments := g.convertSearchResultsToSegments(searchResult, collectionID, options.DocumentID)

	// Step 6: Optionally enrich with graph data (if Graph store is configured)
	if g.Graph != nil && g.Graph.IsConnected() {
		g.reportSearchProgress(callback, types.SearchStatusPending, "Enriching with graph data...", 80)
		segments = g.enrichSegmentsWithGraphData(ctx, segments, collectionIDs.Graph)
	}

	// Step 7: Apply reranking if configured
	if options.Reranker != nil {
		g.reportSearchProgress(callback, types.SearchStatusPending, "Reranking results...", 90)
		segments, err = options.Reranker.Rerank(ctx, segments)
		if err != nil {
			g.Logger.Warnf("Reranking failed: %v, using original order", err)
		}
	}

	g.reportSearchProgress(callback, types.SearchStatusSuccess, fmt.Sprintf("Search completed, found %d segments", len(segments)), 100)
	return segments, nil
}

// MultiSearch performs multiple searches in parallel and returns results grouped by query
func (g *GraphRag) MultiSearch(ctx context.Context, options []types.QueryOptions, callback ...types.SearcherProgress) (map[string][]types.Segment, error) {
	if len(options) == 0 {
		return make(map[string][]types.Segment), nil
	}

	g.reportSearchProgress(callback, types.SearchStatusPending, fmt.Sprintf("Starting multi-search with %d queries...", len(options)), 0)

	results := make(map[string][]types.Segment)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(options))

	// Execute searches in parallel
	for i := range options {
		wg.Add(1)
		go func(idx int, opts types.QueryOptions) {
			defer wg.Done()

			// Generate a unique key for this query
			queryKey := g.generateQueryKey(&opts, idx)

			// Perform the search
			segments, err := g.Search(ctx, &opts)
			if err != nil {
				errChan <- fmt.Errorf("search %d failed: %w", idx, err)
				return
			}

			// Store results
			mu.Lock()
			results[queryKey] = segments
			mu.Unlock()
		}(i, options[i])
	}

	// Wait for all searches to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Return partial results with first error
		g.Logger.Warnf("MultiSearch completed with %d errors", len(errors))
		return results, errors[0]
	}

	g.reportSearchProgress(callback, types.SearchStatusSuccess, fmt.Sprintf("Multi-search completed, %d queries processed", len(options)), 100)
	return results, nil
}

// ================================================================================================
// Internal Helper Methods
// ================================================================================================

// convertSearchResultsToSegments converts vector search results to segments
func (g *GraphRag) convertSearchResultsToSegments(result *types.SearchResult, collectionID string, docID string) []types.Segment {
	segments := make([]types.Segment, 0, len(result.Documents))

	for _, item := range result.Documents {
		if item == nil {
			continue
		}

		doc := &item.Document

		// Extract document ID from metadata if not provided
		documentID := docID
		if documentID == "" && doc.Metadata != nil {
			if metaDocID, ok := doc.Metadata["doc_id"].(string); ok {
				documentID = metaDocID
			}
		}

		segment := types.Segment{
			CollectionID:  collectionID,
			DocumentID:    documentID,
			ID:            doc.ID,
			Text:          doc.Content,
			Metadata:      doc.Metadata,
			Nodes:         []types.GraphNode{},
			Relationships: []types.GraphRelationship{},
			Parents:       []string{},
			Children:      []string{},
			Version:       1,
			Score:         item.Score, // Use the search score
		}

		// Extract additional fields from metadata
		if doc.Metadata != nil {
			// Extract weight/score/vote from metadata
			segment.Weight = types.SafeExtractFloat64(doc.Metadata["weight"], 0.0)
			segment.Positive = types.SafeExtractInt(doc.Metadata["positive"], 0)
			segment.Negative = types.SafeExtractInt(doc.Metadata["negative"], 0)
			segment.Hit = types.SafeExtractInt(doc.Metadata["hit"], 0)

			// Extract timestamps
			if createdAt, ok := doc.Metadata["created_at"]; ok {
				if createdAtStr, ok := createdAt.(string); ok {
					if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
						segment.CreatedAt = t
					}
				}
			}
			if updatedAt, ok := doc.Metadata["updated_at"]; ok {
				if updatedAtStr, ok := updatedAt.(string); ok {
					if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
						segment.UpdatedAt = t
					}
				}
			}

			// Extract parents from metadata
			if parents, ok := doc.Metadata["parents"]; ok {
				switch p := parents.(type) {
				case []string:
					segment.Parents = p
				case []interface{}:
					for _, parent := range p {
						if parentStr, ok := parent.(string); ok {
							segment.Parents = append(segment.Parents, parentStr)
						}
					}
				}
			}

			// Extract score dimensions
			if scoreDimensions, ok := doc.Metadata["score_dimensions"]; ok {
				if dimensionsMap, ok := scoreDimensions.(map[string]interface{}); ok {
					segment.ScoreDimensions = make(map[string]float64)
					for key, value := range dimensionsMap {
						segment.ScoreDimensions[key] = types.SafeExtractFloat64(value, 0.0)
					}
				}
			}

			// Clean up metadata by removing extracted fields
			delete(segment.Metadata, "weight")
			delete(segment.Metadata, "score")
			delete(segment.Metadata, "score_dimensions")
			delete(segment.Metadata, "positive")
			delete(segment.Metadata, "negative")
			delete(segment.Metadata, "hit")
		}

		segments = append(segments, segment)
	}

	return segments
}

// enrichSegmentsWithGraphData enriches segments with node and relationship data from graph database
func (g *GraphRag) enrichSegmentsWithGraphData(ctx context.Context, segments []types.Segment, graphName string) []types.Segment {
	if len(segments) == 0 {
		return segments
	}

	// Check if graph exists
	exists, err := g.Graph.GraphExists(ctx, graphName)
	if err != nil || !exists {
		g.Logger.Debugf("Graph %s does not exist or check failed, skipping graph enrichment", graphName)
		return segments
	}

	// Collect all segment IDs for batch query
	segmentIDs := make([]string, len(segments))
	for i, seg := range segments {
		segmentIDs[i] = seg.ID
	}

	// Query nodes related to these segments
	nodes, err := g.queryNodesForSegments(ctx, graphName, segmentIDs)
	if err != nil {
		g.Logger.Warnf("Failed to query nodes for segments: %v", err)
		return segments
	}

	// Query relationships related to these segments
	relationships, err := g.queryRelationshipsForSegments(ctx, graphName, segmentIDs)
	if err != nil {
		g.Logger.Warnf("Failed to query relationships for segments: %v", err)
		return segments
	}

	// Enrich each segment with its related nodes and relationships
	for i := range segments {
		segmentID := segments[i].ID

		// Add related nodes
		for _, node := range nodes {
			if g.isNodeRelatedToSegment(node, segmentID) {
				segments[i].Nodes = append(segments[i].Nodes, node)
			}
		}

		// Add related relationships
		for _, rel := range relationships {
			if g.isRelationshipRelatedToSegment(rel, segmentID) {
				segments[i].Relationships = append(segments[i].Relationships, rel)
			}
		}
	}

	return segments
}

// queryNodesForSegments queries nodes that are related to the given segment IDs
func (g *GraphRag) queryNodesForSegments(ctx context.Context, graphName string, segmentIDs []string) ([]types.GraphNode, error) {
	// Build Cypher query to find nodes with source_chunks containing any of the segment IDs
	query := `
		MATCH (n)
		WHERE any(chunk IN $segment_ids WHERE chunk IN n.source_chunks)
		RETURN n
	`

	queryOpts := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      query,
		Parameters: map[string]interface{}{"segment_ids": segmentIDs},
		ReturnType: "nodes",
	}

	result, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		return nil, err
	}

	// Convert Node to GraphNode
	graphNodes := make([]types.GraphNode, len(result.Nodes))
	for i, node := range result.Nodes {
		graphNodes[i] = convertNodeToGraphNode(node)
	}

	return graphNodes, nil
}

// queryRelationshipsForSegments queries relationships that are related to the given segment IDs
func (g *GraphRag) queryRelationshipsForSegments(ctx context.Context, graphName string, segmentIDs []string) ([]types.GraphRelationship, error) {
	// Build Cypher query to find relationships with source_chunks containing any of the segment IDs
	query := `
		MATCH ()-[r]->()
		WHERE any(chunk IN $segment_ids WHERE chunk IN r.source_chunks)
		RETURN r
	`

	queryOpts := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      query,
		Parameters: map[string]interface{}{"segment_ids": segmentIDs},
		ReturnType: "relationships",
	}

	result, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		return nil, err
	}

	// Convert Relationship to GraphRelationship
	graphRels := make([]types.GraphRelationship, len(result.Relationships))
	for i, rel := range result.Relationships {
		graphRels[i] = convertRelationshipToGraphRelationship(rel)
	}

	return graphRels, nil
}

// convertNodeToGraphNode converts a Node to GraphNode
func convertNodeToGraphNode(node types.Node) types.GraphNode {
	return types.GraphNode{
		ID:          node.ID,
		Labels:      node.Labels,
		Properties:  node.Properties,
		EntityType:  node.Type,
		Description: node.Description,
		Confidence:  node.Confidence,
		CreatedAt:   time.Unix(node.CreatedAt, 0),
		UpdatedAt:   time.Unix(node.UpdatedAt, 0),
		Version:     node.Version,
	}
}

// convertRelationshipToGraphRelationship converts a Relationship to GraphRelationship
func convertRelationshipToGraphRelationship(rel types.Relationship) types.GraphRelationship {
	return types.GraphRelationship{
		ID:          rel.ID,
		Type:        rel.Type,
		StartNode:   rel.StartNode,
		EndNode:     rel.EndNode,
		Properties:  rel.Properties,
		Description: rel.Description,
		Confidence:  rel.Confidence,
		Weight:      rel.Weight,
		CreatedAt:   time.Unix(rel.CreatedAt, 0),
		UpdatedAt:   time.Unix(rel.UpdatedAt, 0),
		Version:     rel.Version,
	}
}

// generateQueryKey generates a unique key for a query in multi-search
func (g *GraphRag) generateQueryKey(opts *types.QueryOptions, index int) string {
	if opts.Query != "" {
		// Use query text as key (truncated if too long)
		key := opts.Query
		if len(key) > 50 {
			key = key[:50] + "..."
		}
		return key
	}

	// Use index as fallback
	return fmt.Sprintf("query_%d", index)
}

// reportSearchProgress reports search progress via callback
func (g *GraphRag) reportSearchProgress(callback []types.SearcherProgress, status types.SearcherStatus, message string, progress float64) {
	if len(callback) > 0 && callback[0] != nil {
		callback[0](status, types.SearcherPayload{
			Status:   status,
			Message:  message,
			Progress: progress,
		})
	}
}

// ================================================================================================
// Pure Vector Search
// ================================================================================================

// SearchVector performs pure vector similarity search without graph enrichment
// This is useful for application-side orchestration where vector and graph searches are composed separately
func (g *GraphRag) SearchVector(ctx context.Context, options *types.VectorSearchOptions, callback ...types.SearcherProgress) (*types.VectorSearchResult, error) {
	if options == nil {
		return nil, fmt.Errorf("vector search options cannot be nil")
	}

	// Validate collection ID
	if options.CollectionID == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Validate query
	if options.Query == "" && len(options.QueryVector) == 0 {
		return nil, fmt.Errorf("either query text or query vector is required")
	}

	// Connect to vector store if not already connected
	if g.Vector != nil && !g.Vector.IsConnected() {
		err := g.Vector.Connect(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to vector store: %w", err)
		}
		g.Logger.Infof("Connected to vector store")
	}

	g.reportSearchProgress(callback, types.SearchStatusPending, "Starting vector search...", 0)

	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(options.CollectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	// Determine query vector
	var queryVector []float64
	if len(options.QueryVector) > 0 {
		// Use pre-computed vector
		queryVector = options.QueryVector
	} else {
		// Generate embedding
		if options.Embedding == nil {
			return nil, fmt.Errorf("embedding function is required when query vector is not provided")
		}

		g.reportSearchProgress(callback, types.SearchStatusPending, "Generating embedding...", 20)

		embeddingResult, err := options.Embedding.EmbedQuery(ctx, options.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}

		queryVector = embeddingResult.Embedding
		if len(queryVector) == 0 {
			return nil, fmt.Errorf("embedding returned empty vector")
		}
	}

	g.reportSearchProgress(callback, types.SearchStatusPending, "Searching vector database...", 40)

	// Prepare search options
	k := options.K
	if k <= 0 {
		k = defaultSearchK
	}

	searchOpts := &types.SearchOptions{
		CollectionName:  collectionIDs.Vector,
		QueryVector:     queryVector,
		K:               k,
		Filter:          options.Filter,
		MinScore:        options.MinScore,
		IncludeMetadata: true,
		IncludeContent:  true,
	}

	// Add document filter if specified
	if options.DocumentID != "" {
		if searchOpts.Filter == nil {
			searchOpts.Filter = make(map[string]interface{})
		}
		searchOpts.Filter["doc_id"] = options.DocumentID
	}

	// Execute search
	searchResult, err := g.Vector.SearchSimilar(ctx, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	g.reportSearchProgress(callback, types.SearchStatusPending, "Converting results...", 80)

	// Convert to segments
	segments := g.convertSearchResultsToSegments(searchResult, options.CollectionID, options.DocumentID)

	// Apply reranking if configured
	if options.Reranker != nil {
		g.reportSearchProgress(callback, types.SearchStatusPending, "Reranking results...", 90)
		segments, err = options.Reranker.Rerank(ctx, segments)
		if err != nil {
			g.Logger.Warnf("Reranking failed: %v, using original order", err)
		}
	}

	g.reportSearchProgress(callback, types.SearchStatusSuccess, fmt.Sprintf("Vector search completed, found %d segments", len(segments)), 100)

	return &types.VectorSearchResult{
		Segments: segments,
		Total:    len(segments),
	}, nil
}

// ================================================================================================
// Pure Graph Search
// ================================================================================================

// SearchGraph performs pure graph/knowledge search without vector similarity
// This is useful for application-side orchestration where vector and graph searches are composed separately
func (g *GraphRag) SearchGraph(ctx context.Context, options *types.GraphSearchOptions, callback ...types.SearcherProgress) (*types.GraphSearchResult, error) {
	if options == nil {
		return nil, fmt.Errorf("graph search options cannot be nil")
	}

	// Validate collection ID
	if options.CollectionID == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Connect to graph store if not already connected
	if g.Graph != nil && !g.Graph.IsConnected() {
		err := g.Graph.Connect(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to graph store: %w", err)
		}
		g.Logger.Infof("Connected to graph store")
	}

	// Check if graph store is available
	if g.Graph == nil || !g.Graph.IsConnected() {
		return nil, fmt.Errorf("graph store is not available or not connected")
	}

	g.reportSearchProgress(callback, types.SearchStatusPending, "Starting graph search...", 0)

	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(options.CollectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection IDs: %w", err)
	}

	// Check if graph exists
	exists, err := g.Graph.GraphExists(ctx, collectionIDs.Graph)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("graph %s does not exist", collectionIDs.Graph)
	}

	var result *types.GraphSearchResult

	// Determine search mode
	if options.Cypher != "" {
		// Custom Cypher query
		g.reportSearchProgress(callback, types.SearchStatusPending, "Executing Cypher query...", 30)
		result, err = g.executeGraphCypherSearch(ctx, collectionIDs.Graph, options)
	} else if len(options.EntityIDs) > 0 {
		// Search by entity IDs
		g.reportSearchProgress(callback, types.SearchStatusPending, "Searching by entity IDs...", 30)
		result, err = g.executeGraphEntityIDSearch(ctx, collectionIDs.Graph, options)
	} else if len(options.Entities) > 0 {
		// Search by entity names
		g.reportSearchProgress(callback, types.SearchStatusPending, "Searching by entity names...", 30)
		result, err = g.executeGraphEntityNameSearch(ctx, collectionIDs.Graph, options)
	} else if options.Query != "" {
		// Natural language query - extract entities first
		g.reportSearchProgress(callback, types.SearchStatusPending, "Extracting entities from query...", 20)
		result, err = g.executeGraphNLQuerySearch(ctx, collectionIDs.Graph, options, callback)
	} else {
		return nil, fmt.Errorf("at least one of Cypher, EntityIDs, Entities, or Query is required")
	}

	if err != nil {
		return nil, err
	}

	// Optionally fetch related segments
	if len(result.Nodes) > 0 {
		g.reportSearchProgress(callback, types.SearchStatusPending, "Fetching related segments...", 80)
		result.Segments = g.fetchSegmentsForNodes(ctx, collectionIDs.Vector, options.CollectionID, result.Nodes)
	}

	g.reportSearchProgress(callback, types.SearchStatusSuccess, fmt.Sprintf("Graph search completed, found %d nodes, %d relationships", len(result.Nodes), len(result.Relationships)), 100)

	return result, nil
}

// executeGraphCypherSearch executes a custom Cypher query
func (g *GraphRag) executeGraphCypherSearch(ctx context.Context, graphName string, options *types.GraphSearchOptions) (*types.GraphSearchResult, error) {
	queryOpts := &types.GraphQueryOptions{
		GraphName:  graphName,
		QueryType:  "cypher",
		Query:      options.Cypher,
		Parameters: options.Parameters,
		Limit:      options.Limit,
		ReturnType: "all",
		ReadOnly:   true,
	}

	graphResult, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("cypher query failed: %w", err)
	}

	return g.convertGraphResultToSearchResult(graphResult), nil
}

// executeGraphEntityIDSearch searches by entity IDs
func (g *GraphRag) executeGraphEntityIDSearch(ctx context.Context, graphName string, options *types.GraphSearchOptions) (*types.GraphSearchResult, error) {
	// Build Cypher query to find nodes by IDs and their relationships
	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}
	maxDepth := options.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	query := `
		MATCH (n)
		WHERE n.id IN $entity_ids
		OPTIONAL MATCH (n)-[r*1..` + fmt.Sprintf("%d", maxDepth) + `]-(m)
		RETURN n, r, m
		LIMIT $limit
	`

	queryOpts := &types.GraphQueryOptions{
		GraphName: graphName,
		QueryType: "cypher",
		Query:     query,
		Parameters: map[string]interface{}{
			"entity_ids": options.EntityIDs,
			"limit":      limit,
		},
		ReturnType: "all",
		ReadOnly:   true,
	}

	graphResult, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("entity ID search failed: %w", err)
	}

	return g.convertGraphResultToSearchResult(graphResult), nil
}

// executeGraphEntityNameSearch searches by entity names
func (g *GraphRag) executeGraphEntityNameSearch(ctx context.Context, graphName string, options *types.GraphSearchOptions) (*types.GraphSearchResult, error) {
	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}
	maxDepth := options.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	// Build Cypher query to find nodes by name (case-insensitive)
	query := `
		MATCH (n)
		WHERE any(name IN $entity_names WHERE toLower(n.name) CONTAINS toLower(name))
		OPTIONAL MATCH (n)-[r*1..` + fmt.Sprintf("%d", maxDepth) + `]-(m)
		RETURN n, r, m
		LIMIT $limit
	`

	queryOpts := &types.GraphQueryOptions{
		GraphName: graphName,
		QueryType: "cypher",
		Query:     query,
		Parameters: map[string]interface{}{
			"entity_names": options.Entities,
			"limit":        limit,
		},
		ReturnType: "all",
		ReadOnly:   true,
	}

	graphResult, err := g.Graph.Query(ctx, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("entity name search failed: %w", err)
	}

	return g.convertGraphResultToSearchResult(graphResult), nil
}

// executeGraphNLQuerySearch extracts entities from natural language query and searches
func (g *GraphRag) executeGraphNLQuerySearch(ctx context.Context, graphName string, options *types.GraphSearchOptions, callback []types.SearcherProgress) (*types.GraphSearchResult, error) {
	if options.Extraction == nil {
		return nil, fmt.Errorf("extraction function is required for natural language query")
	}

	// Extract entities from query
	extractResult, err := options.Extraction.ExtractQuery(ctx, options.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities from query: %w", err)
	}

	if len(extractResult.Nodes) == 0 {
		g.Logger.Debugf("No entities extracted from query: %s", options.Query)
		return &types.GraphSearchResult{
			Nodes:         []types.GraphNode{},
			Relationships: []types.GraphRelationship{},
			Paths:         []types.GraphPath{},
			Segments:      []types.Segment{},
		}, nil
	}

	// Extract entity names
	entityNames := make([]string, len(extractResult.Nodes))
	for i, node := range extractResult.Nodes {
		entityNames[i] = node.Name
	}

	g.Logger.Debugf("Extracted %d entities from query: %v", len(entityNames), entityNames)
	g.reportSearchProgress(callback, types.SearchStatusPending, fmt.Sprintf("Found %d entities, searching graph...", len(entityNames)), 50)

	// Search by extracted entity names
	searchOpts := &types.GraphSearchOptions{
		CollectionID:  options.CollectionID,
		DocumentID:    options.DocumentID,
		Entities:      entityNames,
		MaxDepth:      options.MaxDepth,
		RelationTypes: options.RelationTypes,
		EntityTypes:   options.EntityTypes,
		Limit:         options.Limit,
	}

	return g.executeGraphEntityNameSearch(ctx, graphName, searchOpts)
}

// convertGraphResultToSearchResult converts GraphResult to GraphSearchResult
func (g *GraphRag) convertGraphResultToSearchResult(result *types.GraphResult) *types.GraphSearchResult {
	// Convert nodes
	nodes := make([]types.GraphNode, len(result.Nodes))
	for i, node := range result.Nodes {
		nodes[i] = convertNodeToGraphNode(node)
	}

	// Convert relationships
	relationships := make([]types.GraphRelationship, len(result.Relationships))
	for i, rel := range result.Relationships {
		relationships[i] = convertRelationshipToGraphRelationship(rel)
	}

	// Convert paths
	paths := make([]types.GraphPath, len(result.Paths))
	for i, path := range result.Paths {
		pathNodes := make([]types.GraphNode, len(path.Nodes))
		for j, node := range path.Nodes {
			pathNodes[j] = convertNodeToGraphNode(node)
		}
		pathRels := make([]types.GraphRelationship, len(path.Relationships))
		for j, rel := range path.Relationships {
			pathRels[j] = convertRelationshipToGraphRelationship(rel)
		}
		paths[i] = types.GraphPath{
			Nodes:         pathNodes,
			Relationships: pathRels,
			Length:        len(pathRels),
		}
	}

	return &types.GraphSearchResult{
		Nodes:         nodes,
		Relationships: relationships,
		Paths:         paths,
		Segments:      []types.Segment{}, // Will be populated by caller if needed
	}
}

// fetchSegmentsForNodes fetches segments related to the given nodes
func (g *GraphRag) fetchSegmentsForNodes(ctx context.Context, vectorCollectionName string, collectionID string, nodes []types.GraphNode) []types.Segment {
	if len(nodes) == 0 {
		return []types.Segment{}
	}

	// Collect all source chunk IDs from nodes
	chunkIDSet := make(map[string]bool)
	for _, node := range nodes {
		if props := node.Properties; props != nil {
			if chunks, ok := props["source_chunks"]; ok {
				switch c := chunks.(type) {
				case []string:
					for _, chunk := range c {
						chunkIDSet[chunk] = true
					}
				case []interface{}:
					for _, chunk := range c {
						if chunkStr, ok := chunk.(string); ok {
							chunkIDSet[chunkStr] = true
						}
					}
				}
			}
		}
	}

	if len(chunkIDSet) == 0 {
		return []types.Segment{}
	}

	// Convert to slice
	chunkIDs := make([]string, 0, len(chunkIDSet))
	for id := range chunkIDSet {
		chunkIDs = append(chunkIDs, id)
	}

	// Fetch documents by IDs
	getOpts := &types.GetDocumentOptions{
		CollectionName: vectorCollectionName,
		IncludePayload: true,
		IncludeVector:  false,
	}

	docs, err := g.Vector.GetDocuments(ctx, chunkIDs, getOpts)
	if err != nil {
		g.Logger.Warnf("Failed to fetch segments for nodes: %v", err)
		return []types.Segment{}
	}

	// Convert to segments
	segments := make([]types.Segment, 0, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}

		// Extract document ID from metadata
		documentID := ""
		if doc.Metadata != nil {
			if metaDocID, ok := doc.Metadata["doc_id"].(string); ok {
				documentID = metaDocID
			}
		}

		segment := types.Segment{
			CollectionID: collectionID,
			DocumentID:   documentID,
			ID:           doc.ID,
			Text:         doc.Content,
			Metadata:     doc.Metadata,
		}

		// Extract additional fields from metadata
		if doc.Metadata != nil {
			segment.Weight = types.SafeExtractFloat64(doc.Metadata["weight"], 0.0)
			segment.Positive = types.SafeExtractInt(doc.Metadata["positive"], 0)
			segment.Negative = types.SafeExtractInt(doc.Metadata["negative"], 0)
			segment.Hit = types.SafeExtractInt(doc.Metadata["hit"], 0)
		}

		segments = append(segments, segment)
	}

	return segments
}
