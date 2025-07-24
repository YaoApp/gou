package graphrag

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// saveCollectionMetadata saves collection metadata with priority: Store > System Collection
func (g *GraphRag) saveCollectionMetadata(ctx context.Context, collection types.CollectionConfig) error {
	// Serialize collection to JSON
	serializedData, err := types.SerializeCollectionConfig(collection)
	if err != nil {
		return fmt.Errorf("failed to serialize collection metadata: %w", err)
	}

	// Priority 1: Store if available
	if g.Store != nil {
		err = g.Store.Set(collection.ID, serializedData, 0) // ttl = 0 means no expiration
		if err != nil {
			return fmt.Errorf("failed to save collection metadata to Store: %w", err)
		}
		g.Logger.Infof("Saved collection metadata to Store: %s", collection.ID)
		return nil
	}

	// Priority 2: System Collection as fallback
	if g.Vector != nil {
		// Create dummy vector for metadata storage (512 dimensions as configured in ensureSystemCollection)
		dummyVector := make([]float64, 512)
		for i := range dummyVector {
			dummyVector[i] = 0.0 // Use zero vector for metadata storage
		}

		// Create document for system collection
		document := &types.Document{
			ID:      collection.ID,
			Content: serializedData,
			Vector:  dummyVector, // Provide dummy vector for Qdrant
			Metadata: map[string]interface{}{
				"type":       "collection_metadata",
				"created_at": collection.ID, // Use collection ID as reference
			},
		}

		opts := &types.AddDocumentOptions{
			CollectionName: g.System,
			Documents:      []*types.Document{document},
			Upsert:         true, // Allow overwrite
		}

		_, err = g.Vector.AddDocuments(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to save collection metadata to System Collection: %w", err)
		}
		g.Logger.Infof("Saved collection metadata to System Collection: %s", collection.ID)
		return nil
	}

	return fmt.Errorf("no storage backend available for collection metadata")
}

// CreateCollection creates a new collection
func (g *GraphRag) CreateCollection(ctx context.Context, collection types.CollectionConfig) (string, error) {
	// Connect to vector store if not already connected and config is provided
	if g.Vector != nil && !g.Vector.IsConnected() {
		err := g.Vector.Connect(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to connect to vector store: %w", err)
		}
		g.Logger.Infof("Connected to vector store")
	}

	// Connect to graph store if not already connected and config is provided
	if g.Graph != nil && !g.Graph.IsConnected() {
		err := g.Graph.Connect(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to connect to graph store: %w", err)
		}
		g.Logger.Infof("Connected to graph store")
	}

	// Check if collection exists, if exists, return error
	if collection.ID != "" {
		exists, err := g.CollectionExists(ctx, collection.ID)
		if err != nil {
			return "", fmt.Errorf("failed to check collection existence: %w", err)
		}
		if exists {
			return "", fmt.Errorf("collection with ID '%s' already exists", collection.ID)
		}
	}

	// Check if system collection exists, if not, create it
	err := g.ensureSystemCollection(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to ensure system collection: %w", err)
	}

	// If collection.ID is not empty, use it, otherwise generate a new one
	collectionID := collection.ID
	if collectionID == "" {
		collectionID = utils.GenDocID()
	}

	// Get IDs using GetCollectionIDs at utils/generate.go
	ids, err := utils.GetCollectionIDs(collectionID)
	if err != nil {
		return "", fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	// Set collection ID for validation
	collection.ID = collectionID

	// Set CollectionName in VectorConfig if it exists
	if collection.Config != nil {
		collection.Config.CollectionName = ids.Vector
	}

	// Validate collection after setting all necessary fields
	if err := types.ValidateCollectionConfig(collection); err != nil {
		return "", fmt.Errorf("invalid collection: %w", err)
	}

	// Create vector collection, if error, return error
	var vectorCreated bool
	if collection.Config != nil {
		err = g.Vector.CreateCollection(ctx, collection.Config)
		if err != nil {
			return "", fmt.Errorf("failed to create vector collection: %w", err)
		}
		vectorCreated = true
	}

	// if graph is not nil and connected, create graph, if error, return error and rollback
	var graphCreated bool
	if g.Graph != nil && g.Graph.IsConnected() {
		err = g.Graph.CreateGraph(ctx, ids.Graph)
		if err != nil {
			// Rollback vector collection if it was created
			if vectorCreated {
				_ = g.Vector.DropCollection(ctx, ids.Vector)
			}
			return "", fmt.Errorf("failed to create graph: %w", err)
		}
		graphCreated = true
	}

	// save collection metadata with priority: Store > System Collection
	err = g.saveCollectionMetadata(ctx, collection)
	if err != nil {
		// Rollback vector collection and graph if they were created
		if vectorCreated {
			_ = g.Vector.DropCollection(ctx, ids.Vector)
		}
		if graphCreated {
			_ = g.Graph.DropGraph(ctx, ids.Graph)
		}
		return "", fmt.Errorf("failed to save collection metadata: %w", err)
	}

	// Return collection.ID
	return collectionID, nil
}

// RemoveCollection removes a collection
func (g *GraphRag) RemoveCollection(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("collection ID cannot be empty")
	}

	// Connect to vector store if not already connected and config is provided
	if g.Vector != nil && !g.Vector.IsConnected() {
		err := g.Vector.Connect(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to connect to vector store: %w", err)
		}
		g.Logger.Infof("Connected to vector store")
	}

	// Connect to graph store if not already connected and config is provided
	if g.Graph != nil && !g.Graph.IsConnected() {
		err := g.Graph.Connect(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to connect to graph store: %w", err)
		}
		g.Logger.Infof("Connected to graph store")
	}

	// Check if collection exists
	exists, err := g.CollectionExists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		return false, nil // Collection doesn't exist, return false (nothing deleted)
	}

	// Get IDs for vector, graph, and store components
	ids, err := utils.GetCollectionIDs(id)
	if err != nil {
		return false, fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	var errors []string
	removed := false

	// Remove vector collection if it exists
	if g.Vector != nil {
		vectorExists, err := g.Vector.CollectionExists(ctx, ids.Vector)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to check vector collection existence: %v", err))
		} else if vectorExists {
			err = g.Vector.DropCollection(ctx, ids.Vector)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to drop vector collection: %v", err))
			} else {
				g.Logger.Infof("Dropped vector collection: %s", ids.Vector)
			}
		}
	}

	// Remove graph if it exists and graph store is connected
	if g.Graph != nil && g.Graph.IsConnected() {
		graphExists, err := g.Graph.GraphExists(ctx, ids.Graph)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to check graph existence: %v", err))
		} else if graphExists {
			err = g.Graph.DropGraph(ctx, ids.Graph)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to drop graph: %v", err))
			} else {
				g.Logger.Infof("Dropped graph: %s", ids.Graph)
			}
		}
	}

	// Remove collection metadata (Store has priority)
	metadataRemoved := false
	if g.Store != nil {
		err = g.Store.Del(id)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete collection metadata from Store: %v", err))
		} else {
			g.Logger.Infof("Removed collection metadata from Store: %s", id)
			metadataRemoved = true
		}
	} else if g.Vector != nil {
		// Try to remove from System Collection
		opts := &types.DeleteDocumentOptions{
			CollectionName: g.System,
			IDs:            []string{id},
		}
		err = g.Vector.DeleteDocuments(ctx, opts)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete collection metadata from System Collection: %v", err))
		} else {
			g.Logger.Infof("Removed collection metadata from System Collection: %s", id)
			metadataRemoved = true
		}
	}

	// If there were any errors, return them but still indicate success if metadata was removed
	if len(errors) > 0 {
		g.Logger.Warnf("Some errors occurred while removing collection %s: %v", id, errors)
		// If we successfully removed the metadata, consider it a success
		if metadataRemoved {
			removed = true
		}
		return removed, fmt.Errorf("partial removal completed with errors: %v", errors)
	}

	// Successfully removed
	removed = metadataRemoved
	if removed {
		g.Logger.Infof("Successfully removed collection: %s", id)
	}
	return removed, nil
}

// CollectionExists checks if a collection exists
func (g *GraphRag) CollectionExists(ctx context.Context, id string) (bool, error) {
	// Check in Store first if available
	if g.Store != nil {
		return g.Store.Has(id), nil
	}

	// Connect to vector store if not already connected and config is provided
	if g.Vector != nil && !g.Vector.IsConnected() {
		err := g.Vector.Connect(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to connect to vector store: %w", err)
		}
		g.Logger.Infof("Connected to vector store")
	}

	// Check in System Collection if Store is not available
	if g.Vector != nil {
		exists, err := g.Vector.CollectionExists(ctx, g.System)
		if err != nil {
			return false, fmt.Errorf("failed to check System Collection existence: %w", err)
		}
		if exists {
			opts := &types.GetDocumentOptions{
				CollectionName: g.System,
				IncludeVector:  false,
				IncludePayload: false,
			}

			docs, err := g.Vector.GetDocuments(ctx, []string{id}, opts)
			if err != nil {
				return false, fmt.Errorf("failed to check collection in System Collection: %w", err)
			}
			return len(docs) > 0 && docs[0] != nil, nil
		}
	}

	// Fallback: check in vector store using generated IDs
	ids, err := utils.GetCollectionIDs(id)
	if err != nil {
		return false, fmt.Errorf("failed to generate collection IDs: %w", err)
	}

	exists, err := g.Vector.CollectionExists(ctx, ids.Vector)
	if err != nil {
		return false, fmt.Errorf("failed to check vector collection existence: %w", err)
	}

	return exists, nil
}

// GetCollections gets all collections with optional metadata filtering
func (g *GraphRag) GetCollections(ctx context.Context, filter map[string]interface{}) ([]types.CollectionInfo, error) {
	if g.Vector == nil {
		return nil, fmt.Errorf("vector store is required for collection management")
	}

	// Connect to vector store if not already connected and config is provided
	if g.Vector != nil && !g.Vector.IsConnected() {
		err := g.Vector.Connect(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to vector store: %w", err)
		}
		g.Logger.Infof("Connected to vector store")
	}

	// Step 1: Get all vector collections to ensure data consistency
	vectorCollections, err := g.Vector.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list vector collections: %w", err)
	}

	// Extract actual collection IDs from vector collection names
	var collectionIDs []string
	for _, vectorCollectionName := range vectorCollections {
		// Skip system collection
		if vectorCollectionName == g.System {
			continue
		}

		// Extract collection ID from vector collection name
		collectionID := utils.ExtractCollectionIDFromVectorName(vectorCollectionName)
		if collectionID != "" {
			collectionIDs = append(collectionIDs, collectionID)
		}
	}

	// Step 2: Get collection metadata based on configuration
	var collections []types.CollectionInfo
	if g.Store != nil {
		// Store-based approach
		collections, err = g.getCollectionsFromStore(ctx, collectionIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get collections from store: %w", err)
		}
	} else {
		// Vector-based approach (using system collection)
		collections, err = g.getCollectionsFromSystemCollection(ctx, collectionIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get collections from system collection: %w", err)
		}
	}

	// Step 3: Apply metadata filtering
	var filteredCollections []types.CollectionInfo
	for _, collection := range collections {
		if g.matchesFilter(collection, filter) {
			filteredCollections = append(filteredCollections, types.CollectionInfo{
				ID:       collection.ID,
				Metadata: collection.Metadata,
				Config:   collection.Config,
			})
		}
	}

	g.Logger.Infof("Retrieved %d collections (total: %d, filter applied: %v)",
		len(filteredCollections), len(collections), len(filter) > 0)

	return filteredCollections, nil
}

// getCollectionsFromStore retrieves collection metadata from Store
func (g *GraphRag) getCollectionsFromStore(_ context.Context, collectionIDs []string) ([]types.CollectionInfo, error) {
	var collections []types.CollectionInfo

	for _, collectionID := range collectionIDs {
		if g.Store.Has(collectionID) {
			data, ok := g.Store.Get(collectionID)
			if !ok {
				g.Logger.Warnf("Failed to get collection data from store for ID: %s", collectionID)
				continue
			}

			serializedData, ok := data.(string)
			if !ok {
				g.Logger.Warnf("Invalid data type in store for collection %s", collectionID)
				continue
			}

			collection, err := types.DeserializeCollectionConfig(serializedData)
			if err != nil {
				g.Logger.Warnf("Failed to deserialize collection from store for ID %s: %v", collectionID, err)
				continue
			}

			collections = append(collections, types.CollectionInfo{
				ID:       collection.ID,
				Metadata: collection.Metadata,
				Config:   collection.Config,
			})
		} else {
			// Collection exists in vector store but not in metadata store
			// Create minimal collection with inferred metadata
			collection := types.CollectionInfo{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"inferred": true,
					"source":   "vector_collection",
				},
				Config: nil, // No config available for inferred collections
			}
			collections = append(collections, collection)
		}
	}

	return collections, nil
}

// getCollectionsFromSystemCollection retrieves collection metadata from System Collection
func (g *GraphRag) getCollectionsFromSystemCollection(ctx context.Context, collectionIDs []string) ([]types.CollectionInfo, error) {
	var collections []types.CollectionInfo

	// Check if system collection exists
	exists, err := g.Vector.CollectionExists(ctx, g.System)
	if err != nil {
		return nil, fmt.Errorf("failed to check system collection existence: %w", err)
	}

	if !exists {
		// System collection doesn't exist, create minimal collections
		for _, collectionID := range collectionIDs {
			collection := types.Collection{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"inferred": true,
					"source":   "vector_collection",
				},
			}
			collections = append(collections, types.CollectionInfo{
				ID:       collection.ID,
				Metadata: collection.Metadata,
			})
		}
		return collections, nil
	}

	// Get collection metadata from system collection
	for _, collectionID := range collectionIDs {
		opts := &types.GetDocumentOptions{
			CollectionName: g.System,
			IncludeVector:  false,
			IncludePayload: true,
		}

		docs, err := g.Vector.GetDocuments(ctx, []string{collectionID}, opts)
		if err != nil {
			g.Logger.Warnf("Failed to get collection metadata from system collection for ID %s: %v", collectionID, err)
			continue
		}

		if len(docs) > 0 && docs[0] != nil && docs[0].Content != "" {
			collection, err := types.DeserializeCollectionConfig(docs[0].Content)
			if err != nil {
				g.Logger.Warnf("Failed to deserialize collection from system collection for ID %s: %v", collectionID, err)
				continue
			}
			collections = append(collections, types.CollectionInfo{
				ID:       collection.ID,
				Metadata: collection.Metadata,
				Config:   collection.Config,
			})
		} else {
			// Collection exists in vector store but not in system collection
			// Create minimal collection with inferred metadata
			collection := types.CollectionInfo{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"inferred": true,
					"source":   "vector_collection",
				},
				Config: nil, // No config available for inferred collections
			}
			collections = append(collections, collection)
		}
	}

	return collections, nil
}

// matchesFilter checks if a collection matches the given metadata filter
func (g *GraphRag) matchesFilter(collection types.CollectionInfo, filter map[string]interface{}) bool {
	// If no filter provided, match all
	if len(filter) == 0 {
		return true
	}

	// Check each filter condition
	for key, expectedValue := range filter {
		if collection.Metadata == nil {
			return false
		}

		actualValue, exists := collection.Metadata[key]
		if !exists {
			return false
		}

		// Simple equality check
		if actualValue != expectedValue {
			return false
		}
	}

	return true
}

// ensureSystemCollection ensures the system collection exists and creates it if not
func (g *GraphRag) ensureSystemCollection(ctx context.Context) error {
	if g.Vector == nil {
		return fmt.Errorf("vector store is required for system collection")
	}

	// Check if system collection exists
	exists, err := g.Vector.CollectionExists(ctx, g.System)
	if err != nil {
		return fmt.Errorf("failed to check system collection existence: %w", err)
	}

	// Create system collection if not exists
	if !exists {
		systemCollectionConfig := &types.CreateCollectionOptions{
			CollectionName: g.System,
			Dimension:      512, // Default dimension for metadata storage
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
		}

		err = g.Vector.CreateCollection(ctx, systemCollectionConfig)
		if err != nil {
			return fmt.Errorf("failed to create system collection: %w", err)
		}

		g.Logger.Infof("Created system collection: %s", g.System)
	}

	return nil
}
