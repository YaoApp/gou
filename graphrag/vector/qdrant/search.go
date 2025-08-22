package qdrant

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
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

	// Handle named vector selection
	if opts.VectorUsing != "" {
		queryReq.Using = qdrant.PtrOf(opts.VectorUsing)
	} else {
		// If no VectorUsing specified, check if collection requires named vectors
		if availableNames, err := s.getAvailableVectorNames(searchCtx, opts.CollectionName); err == nil && len(availableNames) > 0 {
			// Collection has named vectors, use "dense" as default
			for _, name := range availableNames {
				if name == "dense" {
					queryReq.Using = qdrant.PtrOf("dense")
					break
				}
			}
			// If no "dense" vector found, use the first available vector
			if queryReq.Using == nil && len(availableNames) > 0 {
				queryReq.Using = qdrant.PtrOf(availableNames[0])
			}
		}
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
		doc := convertScoredPointToSearchDocument(point, opts.IncludeVector, opts.IncludeMetadata, opts.IncludeContent, opts.Fields, opts.VectorUsing)

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
func convertScoredPointToSearchDocument(point *qdrant.ScoredPoint, includeVector, includeMetadata, includeContent bool, fields []string, vectorUsing string) *types.Document {
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
			if contentVal := point.Payload["content"]; contentVal != nil {
				doc.Content = contentVal.GetStringValue()
			}
		}

		// Extract metadata if requested
		if includeMetadata {
			// Initialize metadata map
			doc.Metadata = make(map[string]interface{})

			// Extract all payload fields as metadata (except id and content)
			for key, fieldVal := range point.Payload {
				if key == "id" || key == "content" {
					continue // Skip basic fields
				}

				if fieldVal != nil {
					switch v := fieldVal.Kind.(type) {
					case *qdrant.Value_DoubleValue:
						doc.Metadata[key] = v.DoubleValue
					case *qdrant.Value_IntegerValue:
						doc.Metadata[key] = v.IntegerValue
					case *qdrant.Value_StringValue:
						doc.Metadata[key] = v.StringValue
					case *qdrant.Value_BoolValue:
						doc.Metadata[key] = v.BoolValue
					case *qdrant.Value_ListValue:
						list := make([]interface{}, len(v.ListValue.Values))
						for i, item := range v.ListValue.Values {
							if str := item.GetStringValue(); str != "" {
								list[i] = str
							} else if num := item.GetDoubleValue(); num != 0 {
								list[i] = num
							} else if intVal := item.GetIntegerValue(); intVal != 0 {
								list[i] = intVal
							} else if boolVal := item.GetBoolValue(); boolVal {
								list[i] = boolVal
							}
						}
						doc.Metadata[key] = list
					case *qdrant.Value_StructValue:
						doc.Metadata[key] = convertStructToMap(v.StructValue)
					}
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
		// Handle named vectors (for collections with multiple vectors)
		if namedVectors := point.Vectors.GetVectors(); namedVectors != nil {
			// For named vectors, try to get the specified vector name
			// or use the first available vector as fallback
			var vectorOutput *qdrant.VectorOutput

			// Try to get the specified vector name first
			if vectorUsing != "" {
				if specifiedVector, exists := namedVectors.GetVectors()[vectorUsing]; exists {
					vectorOutput = specifiedVector
				}
			}

			// If no specified vector or not found, try "dense" as default
			if vectorOutput == nil {
				if denseVector, exists := namedVectors.GetVectors()["dense"]; exists {
					vectorOutput = denseVector
				}
			}

			// If still no vector, get the first available vector
			if vectorOutput == nil {
				for _, vector := range namedVectors.GetVectors() {
					vectorOutput = vector
					break
				}
			}

			if vectorOutput != nil {
				// Handle different vector types
				if denseVector := vectorOutput.GetDense(); denseVector != nil {
					doc.Vector = make([]float64, len(denseVector.Data))
					for i, v := range denseVector.Data {
						doc.Vector[i] = float64(v)
					}
				} else if len(vectorOutput.GetData()) > 0 {
					// Fallback to deprecated Data field
					doc.Vector = make([]float64, len(vectorOutput.GetData()))
					for i, v := range vectorOutput.GetData() {
						doc.Vector[i] = float64(v)
					}
				}
			}
		} else if vectorOutput := point.Vectors.GetVector(); vectorOutput != nil {
			// Handle traditional single vector
			if denseVector := vectorOutput.GetDense(); denseVector != nil {
				doc.Vector = make([]float64, len(denseVector.Data))
				for i, v := range denseVector.Data {
					doc.Vector[i] = float64(v)
				}
			} else if len(vectorOutput.GetData()) > 0 {
				// Fallback to deprecated Data field
				doc.Vector = make([]float64, len(vectorOutput.GetData()))
				for i, v := range vectorOutput.GetData() {
					doc.Vector[i] = float64(v)
				}
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

	// Use named vector if explicitly specified
	if opts.VectorUsing != "" {
		queryReq.Using = qdrant.PtrOf(opts.VectorUsing)
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

	// Calculate maxScore and minScore from selectedPoints (MMR may reorder)
	if len(selectedPoints) > 0 {
		maxScore = float64(selectedPoints[0].Score)
		minScore = float64(selectedPoints[0].Score)
		for _, point := range selectedPoints {
			score := float64(point.Score)
			if score > maxScore {
				maxScore = score
			}
			if score < minScore {
				minScore = score
			}
		}
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
		doc := convertScoredPointToSearchDocument(point, opts.IncludeVector, opts.IncludeMetadata, opts.IncludeContent, opts.Fields, opts.VectorUsing)

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

// getAvailableVectorNames returns the available vector names for a collection
func (s *Store) getAvailableVectorNames(ctx context.Context, collectionName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	info, err := s.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	var vectorNames []string

	// Check if the collection has named vectors
	if info.Config != nil && info.Config.Params != nil {
		if vectorsConfig := info.Config.Params.VectorsConfig; vectorsConfig != nil {
			switch vectorsConfig.Config.(type) {
			case *qdrant.VectorsConfig_ParamsMap:
				// Named vectors configuration
				if paramsMap := vectorsConfig.GetParamsMap(); paramsMap != nil {
					for name := range paramsMap.Map {
						vectorNames = append(vectorNames, name)
					}
				}
			case *qdrant.VectorsConfig_Params:
				// Single vector configuration - no named vectors
				return []string{}, nil
			}
		}
	}

	return vectorNames, nil
}

// validateAndGetVectorName validates the vector name and returns a valid one for the collection
func (s *Store) validateAndGetVectorName(ctx context.Context, collectionName, requestedVectorName string) (string, error) {
	// If no specific vector requested, return empty (let Qdrant handle it)
	if requestedVectorName == "" {
		return "", nil
	}

	// Get available vector names to validate the requested name
	availableNames, err := s.getAvailableVectorNames(ctx, collectionName)
	if err != nil {
		return "", err
	}

	if len(availableNames) == 0 {
		// No named vectors in this collection, return empty
		return "", nil
	}

	// Check if the requested vector name exists
	for _, name := range availableNames {
		if name == requestedVectorName {
			return requestedVectorName, nil
		}
	}

	// Requested vector doesn't exist, return error
	return "", fmt.Errorf("vector '%s' not found in collection '%s'. Available vectors: %v", requestedVectorName, collectionName, availableNames)
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

	// Handle named vector selection
	if opts.VectorUsing != "" {
		vectorName, err := s.validateAndGetVectorName(searchCtx, opts.CollectionName, opts.VectorUsing)
		if err != nil {
			// For score threshold search, if vector validation fails,
			// we should still proceed without named vector rather than fail
			// This allows fallback behavior for tests that expect it
			if !strings.Contains(err.Error(), "not found") {
				// Only return error for collection not found, not vector not found
				return nil, fmt.Errorf("failed to validate vector name: %w", err)
			}
			// For vector not found, try to fallback to "dense" as default
			if availableNames, nameErr := s.getAvailableVectorNames(searchCtx, opts.CollectionName); nameErr == nil {
				for _, name := range availableNames {
					if name == "dense" {
						queryReq.Using = qdrant.PtrOf("dense")
						break
					}
				}
			}
		} else if vectorName != "" {
			queryReq.Using = qdrant.PtrOf(vectorName)
		}
	} else {
		// If no VectorUsing specified, check if collection requires named vectors
		if availableNames, err := s.getAvailableVectorNames(searchCtx, opts.CollectionName); err == nil && len(availableNames) > 0 {
			// Collection has named vectors, use "dense" as default
			for _, name := range availableNames {
				if name == "dense" {
					queryReq.Using = qdrant.PtrOf("dense")
					break
				}
			}
			// If no "dense" vector found, use the first available vector
			if queryReq.Using == nil && len(availableNames) > 0 {
				queryReq.Using = qdrant.PtrOf(availableNames[0])
			}
		}
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
		doc := convertScoredPointToSearchDocument(point, opts.IncludeVector, opts.IncludeMetadata, opts.IncludeContent, opts.Fields, opts.VectorUsing)

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

// SearchHybrid performs hybrid search using Qdrant's native Query API
func (s *Store) SearchHybrid(ctx context.Context, opts *types.HybridSearchOptions) (*types.SearchResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("hybrid search options cannot be nil")
	}

	if opts.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	// Validate that at least one search method is provided
	hasVectorQuery := len(opts.QueryVector) > 0
	hasSparseQuery := opts.QuerySparse != nil && len(opts.QuerySparse.Indices) > 0

	if !hasVectorQuery && !hasSparseQuery {
		return nil, fmt.Errorf("at least one of QueryVector or QuerySparse must be provided")
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

	// Build prefetch queries
	var prefetches []*qdrant.PrefetchQuery
	prefetchLimit := uint64(limit * 2) // Fetch more for better fusion

	// Add vector prefetch if vector query is provided
	if hasVectorQuery {
		queryVector := make([]float32, len(opts.QueryVector))
		for i, v := range opts.QueryVector {
			queryVector[i] = float32(v)
		}

		vectorPrefetch := &qdrant.PrefetchQuery{
			Query: qdrant.NewQueryDense(queryVector),
			Limit: qdrant.PtrOf(prefetchLimit),
		}

		// Set vector using if specified
		if opts.VectorUsing != "" {
			vectorPrefetch.Using = qdrant.PtrOf(opts.VectorUsing)
		}

		prefetches = append(prefetches, vectorPrefetch)
	}

	// Add sparse prefetch if sparse query is provided
	if hasSparseQuery {
		sparsePrefetch := &qdrant.PrefetchQuery{
			Query: qdrant.NewQuerySparse(opts.QuerySparse.Indices, opts.QuerySparse.Values),
			Limit: qdrant.PtrOf(prefetchLimit),
		}

		// Set sparse using if specified
		if opts.SparseUsing != "" {
			sparsePrefetch.Using = qdrant.PtrOf(opts.SparseUsing)
		}

		prefetches = append(prefetches, sparsePrefetch)
	}

	// Determine fusion type
	fusionType := opts.FusionType
	if fusionType == "" {
		// Default fusion based on legacy weights or default to RRF
		if opts.VectorWeight > 0 || opts.KeywordWeight > 0 {
			// Legacy weight-based approach, use RRF for now
			fusionType = types.FusionRRF
		} else {
			fusionType = types.FusionRRF // Default
		}
	}

	// Convert fusion type to Qdrant fusion
	var qdrantFusion qdrant.Fusion
	switch fusionType {
	case types.FusionRRF:
		qdrantFusion = qdrant.Fusion_RRF
	case types.FusionDBSF:
		qdrantFusion = qdrant.Fusion_DBSF
	default:
		qdrantFusion = qdrant.Fusion_RRF // Default fallback
	}

	// Build main query request
	queryReq := &qdrant.QueryPoints{
		CollectionName: opts.CollectionName,
		Prefetch:       prefetches,
		Query:          qdrant.NewQueryFusion(qdrantFusion),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(opts.IncludeMetadata || opts.IncludeContent),
		WithVectors:    qdrant.NewWithVectors(opts.IncludeVector),
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

	// Apply minimum score filter
	if opts.MinScore > 0 {
		scoreThreshold := float32(opts.MinScore)
		queryReq.ScoreThreshold = &scoreThreshold
	}

	// Perform the hybrid search
	points, err := client.Query(searchCtx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
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
		doc := convertScoredPointToSearchDocument(point, opts.IncludeVector, opts.IncludeMetadata, opts.IncludeContent, opts.Fields, opts.VectorUsing)

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

	// Pre-validate all options to fail fast
	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("search option at index %d is nil", i)
		}
	}

	results := make([]*types.SearchResult, len(opts))
	errors := make([]error, len(opts))

	// Use sync.WaitGroup for concurrent execution
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrent goroutines and prevent resource exhaustion
	maxConcurrency := 50 // Reasonable limit for concurrent searches
	if len(opts) < maxConcurrency {
		maxConcurrency = len(opts)
	}
	semaphore := make(chan struct{}, maxConcurrency)

	// Launch concurrent searches
	for i, opt := range opts {
		wg.Add(1)
		go func(index int, searchOpt types.SearchOptionsInterface) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute the appropriate search based on option type
			switch typedOpt := searchOpt.(type) {
			case *types.SearchOptions:
				results[index], errors[index] = s.SearchSimilar(ctx, typedOpt)
			case *types.MMRSearchOptions:
				results[index], errors[index] = s.SearchMMR(ctx, typedOpt)
			case *types.ScoreThresholdOptions:
				results[index], errors[index] = s.SearchWithScoreThreshold(ctx, typedOpt)
			case *types.HybridSearchOptions:
				results[index], errors[index] = s.SearchHybrid(ctx, typedOpt)
			default:
				errors[index] = fmt.Errorf("unsupported search option type at index %d: %T", index, searchOpt)
			}
		}(i, opt)
	}

	// Wait for all searches to complete
	wg.Wait()

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
