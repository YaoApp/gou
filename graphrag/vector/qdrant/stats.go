package qdrant

import (
	"context"
	"fmt"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/yaoapp/gou/graphrag/types"
)

// GetStats returns statistics about the collection
func (s *Store) GetStats(ctx context.Context, collectionName string) (*types.VectorStoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	if collectionName == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}

	// Get collection info from Qdrant
	info, err := s.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	// Get vectors config from the response
	vectorsConfig := info.Config.Params.VectorsConfig
	if vectorsConfig == nil {
		return nil, fmt.Errorf("invalid collection config: missing vectors config")
	}

	// Handle different vector config types
	var vectorParams *qdrant.VectorParams
	var indexSize int64

	switch v := vectorsConfig.Config.(type) {
	case *qdrant.VectorsConfig_Params:
		vectorParams = v.Params
	case *qdrant.VectorsConfig_ParamsMap:
		// For named vectors, use the first vector's params for dimension/distance
		if paramsMap := v.ParamsMap; paramsMap != nil && len(paramsMap.Map) > 0 {
			for _, params := range paramsMap.Map {
				vectorParams = params
				break
			}
		}
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
	indexType := types.IndexTypeHNSW // Qdrant uses HNSW by default

	// Handle PointsCount which might be nil
	var totalVectors int64
	if info.PointsCount != nil {
		totalVectors = int64(*info.PointsCount)
	}

	// Estimate index size based on total vectors and dimension
	if totalVectors > 0 {
		// Rough estimation: each vector takes (dimension * 4 bytes) + metadata overhead
		// HNSW index has additional overhead, so multiply by ~1.5
		bytesPerVector := int64(vectorParams.Size) * 4
		indexSize = totalVectors * bytesPerVector * 3 / 2 // 1.5x overhead for HNSW
	}

	// Calculate memory usage (assume index is loaded in memory)
	memoryUsage := indexSize

	// Build extra stats with additional information
	extraStats := make(map[string]interface{})
	if info.Status != qdrant.CollectionStatus_Green {
		extraStats["status"] = info.Status.String()
	}
	if info.Config != nil && info.Config.Params != nil {
		if info.Config.Params.ReplicationFactor != nil {
			extraStats["replication_factor"] = *info.Config.Params.ReplicationFactor
		}
		if info.Config.Params.WriteConsistencyFactor != nil {
			extraStats["write_consistency_factor"] = *info.Config.Params.WriteConsistencyFactor
		}

	}

	stats := &types.VectorStoreStats{
		TotalVectors:   totalVectors,
		Dimension:      int(vectorParams.Size),
		IndexType:      indexType,
		DistanceMetric: distance,
		IndexSize:      indexSize,
		MemoryUsage:    memoryUsage,
		ExtraStats:     extraStats,
	}

	return stats, nil
}

// GetSearchEngineStats returns search engine performance statistics
func (s *Store) GetSearchEngineStats(ctx context.Context, collectionName string) (*types.SearchEngineStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return nil, fmt.Errorf("not connected to Qdrant server")
	}

	if collectionName == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}

	// Get collection info to verify it exists and get basic stats
	info, err := s.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	// Get document count
	var documentCount int64
	if info.PointsCount != nil {
		documentCount = int64(*info.PointsCount)
	}

	// Calculate index size estimation
	var indexSize int64
	if vectorsConfig := info.Config.Params.VectorsConfig; vectorsConfig != nil {
		if vectorsConfig.Config != nil {
			switch v := vectorsConfig.Config.(type) {
			case *qdrant.VectorsConfig_Params:
				if v.Params != nil && documentCount > 0 {
					bytesPerVector := int64(v.Params.Size) * 4
					indexSize = documentCount * bytesPerVector * 3 / 2 // HNSW overhead
				}
			case *qdrant.VectorsConfig_ParamsMap:
				if paramsMap := v.ParamsMap; paramsMap != nil && len(paramsMap.Map) > 0 {
					for _, params := range paramsMap.Map {
						if params != nil && documentCount > 0 {
							bytesPerVector := int64(params.Size) * 4
							indexSize += documentCount * bytesPerVector * 3 / 2 // HNSW overhead
						}
					}
				}
			}
		}
	}

	// Note: Qdrant doesn't provide detailed search engine statistics
	// So we return basic stats with reasonable defaults/estimates
	stats := &types.SearchEngineStats{
		TotalQueries:     0,          // Not available from Qdrant
		AverageQueryTime: 10.0,       // Reasonable default estimate in ms
		CacheHitRate:     0.0,        // Not available from Qdrant
		PopularQueries:   []string{}, // Not available from Qdrant
		SlowQueries:      []string{}, // Not available from Qdrant
		ErrorRate:        0.0,        // Not available from Qdrant
		IndexSize:        indexSize,
		DocumentCount:    documentCount,
	}

	return stats, nil
}

// Optimize optimizes the collection
func (s *Store) Optimize(ctx context.Context, collectionName string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("not connected to Qdrant server")
	}

	if collectionName == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	// Check if collection exists first
	exists, err := s.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("collection %s does not exist", collectionName)
	}

	// In Qdrant, optimization is typically done through collection updates
	// We can perform several optimization operations:

	// 1. Update collection to optimize the index
	updateReq := &qdrant.UpdateCollection{
		CollectionName: collectionName,
		// Enable optimization parameters
		OptimizersConfig: &qdrant.OptimizersConfigDiff{
			// Default optimization settings
			DefaultSegmentNumber: qdrant.PtrOf(uint64(2)),
			MaxSegmentSize:       qdrant.PtrOf(uint64(200000)),
			MemmapThreshold:      qdrant.PtrOf(uint64(1000000)),
			IndexingThreshold:    qdrant.PtrOf(uint64(20000)),
			FlushIntervalSec:     qdrant.PtrOf(uint64(5)),
		},
	}

	// Apply the optimization configuration
	if err := s.client.UpdateCollection(ctx, updateReq); err != nil {
		return fmt.Errorf("failed to optimize collection: %w", err)
	}

	// Wait a short time for optimization to start
	time.Sleep(100 * time.Millisecond)

	return nil
}
