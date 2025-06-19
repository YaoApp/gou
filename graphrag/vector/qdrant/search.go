package qdrant

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/yaoapp/gou/graphrag/types"
)

// SearchSimilar performs similarity search
func (s *Store) SearchSimilar(ctx context.Context, opts *types.SearchOptions) (*types.SearchResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("search options cannot be nil")
	}

	if opts.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	if len(opts.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector is required")
	}

	// Only hold the lock briefly to check connection status and get client
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected to Qdrant server")
	}
	client := s.client
	s.mu.RUnlock()

	// Start measuring query time
	startTime := time.Now()

	// Handle timeout
	searchCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		searchCtx, cancel = context.WithTimeout(ctx, time.Duration(opts.Timeout)*time.Millisecond)
		defer cancel()
	}

	// Determine the limit for the search
	limit := opts.K
	if opts.PageSize > 0 {
		// If pagination is used, we need to consider the page size
		if opts.Page > 0 {
			// For offset-based pagination, we need to fetch more records to calculate offset
			limit = opts.PageSize + (opts.Page-1)*opts.PageSize
		} else {
			limit = opts.PageSize
		}
	}
	if limit <= 0 {
		limit = 10 // Default limit
	}

	// Apply max results limit
	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 1000 // Default max results
	}
	if limit > maxResults {
		limit = maxResults
	}

	// Convert query vector to float32
	queryVector := make([]float32, len(opts.QueryVector))
	for i, v := range opts.QueryVector {
		queryVector[i] = float32(v)
	}

	// Build query request
	queryReq := &qdrant.QueryPoints{
		CollectionName: opts.CollectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(opts.IncludeMetadata || opts.IncludeContent),
		WithVectors:    qdrant.NewWithVectors(opts.IncludeVector),
	}

	// Apply minimum score filter
	if opts.MinScore > 0 {
		scoreThreshold := float32(opts.MinScore)
		queryReq.ScoreThreshold = &scoreThreshold
	}

	// Apply metadata filter
	if opts.Filter != nil {
		filter, err := convertFilterToQdrant(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter: %w", err)
		}
		queryReq.Filter = filter
	}

	// Apply search parameters
	if opts.EfSearch > 0 || opts.NumProbes > 0 || opts.Approximate {
		queryReq.Params = &qdrant.SearchParams{}

		if opts.EfSearch > 0 {
			queryReq.Params.HnswEf = qdrant.PtrOf(uint64(opts.EfSearch))
		}

		if opts.Approximate {
			queryReq.Params.Exact = qdrant.PtrOf(false)
		}
	}

	// Perform the search
	points, err := client.Query(searchCtx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to perform similarity search: %w", err)
	}

	// Calculate query time
	queryTime := time.Since(startTime).Milliseconds()

	// Convert results to SearchResultItems
	var documents []*types.SearchResultItem
	var maxScore, minScore float64

	if len(points) > 0 {
		maxScore = float64(points[0].Score)
		minScore = float64(points[len(points)-1].Score)
	}

	// Handle pagination
	startIdx := 0
	endIdx := len(points)

	if opts.Page > 0 && opts.PageSize > 0 {
		// Offset-based pagination
		offset := (opts.Page - 1) * opts.PageSize
		startIdx = offset
		endIdx = offset + opts.PageSize

		if startIdx >= len(points) {
			startIdx = len(points)
			endIdx = len(points)
		} else if endIdx > len(points) {
			endIdx = len(points)
		}
	}

	// Convert points to documents
	for i := startIdx; i < endIdx; i++ {
		point := points[i]
		doc := convertScoredPointToSearchDocument(point, opts.IncludeVector, opts.IncludeMetadata, opts.IncludeContent, opts.Fields)

		documents = append(documents, &types.SearchResultItem{
			Document: *doc,
			Score:    float64(point.Score),
		})
	}

	// Build search result
	result := &types.SearchResult{
		Documents: documents,
		QueryTime: queryTime,
		MaxScore:  maxScore,
		MinScore:  minScore,
	}

	// Add pagination metadata if requested
	if opts.Page > 0 && opts.PageSize > 0 {
		result.Page = opts.Page
		result.PageSize = opts.PageSize
		result.HasNext = (opts.Page * opts.PageSize) < len(points)
		result.HasPrevious = opts.Page > 1

		if result.HasNext {
			result.NextPage = opts.Page + 1
		}
		if result.HasPrevious {
			result.PreviousPage = opts.Page - 1
		}

		// Calculate total count if requested (expensive operation)
		if opts.IncludeTotal {
			// For similarity search, we can use the total number of points found
			// In practice, this might be limited by the max results
			result.Total = int64(len(points))
			if len(points) >= maxResults {
				// If we hit the max results limit, we need to do a separate count query
				countReq := &qdrant.CountPoints{
					CollectionName: opts.CollectionName,
					Filter:         queryReq.Filter,
					Exact:          qdrant.PtrOf(false), // Use approximate count for performance
				}

				if count, err := client.Count(searchCtx, countReq); err == nil {
					result.Total = int64(count)
				}
			}

			if result.Total > 0 {
				result.TotalPages = int((result.Total + int64(opts.PageSize) - 1) / int64(opts.PageSize))
			}
		}
	}

	return result, nil
}

// convertScoredPointToSearchDocument converts a Qdrant ScoredPoint to a Document for search results
func convertScoredPointToSearchDocument(point *qdrant.ScoredPoint, includeVector, includeMetadata, includeContent bool, fields []string) *types.Document {
	doc := &types.Document{}

	// Extract ID from point.Id first, then try payload as fallback
	if point.Id != nil {
		if strID := point.Id.GetUuid(); strID != "" {
			doc.ID = strID
		} else if numID := point.Id.GetNum(); numID != 0 {
			doc.ID = fmt.Sprintf("%d", numID)
		}
	}

	// Extract basic fields from payload
	if point.Payload != nil {
		// If ID is still empty, try to get it from payload
		if doc.ID == "" {
			if idVal := point.Payload["id"]; idVal != nil {
				doc.ID = idVal.GetStringValue()
			}
		}

		if includeContent {
			if contentVal := point.Payload["page_content"]; contentVal != nil {
				doc.PageContent = contentVal.GetStringValue()
			}
		}

		// Extract metadata if requested
		if includeMetadata {
			if metadataVal := point.Payload["metadata"]; metadataVal != nil {
				if metadataStruct := metadataVal.GetStructValue(); metadataStruct != nil {
					doc.Metadata = convertStructToMap(metadataStruct)
				}
			}
		}

		// Handle specific fields if requested
		if len(fields) > 0 {
			filteredMetadata := make(map[string]interface{})
			for _, field := range fields {
				if val := point.Payload[field]; val != nil {
					switch v := val.Kind.(type) {
					case *qdrant.Value_StringValue:
						filteredMetadata[field] = v.StringValue
					case *qdrant.Value_DoubleValue:
						filteredMetadata[field] = v.DoubleValue
					case *qdrant.Value_BoolValue:
						filteredMetadata[field] = v.BoolValue
					case *qdrant.Value_StructValue:
						filteredMetadata[field] = convertStructToMap(v.StructValue)
					}
				}
			}
			if len(filteredMetadata) > 0 {
				if doc.Metadata == nil {
					doc.Metadata = make(map[string]interface{})
				}
				for k, v := range filteredMetadata {
					doc.Metadata[k] = v
				}
			}
		}
	}

	// Extract vector if requested
	if includeVector && point.Vectors != nil {
		if vectorData := point.Vectors.GetVector(); vectorData != nil {
			doc.Vector = make([]float64, len(vectorData.Data))
			for i, v := range vectorData.Data {
				doc.Vector[i] = float64(v)
			}
		}
	}

	return doc
}

// SearchMMR performs maximal marginal relevance search
func (s *Store) SearchMMR(ctx context.Context, opts *types.MMRSearchOptions) (*types.SearchResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("MMR search options cannot be nil")
	}

	if opts.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	if len(opts.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector is required")
	}

	// Only hold the lock briefly to check connection status and get client
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected to Qdrant server")
	}
	client := s.client
	s.mu.RUnlock()

	// Start measuring query time
	startTime := time.Now()

	// Handle timeout
	searchCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		searchCtx, cancel = context.WithTimeout(ctx, time.Duration(opts.Timeout)*time.Millisecond)
		defer cancel()
	}

	// For MMR, we need to fetch more documents first (FetchK)
	fetchK := opts.FetchK
	if fetchK <= 0 {
		fetchK = opts.K * 3 // Default to 3x the final K
		if fetchK <= 0 {
			fetchK = 30 // Default fetch size
		}
	}

	// Convert query vector to float32
	queryVector := make([]float32, len(opts.QueryVector))
	for i, v := range opts.QueryVector {
		queryVector[i] = float32(v)
	}

	// Build initial query request to fetch candidates
	queryReq := &qdrant.QueryPoints{
		CollectionName: opts.CollectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          qdrant.PtrOf(uint64(fetchK)),
		WithPayload:    qdrant.NewWithPayload(opts.IncludeMetadata || opts.IncludeContent),
		WithVectors:    qdrant.NewWithVectors(true), // Need vectors for MMR calculation
	}

	// Apply metadata filter
	if opts.Filter != nil {
		filter, err := convertFilterToQdrant(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter: %w", err)
		}
		queryReq.Filter = filter
	}

	// Apply minimum score filter
	if opts.MinScore > 0 {
		scoreThreshold := float32(opts.MinScore)
		queryReq.ScoreThreshold = &scoreThreshold
	}

	// Apply search parameters
	if opts.EfSearch > 0 || opts.Approximate {
		queryReq.Params = &qdrant.SearchParams{}

		if opts.EfSearch > 0 {
			queryReq.Params.HnswEf = qdrant.PtrOf(uint64(opts.EfSearch))
		}

		if opts.Approximate {
			queryReq.Params.Exact = qdrant.PtrOf(false)
		}
	}

	// Perform the initial search to get candidates
	candidates, err := client.Query(searchCtx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MMR candidates: %w", err)
	}

	// Apply MMR algorithm to select diverse results
	finalK := opts.K
	if finalK <= 0 {
		finalK = 10 // Default K
	}

	if opts.PageSize > 0 {
		if opts.Page > 0 {
			finalK = opts.PageSize + (opts.Page-1)*opts.PageSize
		} else {
			finalK = opts.PageSize
		}
	}

	lambdaMult := opts.LambdaMult
	if lambdaMult <= 0 {
		lambdaMult = 0.5 // Default balance between similarity and diversity
	}

	selectedPoints := selectMMRResults(candidates, queryVector, finalK, lambdaMult)

	// Calculate query time
	queryTime := time.Since(startTime).Milliseconds()

	// Convert results to SearchResultItems
	var documents []*types.SearchResultItem
	var maxScore, minScore float64

	if len(selectedPoints) > 0 {
		maxScore = float64(selectedPoints[0].Score)
		minScore = float64(selectedPoints[len(selectedPoints)-1].Score)
	}

	// Handle pagination for selected results
	startIdx := 0
	endIdx := len(selectedPoints)

	if opts.Page > 0 && opts.PageSize > 0 {
		offset := (opts.Page - 1) * opts.PageSize
		startIdx = offset
		endIdx = offset + opts.PageSize

		if startIdx >= len(selectedPoints) {
			startIdx = len(selectedPoints)
			endIdx = len(selectedPoints)
		} else if endIdx > len(selectedPoints) {
			endIdx = len(selectedPoints)
		}
	}

	// Convert points to documents
	for i := startIdx; i < endIdx; i++ {
		point := selectedPoints[i]
		doc := convertScoredPointToSearchDocument(point, opts.IncludeVector, opts.IncludeMetadata, opts.IncludeContent, opts.Fields)

		documents = append(documents, &types.SearchResultItem{
			Document: *doc,
			Score:    float64(point.Score),
		})
	}

	// Build search result
	result := &types.SearchResult{
		Documents: documents,
		QueryTime: queryTime,
		MaxScore:  maxScore,
		MinScore:  minScore,
	}

	// Add pagination metadata if requested
	if opts.Page > 0 && opts.PageSize > 0 {
		result.Page = opts.Page
		result.PageSize = opts.PageSize
		result.HasNext = (opts.Page * opts.PageSize) < len(selectedPoints)
		result.HasPrevious = opts.Page > 1

		if result.HasNext {
			result.NextPage = opts.Page + 1
		}
		if result.HasPrevious {
			result.PreviousPage = opts.Page - 1
		}

		if opts.IncludeTotal {
			result.Total = int64(len(selectedPoints))
			if result.Total > 0 {
				result.TotalPages = int((result.Total + int64(opts.PageSize) - 1) / int64(opts.PageSize))
			}
		}
	}

	return result, nil
}

// selectMMRResults implements the MMR algorithm for diverse result selection
func selectMMRResults(candidates []*qdrant.ScoredPoint, queryVector []float32, k int, lambdaMult float64) []*qdrant.ScoredPoint {
	if len(candidates) == 0 || k <= 0 {
		return []*qdrant.ScoredPoint{}
	}

	if k >= len(candidates) {
		return candidates
	}

	selected := make([]*qdrant.ScoredPoint, 0, k)
	remaining := make([]*qdrant.ScoredPoint, len(candidates))
	copy(remaining, candidates)

	// Select the first document (highest similarity)
	if len(remaining) > 0 {
		selected = append(selected, remaining[0])
		remaining = remaining[1:]
	}

	// Select remaining documents using MMR
	for len(selected) < k && len(remaining) > 0 {
		bestIdx := -1
		bestScore := float64(-1)

		for i, candidate := range remaining {
			// Calculate similarity to query
			querySim := float64(candidate.Score)

			// Calculate maximum similarity to already selected documents
			maxSelectedSim := 0.0
			if candidate.Vectors != nil && candidate.Vectors.GetVector() != nil {
				candidateVector := candidate.Vectors.GetVector().Data
				for _, selectedDoc := range selected {
					if selectedDoc.Vectors != nil && selectedDoc.Vectors.GetVector() != nil {
						selectedVector := selectedDoc.Vectors.GetVector().Data
						sim := cosineSimilarity(candidateVector, selectedVector)
						if sim > maxSelectedSim {
							maxSelectedSim = sim
						}
					}
				}
			}

			// Calculate MMR score
			mmrScore := lambdaMult*querySim - (1-lambdaMult)*maxSelectedSim

			if mmrScore > bestScore {
				bestScore = mmrScore
				bestIdx = i
			}
		}

		if bestIdx >= 0 {
			selected = append(selected, remaining[bestIdx])
			// Remove selected document from remaining
			remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
		} else {
			break
		}
	}

	return selected
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SearchWithScoreThreshold performs similarity search with score threshold
func (s *Store) SearchWithScoreThreshold(ctx context.Context, opts *types.ScoreThresholdOptions) (*types.SearchResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("score threshold search options cannot be nil")
	}

	if opts.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	if len(opts.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector is required")
	}

	// Only hold the lock briefly to check connection status and get client
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected to Qdrant server")
	}
	client := s.client
	s.mu.RUnlock()

	// Start measuring query time
	startTime := time.Now()

	// Handle timeout
	searchCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		searchCtx, cancel = context.WithTimeout(ctx, time.Duration(opts.Timeout)*time.Millisecond)
		defer cancel()
	}

	// Determine the limit for the search
	limit := opts.K
	if opts.PageSize > 0 {
		if opts.Page > 0 {
			limit = opts.PageSize + (opts.Page-1)*opts.PageSize
		} else {
			limit = opts.PageSize
		}
	}
	if limit <= 0 {
		limit = 10 // Default limit
	}

	// Apply max results limit
	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 1000 // Default max results
	}
	if limit > maxResults {
		limit = maxResults
	}

	// Convert query vector to float32
	queryVector := make([]float32, len(opts.QueryVector))
	for i, v := range opts.QueryVector {
		queryVector[i] = float32(v)
	}

	// Build query request
	queryReq := &qdrant.QueryPoints{
		CollectionName: opts.CollectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(opts.IncludeMetadata || opts.IncludeContent),
		WithVectors:    qdrant.NewWithVectors(opts.IncludeVector),
	}

	// Apply score threshold - this is the key difference from regular similarity search
	scoreThreshold := float32(opts.ScoreThreshold)
	queryReq.ScoreThreshold = &scoreThreshold

	// Apply metadata filter
	if opts.Filter != nil {
		filter, err := convertFilterToQdrant(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter: %w", err)
		}
		queryReq.Filter = filter
	}

	// Apply search parameters
	if opts.EfSearch > 0 || opts.NumProbes > 0 || opts.Approximate {
		queryReq.Params = &qdrant.SearchParams{}

		if opts.EfSearch > 0 {
			queryReq.Params.HnswEf = qdrant.PtrOf(uint64(opts.EfSearch))
		}

		if opts.Approximate {
			queryReq.Params.Exact = qdrant.PtrOf(false)
		}
	}

	// Perform the search
	points, err := client.Query(searchCtx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to perform score threshold search: %w", err)
	}

	// Calculate query time
	queryTime := time.Since(startTime).Milliseconds()

	// Convert results to SearchResultItems
	var documents []*types.SearchResultItem
	var maxScore, minScore float64

	if len(points) > 0 {
		maxScore = float64(points[0].Score)
		minScore = float64(points[len(points)-1].Score)
	}

	// Handle pagination
	startIdx := 0
	endIdx := len(points)

	if opts.Page > 0 && opts.PageSize > 0 {
		offset := (opts.Page - 1) * opts.PageSize
		startIdx = offset
		endIdx = offset + opts.PageSize

		if startIdx >= len(points) {
			startIdx = len(points)
			endIdx = len(points)
		} else if endIdx > len(points) {
			endIdx = len(points)
		}
	}

	// Convert points to documents
	for i := startIdx; i < endIdx; i++ {
		point := points[i]
		doc := convertScoredPointToSearchDocument(point, opts.IncludeVector, opts.IncludeMetadata, opts.IncludeContent, opts.Fields)

		documents = append(documents, &types.SearchResultItem{
			Document: *doc,
			Score:    float64(point.Score),
		})
	}

	// Build search result
	result := &types.SearchResult{
		Documents: documents,
		QueryTime: queryTime,
		MaxScore:  maxScore,
		MinScore:  minScore,
	}

	// Add pagination metadata if requested
	if opts.Page > 0 && opts.PageSize > 0 {
		result.Page = opts.Page
		result.PageSize = opts.PageSize
		result.HasNext = (opts.Page * opts.PageSize) < len(points)
		result.HasPrevious = opts.Page > 1

		if result.HasNext {
			result.NextPage = opts.Page + 1
		}
		if result.HasPrevious {
			result.PreviousPage = opts.Page - 1
		}

		if opts.IncludeTotal {
			result.Total = int64(len(points))
			if len(points) >= maxResults {
				// If we hit the max results limit, we need to do a separate count query
				countReq := &qdrant.CountPoints{
					CollectionName: opts.CollectionName,
					Filter:         queryReq.Filter,
					Exact:          qdrant.PtrOf(false), // Use approximate count for performance
				}

				if count, err := client.Count(searchCtx, countReq); err == nil {
					result.Total = int64(count)
				}
			}

			if result.Total > 0 {
				result.TotalPages = int((result.Total + int64(opts.PageSize) - 1) / int64(opts.PageSize))
			}
		}
	}

	return result, nil
}

// SearchHybrid performs hybrid search (vector + keyword)
func (s *Store) SearchHybrid(ctx context.Context, opts *types.HybridSearchOptions) (*types.SearchResult, error) {
	// Note: Qdrant doesn't natively support hybrid search combining vector and keyword search
	// This is a simplified implementation that performs vector search only
	// For true hybrid search, you would need to implement keyword search separately
	// and combine results using the specified weights

	if opts == nil {
		return nil, fmt.Errorf("hybrid search options cannot be nil")
	}

	if opts.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	// For now, if no vector query is provided, return an error
	// In a full implementation, you could perform keyword-only search
	if len(opts.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector is required for Qdrant hybrid search")
	}

	// Convert to similarity search options (simplified approach)
	searchOpts := &types.SearchOptions{
		CollectionName:  opts.CollectionName,
		QueryVector:     opts.QueryVector,
		K:               opts.K,
		Filter:          opts.Filter,
		Page:            opts.Page,
		PageSize:        opts.PageSize,
		Cursor:          opts.Cursor,
		IncludeVector:   opts.IncludeVector,
		IncludeMetadata: opts.IncludeMetadata,
		IncludeContent:  opts.IncludeContent,
		Fields:          opts.Fields,
		IncludeTotal:    opts.IncludeTotal,
		EfSearch:        opts.EfSearch,
		NumProbes:       opts.NumProbes,
		Rescore:         opts.Rescore,
		Approximate:     opts.Approximate,
		Timeout:         opts.Timeout,
		MinScore:        opts.MinScore,
		MaxResults:      opts.MaxResults,
		SortBy:          opts.SortBy,
		FacetFields:     opts.FacetFields,
		HighlightFields: opts.HighlightFields,
	}

	// Perform vector similarity search
	result, err := s.SearchSimilar(ctx, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}

	// TODO: Implement actual hybrid functionality:
	// 1. Perform keyword search if QueryText is provided
	// 2. Combine vector and keyword results using VectorWeight and KeywordWeight
	// 3. Apply boost fields for keyword search
	// 4. Handle fuzzy matching for keywords

	return result, nil
}

// SearchBatch performs unified batch search for multiple search types
func (s *Store) SearchBatch(ctx context.Context, opts []types.SearchOptionsInterface) ([]*types.SearchResult, error) {
	if len(opts) == 0 {
		return []*types.SearchResult{}, nil
	}

	// Only hold the lock briefly to check connection status
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected to Qdrant server")
	}
	s.mu.RUnlock()

	results := make([]*types.SearchResult, len(opts))
	errors := make([]error, len(opts))

	// TODO: Implement parallel execution for better performance
	// For now, execute searches sequentially but this could be optimized
	// to run multiple searches in parallel using goroutines

	for i, opt := range opts {
		if opt == nil {
			errors[i] = fmt.Errorf("search option at index %d is nil", i)
			continue
		}

		switch searchOpt := opt.(type) {
		case *types.SearchOptions:
			results[i], errors[i] = s.SearchSimilar(ctx, searchOpt)
		case *types.MMRSearchOptions:
			results[i], errors[i] = s.SearchMMR(ctx, searchOpt)
		case *types.ScoreThresholdOptions:
			results[i], errors[i] = s.SearchWithScoreThreshold(ctx, searchOpt)
		case *types.HybridSearchOptions:
			results[i], errors[i] = s.SearchHybrid(ctx, searchOpt)
		default:
			errors[i] = fmt.Errorf("unsupported search option type at index %d: %T", i, searchOpt)
		}
	}

	// Check if any errors occurred
	var hasErrors bool
	errorMessages := make([]string, 0)
	for i, err := range errors {
		if err != nil {
			hasErrors = true
			errorMessages = append(errorMessages, fmt.Sprintf("search %d: %v", i, err))
		}
	}

	if hasErrors {
		return results, fmt.Errorf("batch search completed with errors: %s", strings.Join(errorMessages, "; "))
	}

	return results, nil
}
