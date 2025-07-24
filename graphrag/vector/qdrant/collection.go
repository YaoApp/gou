package qdrant

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
	"github.com/yaoapp/gou/graphrag/types"
)

// CreateCollection creates a new collection in Qdrant
func (s *Store) CreateCollection(ctx context.Context, opts *types.CreateCollectionOptions) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("not connected to Qdrant server")
	}

	// Convert distance metric
	var distance qdrant.Distance
	switch opts.Distance {
	case types.DistanceCosine:
		distance = qdrant.Distance_Cosine
	case types.DistanceEuclidean:
		distance = qdrant.Distance_Euclid
	case types.DistanceDot:
		distance = qdrant.Distance_Dot
	case types.DistanceManhattan:
		distance = qdrant.Distance_Manhattan
	default:
		distance = qdrant.Distance_Cosine
	}

	var vectorsConfig *qdrant.VectorsConfig
	var sparseVectorsConfig *qdrant.SparseVectorConfig

	if opts.EnableSparseVectors {
		// Create named vectors for hybrid search
		denseVectorName := opts.DenseVectorName
		if denseVectorName == "" {
			denseVectorName = "dense"
		}

		sparseVectorName := opts.SparseVectorName
		if sparseVectorName == "" {
			sparseVectorName = "sparse"
		}

		// Create dense vector configuration with named vectors
		denseVectorParams := &qdrant.VectorParams{
			Size:     uint64(opts.Dimension),
			Distance: distance,
		}

		namedVectors := map[string]*qdrant.VectorParams{
			denseVectorName: denseVectorParams,
		}

		vectorsConfig = qdrant.NewVectorsConfigMap(namedVectors)

		// Create sparse vector configuration
		sparseVectorParams := &qdrant.SparseVectorParams{}
		namedSparseVectors := map[string]*qdrant.SparseVectorParams{
			sparseVectorName: sparseVectorParams,
		}

		sparseVectorsConfig = qdrant.NewSparseVectorsConfig(namedSparseVectors)
	} else {
		// Create standard single vector configuration
		vectorParams := &qdrant.VectorParams{
			Size:     uint64(opts.Dimension),
			Distance: distance,
		}
		vectorsConfig = qdrant.NewVectorsConfig(vectorParams)
	}

	// Create HNSW config if specified
	var hnswConfig *qdrant.HnswConfigDiff
	if opts.IndexType == types.IndexTypeHNSW {
		hnswConfig = &qdrant.HnswConfigDiff{}
		if opts.M > 0 {
			hnswConfig.M = qdrant.PtrOf(uint64(opts.M))
		}
		if opts.EfConstruction > 0 {
			hnswConfig.EfConstruct = qdrant.PtrOf(uint64(opts.EfConstruction))
		}
	}

	// Create collection request
	req := &qdrant.CreateCollection{
		CollectionName:      opts.CollectionName,
		VectorsConfig:       vectorsConfig,
		SparseVectorsConfig: sparseVectorsConfig,
		HnswConfig:          hnswConfig,
	}

	// Execute create collection
	err := s.client.CreateCollection(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", opts.CollectionName, err)
	}

	return nil
}

// ListCollections returns a list of all collections
func (s *Store) ListCollections(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	collections, err := s.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	return collections, nil
}

// DropCollection deletes a collection
func (s *Store) DropCollection(ctx context.Context, collectionName string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("not connected to Qdrant server")
	}

	err := s.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to drop collection %s: %w", collectionName, err)
	}

	return nil
}

// CollectionExists checks if a collection exists
func (s *Store) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return false, fmt.Errorf("not connected to Qdrant server")
	}

	exists, err := s.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return false, fmt.Errorf("failed to check if collection exists: %w", err)
	}

	return exists, nil
}

// DescribeCollection returns statistics about a collection
func (s *Store) DescribeCollection(ctx context.Context, collectionName string) (*types.VectorStoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	info, err := s.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe collection %s: %w", collectionName, err)
	}

	// Get vectors config from the response
	vectorsConfig := info.Config.Params.VectorsConfig
	if vectorsConfig == nil {
		return nil, fmt.Errorf("invalid collection config: missing vectors config")
	}

	// Handle different vector config types
	var vectorParams *qdrant.VectorParams
	switch v := vectorsConfig.Config.(type) {
	case *qdrant.VectorsConfig_Params:
		vectorParams = v.Params
	default:
		return nil, fmt.Errorf("unsupported vector config type")
	}

	if vectorParams == nil {
		return nil, fmt.Errorf("invalid collection config: missing vector params")
	}

	// Convert distance metric
	var distance types.DistanceMetric
	switch vectorParams.Distance {
	case qdrant.Distance_Cosine:
		distance = types.DistanceCosine
	case qdrant.Distance_Euclid:
		distance = types.DistanceEuclidean
	case qdrant.Distance_Dot:
		distance = types.DistanceDot
	case qdrant.Distance_Manhattan:
		distance = types.DistanceManhattan
	default:
		distance = types.DistanceCosine
	}

	// Convert index type
	indexType := types.IndexTypeHNSW // Qdrant default

	// Handle PointsCount which might be nil
	var totalVectors int64
	if info.PointsCount != nil {
		totalVectors = int64(*info.PointsCount)
	}

	stats := &types.VectorStoreStats{
		TotalVectors:   totalVectors,
		Dimension:      int(vectorParams.Size),
		IndexType:      indexType,
		DistanceMetric: distance,
	}

	return stats, nil
}

// LoadCollection loads a collection (for databases that support it)
// For Qdrant, this is a no-op as collections are always loaded
func (s *Store) LoadCollection(ctx context.Context, collectionName string) error {
	// Qdrant collections are always loaded, so this is a no-op
	return nil
}

// ReleaseCollection releases a collection (for databases that support it)
// For Qdrant, this is a no-op as collections cannot be unloaded
func (s *Store) ReleaseCollection(ctx context.Context, collectionName string) error {
	// Qdrant collections cannot be unloaded, so this is a no-op
	return nil
}

// GetLoadState returns the load state of a collection
// For Qdrant, collections are always loaded if they exist
func (s *Store) GetLoadState(ctx context.Context, collectionName string) (types.LoadState, error) {
	exists, err := s.CollectionExists(ctx, collectionName)
	if err != nil {
		return types.LoadStateNotExist, err
	}

	if !exists {
		return types.LoadStateNotExist, nil
	}

	return types.LoadStateLoaded, nil
}
