package qdrant

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// =============================================================================
// SearchBatch Tests
// =============================================================================

// UnsupportedSearchOptions is a mock type for testing unsupported search options
type UnsupportedSearchOptions struct{}

// GetType implements the SearchOptionsInterface
func (u *UnsupportedSearchOptions) GetType() types.SearchType {
	return types.SearchType("unsupported")
}

// TestSearchBatch_BasicFunctionality tests basic batch search functionality
func TestSearchBatch_BasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) < 3 {
		t.Skip("Need at least 3 test documents for batch search tests")
	}

	ctx := context.Background()

	t.Run("EmptyBatch", func(t *testing.T) {
		opts := []types.SearchOptionsInterface{}

		results, err := env.Store.SearchBatch(ctx, opts)
		if err != nil {
			t.Fatalf("SearchBatch with empty options failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty batch, got %d", len(results))
		}
	})

	t.Run("SingleSearch", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		opts := []types.SearchOptionsInterface{
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		results, err := env.Store.SearchBatch(ctx, opts)
		if err != nil {
			t.Fatalf("SearchBatch with single search failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if results[0] == nil {
			t.Fatal("First result is nil")
		}

		if len(results[0].Documents) == 0 {
			t.Error("First result has no documents")
		}
	})

	t.Run("MultipleSimilaritySearches", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		var opts []types.SearchOptionsInterface

		// Create multiple similarity search options
		for i := 0; i < 3; i++ {
			opts = append(opts, &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			})
		}

		results, err := env.Store.SearchBatch(ctx, opts)
		if err != nil {
			t.Fatalf("SearchBatch with multiple searches failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Verify each result
		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d is nil", i)
				continue
			}

			if len(result.Documents) == 0 {
				t.Errorf("Result %d has no documents", i)
			}

			// Verify score ordering
			for j := 1; j < len(result.Documents); j++ {
				if result.Documents[j-1].Score < result.Documents[j].Score {
					t.Errorf("Result %d: documents not ordered by score at positions %d, %d", i, j-1, j)
				}
			}
		}
	})

	t.Run("MixedSearchTypes", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		opts := []types.SearchOptionsInterface{
			// Similarity search
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			// Score threshold search
			&types.ScoreThresholdOptions{
				CollectionName:  testDataSet.CollectionName,
				QueryVector:     queryVector,
				K:               5,
				ScoreThreshold:  0.1,
				IncludeMetadata: true,
				VectorUsing:     "dense", // Specify vector name for named vector collections
			},
			// MMR search
			&types.MMRSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              3,
				FetchK:         10,
				LambdaMult:     0.5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		results, err := env.Store.SearchBatch(ctx, opts)
		if err != nil {
			t.Fatalf("SearchBatch with mixed search types failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Verify all results are non-nil
		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d is nil", i)
			}
		}
	})

	t.Run("LargeBatch", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		var opts []types.SearchOptionsInterface

		// Create a large batch of search options
		batchSize := 20
		for i := 0; i < batchSize; i++ {
			opts = append(opts, &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              3,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			})
		}

		startTime := time.Now()
		results, err := env.Store.SearchBatch(ctx, opts)
		duration := time.Since(startTime)

		if err != nil {
			t.Fatalf("SearchBatch with large batch failed: %v", err)
		}

		if len(results) != batchSize {
			t.Errorf("Expected %d results, got %d", batchSize, len(results))
		}

		// Verify all results
		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d is nil", i)
				continue
			}

			if len(result.Documents) == 0 {
				t.Errorf("Result %d has no documents", i)
			}
		}

		t.Logf("Large batch (%d searches) completed in %v (%.2f searches/sec)",
			batchSize, duration, float64(batchSize)/duration.Seconds())
	})
}

// TestSearchBatch_ErrorScenarios tests error conditions
func TestSearchBatch_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	ctx := context.Background()

	t.Run("NilOptionInBatch", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		opts := []types.SearchOptionsInterface{
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			nil, // This should cause an error
		}

		results, err := env.Store.SearchBatch(ctx, opts)

		if err == nil {
			t.Error("SearchBatch should fail with nil option")
		}

		if !stringContains(err.Error(), "search option at index 1 is nil") {
			t.Errorf("Expected specific error message about nil option, got: %v", err)
		}

		if results != nil {
			t.Error("Results should be nil when validation fails")
		}
	})

	t.Run("UnsupportedSearchType", func(t *testing.T) {
		opts := []types.SearchOptionsInterface{
			&UnsupportedSearchOptions{},
		}

		results, err := env.Store.SearchBatch(ctx, opts)

		if err == nil {
			t.Error("SearchBatch should fail with unsupported option type")
		}

		if !stringContains(err.Error(), "unsupported search option type") {
			t.Errorf("Expected unsupported type error, got: %v", err)
		}

		// Results should still be returned even with errors
		if results == nil {
			t.Error("Results should not be nil even with errors")
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result slot, got %d", len(results))
		}
	})

	t.Run("MixedSuccessAndFailure", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		opts := []types.SearchOptionsInterface{
			// Valid search
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			// Invalid search (nonexistent collection)
			&types.SearchOptions{
				CollectionName: "nonexistent_collection_12345",
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			// Another valid search
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              3,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		results, err := env.Store.SearchBatch(ctx, opts)

		// Should return error but also results
		if err == nil {
			t.Error("SearchBatch should return error when some searches fail")
		}

		if results == nil {
			t.Error("Results should not be nil even with partial failures")
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 result slots, got %d", len(results))
		}

		// First and third results should be valid, second should be nil
		if results[0] == nil {
			t.Error("First result should be valid")
		}

		if results[2] == nil {
			t.Error("Third result should be valid")
		}

		// Error message should mention the failed search
		if !stringContains(err.Error(), "search 1:") {
			t.Errorf("Error should mention failed search 1, got: %v", err)
		}
	})

	t.Run("NotConnectedStore", func(t *testing.T) {
		disconnectedStore := NewStore()

		opts := []types.SearchOptionsInterface{
			&types.SearchOptions{
				CollectionName: "test_collection",
				QueryVector:    []float64{1.0, 2.0, 3.0},
				K:              5,
			},
		}

		results, err := disconnectedStore.SearchBatch(ctx, opts)

		if err == nil {
			t.Error("SearchBatch should fail when store is not connected")
		}

		if !stringContains(err.Error(), "not connected") {
			t.Errorf("Expected 'not connected' error, got: %v", err)
		}

		if results != nil {
			t.Error("Results should be nil when not connected")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(testDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from test data")
		}

		// Create a context that will be cancelled
		cancelCtx, cancel := context.WithCancel(context.Background())

		opts := []types.SearchOptionsInterface{
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		// Cancel the context immediately
		cancel()

		results, err := env.Store.SearchBatch(cancelCtx, opts)

		// Behavior depends on timing - context might be cancelled before or during search
		if err != nil && stringContains(err.Error(), "context canceled") {
			t.Log("Context cancellation worked as expected")
		} else {
			// If search completed before cancellation, that's also acceptable
			t.Log("Search completed before context cancellation")
		}

		// Results behavior is implementation-dependent when context is cancelled
		_ = results
	})
}

// TestSearchBatch_ConcurrentStress tests concurrent batch search operations
func TestSearchBatch_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) < 5 {
		t.Skip("Need at least 5 test documents for concurrent stress test")
	}

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	// Test parameters
	numGoroutines := 10
	batchesPerGoroutine := 5
	searchesPerBatch := 3

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*batchesPerGoroutine)
	results := make(chan []*types.SearchResult, numGoroutines*batchesPerGoroutine)

	startTime := time.Now()

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < batchesPerGoroutine; j++ {
				// Create batch of searches
				var opts []types.SearchOptionsInterface
				for k := 0; k < searchesPerBatch; k++ {
					opts = append(opts, &types.SearchOptions{
						CollectionName: testDataSet.CollectionName,
						QueryVector:    queryVector,
						K:              5,
						VectorUsing:    "dense", // Specify vector name for named vector collections
					})
				}

				result, err := env.Store.SearchBatch(ctx, opts)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, batch %d: %w", goroutineID, j, err)
					continue
				}

				if result == nil {
					errors <- fmt.Errorf("goroutine %d, batch %d: nil result", goroutineID, j)
					continue
				}

				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(results)

	duration := time.Since(startTime)

	// Collect results and errors
	var errorList []error
	var resultList [][]*types.SearchResult

	for err := range errors {
		errorList = append(errorList, err)
	}

	for result := range results {
		resultList = append(resultList, result)
	}

	// Calculate statistics
	totalBatches := numGoroutines * batchesPerGoroutine
	totalSearches := totalBatches * searchesPerBatch
	successfulBatches := len(resultList)
	errorRate := float64(len(errorList)) / float64(totalBatches) * 100
	batchesPerSecond := float64(successfulBatches) / duration.Seconds()
	searchesPerSecond := float64(successfulBatches*searchesPerBatch) / duration.Seconds()

	t.Logf("Concurrent stress test completed:")
	t.Logf("  Total batches: %d", totalBatches)
	t.Logf("  Total searches: %d", totalSearches)
	t.Logf("  Successful batches: %d", successfulBatches)
	t.Logf("  Errors: %d", len(errorList))
	t.Logf("  Error rate: %.2f%%", errorRate)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Batches per second: %.2f", batchesPerSecond)
	t.Logf("  Searches per second: %.2f", searchesPerSecond)

	// Verify that error rate is reasonable (allow up to 5% error rate)
	if errorRate > 5.0 {
		t.Errorf("Error rate too high: %.2f%% (expected <= 5%%)", errorRate)

		// Log first few errors for debugging
		for i, err := range errorList {
			if i >= 5 {
				t.Logf("... and %d more errors", len(errorList)-5)
				break
			}
			t.Logf("Error %d: %v", i+1, err)
		}
	}

	// Verify that results are reasonable
	for i, batchResult := range resultList {
		if i >= 10 { // Only check first 10 batch results
			break
		}

		if len(batchResult) != searchesPerBatch {
			t.Errorf("Batch result %d has %d results, expected %d", i, len(batchResult), searchesPerBatch)
		}

		for j, searchResult := range batchResult {
			if searchResult == nil {
				t.Errorf("Batch %d, search %d: nil result", i, j)
				continue
			}

			if len(searchResult.Documents) == 0 {
				t.Errorf("Batch %d, search %d: no documents", i, j)
			}

			// Verify score ordering for first few results
			if j < 3 && len(searchResult.Documents) > 1 {
				for k := 1; k < len(searchResult.Documents); k++ {
					if searchResult.Documents[k-1].Score < searchResult.Documents[k].Score {
						t.Errorf("Batch %d, search %d: documents not ordered by score at positions %d, %d", i, j, k-1, k)
						break
					}
				}
			}
		}
	}
}

// TestSearchBatch_MemoryLeakDetection tests for memory leaks during batch searches
func TestSearchBatch_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) < 3 {
		t.Skip("Need at least 3 test documents for memory leak test")
	}

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	// Get initial memory stats
	var initialStats, finalStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialStats)

	// Perform many batch searches
	iterations := 200
	batchSize := 5

	for i := 0; i < iterations; i++ {
		// Create batch of searches
		var opts []types.SearchOptionsInterface
		for j := 0; j < batchSize; j++ {
			// Vary search types and options
			switch j % 3 {
			case 0:
				opts = append(opts, &types.SearchOptions{
					CollectionName:  testDataSet.CollectionName,
					QueryVector:     queryVector,
					K:               5,
					IncludeVector:   i%2 == 0,
					IncludeMetadata: i%3 == 0,
					VectorUsing:     "dense", // Specify vector name for named vector collections
				})
			case 1:
				opts = append(opts, &types.ScoreThresholdOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              3,
					ScoreThreshold: 0.1,
					IncludeContent: i%4 == 0,
					VectorUsing:    "dense", // Specify vector name for named vector collections
				})
			case 2:
				opts = append(opts, &types.MMRSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              3,
					FetchK:         10,
					LambdaMult:     0.5,
					VectorUsing:    "dense", // Specify vector name for named vector collections
				})
			}
		}

		result, err := env.Store.SearchBatch(ctx, opts)
		if err != nil {
			t.Fatalf("SearchBatch failed at iteration %d: %v", i, err)
		}

		if len(result) != batchSize {
			t.Fatalf("Expected %d results at iteration %d, got %d", batchSize, i, len(result))
		}

		// Force result to go out of scope
		result = nil
		opts = nil

		// Periodic cleanup and progress reporting
		if i%50 == 0 {
			runtime.GC()
			var currentStats runtime.MemStats
			runtime.ReadMemStats(&currentStats)
			t.Logf("Iteration %d/%d: HeapAlloc=%d KB, NumGC=%d",
				i, iterations, currentStats.HeapAlloc/1024, currentStats.NumGC)
		}
	}

	// Final memory check
	runtime.GC()
	runtime.ReadMemStats(&finalStats)

	// Calculate memory growth
	heapGrowth := int64(finalStats.HeapAlloc) - int64(initialStats.HeapAlloc)
	totalAllocGrowth := int64(finalStats.TotalAlloc) - int64(initialStats.TotalAlloc)

	t.Logf("Memory leak test completed:")
	t.Logf("  Operations: %d batch searches (%d total searches)", iterations, iterations*batchSize)
	t.Logf("  Initial HeapAlloc: %d KB", initialStats.HeapAlloc/1024)
	t.Logf("  Final HeapAlloc: %d KB", finalStats.HeapAlloc/1024)
	t.Logf("  Heap Growth: %d KB", heapGrowth/1024)
	t.Logf("  Total Alloc Growth: %d KB", totalAllocGrowth/1024)
	t.Logf("  GC Runs: %d", finalStats.NumGC-initialStats.NumGC)

	// Check for excessive memory growth
	// Allow up to 150MB heap growth and 1.5GB total allocation growth for batch operations
	if heapGrowth > 150*1024*1024 {
		t.Errorf("Excessive heap growth: %d MB", heapGrowth/(1024*1024))
	}
	if totalAllocGrowth > 1536*1024*1024 { // 1.5GB
		t.Errorf("Excessive total allocation growth: %d MB", totalAllocGrowth/(1024*1024))
	}
}

// TestSearchBatch_Performance tests the performance characteristics of batch search
func TestSearchBatch_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) < 10 {
		t.Skip("Need at least 10 test documents for performance test")
	}

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	// Test different batch sizes to understand performance characteristics
	batchSizes := []int{1, 5, 10, 20, 50}

	for _, batchSize := range batchSizes {
		t.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(t *testing.T) {
			var opts []types.SearchOptionsInterface

			// Create batch of searches
			for i := 0; i < batchSize; i++ {
				opts = append(opts, &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              10,
					VectorUsing:    "dense", // Specify vector name for named vector collections
				})
			}

			// Warm up
			_, err := env.Store.SearchBatch(ctx, opts)
			if err != nil {
				t.Fatalf("Warm-up batch search failed: %v", err)
			}

			// Measure performance
			iterations := 10
			totalDuration := time.Duration(0)

			for i := 0; i < iterations; i++ {
				startTime := time.Now()
				result, err := env.Store.SearchBatch(ctx, opts)
				duration := time.Since(startTime)
				totalDuration += duration

				if err != nil {
					t.Fatalf("Batch search failed at iteration %d: %v", i, err)
				}

				if len(result) != batchSize {
					t.Fatalf("Expected %d results, got %d", batchSize, len(result))
				}
			}

			avgDuration := totalDuration / time.Duration(iterations)
			searchesPerSecond := float64(batchSize) / avgDuration.Seconds()

			t.Logf("Batch size %d: avg duration=%v, searches/sec=%.2f",
				batchSize, avgDuration, searchesPerSecond)

			// Performance expectations (these are rough guidelines)
			// Larger batches should generally have higher throughput
			if batchSize == 1 && searchesPerSecond < 5 {
				t.Logf("Warning: Single search performance seems low: %.2f searches/sec", searchesPerSecond)
			}
			if batchSize >= 10 && searchesPerSecond < 15 {
				t.Logf("Warning: Batch search performance seems low: %.2f searches/sec", searchesPerSecond)
			}
		})
	}
}

// TestSearchBatch_EdgeCases tests edge cases and boundary conditions
func TestSearchBatch_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping edge case tests in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	t.Run("VeryLargeBatch", func(t *testing.T) {
		// Test with a very large batch to ensure the semaphore works correctly
		var opts []types.SearchOptionsInterface
		largeBatchSize := 100

		for i := 0; i < largeBatchSize; i++ {
			opts = append(opts, &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              3,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			})
		}

		startTime := time.Now()
		results, err := env.Store.SearchBatch(ctx, opts)
		duration := time.Since(startTime)

		if err != nil {
			t.Fatalf("Very large batch search failed: %v", err)
		}

		if len(results) != largeBatchSize {
			t.Errorf("Expected %d results, got %d", largeBatchSize, len(results))
		}

		// Verify all results are non-nil
		nullCount := 0
		for i, result := range results {
			if result == nil {
				nullCount++
				if nullCount <= 5 {
					t.Errorf("Result %d is nil", i)
				}
			}
		}
		if nullCount > 5 {
			t.Errorf("... and %d more nil results", nullCount-5)
		}

		t.Logf("Very large batch (%d searches) completed in %v (%.2f searches/sec)",
			largeBatchSize, duration, float64(largeBatchSize)/duration.Seconds())
	})

	t.Run("MixedValidAndInvalidOptions", func(t *testing.T) {
		opts := []types.SearchOptionsInterface{
			// Valid search
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			// Invalid search - empty query vector
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    []float64{},
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			// Valid search
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              3,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			// Invalid search - empty collection name
			&types.SearchOptions{
				CollectionName: "",
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		results, err := env.Store.SearchBatch(ctx, opts)

		// Should return error due to invalid options
		if err == nil {
			t.Error("Expected error due to invalid search options")
		}

		if results == nil {
			t.Error("Results should not be nil even with errors")
		}

		if len(results) != 4 {
			t.Errorf("Expected 4 result slots, got %d", len(results))
		}

		// First and third results should be valid, second and fourth should be nil
		if results[0] == nil {
			t.Error("First result should be valid")
		}
		if results[2] == nil {
			t.Error("Third result should be valid")
		}
	})

	t.Run("AllInvalidOptions", func(t *testing.T) {
		opts := []types.SearchOptionsInterface{
			&types.SearchOptions{
				CollectionName: "nonexistent",
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    []float64{},
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		results, err := env.Store.SearchBatch(ctx, opts)

		// Should return error
		if err == nil {
			t.Error("Expected error when all options are invalid")
		}

		// Should still return results array
		if results == nil {
			t.Error("Results should not be nil")
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 result slots, got %d", len(results))
		}
	})

	t.Run("TimeoutScenario", func(t *testing.T) {
		// Create a very short timeout context
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		var opts []types.SearchOptionsInterface
		for i := 0; i < 5; i++ {
			opts = append(opts, &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			})
		}

		results, err := env.Store.SearchBatch(timeoutCtx, opts)

		// This might timeout or complete successfully depending on timing
		if err != nil && (stringContains(err.Error(), "timeout") || stringContains(err.Error(), "context deadline exceeded")) {
			t.Log("Timeout occurred as expected")
		} else if err == nil {
			t.Log("Batch completed before timeout")
		} else {
			t.Logf("Unexpected error (not timeout): %v", err)
		}

		// Results behavior is implementation-dependent when timeout occurs
		_ = results
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// BenchmarkSearchBatch benchmarks the SearchBatch method
func BenchmarkSearchBatch(b *testing.B) {
	// Setup test environment
	env := getOrCreateSearchTestEnvironment(&testing.T{})
	testDataSet := getOrCreateTestDataSet(&testing.T{}, "en")

	if len(testDataSet.Documents) < 5 {
		b.Skip("Need at least 5 test documents for benchmarking")
	}

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		b.Skip("No dense query vector available from test data")
	}

	b.Run("BatchSize_1", func(b *testing.B) {
		opts := []types.SearchOptionsInterface{
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchBatch(ctx, opts)
			if err != nil {
				b.Fatalf("SearchBatch failed: %v", err)
			}
		}
	})

	b.Run("BatchSize_5", func(b *testing.B) {
		var opts []types.SearchOptionsInterface
		for i := 0; i < 5; i++ {
			opts = append(opts, &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchBatch(ctx, opts)
			if err != nil {
				b.Fatalf("SearchBatch failed: %v", err)
			}
		}
	})

	b.Run("BatchSize_10", func(b *testing.B) {
		var opts []types.SearchOptionsInterface
		for i := 0; i < 10; i++ {
			opts = append(opts, &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchBatch(ctx, opts)
			if err != nil {
				b.Fatalf("SearchBatch failed: %v", err)
			}
		}
	})

	b.Run("BatchSize_20", func(b *testing.B) {
		var opts []types.SearchOptionsInterface
		for i := 0; i < 20; i++ {
			opts = append(opts, &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchBatch(ctx, opts)
			if err != nil {
				b.Fatalf("SearchBatch failed: %v", err)
			}
		}
	})

	b.Run("MixedSearchTypes", func(b *testing.B) {
		opts := []types.SearchOptionsInterface{
			&types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			&types.ScoreThresholdOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              10,
				ScoreThreshold: 0.1,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			&types.MMRSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				FetchK:         15,
				LambdaMult:     0.5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchBatch(ctx, opts)
			if err != nil {
				b.Fatalf("SearchBatch with mixed types failed: %v", err)
			}
		}
	})
}
