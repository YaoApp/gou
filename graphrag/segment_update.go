package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// ================================================================================================
// CRUD Operations - Update Segments
// ================================================================================================

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

	// Step 10: Extract entities and relationships from chunks and store to graph (if Graph is configured)
	var extractionResults []*types.ExtractionResult
	var entityDeduplicationResults map[string]*EntityDeduplicationResult
	var relationshipDeduplicationResults map[string]*RelationshipDeduplicationResult

	extractionResults, entityDeduplicationResults, relationshipDeduplicationResults, err = g.extractAndStoreEntitiesAndRelationships(ctx, chunks, opts, cb, &embeddingTexts, collectionIDs.Graph, docID)
	if err != nil {
		return 0, fmt.Errorf("failed to extract and store entities and relationships: %w", err)
	}

	// Step 11: Embed all texts (chunks + entities + relationships)
	embeddings, err := opts.Embedding.EmbedDocuments(ctx, embeddingTexts, cb.Embedding)
	if err != nil {
		return 0, fmt.Errorf("failed to embed the documents: %w", err)
	}

	// Step 13: Update all documents in vector store (chunks + entities + relationships)
	// Extract entities and relationships from extraction results for vector store
	var allEntities []types.Node
	var allRelationships []types.Relationship
	var entityIndexMap map[*types.Node]int = make(map[*types.Node]int)
	var relationshipIndexMap map[*types.Relationship]int = make(map[*types.Relationship]int)

	// Calculate the starting index for entities and relationships in embeddingTexts
	// embeddingTexts contains: [chunks, entities, relationships]
	chunksCount := len(chunks)
	entityStartIndex := chunksCount

	// Collect entities and relationships from extraction results
	entityIndex := entityStartIndex
	for _, extractionResult := range extractionResults {
		for i := range extractionResult.Nodes {
			entity := extractionResult.Nodes[i]
			allEntities = append(allEntities, entity)
			entityIndexMap[&extractionResult.Nodes[i]] = entityIndex
			entityIndex++
		}
	}

	relationshipStartIndex := entityIndex
	relationshipIndex := relationshipStartIndex
	for _, extractionResult := range extractionResults {
		for i := range extractionResult.Relationships {
			relationship := extractionResult.Relationships[i]
			allRelationships = append(allRelationships, relationship)
			relationshipIndexMap[&extractionResult.Relationships[i]] = relationshipIndex
			relationshipIndex++
		}
	}

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
