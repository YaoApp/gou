package qdrant

import (
	"context"
	"crypto/md5"
	"fmt"
	"strconv"

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
		if contentVal := point.Payload["page_content"]; contentVal != nil {
			doc.PageContent = contentVal.GetStringValue()
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
		if contentVal := point.Payload["page_content"]; contentVal != nil {
			doc.PageContent = contentVal.GetStringValue()
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	if opts == nil || len(opts.Documents) == 0 {
		return nil, fmt.Errorf("no documents provided")
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
				docID = fmt.Sprintf("doc_%d_%d", i+j, stringToUint64ID(doc.PageContent)%1000000)
			}
			batchIDs[j] = docID

			// Create point payload
			payload := map[string]*qdrant.Value{
				"id":           qdrant.NewValueString(docID),
				"page_content": qdrant.NewValueString(doc.PageContent),
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

			// Add vector if provided
			if len(doc.Vector) > 0 {
				vectorData := make([]float32, len(doc.Vector))
				for k, v := range doc.Vector {
					vectorData[k] = float32(v)
				}
				point.Vectors = qdrant.NewVectors(vectorData...)
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

// GetDocuments retrieves documents by IDs
func (s *Store) GetDocuments(ctx context.Context, ids []string, opts *types.GetDocumentOptions) ([]*types.Document, error) {
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

// ListDocuments lists documents with pagination
func (s *Store) ListDocuments(ctx context.Context, opts *types.ListDocumentsOptions) (*types.PaginatedDocumentsResult, error) {
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	if opts == nil {
		return nil, fmt.Errorf("scroll options cannot be nil")
	}

	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	// Prepare scroll request
	req := &qdrant.ScrollPoints{
		CollectionName: opts.CollectionName,
		Limit:          qdrant.PtrOf(uint32(batchSize)),
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

	// Handle continuation with scroll ID
	if opts.ScrollID != "" {
		// Convert scroll ID back to offset (this is a simplified approach)
		if offset, err := strconv.ParseUint(opts.ScrollID, 10, 64); err == nil {
			req.Offset = qdrant.NewIDNum(offset)
		}
	}

	// Collect all documents by scrolling through all batches
	var allDocuments []*types.Document
	var nextOffset *qdrant.PointId

	for {
		// Set offset for continuation
		if nextOffset != nil {
			req.Offset = nextOffset
		}

		// Perform scroll
		result, err := s.client.Scroll(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to scroll documents: %w", err)
		}

		// Convert results to documents
		for _, point := range result {
			doc := convertRetrievedPointToDocument(point, opts.IncludeVector, opts.IncludePayload)
			allDocuments = append(allDocuments, doc)
		}

		// Check if we have more results
		if len(result) < batchSize {
			// No more results
			break
		}

		// Set next offset to the last point ID for continuation
		if len(result) > 0 {
			lastPoint := result[len(result)-1]
			nextOffset = lastPoint.Id
		} else {
			break
		}
	}

	// Prepare scroll result
	scrollResult := &types.ScrollResult{
		Documents: allDocuments,
		HasMore:   false, // We've collected all documents
	}

	return scrollResult, nil
}
