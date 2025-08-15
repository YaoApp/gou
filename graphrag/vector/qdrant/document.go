package qdrant

import (
	"context"
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"

	"github.com/qdrant/go-client/qdrant"
	"github.com/yaoapp/gou/graphrag/types"
)

// stringToUint64ID converts a string ID to uint64 using MD5 hash for consistency
func stringToUint64ID(s string) uint64 {
	hasher := md5.New()
	hasher.Write([]byte(s))
	hash := hasher.Sum(nil)

	// Convert first 8 bytes of MD5 hash to uint64
	result := uint64(0)
	for i := 0; i < 8 && i < len(hash); i++ {
		result = (result << 8) | uint64(hash[i])
	}
	return result
}

// convertRetrievedPointToDocument converts Qdrant RetrievedPoint to Document
func convertRetrievedPointToDocument(point *qdrant.RetrievedPoint, includeVector, includePayload bool) *types.Document {
	doc := &types.Document{}

	// Extract basic fields
	if point.Payload != nil {
		if idVal := point.Payload["id"]; idVal != nil {
			doc.ID = idVal.GetStringValue()
		}
		if contentVal := point.Payload["content"]; contentVal != nil {
			doc.Content = contentVal.GetStringValue()
		}

		// Extract metadata if requested
		if includePayload {
			if metadataVal := point.Payload["metadata"]; metadataVal != nil {
				if metadataStruct := metadataVal.GetStructValue(); metadataStruct != nil {
					doc.Metadata = convertStructToMap(metadataStruct)
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

// convertScoredPointToDocument converts Qdrant ScoredPoint to Document (for scroll results)
func convertScoredPointToDocument(point *qdrant.ScoredPoint, includeVector, includePayload bool) *types.Document {
	doc := &types.Document{}

	// Extract basic fields
	if point.Payload != nil {
		if idVal := point.Payload["id"]; idVal != nil {
			doc.ID = idVal.GetStringValue()
		}
		if contentVal := point.Payload["content"]; contentVal != nil {
			doc.Content = contentVal.GetStringValue()
		}

		// Extract metadata if requested
		if includePayload {
			if metadataVal := point.Payload["metadata"]; metadataVal != nil {
				if metadataStruct := metadataVal.GetStructValue(); metadataStruct != nil {
					doc.Metadata = convertStructToMap(metadataStruct)
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

// convertStructToMap converts Qdrant Struct to map[string]interface{}
func convertStructToMap(s *qdrant.Struct) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range s.Fields {
		switch v := value.Kind.(type) {
		case *qdrant.Value_StringValue:
			result[key] = v.StringValue
		case *qdrant.Value_DoubleValue:
			result[key] = v.DoubleValue
		case *qdrant.Value_IntegerValue:
			result[key] = v.IntegerValue
		case *qdrant.Value_BoolValue:
			result[key] = v.BoolValue
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
			result[key] = list
		case *qdrant.Value_StructValue:
			result[key] = convertStructToMap(v.StructValue)
		}
	}
	return result
}

// convertMetadataToPayload converts metadata map to Qdrant payload
func convertMetadataToPayload(metadata map[string]interface{}) (map[string]*qdrant.Value, error) {
	payload := make(map[string]*qdrant.Value)

	for key, value := range metadata {
		switch v := value.(type) {
		case string:
			payload[key] = qdrant.NewValueString(v)
		case float64:
			payload[key] = qdrant.NewValueDouble(v)
		case float32:
			payload[key] = qdrant.NewValueDouble(float64(v))
		case int:
			payload[key] = qdrant.NewValueInt(int64(v))
		case int64:
			payload[key] = qdrant.NewValueInt(v)
		case bool:
			payload[key] = qdrant.NewValueBool(v)
		case []string:
			values := make([]*qdrant.Value, len(v))
			for i, s := range v {
				values[i] = qdrant.NewValueString(s)
			}
			payload[key] = &qdrant.Value{
				Kind: &qdrant.Value_ListValue{
					ListValue: &qdrant.ListValue{Values: values},
				},
			}
		case map[string]interface{}:
			if nestedPayload, err := convertMetadataToPayload(v); err == nil {
				payload[key] = qdrant.NewValueStruct(&qdrant.Struct{Fields: nestedPayload})
			}
		default:
			// Convert unknown types to string
			payload[key] = qdrant.NewValueString(fmt.Sprintf("%v", v))
		}
	}

	return payload, nil
}

// AddDocuments adds documents to the collection
func (s *Store) AddDocuments(ctx context.Context, opts *types.AddDocumentOptions) ([]string, error) {

	// Auto connect
	err := s.tryConnect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant server: %w", err)
	}

	// Lock the mutex
	s.mu.RLock()
	defer s.mu.RUnlock()

	if opts == nil || len(opts.Documents) == 0 {
		return nil, fmt.Errorf("no documents provided")
	}

	// Validate vector mode
	if opts.VectorMode != "" && !opts.VectorMode.IsValid() {
		return nil, fmt.Errorf("invalid vector mode: %s", opts.VectorMode)
	}

	var addedIDs []string
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	// Process documents in batches
	for i := 0; i < len(opts.Documents); i += batchSize {
		end := i + batchSize
		if end > len(opts.Documents) {
			end = len(opts.Documents)
		}

		batch := opts.Documents[i:end]
		points := make([]*qdrant.PointStruct, len(batch))
		batchIDs := make([]string, len(batch))

		for j, doc := range batch {
			// Generate ID if not provided
			docID := doc.ID
			if docID == "" {
				docID = fmt.Sprintf("doc_%d_%d", i+j, stringToUint64ID(doc.Content)%1000000)
			}
			batchIDs[j] = docID

			// Create point payload
			payload := map[string]*qdrant.Value{
				"id":      qdrant.NewValueString(docID),
				"content": qdrant.NewValueString(doc.Content),
			}

			// Add metadata if provided
			if doc.Metadata != nil {
				if metadataPayload, err := convertMetadataToPayload(doc.Metadata); err == nil {
					payload["metadata"] = qdrant.NewValueStruct(&qdrant.Struct{Fields: metadataPayload})
				}
			}

			// Create point
			point := &qdrant.PointStruct{
				Id:      qdrant.NewIDNum(stringToUint64ID(docID)),
				Payload: payload,
			}

			// Determine vector mode
			vectorMode := opts.VectorMode
			if vectorMode == "" {
				vectorMode = types.VectorModeAuto
			}

			// Handle vector addition based on mode
			if err := s.addVectorsToPoint(point, doc, vectorMode, opts); err != nil {
				return addedIDs, fmt.Errorf("failed to add vectors to point: %w", err)
			}

			points[j] = point
		}

		// Upsert the batch
		req := &qdrant.UpsertPoints{
			CollectionName: opts.CollectionName,
			Points:         points,
			Wait:           qdrant.PtrOf(!opts.Upsert), // Wait for sync if not upserting
		}

		_, err := s.client.Upsert(ctx, req)
		if err != nil {
			return addedIDs, fmt.Errorf("failed to add documents batch %d: %w", i/batchSize, err)
		}

		addedIDs = append(addedIDs, batchIDs...)
	}

	return addedIDs, nil
}

// addVectorsToPoint adds vectors to a point based on the vector mode
func (s *Store) addVectorsToPoint(point *qdrant.PointStruct, doc *types.Document, vectorMode types.VectorMode, opts *types.AddDocumentOptions) error {
	// First check if the collection actually supports named vectors
	ctx := context.Background()
	collectionSupportsNamedVectors, err := s.collectionUsesNamedVectors(ctx, opts.CollectionName)
	if err != nil {
		// If we can't determine collection info, fall back to content-based logic for backwards compatibility
		collectionSupportsNamedVectors = true
	}

	// Determine if we should use named vectors
	var usesNamedVectors bool
	if !collectionSupportsNamedVectors {
		// Collection doesn't support named vectors, must use single vector mode
		usesNamedVectors = false
	} else {
		// Collection supports named vectors, check if we need them
		usesNamedVectors = s.shouldUseNamedVectors(doc, opts)
	}

	// Get available vectors from document
	hasDense := doc.HasDenseVector()
	hasSparse := doc.HasSparseVector()

	switch vectorMode {
	case types.VectorModeAuto:
		// Add whatever vectors are available
		if hasDense && hasSparse {
			return s.addBothVectors(point, doc, opts, usesNamedVectors)
		} else if hasDense {
			return s.addDenseVector(point, doc, opts, usesNamedVectors)
		} else if hasSparse {
			return s.addSparseVector(point, doc, opts, usesNamedVectors)
		}
		// No vectors available - continue without vectors
		return nil

	case types.VectorModeDenseOnly:
		if hasDense {
			return s.addDenseVector(point, doc, opts, usesNamedVectors)
		}
		// No dense vector available - continue without vectors
		return nil

	case types.VectorModeSparseOnly:
		if hasSparse {
			return s.addSparseVector(point, doc, opts, usesNamedVectors)
		}
		// No sparse vector available - continue without vectors
		return nil

	case types.VectorModeBoth:
		if !hasDense || !hasSparse {
			return fmt.Errorf("vector mode 'both' requires both dense and sparse vectors to be present")
		}
		return s.addBothVectors(point, doc, opts, usesNamedVectors)

	default:
		return fmt.Errorf("unsupported vector mode: %s", vectorMode)
	}
}

// addDenseVector adds dense vector to a point
func (s *Store) addDenseVector(point *qdrant.PointStruct, doc *types.Document, opts *types.AddDocumentOptions, usesNamedVectors bool) error {
	denseVector := doc.GetDenseVector()
	if len(denseVector) == 0 {
		return nil
	}

	vectorData := make([]float32, len(denseVector))
	for k, v := range denseVector {
		vectorData[k] = float32(v)
	}

	if usesNamedVectors {
		// Use named vectors for hybrid search collections
		vectorName := opts.DenseVectorName
		if vectorName == "" {
			// Fall back to legacy VectorUsing field for backward compatibility
			vectorName = opts.VectorUsing
		}
		if vectorName == "" {
			vectorName = "dense" // Default dense vector name
		}

		namedVectors := map[string]*qdrant.Vector{
			vectorName: qdrant.NewVector(vectorData...),
		}
		point.Vectors = qdrant.NewVectorsMap(namedVectors)
	} else {
		// Use single vector for traditional collections
		point.Vectors = qdrant.NewVectors(vectorData...)
	}

	return nil
}

// addSparseVector adds sparse vector to a point
func (s *Store) addSparseVector(point *qdrant.PointStruct, doc *types.Document, opts *types.AddDocumentOptions, usesNamedVectors bool) error {
	if doc.SparseVector == nil {
		return nil
	}

	if !usesNamedVectors {
		// Sparse vectors can only be used with named vector collections
		return fmt.Errorf("sparse vectors require named vector collections")
	}

	vectorName := opts.SparseVectorName
	if vectorName == "" {
		vectorName = "sparse" // Default sparse vector name
	}

	namedVectors := map[string]*qdrant.Vector{
		vectorName: qdrant.NewVectorSparse(doc.SparseVector.Indices, doc.SparseVector.Values),
	}
	point.Vectors = qdrant.NewVectorsMap(namedVectors)

	return nil
}

// addBothVectors adds both dense and sparse vectors to a point
func (s *Store) addBothVectors(point *qdrant.PointStruct, doc *types.Document, opts *types.AddDocumentOptions, usesNamedVectors bool) error {
	if !usesNamedVectors {
		// Both vectors require named vector collections
		return fmt.Errorf("both dense and sparse vectors require named vector collections")
	}

	namedVectors := make(map[string]*qdrant.Vector)

	// Add dense vector
	denseVector := doc.GetDenseVector()
	if len(denseVector) > 0 {
		vectorData := make([]float32, len(denseVector))
		for k, v := range denseVector {
			vectorData[k] = float32(v)
		}

		denseName := opts.DenseVectorName
		if denseName == "" {
			denseName = "dense"
		}
		namedVectors[denseName] = qdrant.NewVector(vectorData...)
	}

	// Add sparse vector
	if doc.SparseVector != nil {
		sparseName := opts.SparseVectorName
		if sparseName == "" {
			sparseName = "sparse"
		}
		namedVectors[sparseName] = qdrant.NewVectorSparse(doc.SparseVector.Indices, doc.SparseVector.Values)
	}

	if len(namedVectors) > 0 {
		point.Vectors = qdrant.NewVectorsMap(namedVectors)
	}

	return nil
}

// collectionUsesNamedVectors checks if a collection uses named vectors by querying collection info
func (s *Store) collectionUsesNamedVectors(ctx context.Context, collectionName string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return false, fmt.Errorf("not connected to Qdrant server")
	}

	info, err := s.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get collection info: %w", err)
	}

	// Check if the collection has named vectors (VectorsConfig is a map)
	if info.Config != nil && info.Config.Params != nil {
		if vectorsConfig := info.Config.Params.VectorsConfig; vectorsConfig != nil {
			// If VectorsConfig has a Map field, it uses named vectors
			switch vectorsConfig.Config.(type) {
			case *qdrant.VectorsConfig_ParamsMap:
				return true, nil
			case *qdrant.VectorsConfig_Params:
				return false, nil
			}
		}
	}

	return false, nil
}

// shouldUseNamedVectors determines if named vectors should be used based on document content and options
// This function assumes the collection supports named vectors
func (s *Store) shouldUseNamedVectors(doc *types.Document, opts *types.AddDocumentOptions) bool {
	// If we have both dense and sparse vectors, we must use named vectors
	if doc.HasDenseVector() && doc.HasSparseVector() {
		return true
	}

	// If user explicitly specified vector names, use named vectors
	if opts.DenseVectorName != "" || opts.SparseVectorName != "" || opts.VectorUsing != "" {
		return true
	}

	// If we only have sparse vectors, we must use named vectors (sparse vectors require naming)
	if doc.HasSparseVector() {
		return true
	}

	// For collections that support named vectors, default to using them even for dense-only vectors
	// This provides consistency and allows for future expansion
	if doc.HasDenseVector() {
		return true
	}

	// No vectors at all
	return false
}

// GetDocuments retrieves documents by IDs
func (s *Store) GetDocuments(ctx context.Context, ids []string, opts *types.GetDocumentOptions) ([]*types.Document, error) {
	// Auto connect
	err := s.tryConnect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant server: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	if len(ids) == 0 {
		return []*types.Document{}, nil
	}

	// Convert string IDs to Qdrant point IDs
	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = qdrant.NewIDNum(stringToUint64ID(id))
	}

	// Prepare request
	req := &qdrant.GetPoints{
		CollectionName: opts.CollectionName,
		Ids:            pointIDs,
		WithPayload:    qdrant.NewWithPayload(opts.IncludePayload),
		WithVectors:    qdrant.NewWithVectors(opts.IncludeVector),
	}

	// Get points from Qdrant
	points, err := s.client.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	// Convert to documents
	documents := make([]*types.Document, len(points))
	for i, point := range points {
		documents[i] = convertRetrievedPointToDocument(point, opts.IncludeVector, opts.IncludePayload)
	}

	return documents, nil
}

// DeleteDocuments deletes documents from the collection
func (s *Store) DeleteDocuments(ctx context.Context, opts *types.DeleteDocumentOptions) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("not connected to Qdrant server")
	}

	if opts == nil {
		return fmt.Errorf("delete options cannot be nil")
	}

	var pointsSelector *qdrant.PointsSelector

	if len(opts.IDs) > 0 {
		// Delete by specific IDs
		pointIDs := make([]*qdrant.PointId, len(opts.IDs))
		for i, id := range opts.IDs {
			pointIDs[i] = qdrant.NewIDNum(stringToUint64ID(id))
		}

		pointsSelector = &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointIDs,
				},
			},
		}
	} else if opts.Filter != nil {
		// Delete by filter
		filter, err := convertFilterToQdrant(opts.Filter)
		if err != nil {
			return fmt.Errorf("failed to convert filter: %w", err)
		}

		pointsSelector = &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: filter,
			},
		}
	} else {
		return fmt.Errorf("either IDs or filter must be provided")
	}

	// Perform deletion (dry run check)
	if opts.DryRun {
		// For dry run, we could use scroll to see what would be deleted
		// but for now, just return success without actually deleting
		return nil
	}

	req := &qdrant.DeletePoints{
		CollectionName: opts.CollectionName,
		Points:         pointsSelector,
		Wait:           qdrant.PtrOf(true), // Wait for completion
	}

	_, err := s.client.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// convertFilterToQdrant converts generic filter to Qdrant filter
func convertFilterToQdrant(filter map[string]interface{}) (*qdrant.Filter, error) {
	conditions := make([]*qdrant.Condition, 0)

	for key, value := range filter {
		var condition *qdrant.Condition

		switch v := value.(type) {
		case string:
			condition = &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: fmt.Sprintf("metadata.%s", key), // Access nested metadata field
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: v,
							},
						},
					},
				},
			}
		case float64, int, int64:
			var floatVal float64
			switch vt := v.(type) {
			case float64:
				floatVal = vt
			case int:
				floatVal = float64(vt)
			case int64:
				floatVal = float64(vt)
			}

			condition = &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: fmt.Sprintf("metadata.%s", key), // Access nested metadata field
						Range: &qdrant.Range{
							Gte: &floatVal,
							Lte: &floatVal,
						},
					},
				},
			}
		case bool:
			condition = &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: fmt.Sprintf("metadata.%s", key), // Access nested metadata field
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Boolean{
								Boolean: v,
							},
						},
					},
				},
			}
		}

		if condition != nil {
			conditions = append(conditions, condition)
		}
	}

	if len(conditions) == 0 {
		return nil, fmt.Errorf("no valid filter conditions found")
	}

	return &qdrant.Filter{
		Must: conditions,
	}, nil
}

// ListDocuments lists documents with pagination (Deprecated)
func (s *Store) ListDocuments(ctx context.Context, opts *types.ListDocumentsOptions) (*types.PaginatedDocumentsResult, error) {
	// Auto connect
	err := s.tryConnect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant server: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	if opts == nil {
		return nil, fmt.Errorf("list options cannot be nil")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Use scroll API for listing with pagination
	req := &qdrant.ScrollPoints{
		CollectionName: opts.CollectionName,
		Limit:          qdrant.PtrOf(uint32(limit)),
		WithPayload:    qdrant.NewWithPayload(opts.IncludePayload),
		WithVectors:    qdrant.NewWithVectors(opts.IncludeVector),
	}

	// Add filter if provided
	if opts.Filter != nil {
		filter, err := convertFilterToQdrant(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter: %w", err)
		}
		req.Filter = filter
	}

	// Handle offset using scroll API - this is simplified pagination
	// For proper offset support, we would need to implement cursor-based pagination
	if opts.Offset > 0 {
		// Create a temporary scroll ID based on offset
		req.Offset = qdrant.NewIDNum(uint64(opts.Offset))
	}

	// Perform the scroll
	result, err := s.client.Scroll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	// Convert results to documents
	documents := make([]*types.Document, len(result))
	for i, point := range result {
		documents[i] = convertRetrievedPointToDocument(point, opts.IncludeVector, opts.IncludePayload)
	}

	// Get total count by performing a count query
	countReq := &qdrant.CountPoints{
		CollectionName: opts.CollectionName,
		Filter:         req.Filter,          // Use same filter for count
		Exact:          qdrant.PtrOf(false), // Use approximate count for performance
	}

	countResult, err := s.client.Count(ctx, countReq)
	var totalCount int64
	if err != nil {
		// If count fails, set total to 0 but don't fail the request
		totalCount = 0
	} else {
		totalCount = int64(countResult)
	}

	// Prepare pagination result
	paginatedResult := &types.PaginatedDocumentsResult{
		Documents: documents,
		Total:     totalCount,
		HasMore:   len(result) == limit, // Simplified: assume more if we got exactly the limit
	}

	if len(documents) > 0 {
		paginatedResult.NextOffset = opts.Offset + len(documents)
	}

	return paginatedResult, nil
}

// ScrollDocuments provides iterator-style access to documents
func (s *Store) ScrollDocuments(ctx context.Context, opts *types.ScrollOptions) (*types.ScrollResult, error) {
	// Auto connect
	err := s.tryConnect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant server: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	if opts == nil {
		return nil, fmt.Errorf("scroll options cannot be nil")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Request limit+1 documents to determine if there are more results
	// If limit+1 documents are returned, the last one serves as cursor but is not returned to user
	queryLimit := limit + 1

	// Prepare scroll request
	req := &qdrant.ScrollPoints{
		CollectionName: opts.CollectionName,
		Limit:          qdrant.PtrOf(uint32(queryLimit)), // Request limit+1 documents
		WithPayload:    qdrant.NewWithPayload(opts.IncludePayload),
		WithVectors:    qdrant.NewWithVectors(opts.IncludeVector),
	}

	// Add filter if provided
	if opts.Filter != nil {
		filter, err := convertFilterToQdrant(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter: %w", err)
		}
		req.Filter = filter
	}

	// Add ordering if provided
	if len(opts.OrderBy) > 0 {
		// Use the first OrderBy field (Qdrant ScrollPoints supports single field ordering)
		orderField := opts.OrderBy[0]

		// Parse direction from field name (e.g., "score:desc" or just "score" for asc)
		direction := qdrant.Direction_Asc // Default to ascending
		fieldKey := orderField

		if strings.Contains(orderField, ":") {
			parts := strings.Split(orderField, ":")
			if len(parts) == 2 {
				fieldKey = parts[0]
				if strings.ToLower(parts[1]) == "desc" {
					direction = qdrant.Direction_Desc
				}
			}
		}

		// Set OrderBy for the request
		req.OrderBy = &qdrant.OrderBy{
			Key:       fmt.Sprintf("metadata.%s", fieldKey), // Access nested metadata field
			Direction: &direction,
		}
	}

	// Handle continuation with scroll ID
	if opts.ScrollID != "" {

		// Remove the order by from the request
		req.OrderBy = nil

		// Convert scroll ID back to offset (this is a simplified approach)
		if offset, err := strconv.ParseUint(opts.ScrollID, 10, 64); err == nil {
			req.Offset = qdrant.NewIDNum(offset)
		}
	}

	// Perform single scroll request (not collecting all documents)
	result, err := s.client.Scroll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to scroll documents: %w", err)
	}

	// Process results: if limit+1 documents are returned, use the last one as cursor but don't return to user
	var documents []*types.Document
	var hasMore bool
	var nextScrollID string

	if len(result) > limit {
		// Received limit+1 documents, indicating there are more results
		hasMore = true

		// Convert only the first 'limit' documents for the user
		for i := 0; i < limit; i++ {
			doc := convertRetrievedPointToDocument(result[i], opts.IncludeVector, opts.IncludePayload)
			documents = append(documents, doc)
		}

		// Use the last document (limit+1-th) as cursor
		lastPoint := result[limit] // This is the (limit+1)-th document, used as cursor
		if lastPoint.Id != nil {
			if numID := lastPoint.Id.GetNum(); numID != 0 {
				nextScrollID = fmt.Sprintf("%d", numID)
			} else if uuidID := lastPoint.Id.GetUuid(); uuidID != "" {
				nextScrollID = uuidID
			}
		}
	} else {
		// Received <= limit documents, indicating no more results
		hasMore = false
		nextScrollID = ""

		// Convert all returned documents
		for _, point := range result {
			doc := convertRetrievedPointToDocument(point, opts.IncludeVector, opts.IncludePayload)
			documents = append(documents, doc)
		}
	}

	// Prepare scroll result
	scrollResult := &types.ScrollResult{
		Documents: documents,
		ScrollID:  nextScrollID,
		HasMore:   hasMore,
	}

	return scrollResult, nil
}
