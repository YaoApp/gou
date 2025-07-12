package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// AddSegments adds segments to a collection manually
func (g *GraphRag) AddSegments(ctx context.Context, docID string, segmentTexts []types.SegmentText, options *types.UpsertOptions) ([]string, error) {
	// Step 1: Parse GraphName from docID
	graphName, _ := utils.ExtractGraphNameFromDocID(docID)
	if graphName == "" {
		graphName = "default"
	}

	// Step 2: Prepare options by copying and setting necessary fields
	opts := &types.UpsertOptions{}
	if options != nil {
		*opts = *options
	}
	opts.GraphName = graphName
	opts.DocID = docID

	// Step 3: Get collection IDs for vector and graph storage
	collectionIDs, err := utils.GetCollectionIDs(graphName)
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

	// Step 6: Prepare embedding and extraction if not provided
	if opts.Embedding == nil {
		embedding, err := DetectEmbedding("")
		if err != nil {
			return nil, fmt.Errorf("failed to detect embedding: %w", err)
		}
		opts.Embedding = embedding
	}

	if g.Graph != nil && opts.Extraction == nil {
		extraction, err := DetectExtractor("")
		if err != nil {
			return nil, fmt.Errorf("failed to detect extraction: %w", err)
		}
		opts.Extraction = extraction
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
		CollectionID:                     graphName,
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

	docID, graphName, err := g.getDocIDFromExistingSegments(ctx, segmentIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to get doc_id from existing segments: %w", err)
	}

	// Step 3: Prepare options by copying and setting necessary fields
	opts := &types.UpsertOptions{}
	if options != nil {
		*opts = *options
	}
	opts.GraphName = graphName
	opts.DocID = docID

	// Step 4: Get collection IDs for vector and graph storage
	collectionIDs, err := utils.GetCollectionIDs(graphName)
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

	// Step 7: Prepare embedding and extraction if not provided
	if opts.Embedding == nil {
		embedding, err := DetectEmbedding("")
		if err != nil {
			return 0, fmt.Errorf("failed to detect embedding: %w", err)
		}
		opts.Embedding = embedding
	}

	if g.Graph != nil && opts.Extraction == nil {
		extraction, err := DetectExtractor("")
		if err != nil {
			return 0, fmt.Errorf("failed to detect extraction: %w", err)
		}
		opts.Extraction = extraction
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
		CollectionID:                     graphName,
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
	// TODO: Implement RemoveSegments
	return 0, nil
}

// RemoveSegmentsByDocID removes all segments of a document
func (g *GraphRag) RemoveSegmentsByDocID(ctx context.Context, docID string) (int, error) {
	// TODO: Implement RemoveSegmentsByDocID
	return 0, nil
}

// GetSegments gets all segments of a collection
func (g *GraphRag) GetSegments(ctx context.Context, docID string) ([]types.Segment, error) {
	// TODO: Implement GetSegments
	return nil, nil
}

// GetSegment gets a single segment by ID
func (g *GraphRag) GetSegment(ctx context.Context, segmentID string) (*types.Segment, error) {
	// TODO: Implement GetSegment
	return nil, nil
}

// convertSegmentTextsToChunks converts SegmentTexts to Chunks for processing
func (g *GraphRag) convertSegmentTextsToChunks(segmentTexts []types.SegmentText, docID string) ([]*types.Chunk, error) {
	chunks := make([]*types.Chunk, 0, len(segmentTexts))

	for i, segmentText := range segmentTexts {
		chunkID := segmentText.ID
		if chunkID == "" {
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

// storeSegmentMetadataToStore stores segment metadata (Weight, Score, Vote) to Store
func (g *GraphRag) storeSegmentMetadataToStore(ctx context.Context, docID string, chunks []*types.Chunk, storeCollectionName string) error {
	if g.Store == nil {
		return nil
	}

	// Store default metadata for each segment chunk
	for _, chunk := range chunks {
		segmentID := chunk.ID

		// Store Weight
		weightKey := fmt.Sprintf("segment_weight_%s_%s", docID, segmentID)
		err := g.Store.Set(weightKey, 0.0, 0) // Default weight: 0.0
		if err != nil {
			g.Logger.Warnf("Failed to store weight for segment %s: %v", segmentID, err)
		}

		// Store Score
		scoreKey := fmt.Sprintf("segment_score_%s_%s", docID, segmentID)
		err = g.Store.Set(scoreKey, 0.0, 0) // Default score: 0.0
		if err != nil {
			g.Logger.Warnf("Failed to store score for segment %s: %v", segmentID, err)
		}

		// Store Vote
		voteKey := fmt.Sprintf("segment_vote_%s_%s", docID, segmentID)
		err = g.Store.Set(voteKey, 0, 0) // Default vote: 0
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
	hasVote := false
	hasWeight := false
	hasScore := false

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
			voteKey := fmt.Sprintf("segment_vote_%s_%s", docID, segmentID)
			err := g.Store.Set(voteKey, vote, 0)
			if err != nil {
				g.Logger.Warnf("Failed to update vote for segment %s: %v", segmentID, err)
			}
		}

		// Update Weight if provided
		if hasWeight {
			weightKey := fmt.Sprintf("segment_weight_%s_%s", docID, segmentID)
			err := g.Store.Set(weightKey, weight, 0)
			if err != nil {
				g.Logger.Warnf("Failed to update weight for segment %s: %v", segmentID, err)
			}
		}

		// Update Score if provided
		if hasScore {
			scoreKey := fmt.Sprintf("segment_score_%s_%s", docID, segmentID)
			err := g.Store.Set(scoreKey, score, 0)
			if err != nil {
				g.Logger.Warnf("Failed to update score for segment %s: %v", segmentID, err)
			}
		}
	}

	return nil
}
