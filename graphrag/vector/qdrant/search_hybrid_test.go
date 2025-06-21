package qdrant

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestSearchHybrid_BasicFunctionality tests basic hybrid search functionality
func TestSearchHybrid_BasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	tests := []struct {
		name        string
		opts        *types.HybridSearchOptions
		expectError bool
		minResults  int
		maxResults  int
	}{
		{
			name: "Vector-only search",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				IncludeContent: true,
				VectorWeight:   1.0,
				VectorUsing:    "dense",
			},
			expectError: false,
			minResults:  1,
			maxResults:  5,
		},
		{
			name: "Sparse vector search",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QuerySparse: &types.SparseVector{
					Indices: []uint32{1, 42, 100},
					Values:  []float32{0.22, 0.8, 0.5},
				},
				K:              5,
				IncludeContent: true,
				KeywordWeight:  1.0,
				SparseUsing:    "sparse",
			},
			expectError: false,
			minResults:  0, // Sparse vectors might not match any documents in test data
			maxResults:  5,
		},
		{
			name: "Hybrid search with RRF fusion",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse: &types.SparseVector{
					Indices: []uint32{1, 42, 100, 200},
					Values:  []float32{0.22, 0.8, 0.5, 0.3},
				},
				K:              5,
				FusionType:     types.FusionRRF,
				IncludeContent: true,
				VectorWeight:   0.7,
				KeywordWeight:  0.3,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError: false,
			minResults:  0,
			maxResults:  5,
		},
		{
			name: "Search with metadata filter",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				Filter: map[string]interface{}{
					"mapping_is_leaf": true,
				},
				IncludeContent: true,
				VectorWeight:   1.0,
				VectorUsing:    "dense",
			},
			expectError: false,
			minResults:  0,
			maxResults:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()
			result, err := env.Store.SearchHybrid(ctx, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			// Validate result structure
			if len(result.Documents) < tt.minResults {
				t.Errorf("Expected at least %d results, got %d", tt.minResults, len(result.Documents))
			}

			if len(result.Documents) > tt.maxResults {
				t.Errorf("Expected at most %d results, got %d", tt.maxResults, len(result.Documents))
			}

			// Validate query time
			if result.QueryTime < 0 {
				t.Errorf("Query time should be non-negative, got %d", result.QueryTime)
			}

			queryDuration := time.Since(startTime).Milliseconds()
			if result.QueryTime > queryDuration+100 { // Allow 100ms tolerance
				t.Errorf("Query time %d seems too high compared to actual duration %d", result.QueryTime, queryDuration)
			}

			// Validate score consistency
			if len(result.Documents) > 0 {
				if result.MaxScore <= 0 {
					t.Errorf("MaxScore should be positive when results exist, got %f", result.MaxScore)
				}
				if result.MinScore < 0 {
					t.Errorf("MinScore should be non-negative, got %f", result.MinScore)
				}
				if result.MaxScore < result.MinScore {
					t.Errorf("MaxScore (%f) should be >= MinScore (%f)", result.MaxScore, result.MinScore)
				}

				// Check score ordering (should be descending)
				for i := 1; i < len(result.Documents); i++ {
					if result.Documents[i-1].Score < result.Documents[i].Score {
						t.Errorf("Results should be ordered by score (descending), but result[%d].Score=%f < result[%d].Score=%f",
							i-1, result.Documents[i-1].Score, i, result.Documents[i].Score)
					}
				}
			}

			// Validate document structure
			for i, doc := range result.Documents {
				if doc.Document.ID == "" {
					t.Errorf("Document %d has empty ID", i)
				}

				if tt.opts.IncludeContent && doc.Document.Content == "" {
					t.Logf("Warning: Document %d has empty content (ID: %s)", i, doc.Document.ID)
				}

				if tt.opts.IncludeMetadata && doc.Document.Metadata == nil {
					t.Logf("Warning: Document %d has nil metadata (ID: %s)", i, doc.Document.ID)
				}

				if tt.opts.IncludeVector && len(doc.Document.Vector) == 0 {
					t.Logf("Warning: Document %d has empty vector (ID: %s)", i, doc.Document.ID)
				}
			}
		})
	}
}

// TestSearchHybrid_Pagination tests pagination functionality
func TestSearchHybrid_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	tests := []struct {
		name     string
		page     int
		pageSize int
		total    bool
	}{
		{
			name:     "Page 1 with size 3",
			page:     1,
			pageSize: 3,
			total:    true,
		},
		{
			name:     "Page 2 with size 2",
			page:     2,
			pageSize: 2,
			total:    false,
		},
		{
			name:     "Page beyond available data",
			page:     100,
			pageSize: 5,
			total:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				Page:           tt.page,
				PageSize:       tt.pageSize,
				IncludeTotal:   tt.total,
				IncludeContent: true,
				VectorWeight:   1.0,
				VectorUsing:    "dense",
			}

			result, err := env.Store.SearchHybrid(ctx, opts)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Validate pagination metadata
			if result.Page != tt.page {
				t.Errorf("Expected page %d, got %d", tt.page, result.Page)
			}

			if result.PageSize != tt.pageSize {
				t.Errorf("Expected page size %d, got %d", tt.pageSize, result.PageSize)
			}

			// Check document count
			if len(result.Documents) > tt.pageSize {
				t.Errorf("Expected at most %d documents, got %d", tt.pageSize, len(result.Documents))
			}

			// Check pagination flags
			if tt.page == 1 && result.HasPrevious {
				t.Errorf("Page 1 should not have previous page")
			}

			if tt.page > 1 && !result.HasPrevious {
				t.Errorf("Page %d should have previous page", tt.page)
			}

			if result.HasNext && result.NextPage != tt.page+1 {
				t.Errorf("Next page should be %d, got %d", tt.page+1, result.NextPage)
			}

			if result.HasPrevious && result.PreviousPage != tt.page-1 {
				t.Errorf("Previous page should be %d, got %d", tt.page-1, result.PreviousPage)
			}

			t.Logf("Pagination test '%s' returned %d documents", tt.name, len(result.Documents))
		})
	}
}

// TestSearchHybrid_WeightCombinations tests different fusion algorithms
func TestSearchHybrid_WeightCombinations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	// Create a test sparse vector
	testSparseVector := &types.SparseVector{
		Indices: []uint32{1, 42, 100, 200, 300},
		Values:  []float32{0.22, 0.8, 0.5, 0.3, 0.6},
	}

	tests := []struct {
		name          string
		opts          *types.HybridSearchOptions
		expectError   bool
		validateScore bool
	}{
		{
			name: "RRF Fusion (Reciprocal Rank Fusion)",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse:    testSparseVector,
				K:              5,
				FusionType:     types.FusionRRF,
				IncludeContent: true,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError:   false,
			validateScore: true,
		},
		{
			name: "DBSF Fusion (Distribution-Based Score Fusion)",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse:    testSparseVector,
				K:              5,
				FusionType:     types.FusionDBSF,
				IncludeContent: true,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError:   false,
			validateScore: true,
		},
		{
			name: "Vector only with named vector",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Use dense vector
				IncludeContent: true,
			},
			expectError:   false,
			validateScore: true,
		},
		{
			name: "Sparse only with named vector",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QuerySparse:    testSparseVector,
				K:              5,
				SparseUsing:    "sparse", // Use sparse vector
				IncludeContent: true,
			},
			expectError:   false, // May return no results but shouldn't error
			validateScore: false,
		},
		{
			name: "Legacy weight support (should use RRF)",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse:    testSparseVector,
				K:              5,
				VectorWeight:   0.7, // Legacy weights should trigger RRF
				KeywordWeight:  0.3,
				IncludeContent: true,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError:   false,
			validateScore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchHybrid(ctx, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			t.Logf("Test '%s' returned %d results", tt.name, len(result.Documents))

			if tt.validateScore && len(result.Documents) > 0 {
				// Validate basic score properties
				if result.MaxScore <= 0 {
					t.Errorf("MaxScore should be positive when results exist, got %f", result.MaxScore)
				}

				// Check that all document scores are within the valid range
				for i, doc := range result.Documents {
					if doc.Score < 0 {
						t.Errorf("Document %d has negative score: %f", i, doc.Score)
					}
				}

				// Verify score ordering
				for i := 1; i < len(result.Documents); i++ {
					if result.Documents[i-1].Score < result.Documents[i].Score {
						t.Errorf("Results should be ordered by score (descending)")
						break
					}
				}
			}

			// Validate query time
			if result.QueryTime < 0 {
				t.Errorf("Query time should be non-negative, got %d", result.QueryTime)
			}
		})
	}
}

// TestSearchHybrid_ErrorScenarios tests error handling
func TestSearchHybrid_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	ctx := context.Background()

	tests := []struct {
		name string
		opts *types.HybridSearchOptions
	}{
		{
			name: "Nil options",
			opts: nil,
		},
		{
			name: "Empty collection name",
			opts: &types.HybridSearchOptions{
				CollectionName: "",
				QueryVector:    []float64{0.1, 0.2, 0.3},
				K:              5,
			},
		},
		{
			name: "No query provided",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				K:              5,
			},
		},
		{
			name: "Neither vector nor sparse query provided",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				K:              5,
			},
		},
		{
			name: "Nonexistent collection",
			opts: &types.HybridSearchOptions{
				CollectionName: "nonexistent_collection_12345",
				QueryVector:    []float64{0.1, 0.2, 0.3},
				K:              5,
			},
		},
		{
			name: "Invalid sparse vector (empty indices)",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QuerySparse: &types.SparseVector{
					Indices: []uint32{},
					Values:  []float32{0.5, 0.3},
				},
				K: 5,
			},
		},
		{
			name: "Invalid sparse vector (mismatched indices and values)",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QuerySparse: &types.SparseVector{
					Indices: []uint32{1, 2, 3},
					Values:  []float32{0.5}, // Only one value for 3 indices
				},
				K: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchHybrid(ctx, tt.opts)

			// All these scenarios should result in an error
			if err == nil {
				t.Errorf("Expected error for %s, but got none. Result: %+v", tt.name, result)
			}

			// Result should be nil when there's an error
			if result != nil {
				t.Errorf("Expected nil result for %s when error occurs, got: %+v", tt.name, result)
			}

			t.Logf("Test '%s' correctly returned error: %v", tt.name, err)
		})
	}
}

// TestSearchHybrid_Timeout tests timeout functionality
func TestSearchHybrid_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	tests := []struct {
		name        string
		timeout     int
		expectError bool
	}{
		{
			name:        "Normal timeout",
			timeout:     5000, // 5 seconds
			expectError: false,
		},
		{
			name:        "Very short timeout",
			timeout:     1,    // 1 millisecond
			expectError: true, // May timeout
		},
		{
			name:        "No timeout specified",
			timeout:     0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				Timeout:        tt.timeout,
				IncludeContent: true,
				VectorWeight:   1.0,
				VectorUsing:    "dense",
			}

			result, err := env.Store.SearchHybrid(ctx, opts)

			if tt.expectError {
				// For very short timeout, we might get either a timeout error or success
				// depending on system performance, so we'll just log the result
				var resultCount int
				var resultStr string
				if err != nil {
					resultCount = -1
					resultStr = err.Error()
				} else {
					resultCount = len(result.Documents)
					resultStr = "success"
				}
				t.Logf("Short timeout test result: results=%d, status=%s", resultCount, resultStr)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			t.Logf("Timeout test '%s' returned %d results", tt.name, len(result.Documents))
		})
	}
}

// TestSearchHybrid_Unconnected tests behavior with unconnected store
func TestSearchHybrid_Unconnected(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create an unconnected store
	store := &Store{connected: false}
	ctx := context.Background()

	opts := &types.HybridSearchOptions{
		CollectionName: "test_collection",
		QueryVector:    []float64{0.1, 0.2, 0.3},
		K:              5,
	}

	result, err := store.SearchHybrid(ctx, opts)

	// Should return an error
	if err == nil {
		t.Errorf("Expected error for unconnected store, but got none. Result: %+v", result)
	}

	// Result should be nil
	if result != nil {
		t.Errorf("Expected nil result for unconnected store, got: %+v", result)
	}

	// Check error message
	if err != nil && !strings.Contains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}

	t.Logf("Unconnected store test correctly returned error: %v", err)
}

// TestSearchHybrid_MemoryLeak tests for memory leaks during repeated operations
func TestSearchHybrid_MemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	// Force garbage collection and get initial memory stats
	runtime.GC()
	runtime.GC()
	var initialStats, finalStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)

	// Perform many search operations
	const numOperations = 1500
	successCount := 0

	opts := &types.HybridSearchOptions{
		CollectionName: testDataSet.CollectionName,
		QueryVector:    queryVector,
		K:              5,
		IncludeContent: true,
		VectorWeight:   1.0,
		VectorUsing:    "dense",
	}

	for i := 0; i < numOperations; i++ {
		result, err := env.Store.SearchHybrid(ctx, opts)
		if err == nil && result != nil {
			successCount++
		} else {
			t.Logf("Search %d failed: %v", i+1, err)
		}
	}

	// Force garbage collection and get final memory stats
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&finalStats)

	// Calculate memory growth
	heapGrowth := int64(finalStats.HeapInuse) - int64(initialStats.HeapInuse)

	t.Logf("Performed %d search operations (%d successful)", numOperations, successCount)
	t.Logf("Initial heap: %d bytes", initialStats.HeapInuse)
	t.Logf("Final heap: %d bytes", finalStats.HeapInuse)
	t.Logf("Memory growth: %d bytes (%.2f MB)", heapGrowth, float64(heapGrowth)/(1024*1024))
	t.Logf("GC runs: %d", finalStats.NumGC-initialStats.NumGC)

	// Memory growth should be reasonable (less than 10MB for 1500 operations)
	const maxMemoryGrowthMB = 10
	const maxMemoryGrowthBytes = maxMemoryGrowthMB * 1024 * 1024

	if heapGrowth > maxMemoryGrowthBytes {
		t.Errorf("Memory growth of %.2f MB exceeds maximum allowed %d MB",
			float64(heapGrowth)/(1024*1024), maxMemoryGrowthMB)
	}

	if successCount == 0 {
		t.Errorf("No successful search operations out of %d attempts", numOperations)
	}

	successRate := float64(successCount) / float64(numOperations) * 100
	t.Logf("Success rate: %.1f%%", successRate)
}

// TestSearchHybrid_ConcurrentStress tests concurrent access
func TestSearchHybrid_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	// Test parameters
	const numGoroutines = 15
	const operationsPerGoroutine = 25
	const totalOperations = numGoroutines * operationsPerGoroutine

	// Results tracking
	var (
		successCount int64
		errorCount   int64
		errors       []string
		mu           sync.Mutex
	)

	// Create wait group for goroutines
	var wg sync.WaitGroup

	startTime := time.Now()

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				opts := &types.HybridSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              5,
					IncludeContent: true,
					VectorWeight:   1.0,
					VectorUsing:    "dense",
				}

				result, err := env.Store.SearchHybrid(ctx, opts)

				mu.Lock()
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					if len(errors) < 10 { // Keep first 10 errors
						errors = append(errors, err.Error())
					}
				} else if result != nil {
					atomic.AddInt64(&successCount, 1)
				}
				mu.Unlock()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	duration := time.Since(startTime)

	// Calculate statistics
	successRate := float64(successCount) / float64(totalOperations) * 100
	errorRate := float64(errorCount) / float64(totalOperations) * 100
	operationsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("Concurrent stress test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Successful operations: %d (%.1f%%)", successCount, successRate)
	t.Logf("  Failed operations: %d (%.1f%%)", errorCount, errorRate)
	t.Logf("  Operations per second: %.1f", operationsPerSecond)

	// Log first few errors for debugging
	if len(errors) > 0 {
		t.Logf("First few errors:")
		for i, err := range errors {
			if i >= 5 { // Limit to first 5 errors in log
				break
			}
			t.Logf("  Error %d: %s", i+1, err)
		}
	}

	// Validate results
	if successCount == 0 {
		t.Errorf("No successful operations in concurrent test")
	}

	// Allow up to 10% error rate for concurrent operations
	const maxErrorRate = 10.0
	if errorRate > maxErrorRate {
		t.Errorf("Error rate %.1f%% exceeds maximum allowed %.1f%%", errorRate, maxErrorRate)
	}

	// Should achieve reasonable throughput (at least 50 operations per second)
	const minOperationsPerSecond = 50.0
	if operationsPerSecond < minOperationsPerSecond {
		t.Errorf("Operations per second %.1f is below minimum required %.1f",
			operationsPerSecond, minOperationsPerSecond)
	}
}

// TestSearchHybrid_Benchmark tests performance
func TestSearchHybrid_Benchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	benchmarks := []struct {
		name string
		opts *types.HybridSearchOptions
	}{
		{
			name: "Vector only search",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				IncludeContent: true,
				VectorWeight:   1.0,
				VectorUsing:    "dense",
			},
		},
		{
			name: "Vector search with metadata",
			opts: &types.HybridSearchOptions{
				CollectionName:  testDataSet.CollectionName,
				QueryVector:     queryVector,
				K:               10,
				IncludeMetadata: true,
				IncludeContent:  true,
				VectorWeight:    1.0,
				VectorUsing:     "dense",
			},
		},
		{
			name: "Vector search with vectors",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				IncludeVector:  true,
				IncludeContent: true,
				VectorWeight:   1.0,
				VectorUsing:    "dense",
			},
		},
		{
			name: "Vector search with pagination",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				Page:           1,
				PageSize:       5,
				IncludeContent: true,
				VectorWeight:   1.0,
				VectorUsing:    "dense",
			},
		},
	}

	for _, bm := range benchmarks {
		t.Run(bm.name, func(t *testing.T) {
			const numRuns = 10
			var totalDuration time.Duration
			var successfulRuns int

			for i := 0; i < numRuns; i++ {
				startTime := time.Now()
				result, err := env.Store.SearchHybrid(ctx, bm.opts)
				duration := time.Since(startTime)

				if err != nil {
					t.Logf("Run %d failed: %v", i+1, err)
					continue
				}

				if result == nil {
					t.Logf("Run %d returned nil result", i+1)
					continue
				}

				totalDuration += duration
				successfulRuns++
			}

			if successfulRuns == 0 {
				t.Errorf("All benchmark runs failed for %s", bm.name)
				return
			}

			avgDuration := totalDuration / time.Duration(successfulRuns)
			t.Logf("Benchmark %s: %d/%d successful runs, avg duration: %v",
				bm.name, successfulRuns, numRuns, avgDuration)

			// Validate reasonable performance (should complete within 10 seconds on average)
			const maxAvgDuration = 10 * time.Second
			if avgDuration > maxAvgDuration {
				t.Errorf("Average duration %v exceeds maximum allowed %v", avgDuration, maxAvgDuration)
			}
		})
	}
}

// TestSearchHybrid_AdvancedFeatures tests advanced search features
func TestSearchHybrid_AdvancedFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}
	ctx := context.Background()

	testSparseVector := &types.SparseVector{
		Indices: []uint32{1, 42, 100, 200, 300},
		Values:  []float32{0.22, 0.8, 0.5, 0.3, 0.6},
	}

	tests := []struct {
		name        string
		opts        *types.HybridSearchOptions
		expectError bool
	}{
		{
			name: "Custom EfSearch parameter",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse:    testSparseVector,
				K:              5,
				EfSearch:       128,
				IncludeContent: true,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError: false,
		},
		{
			name: "Approximate search enabled",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse:    testSparseVector,
				K:              5,
				Approximate:    true,
				IncludeContent: true,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError: false,
		},
		{
			name: "With minimum score threshold",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse:    testSparseVector,
				K:              10,
				MinScore:       0.3,
				IncludeContent: true,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError: false,
		},
		{
			name: "Include all data types",
			opts: &types.HybridSearchOptions{
				CollectionName:  testDataSet.CollectionName,
				QueryVector:     queryVector,
				QuerySparse:     testSparseVector,
				K:               5,
				IncludeVector:   true,
				IncludeMetadata: true,
				IncludeContent:  true,
				VectorUsing:     "dense",
				SparseUsing:     "sparse",
			},
			expectError: false,
		},
		{
			name: "High MaxResults limit",
			opts: &types.HybridSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				QuerySparse:    testSparseVector,
				K:              20,
				MaxResults:     1000,
				IncludeContent: true,
				VectorUsing:    "dense",
				SparseUsing:    "sparse",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchHybrid(ctx, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			t.Logf("Test '%s' returned %d results", tt.name, len(result.Documents))

			// Validate query time
			if result.QueryTime < 0 {
				t.Errorf("Query time should be non-negative, got %d", result.QueryTime)
			}

			// Validate include options
			for i, doc := range result.Documents {
				if tt.opts.IncludeVector && len(doc.Document.Vector) == 0 {
					t.Logf("Warning: Document %d has empty vector despite IncludeVector=true", i)
				}

				if tt.opts.IncludeMetadata && doc.Document.Metadata == nil {
					t.Logf("Warning: Document %d has nil metadata despite IncludeMetadata=true", i)
				}

				if tt.opts.IncludeContent && doc.Document.Content == "" {
					t.Logf("Warning: Document %d has empty content despite IncludeContent=true", i)
				}
			}

			// Validate minimum score if specified
			if tt.opts.MinScore > 0 {
				for i, doc := range result.Documents {
					if doc.Score < tt.opts.MinScore {
						t.Errorf("Document %d has score %f below minimum threshold %f",
							i, doc.Score, tt.opts.MinScore)
					}
				}
			}
		})
	}
}

// TestSearchHybrid_MultiLanguageSupport tests search across different language datasets
func TestSearchHybrid_MultiLanguageSupport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	ctx := context.Background()

	// Test with English dataset
	t.Run("English dataset", func(t *testing.T) {
		testDataSet := getOrCreateTestDataSet(t, "en")

		if len(testDataSet.Documents) == 0 {
			t.Skip("No English test documents available")
		}

		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		opts := &types.HybridSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			QuerySparse: &types.SparseVector{
				Indices: []uint32{1, 42, 100},
				Values:  []float32{0.5, 0.8, 0.3},
			},
			K:              5,
			FusionType:     types.FusionRRF,
			IncludeContent: true,
			VectorUsing:    "dense",
			SparseUsing:    "sparse",
		}

		result, err := env.Store.SearchHybrid(ctx, opts)
		if err != nil {
			t.Fatalf("English hybrid search failed: %v", err)
		}

		if result == nil {
			t.Fatal("English search result is nil")
		}

		t.Logf("English hybrid search returned %d results", len(result.Documents))

		// Validate basic properties
		if result.QueryTime < 0 {
			t.Errorf("Query time should be non-negative, got %d", result.QueryTime)
		}
	})

	// Test with Chinese dataset
	t.Run("Chinese dataset", func(t *testing.T) {
		testDataSet := getOrCreateTestDataSet(t, "zh")

		if len(testDataSet.Documents) == 0 {
			t.Skip("No Chinese test documents available")
		}

		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		opts := &types.HybridSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			QuerySparse: &types.SparseVector{
				Indices: []uint32{2, 50, 150},
				Values:  []float32{0.6, 0.9, 0.4},
			},
			K:              5,
			FusionType:     types.FusionRRF,
			IncludeContent: true,
			VectorUsing:    "dense",
			SparseUsing:    "sparse",
		}

		result, err := env.Store.SearchHybrid(ctx, opts)
		if err != nil {
			t.Fatalf("Chinese hybrid search failed: %v", err)
		}

		if result == nil {
			t.Fatal("Chinese search result is nil")
		}

		t.Logf("Chinese hybrid search returned %d results", len(result.Documents))

		// Validate basic properties
		if result.QueryTime < 0 {
			t.Errorf("Query time should be non-negative, got %d", result.QueryTime)
		}
	})

	// Cross-language validation
	t.Run("Cross-language consistency", func(t *testing.T) {
		enDataSet := getOrCreateTestDataSet(t, "en")
		zhDataSet := getOrCreateTestDataSet(t, "zh")

		if len(enDataSet.Documents) == 0 || len(zhDataSet.Documents) == 0 {
			t.Skip("Skipping cross-language test: datasets not available")
		}

		// Test same query on different language datasets
		sameQueryVector := getQueryVectorFromDataSet(enDataSet)
		if len(sameQueryVector) == 0 {
			t.Skip("No dense query vector available from English test data")
		}
		sameSparseVector := &types.SparseVector{
			Indices: []uint32{1, 42, 100},
			Values:  []float32{0.5, 0.8, 0.3},
		}

		// English search
		enOpts := &types.HybridSearchOptions{
			CollectionName: enDataSet.CollectionName,
			QueryVector:    sameQueryVector,
			QuerySparse:    sameSparseVector,
			K:              3,
			FusionType:     types.FusionRRF,
			IncludeContent: true,
			VectorUsing:    "dense",
			SparseUsing:    "sparse",
		}

		enResult, err := env.Store.SearchHybrid(ctx, enOpts)
		if err != nil {
			t.Fatalf("English search failed: %v", err)
		}

		// Chinese search with same parameters
		zhOpts := &types.HybridSearchOptions{
			CollectionName: zhDataSet.CollectionName,
			QueryVector:    sameQueryVector,
			QuerySparse:    sameSparseVector,
			K:              3,
			FusionType:     types.FusionRRF,
			IncludeContent: true,
			VectorUsing:    "dense",
			SparseUsing:    "sparse",
		}

		zhResult, err := env.Store.SearchHybrid(ctx, zhOpts)
		if err != nil {
			t.Fatalf("Chinese search failed: %v", err)
		}

		// Compare results (should both work, but may have different scores)
		t.Logf("Cross-language test: EN results=%d, ZH results=%d",
			len(enResult.Documents), len(zhResult.Documents))

		// Both searches should complete successfully
		if enResult.QueryTime < 0 {
			t.Errorf("English query time should be non-negative")
		}
		if zhResult.QueryTime < 0 {
			t.Errorf("Chinese query time should be non-negative")
		}
	})
}
