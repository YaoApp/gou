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
