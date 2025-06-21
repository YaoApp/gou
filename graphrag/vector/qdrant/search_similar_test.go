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
// SearchSimilar Tests
// =============================================================================

// TestSearchSimilar_BasicFunctionality tests basic search functionality
func TestSearchSimilar_BasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	ctx := context.Background()

	// Find a document with dense vector for query
	var queryVector []float64
	for _, doc := range testDataSet.Documents {
		if len(doc.Vector) > 0 {
			queryVector = doc.Vector
			break
		} else if len(doc.DenseVector) > 0 {
			queryVector = doc.DenseVector
			break
		}
	}

	// Skip if no dense query vector available
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	t.Run("BasicSimilaritySearch", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned")
		}

		if len(result.Documents) > 5 {
			t.Errorf("Expected at most 5 documents, got %d", len(result.Documents))
		}

		// Verify that scores are in descending order
		for i := 1; i < len(result.Documents); i++ {
			if result.Documents[i-1].Score < result.Documents[i].Score {
				t.Errorf("Documents not ordered by score: doc[%d].Score=%.6f < doc[%d].Score=%.6f",
					i-1, result.Documents[i-1].Score, i, result.Documents[i].Score)
			}
		}

		// Verify MaxScore and MinScore
		if len(result.Documents) > 0 {
			firstScore := result.Documents[0].Score
			lastScore := result.Documents[len(result.Documents)-1].Score

			if result.MaxScore != firstScore {
				t.Errorf("MaxScore mismatch: expected %.6f, got %.6f", firstScore, result.MaxScore)
			}
			if result.MinScore != lastScore {
				t.Errorf("MinScore mismatch: expected %.6f, got %.6f", lastScore, result.MinScore)
			}
		}
	})

	t.Run("SearchWithDifferentK", func(t *testing.T) {
		for _, k := range []int{1, 3, 10, 20} {
			t.Run(fmt.Sprintf("K=%d", k), func(t *testing.T) {
				opts := &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              k,
					VectorUsing:    "dense", // Specify vector name for named vector collections
				}

				result, err := env.Store.SearchSimilar(ctx, opts)
				if err != nil {
					t.Fatalf("SearchSimilar with K=%d failed: %v", k, err)
				}

				maxExpected := min(k, len(testDataSet.Documents))
				if len(result.Documents) > maxExpected {
					t.Errorf("With K=%d, expected at most %d documents, got %d",
						k, maxExpected, len(result.Documents))
				}
			})
		}
	})

	t.Run("PaginationTest", func(t *testing.T) {
		pageSize := 3
		totalPages := 2

		var allResults []*types.SearchResultItem

		for page := 1; page <= totalPages; page++ {
			opts := &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              pageSize * totalPages,
				Page:           page,
				PageSize:       pageSize,
				IncludeTotal:   true,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				t.Fatalf("Paginated search (page %d) failed: %v", page, err)
			}

			// Verify pagination metadata
			if result.Page != page {
				t.Errorf("Page mismatch: expected %d, got %d", page, result.Page)
			}
			if result.PageSize != pageSize {
				t.Errorf("PageSize mismatch: expected %d, got %d", pageSize, result.PageSize)
			}

			if page == 1 {
				if result.HasPrevious {
					t.Error("First page should not have previous page")
				}
			} else {
				if !result.HasPrevious {
					t.Error("Non-first page should have previous page")
				}
				if result.PreviousPage != page-1 {
					t.Errorf("PreviousPage mismatch: expected %d, got %d", page-1, result.PreviousPage)
				}
			}

			allResults = append(allResults, result.Documents...)
		}

		// Verify no duplicates across pages
		seenIDs := make(map[string]bool)
		for i, doc := range allResults {
			if seenIDs[doc.Document.ID] {
				t.Errorf("Duplicate document ID across pages: %s (found at result index %d)", doc.Document.ID, i)
			}
			seenIDs[doc.Document.ID] = true
		}

		t.Logf("Pagination test completed: total results across pages: %d, unique IDs: %d",
			len(allResults), len(seenIDs))
	})
}

// TestSearchSimilar_ErrorScenarios tests error conditions
func TestSearchSimilar_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	ctx := context.Background()

	// Find a document with dense vector for query
	var queryVector []float64
	for _, doc := range testDataSet.Documents {
		if len(doc.Vector) > 0 {
			queryVector = doc.Vector
			break
		} else if len(doc.DenseVector) > 0 {
			queryVector = doc.DenseVector
			break
		}
	}

	// Use a dummy vector if no test data vector available
	if len(queryVector) == 0 {
		queryVector = make([]float64, 1536) // Use actual dimension from test data
		for i := range queryVector {
			queryVector[i] = 0.1
		}
	}

	tests := []struct {
		name     string
		opts     *types.SearchOptions
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "NilOptions",
			opts:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "search options cannot be nil")
			},
		},
		{
			name: "EmptyCollectionName",
			opts: &types.SearchOptions{
				CollectionName: "",
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "collection name is required")
			},
		},
		{
			name: "EmptyQueryVector",
			opts: &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    []float64{},
				K:              5,
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "query vector is required")
			},
		},
		{
			name: "NilQueryVector",
			opts: &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    nil,
				K:              5,
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "query vector is required")
			},
		},
		{
			name: "NonexistentCollection",
			opts: &types.SearchOptions{
				CollectionName: "nonexistent_collection_12345",
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "failed to perform similarity search")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchSimilar(ctx, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchSimilar() expected error, got nil")
				} else if tt.errCheck != nil && !tt.errCheck(err) {
					t.Errorf("SearchSimilar() error = %v, error check failed", err)
				}
			} else {
				if err != nil {
					t.Errorf("SearchSimilar() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("SearchSimilar() returned nil result")
				}
			}
		})
	}
}

// TestSearchSimilar_EdgeCases tests edge cases
func TestSearchSimilar_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	ctx := context.Background()

	// Find a document with dense vector for query
	var queryVector []float64
	for _, doc := range testDataSet.Documents {
		if len(doc.Vector) > 0 {
			queryVector = doc.Vector
			break
		} else if len(doc.DenseVector) > 0 {
			queryVector = doc.DenseVector
			break
		}
	}

	// Skip if no dense query vector available
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	t.Run("ZeroK", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              0,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with K=0 failed: %v", err)
		}

		// Should return some default number of results
		if len(result.Documents) == 0 {
			t.Log("K=0 returned no documents (acceptable)")
		}
	})

	t.Run("VeryHighK", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10000,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with very high K failed: %v", err)
		}

		// Should not return more documents than available
		maxPossible := len(testDataSet.Documents)
		if len(result.Documents) > maxPossible {
			t.Errorf("Expected at most %d documents, got %d", maxPossible, len(result.Documents))
		}
	})

	t.Run("VeryHighMinScore", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			MinScore:       0.99999,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with very high MinScore failed: %v", err)
		}

		// Might return no documents if none meet the threshold
		for _, doc := range result.Documents {
			if doc.Score < 0.99999 {
				t.Errorf("Document score %.6f should be >= 0.99999", doc.Score)
			}
		}
	})

	t.Run("ZeroVector", func(t *testing.T) {
		zeroVector := make([]float64, len(queryVector))

		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    zeroVector,
			K:              5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with zero vector failed: %v", err)
		}

		// Should still return results, just with different scores
		if len(result.Documents) == 0 {
			t.Log("Zero vector returned no documents (acceptable)")
		}
	})

	t.Run("PaginationBeyondData", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			Page:           1000,
			PageSize:       10,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with pagination beyond data failed: %v", err)
		}

		// Should return empty results
		if len(result.Documents) > 0 {
			t.Logf("Pagination beyond data returned %d documents", len(result.Documents))
		}
	})

	t.Run("VeryShortTimeout", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			Timeout:        1,       // 1 millisecond
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		// This might timeout, which is acceptable
		if err != nil {
			if stringContains(err.Error(), "timeout") || stringContains(err.Error(), "context deadline exceeded") {
				t.Log("Very short timeout caused timeout error (acceptable)")
				return
			}
			t.Fatalf("SearchSimilar with very short timeout failed with unexpected error: %v", err)
		}

		// If it didn't timeout, it should still return valid results
		if result != nil && len(result.Documents) > 0 {
			t.Log("Very short timeout completed successfully")
		}
	})

	t.Run("FilterNoMatches", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			Filter: map[string]interface{}{
				"nonexistent_field": "nonexistent_value",
			},
			VectorUsing: "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with no-match filter failed: %v", err)
		}

		// Should return empty results or very few results
		if len(result.Documents) > 0 {
			t.Logf("Filter with no matches returned %d documents", len(result.Documents))
		}
	})
}

// TestSearchSimilar_NotConnectedStore tests error when store is not connected
func TestSearchSimilar_NotConnectedStore(t *testing.T) {
	store := NewStore()

	opts := &types.SearchOptions{
		CollectionName: "test_collection",
		QueryVector:    []float64{1.0, 2.0, 3.0},
		K:              5,
	}

	result, err := store.SearchSimilar(context.Background(), opts)

	if err == nil {
		t.Error("SearchSimilar() should fail when store is not connected")
	}

	if !stringContains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}

	if result != nil {
		t.Error("SearchSimilar() should return nil result when not connected")
	}
}

// TestSearchSimilar_MultiLanguageData tests cross-language search
func TestSearchSimilar_MultiLanguageData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load both English and Chinese datasets
	enDataSet := getOrCreateTestDataSet(t, "en")
	zhDataSet := getOrCreateTestDataSet(t, "zh")
	env := getOrCreateSearchTestEnvironment(t)

	if len(enDataSet.Documents) == 0 || len(zhDataSet.Documents) == 0 {
		t.Skip("Not enough test documents available for multi-language test")
	}

	ctx := context.Background()

	t.Run("SearchEnglishData", func(t *testing.T) {
		// Find a document with dense vector for query
		var queryVector []float64
		for _, doc := range enDataSet.Documents {
			if len(doc.Vector) > 0 {
				queryVector = doc.Vector
				break
			} else if len(doc.DenseVector) > 0 {
				queryVector = doc.DenseVector
				break
			}
		}

		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from English test data")
		}

		opts := &types.SearchOptions{
			CollectionName:  enDataSet.CollectionName,
			QueryVector:     queryVector,
			K:               5,
			IncludeMetadata: true,
			VectorUsing:     "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar on English data failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned for English search")
		}

		// Verify that returned documents are from English dataset
		for i, doc := range result.Documents {
			if lang, ok := doc.Document.Metadata["language"]; ok {
				if lang != "en" {
					t.Errorf("Document %d should be English, got language: %v", i, lang)
				}
			}
		}
	})

	t.Run("SearchChineseData", func(t *testing.T) {
		// Find a document with dense vector for query
		var queryVector []float64
		for _, doc := range zhDataSet.Documents {
			if len(doc.Vector) > 0 {
				queryVector = doc.Vector
				break
			} else if len(doc.DenseVector) > 0 {
				queryVector = doc.DenseVector
				break
			}
		}

		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from Chinese test data")
		}

		opts := &types.SearchOptions{
			CollectionName:  zhDataSet.CollectionName,
			QueryVector:     queryVector,
			K:               5,
			IncludeMetadata: true,
			VectorUsing:     "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar on Chinese data failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned for Chinese search")
		}

		// Verify that returned documents are from Chinese dataset
		for i, doc := range result.Documents {
			if lang, ok := doc.Document.Metadata["language"]; ok {
				if lang != "zh" {
					t.Errorf("Document %d should be Chinese, got language: %v", i, lang)
				}
			}
		}
	})

	t.Run("CrossLanguageQuery", func(t *testing.T) {
		// Use English vector to search Chinese collection
		// Find a document with dense vector for query
		var enVector []float64
		for _, doc := range enDataSet.Documents {
			if len(doc.Vector) > 0 {
				enVector = doc.Vector
				break
			} else if len(doc.DenseVector) > 0 {
				enVector = doc.DenseVector
				break
			}
		}

		if len(enVector) == 0 {
			t.Skip("No dense query vector available from English test data")
		}

		opts := &types.SearchOptions{
			CollectionName:  zhDataSet.CollectionName,
			QueryVector:     enVector,
			K:               3,
			IncludeMetadata: true,
			VectorUsing:     "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("Cross-language search failed: %v", err)
		}

		// Results should still be from Chinese collection
		for i, doc := range result.Documents {
			if lang, ok := doc.Document.Metadata["language"]; ok {
				if lang != "zh" {
					t.Errorf("Cross-language search result %d should be Chinese, got: %v", i, lang)
				}
			}
		}
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// BenchmarkSearchSimilar benchmarks the SearchSimilar method
func BenchmarkSearchSimilar(b *testing.B) {
	// Setup test environment
	env := getOrCreateSearchTestEnvironment(&testing.T{})
	testDataSet := getOrCreateTestDataSet(&testing.T{}, "en")

	if len(testDataSet.Documents) == 0 {
		b.Skip("No test documents available for benchmarking")
	}

	ctx := context.Background()

	// Find a document with dense vector for query
	var queryVector []float64
	for _, doc := range testDataSet.Documents {
		if len(doc.Vector) > 0 {
			queryVector = doc.Vector
			break
		} else if len(doc.DenseVector) > 0 {
			queryVector = doc.DenseVector
			break
		}
	}

	if len(queryVector) == 0 {
		b.Skip("No dense query vector available from test data")
	}

	b.Run("BasicSearch", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar failed: %v", err)
			}
		}
	})

	b.Run("SearchWithVectors", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			IncludeVector:  true,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with vectors failed: %v", err)
			}
		}
	})

	b.Run("SearchWithFilter", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			Filter: map[string]interface{}{
				"language": "en",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with filter failed: %v", err)
			}
		}
	})

	b.Run("SearchWithPagination", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              20,
			Page:           1,
			PageSize:       5,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with pagination failed: %v", err)
			}
		}
	})

	b.Run("HighK", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with high K failed: %v", err)
			}
		}
	})
}

// =============================================================================
// Memory and Stress Tests
// =============================================================================

// TestSearchSimilar_MemoryLeakDetection tests for memory leaks during searches
func TestSearchSimilar_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available for memory leak test")
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

	// Perform many searches
	iterations := 500
	searchesPerIteration := 5

	for i := 0; i < iterations; i++ {
		for j := 0; j < searchesPerIteration; j++ {
			opts := &types.SearchOptions{
				CollectionName:  testDataSet.CollectionName,
				QueryVector:     queryVector,
				K:               10,
				IncludeVector:   j%2 == 0,
				IncludeMetadata: j%3 == 0,
				IncludeContent:  j%4 == 0,
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				t.Fatalf("SearchSimilar failed at iteration %d, search %d: %v", i, j, err)
			}

			if len(result.Documents) == 0 {
				t.Fatalf("No documents returned at iteration %d, search %d", i, j)
			}

			// Force result to go out of scope
			result = nil
		}

		// Periodic cleanup and progress reporting
		if i%100 == 0 {
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
	t.Logf("  Operations: %d searches", iterations*searchesPerIteration)
	t.Logf("  Initial HeapAlloc: %d KB", initialStats.HeapAlloc/1024)
	t.Logf("  Final HeapAlloc: %d KB", finalStats.HeapAlloc/1024)
	t.Logf("  Heap Growth: %d KB", heapGrowth/1024)
	t.Logf("  Total Alloc Growth: %d KB", totalAllocGrowth/1024)
	t.Logf("  GC Runs: %d", finalStats.NumGC-initialStats.NumGC)

	// Check for excessive memory growth
	// Allow up to 100MB heap growth and 1GB total allocation growth
	if heapGrowth > 100*1024*1024 {
		t.Errorf("Excessive heap growth: %d MB", heapGrowth/(1024*1024))
	}
	if totalAllocGrowth > 1024*1024*1024 {
		t.Errorf("Excessive total allocation growth: %d MB", totalAllocGrowth/(1024*1024))
	}
}

// TestSearchSimilar_ConcurrentStress tests concurrent search operations
func TestSearchSimilar_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available for concurrent stress test")
	}

	ctx := context.Background()

	// Test parameters
	numGoroutines := 20
	operationsPerGoroutine := 50

	// Different test scenarios
	testScenarios := []struct {
		name string
		opts func(int) *types.SearchOptions
	}{
		{
			name: "basic",
			opts: func(i int) *types.SearchOptions {
				queryVector := getQueryVectorFromDataSet(testDataSet)
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              10,
					VectorUsing:    "dense",
				}
			},
		},
		{
			name: "paginated",
			opts: func(i int) *types.SearchOptions {
				queryVector := getQueryVectorFromDataSet(testDataSet)
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              20,
					Page:           (i % 3) + 1,
					PageSize:       5,
					VectorUsing:    "dense",
				}
			},
		},
		{
			name: "filtered",
			opts: func(i int) *types.SearchOptions {
				queryVector := getQueryVectorFromDataSet(testDataSet)
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              10,
					Filter: map[string]interface{}{
						"language": "en",
					},
					VectorUsing: "dense",
				}
			},
		},
		{
			name: "high_k",
			opts: func(i int) *types.SearchOptions {
				queryVector := getQueryVectorFromDataSet(testDataSet)
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              50,
					VectorUsing:    "dense",
				}
			},
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			var wg sync.WaitGroup
			errors := make(chan error, numGoroutines*operationsPerGoroutine)
			results := make(chan *types.SearchResult, numGoroutines*operationsPerGoroutine)

			startTime := time.Now()

			// Launch concurrent goroutines
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()

					for j := 0; j < operationsPerGoroutine; j++ {
						opID := goroutineID*operationsPerGoroutine + j
						opts := scenario.opts(opID)

						result, err := env.Store.SearchSimilar(ctx, opts)
						if err != nil {
							errors <- fmt.Errorf("goroutine %d, operation %d: %w", goroutineID, j, err)
							continue
						}

						if result == nil {
							errors <- fmt.Errorf("goroutine %d, operation %d: nil result", goroutineID, j)
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
			var resultList []*types.SearchResult

			for err := range errors {
				errorList = append(errorList, err)
			}

			for result := range results {
				resultList = append(resultList, result)
			}

			// Calculate statistics
			totalOperations := numGoroutines * operationsPerGoroutine
			successfulOperations := len(resultList)
			errorRate := float64(len(errorList)) / float64(totalOperations) * 100
			opsPerSecond := float64(successfulOperations) / duration.Seconds()

			t.Logf("Concurrent stress test (%s) completed:", scenario.name)
			t.Logf("  Total operations: %d", totalOperations)
			t.Logf("  Successful operations: %d", successfulOperations)
			t.Logf("  Errors: %d", len(errorList))
			t.Logf("  Error rate: %.2f%%", errorRate)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Operations per second: %.2f", opsPerSecond)

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
			for i, result := range resultList {
				if i >= 10 { // Only check first 10 results
					break
				}

				if len(result.Documents) == 0 {
					t.Errorf("Result %d has no documents", i)
				}

				// Verify score ordering
				for j := 1; j < len(result.Documents); j++ {
					if result.Documents[j-1].Score < result.Documents[j].Score {
						t.Errorf("Result %d: documents not ordered by score at positions %d, %d", i, j-1, j)
						break
					}
				}
			}
		})
	}
}

// TestSearchSimilar_ConcurrentWithDifferentCollections tests concurrent access to different collections
func TestSearchSimilar_ConcurrentWithDifferentCollections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent collections test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	enDataSet := getOrCreateTestDataSet(t, "en")
	zhDataSet := getOrCreateTestDataSet(t, "zh")

	if len(enDataSet.Documents) == 0 || len(zhDataSet.Documents) == 0 {
		t.Skip("Not enough test documents for concurrent collections test")
	}

	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent searches on English collection
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			queryVector := getQueryVectorFromDataSet(enDataSet)
			opts := &types.SearchOptions{
				CollectionName: enDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense",
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				errors <- fmt.Errorf("EN search %d: %w", idx, err)
				return
			}

			if len(result.Documents) == 0 {
				errors <- fmt.Errorf("EN search %d: no results", idx)
			}
		}(i)
	}

	// Concurrent searches on Chinese collection
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			queryVector := getQueryVectorFromDataSet(zhDataSet)
			opts := &types.SearchOptions{
				CollectionName: zhDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense",
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				errors <- fmt.Errorf("ZH search %d: %w", idx, err)
				return
			}

			if len(result.Documents) == 0 {
				errors <- fmt.Errorf("ZH search %d: no results", idx)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Errorf("Concurrent collections test had %d errors:", len(errorList))
		for i, err := range errorList {
			if i >= 5 {
				t.Logf("... and %d more errors", len(errorList)-5)
				break
			}
			t.Logf("Error %d: %v", i+1, err)
		}
	}
}
