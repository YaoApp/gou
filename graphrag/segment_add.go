package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// ================================================================================================
// CRUD Operations - Add Segments
// ================================================================================================

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

	// Step 8: Extract entities and relationships from chunks and store to graph (equivalent to AddFile step 4.3)
	var extractionResults []*types.ExtractionResult
	var entityDeduplicationResults map[string]*EntityDeduplicationResult
	var relationshipDeduplicationResults map[string]*RelationshipDeduplicationResult

	extractionResults, entityDeduplicationResults, relationshipDeduplicationResults, err = g.extractAndStoreEntitiesAndRelationships(ctx, chunks, opts, cb, &embeddingTexts, collectionIDs.Graph, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract and store entities and relationships: %w", err)
	}

	// Step 9: Embed all texts (chunks + entities + relationships) (equivalent to AddFile step 5)
	embeddings, err := opts.Embedding.EmbedDocuments(ctx, embeddingTexts, cb.Embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to embed the documents: %w", err)
	}

	// Step 11: Store all documents to vector store (equivalent to AddFile step 7)
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
