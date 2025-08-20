package graphrag

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// AddSegments adds segments to a collection manually
func (g *GraphRag) AddSegments(ctx context.Context, docID string, segmentTexts []types.SegmentText, options *types.UpsertOptions) ([]string, error) {
	// Step 1: Parse CollectionID from docID
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
	}

	// Step 2: Prepare options by copying and setting necessary fields
	opts := &types.UpsertOptions{}
	if options != nil {
		*opts = *options
	}
	opts.CollectionID = collectionID
	opts.DocID = docID

	// Step 3: Get collection IDs for vector and graph storage
	collectionIDs, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	// Step 4: Convert SegmentTexts to Chunks
	chunks, err := g.convertSegmentTextsToChunks(segmentTexts, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert segment texts to chunks: %w", err)
	}

	// Step 5: Create the callback for progress tracking
	var embeddingTexts []string = []string{}
	var embeddingIndexesMap map[*types.Chunk]int = map[*types.Chunk]int{}
	var cb = MakeUpsertCallback(docID, nil, opts.Progress)

	// Step 6: Require embedding and extraction configurations
	if opts.Embedding == nil {
		return nil, fmt.Errorf("embedding configuration is required for AddSegments operation")
	}

	if g.Graph != nil && opts.Extraction == nil {
		return nil, fmt.Errorf("extraction configuration is required when graph store is configured")
	}

	// Step 7: Store embedding indexes for chunks (equivalent to AddFile step 4.2)
	for _, chunk := range chunks {
		embeddingTexts = append(embeddingTexts, chunk.Text)
		embeddingIndexesMap[chunk] = len(embeddingTexts) - 1
	}

	// Step 8: Extract entities and relationships from chunks (equivalent to AddFile step 4.3)
	allEntities, allRelationships, entityIndexMap, relationshipIndexMap, err := g.extractEntitiesAndRelationships(ctx, chunks, opts, cb, &embeddingTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities and relationships: %w", err)
	}

	// Step 9: Embed all texts (chunks + entities + relationships) (equivalent to AddFile step 5)
	embeddings, err := opts.Embedding.EmbedDocuments(ctx, embeddingTexts, cb.Embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to embed the documents: %w", err)
	}

	// Step 10: Store entities and relationships to graph store first for deduplication (equivalent to AddFile step 6)
	var actualEntityIDs []string
	var actualRelationshipIDs []string
	var entityIDMap = make(map[string]string)
	var relationshipIDMap = make(map[string]string)

	var entityDeduplicationResults map[string]*EntityDeduplicationResult
	var relationshipDeduplicationResults map[string]*RelationshipDeduplicationResult

	if g.Graph != nil && opts.Extraction != nil && (len(allEntities) > 0 || len(allRelationships) > 0) {
		// Store entities to graph store
		if len(allEntities) > 0 {
			actualEntityIDs, entityDeduplicationResults, err = g.storeEntitiesToGraphStore(ctx, allEntities, collectionIDs.Graph, docID)
			if err != nil {
				return nil, fmt.Errorf("failed to store entities to graph store: %w", err)
			}

			// Create mapping from original IDs to actual IDs
			for i, entity := range allEntities {
				if i < len(actualEntityIDs) {
					entityIDMap[entity.ID] = actualEntityIDs[i]
				}
			}
		}

		// Store relationships to graph store
		if len(allRelationships) > 0 {
			actualRelationshipIDs, relationshipDeduplicationResults, err = g.storeRelationshipsToGraphStore(ctx, allRelationships, collectionIDs.Graph, docID)
			if err != nil {
				return nil, fmt.Errorf("failed to store relationships to graph store: %w", err)
			}

			// Create mapping from original IDs to actual IDs
			for i, relationship := range allRelationships {
				if i < len(actualRelationshipIDs) {
					relationshipIDMap[relationship.ID] = actualRelationshipIDs[i]
				}
			}
		}

		// Update chunks with actual IDs from graph database
		g.updateChunksWithActualIds(chunks, entityIDMap, relationshipIDMap)
	}

	// Step 11: Store all documents to vector store (equivalent to AddFile step 7)
	storeOptions := &StoreDocumentsOptions{
		Chunks:                           chunks,
		Entities:                         allEntities,
		Relationships:                    allRelationships,
		Embeddings:                       embeddings,
		EmbeddingIndexesMap:              embeddingIndexesMap,
		EntityIndexMap:                   entityIndexMap,
		RelationshipIndexMap:             relationshipIndexMap,
		SourceFile:                       "",
		ConvertMetadata:                  make(map[string]interface{}),
		UserMetadata:                     opts.Metadata,
		VectorCollectionName:             collectionIDs.Vector,
		CollectionID:                     collectionID,
		DocID:                            docID,
		EntityDeduplicationResults:       entityDeduplicationResults,
		RelationshipDeduplicationResults: relationshipDeduplicationResults,
	}

	err = g.storeAllDocumentsToVectorStore(ctx, storeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to store documents to vector store: %w", err)
	}

	// Step 12: Store segment metadata (Weight, Score, Vote) to Store if configured
	if g.Store != nil {
		err = g.storeSegmentMetadataToStore(ctx, docID, chunks, collectionIDs.Store)
		if err != nil {
			g.Logger.Warnf("Failed to store segment metadata to Store: %v", err)
		}
	}

	// Collect and return all segment IDs
	segmentIDs := make([]string, len(chunks))
	for i, chunk := range chunks {
		segmentIDs[i] = chunk.ID
	}

	return segmentIDs, nil
}

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

// removeSegmentsFromStore removes segment metadata from Store
func (g *GraphRag) removeSegmentsFromStore(ctx context.Context, docID string, segmentIDs []string) {
	for _, segmentID := range segmentIDs {
		// Delete Weight
		err := g.deleteSegmentValue(segmentID, StoreKeyWeight)
		if err != nil {
			g.Logger.Warnf("Failed to delete weight for segment %s: %v", segmentID, err)
		}

		// Delete Score
		err = g.deleteSegmentValue(segmentID, StoreKeyScore)
		if err != nil {
			g.Logger.Warnf("Failed to delete score for segment %s: %v", segmentID, err)
		}

		// Delete Vote
		err = g.deleteSegmentValue(segmentID, StoreKeyVote)
		if err != nil {
			g.Logger.Warnf("Failed to delete vote for segment %s: %v", segmentID, err)
		}
	}
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
		err := g.deleteSegmentValue(segmentID, StoreKeyWeight)
		if err != nil {
			g.Logger.Warnf("Failed to delete weight for segment %s: %v", segmentID, err)
		} else {
			removedCount++
		}

		// Delete Score
		err = g.deleteSegmentValue(segmentID, StoreKeyScore)
		if err != nil {
			g.Logger.Warnf("Failed to delete score for segment %s: %v", segmentID, err)
		} else {
			removedCount++
		}

		// Delete Vote
		err = g.deleteSegmentValue(segmentID, StoreKeyVote)
		if err != nil {
			g.Logger.Warnf("Failed to delete vote for segment %s: %v", segmentID, err)
		} else {
			removedCount++
		}
	}

	g.Logger.Infof("Segment metadata cleanup completed for document %s: removed %d Store entries for %d segments", docID, removedCount, len(segmentIDs))
}

// UpdateSegments updates segments manually
func (g *GraphRag) UpdateSegments(ctx context.Context, segmentTexts []types.SegmentText, options *types.UpsertOptions) (int, error) {
	// Step 1: Validate input - all segments must have IDs
	for i, segmentText := range segmentTexts {
		if segmentText.ID == "" {
			return 0, fmt.Errorf("segment at index %d is missing ID - all segments must have IDs for update", i)
		}
	}

	if len(segmentTexts) == 0 {
		return 0, nil
	}

	// Step 2: Get existing chunks to read doc_id and validate segments exist
	segmentIDs := make([]string, 0, len(segmentTexts))
	for _, segment := range segmentTexts {
		segmentIDs = append(segmentIDs, segment.ID)
	}

	// docID, collectionID, err := g.getDocIDFromExistingSegments(ctx, segmentIDs)
	// if err != nil {
	// 	return 0, fmt.Errorf("failed to get doc_id from existing segments: %w", err)
	// }

	// Step 3: Get collection ID and doc ID from options (for update segments)
	collectionID := options.CollectionID
	docID := options.DocID

	// Step 3: Prepare options by copying and setting necessary fields
	opts := &types.UpsertOptions{}
	if options != nil {
		*opts = *options
	}
	opts.CollectionID = collectionID
	opts.DocID = docID

	// Step 4: Get collection IDs for vector and graph storage
	collectionIDs, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		return 0, fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	// Step 5: Convert SegmentTexts to Chunks for processing
	chunks, err := g.convertSegmentTextsToChunks(segmentTexts, docID)
	if err != nil {
		return 0, fmt.Errorf("failed to convert segment texts to chunks: %w", err)
	}

	// Step 6: Create the callback for progress tracking
	var embeddingTexts []string = []string{}
	var embeddingIndexesMap map[*types.Chunk]int = map[*types.Chunk]int{}
	var cb = MakeUpsertCallback(docID, nil, opts.Progress)

	// Step 7: Require embedding and extraction configurations
	if opts.Embedding == nil {
		return 0, fmt.Errorf("embedding configuration is required for UpdateSegments operation")
	}

	if g.Graph != nil && opts.Extraction == nil {
		return 0, fmt.Errorf("extraction configuration is required when graph store is configured")
	}

	// Step 8: If Graph is configured, remove existing entities and relationships for these segments
	if g.Graph != nil && opts.Extraction != nil {
		err = g.removeEntitiesAndRelationshipsForSegments(ctx, collectionIDs.Graph, segmentIDs)
		if err != nil {
			g.Logger.Warnf("Failed to remove existing entities and relationships: %v", err)
		}
	}

	// Step 9: Store embedding indexes for chunks
	for _, chunk := range chunks {
		embeddingTexts = append(embeddingTexts, chunk.Text)
		embeddingIndexesMap[chunk] = len(embeddingTexts) - 1
	}

	// Step 10: Extract entities and relationships from chunks (if Graph is configured)
	allEntities, allRelationships, entityIndexMap, relationshipIndexMap, err := g.extractEntitiesAndRelationships(ctx, chunks, opts, cb, &embeddingTexts)
	if err != nil {
		return 0, fmt.Errorf("failed to extract entities and relationships: %w", err)
	}

	// Step 11: Embed all texts (chunks + entities + relationships)
	embeddings, err := opts.Embedding.EmbedDocuments(ctx, embeddingTexts, cb.Embedding)
	if err != nil {
		return 0, fmt.Errorf("failed to embed the documents: %w", err)
	}

	// Step 12: Store entities and relationships to graph store (if available)
	var actualEntityIDs []string
	var actualRelationshipIDs []string
	var entityIDMap = make(map[string]string)
	var relationshipIDMap = make(map[string]string)

	var entityDeduplicationResults map[string]*EntityDeduplicationResult
	var relationshipDeduplicationResults map[string]*RelationshipDeduplicationResult

	if g.Graph != nil && opts.Extraction != nil && (len(allEntities) > 0 || len(allRelationships) > 0) {
		// Store entities to graph store
		if len(allEntities) > 0 {
			actualEntityIDs, entityDeduplicationResults, err = g.storeEntitiesToGraphStore(ctx, allEntities, collectionIDs.Graph, docID)
			if err != nil {
				return 0, fmt.Errorf("failed to store entities to graph store: %w", err)
			}

			// Create mapping from original IDs to actual IDs
			for i, entity := range allEntities {
				if i < len(actualEntityIDs) {
					entityIDMap[entity.ID] = actualEntityIDs[i]
				}
			}
		}

		// Store relationships to graph store
		if len(allRelationships) > 0 {
			actualRelationshipIDs, relationshipDeduplicationResults, err = g.storeRelationshipsToGraphStore(ctx, allRelationships, collectionIDs.Graph, docID)
			if err != nil {
				return 0, fmt.Errorf("failed to store relationships to graph store: %w", err)
			}

			// Create mapping from original IDs to actual IDs
			for i, relationship := range allRelationships {
				if i < len(actualRelationshipIDs) {
					relationshipIDMap[relationship.ID] = actualRelationshipIDs[i]
				}
			}
		}

		// Update chunks with actual IDs from graph database
		g.updateChunksWithActualIds(chunks, entityIDMap, relationshipIDMap)
	}

	// Step 13: Update all documents in vector store (chunks + entities + relationships)
	storeOptions := &StoreDocumentsOptions{
		Chunks:                           chunks,
		Entities:                         allEntities,
		Relationships:                    allRelationships,
		Embeddings:                       embeddings,
		EmbeddingIndexesMap:              embeddingIndexesMap,
		EntityIndexMap:                   entityIndexMap,
		RelationshipIndexMap:             relationshipIndexMap,
		SourceFile:                       "",
		ConvertMetadata:                  make(map[string]interface{}),
		UserMetadata:                     opts.Metadata,
		VectorCollectionName:             collectionIDs.Vector,
		CollectionID:                     collectionID,
		DocID:                            docID,
		EntityDeduplicationResults:       entityDeduplicationResults,
		RelationshipDeduplicationResults: relationshipDeduplicationResults,
	}

	err = g.storeAllDocumentsToVectorStore(ctx, storeOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to update documents in vector store: %w", err)
	}

	// Step 14: Update segment metadata in Store if configured and user metadata contains vote/weight/score
	if g.Store != nil && opts.Metadata != nil {
		err = g.updateSegmentMetadataInStore(ctx, docID, segmentTexts, opts.Metadata, collectionIDs.Store)
		if err != nil {
			g.Logger.Warnf("Failed to update segment metadata in Store: %v", err)
		}
	}

	return len(segmentTexts), nil
}

// RemoveSegments removes segments by IDs
func (g *GraphRag) RemoveSegments(ctx context.Context, segmentIDs []string) (int, error) {
	if len(segmentIDs) == 0 {
		return 0, nil
	}

	g.Logger.Infof("Starting to remove %d segments", len(segmentIDs))

	// Step 1: Get docID and graphName from existing segments
	// For segments removal, we need to determine which document and graph they belong to
	docID, graphName, err := g.getDocIDFromExistingSegments(ctx, segmentIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to get document info from segments: %w", err)
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

// GetSegments gets segments by IDs
func (g *GraphRag) GetSegments(ctx context.Context, segmentIDs []string) ([]types.Segment, error) {
	if len(segmentIDs) == 0 {
		return []types.Segment{}, nil
	}

	g.Logger.Debugf("Getting %d segments by IDs", len(segmentIDs))

	// Get docID and graphName from existing segments
	docID, graphName, err := g.getDocIDFromExistingSegments(ctx, segmentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get document info from segments: %w", err)
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

// GetSegment gets a single segment by ID
func (g *GraphRag) GetSegment(ctx context.Context, segmentID string) (*types.Segment, error) {
	if segmentID == "" {
		return nil, fmt.Errorf("segmentID cannot be empty")
	}

	g.Logger.Debugf("Getting segment by ID: %s", segmentID)

	// Get segments using GetSegments
	segments, err := g.GetSegments(ctx, []string{segmentID})
	if err != nil {
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("segment %s not found", segmentID)
	}

	return &segments[0], nil
}

// convertSegmentTextsToChunks converts SegmentTexts to Chunks for processing
func (g *GraphRag) convertSegmentTextsToChunks(segmentTexts []types.SegmentText, docID string) ([]*types.Chunk, error) {
	chunks := make([]*types.Chunk, 0, len(segmentTexts))

	for i, segmentText := range segmentTexts {
		chunkID := segmentText.ID
		if chunkID == "" {
			// Generate UUID for chunk ID
			chunkID = utils.GenChunkID()
		}

		chunk := &types.Chunk{
			ID:      chunkID,
			Text:    segmentText.Text,
			Type:    types.ChunkingTypeText,
			Depth:   0,
			Index:   i,
			Leaf:    true,
			Root:    true,
			Status:  types.ChunkingStatusCompleted,
			Parents: []types.Chunk{},
			TextPos: &types.TextPosition{
				StartIndex: 0,
				EndIndex:   len(segmentText.Text),
				StartLine:  1,
				EndLine:    1,
			},
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

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
		err := g.storeSegmentValue(segmentID, StoreKeyWeight, 0.0)
		if err != nil {
			g.Logger.Warnf("Failed to store weight for segment %s: %v", segmentID, err)
		}

		// Store default Score
		err = g.storeSegmentValue(segmentID, StoreKeyScore, 0.0)
		if err != nil {
			g.Logger.Warnf("Failed to store score for segment %s: %v", segmentID, err)
		}

		// Store default Vote
		err = g.storeSegmentValue(segmentID, StoreKeyVote, 0)
		if err != nil {
			g.Logger.Warnf("Failed to store vote for segment %s: %v", segmentID, err)
		}
	}

	return nil
}

// getDocIDFromExistingSegments retrieves doc_id and graph_name from existing segments in vector store
func (g *GraphRag) getDocIDFromExistingSegments(ctx context.Context, segmentIDs []string) (string, string, error) {
	if len(segmentIDs) == 0 {
		return "", "", fmt.Errorf("no segment IDs provided")
	}

	// We need to search across all possible collections since we don't know the graph name yet
	// Try to get documents from vector store using the first segment ID
	firstSegmentID := segmentIDs[0]

	// Since we don't know which collection the segments are in, we need to search
	// This is a limitation - ideally the user should provide the graphName or we should store it elsewhere
	// For now, we'll try a few common approaches:

	// Try to extract graphName from segmentID if it follows a pattern
	graphName := "default" // Default fallback

	// Try to get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	// Check if vector collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionIDs.Vector)
	if err != nil {
		return "", "", fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		return "", "", fmt.Errorf("vector collection %s does not exist", collectionIDs.Vector)
	}

	// Get the first segment document to read metadata
	getOpts := &types.GetDocumentOptions{
		CollectionName: collectionIDs.Vector,
		IncludeVector:  false,
		IncludePayload: true,
	}

	docs, err := g.Vector.GetDocuments(ctx, []string{firstSegmentID}, getOpts)
	if err != nil {
		return "", "", fmt.Errorf("failed to get segment documents: %w", err)
	}

	if len(docs) == 0 || docs[0] == nil {
		return "", "", fmt.Errorf("segment %s not found in vector store", firstSegmentID)
	}

	doc := docs[0]

	// Extract doc_id from metadata
	docID, ok := doc.Metadata["doc_id"].(string)
	if !ok {
		return "", "", fmt.Errorf("doc_id not found in segment metadata")
	}

	// Extract collection_id (graph_name) from metadata
	if collectionID, ok := doc.Metadata["collection_id"].(string); ok {
		graphName = collectionID
	}

	// Validate all segments exist
	allDocs, err := g.Vector.GetDocuments(ctx, segmentIDs, getOpts)
	if err != nil {
		return "", "", fmt.Errorf("failed to validate all segments exist: %w", err)
	}

	for i, segmentID := range segmentIDs {
		if i >= len(allDocs) || allDocs[i] == nil {
			return "", "", fmt.Errorf("segment %s not found in vector store", segmentID)
		}

		// Verify doc_id consistency
		segmentDocID, ok := allDocs[i].Metadata["doc_id"].(string)
		if !ok || segmentDocID != docID {
			return "", "", fmt.Errorf("segment %s has inconsistent doc_id", segmentID)
		}
	}

	return docID, graphName, nil
}

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

// updateSegmentMetadataInStore updates segment metadata (Weight, Score, Vote) in Store from user metadata
func (g *GraphRag) updateSegmentMetadataInStore(ctx context.Context, docID string, segmentTexts []types.SegmentText, userMetadata map[string]interface{}, storeCollectionName string) error {
	if g.Store == nil {
		return nil
	}

	// Check if user metadata contains vote, weight, or score
	vote, hasVote := userMetadata["vote"]
	weight, hasWeight := userMetadata["weight"]
	score, hasScore := userMetadata["score"]

	if !hasVote && !hasWeight && !hasScore {
		// No relevant metadata to update
		return nil
	}

	// Update metadata for each segment
	for _, segmentText := range segmentTexts {
		segmentID := segmentText.ID

		// Update Vote if provided
		if hasVote {
			err := g.storeSegmentValue(segmentID, StoreKeyVote, vote)
			if err != nil {
				g.Logger.Warnf("Failed to update vote for segment %s: %v", segmentID, err)
			}
		}

		// Update Weight if provided
		if hasWeight {
			err := g.storeSegmentValue(segmentID, StoreKeyWeight, weight)
			if err != nil {
				g.Logger.Warnf("Failed to update weight for segment %s: %v", segmentID, err)
			}
		}

		// Update Score if provided
		if hasScore {
			err := g.storeSegmentValue(segmentID, StoreKeyScore, score)
			if err != nil {
				g.Logger.Warnf("Failed to update score for segment %s: %v", segmentID, err)
			}
		}
	}

	return nil
}

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
		storeDataFromStore = g.queryMetadataFromStoreOnly(segmentIDs)
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

			// Extract Vote, Score, Weight from metadata
			if vote, ok := doc.Metadata["vote"]; ok {
				segmentData["vote"] = vote
			}
			if score, ok := doc.Metadata["score"]; ok {
				segmentData["score"] = score
			}
			if weight, ok := doc.Metadata["weight"]; ok {
				segmentData["weight"] = weight
			}

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
func (g *GraphRag) queryMetadataFromStoreOnly(segmentIDs []string) map[string]interface{} {
	storeData := make(map[string]interface{})

	if g.Store == nil {
		return storeData
	}

	// Query metadata for each segment
	for _, segmentID := range segmentIDs {
		segmentData := make(map[string]interface{})

		// Query Weight
		weight, ok := g.getSegmentValue(segmentID, StoreKeyWeight)
		if ok {
			segmentData["weight"] = weight
		}

		// Query Score
		score, ok := g.getSegmentValue(segmentID, StoreKeyScore)
		if ok {
			segmentData["score"] = score
		}

		// Query Vote
		vote, ok := g.getSegmentValue(segmentID, StoreKeyVote)
		if ok {
			segmentData["vote"] = vote
		}

		if len(segmentData) > 0 {
			storeData[segmentID] = segmentData
		}
	}

	return storeData
}

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
			Vote:          0,
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

		// Add metadata from store
		if segmentData, ok := data.StoreData[chunk.ID]; ok {
			if segmentMap, ok := segmentData.(map[string]interface{}); ok {
				if weight, ok := segmentMap["weight"]; ok {
					if weightFloat, ok := weight.(float64); ok {
						segment.Weight = weightFloat
					}
				}
				if score, ok := segmentMap["score"]; ok {
					if scoreFloat, ok := score.(float64); ok {
						segment.Score = scoreFloat
					}
				}
				if vote, ok := segmentMap["vote"]; ok {
					if voteInt, ok := vote.(int); ok {
						segment.Vote = voteInt
					}
				}
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
