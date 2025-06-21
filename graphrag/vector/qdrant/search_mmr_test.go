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
// SearchMMR Tests
// =============================================================================

// TestSearchMMR_BasicFunctionality tests basic MMR search functionality
func TestSearchMMR_BasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
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

	t.Run("BasicMMRSearch", func(t *testing.T) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			FetchK:         15,
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned")
		}

		if len(result.Documents) > 5 {
			t.Errorf("Expected at most 5 documents, got %d", len(result.Documents))
		}

		// Verify that scores are in descending order (MMR might modify order but should still be reasonable)
		for i := 1; i < len(result.Documents); i++ {
			if result.Documents[i-1].Score < result.Documents[i].Score {
				// This is acceptable for MMR as it balances similarity and diversity
				t.Logf("MMR reordered documents: doc[%d].Score=%.6f < doc[%d].Score=%.6f (diversity optimization)",
					i-1, result.Documents[i-1].Score, i, result.Documents[i].Score)
			}
		}

		// Verify MaxScore and MinScore
		if len(result.Documents) > 0 {
			// For MMR, result.MaxScore and MinScore are calculated from the actual returned documents
			// which might be reordered due to diversity optimization
			actualMaxScore := result.Documents[0].Score
			actualMinScore := result.Documents[0].Score
			for _, doc := range result.Documents {
				if doc.Score > actualMaxScore {
					actualMaxScore = doc.Score
				}
				if doc.Score < actualMinScore {
					actualMinScore = doc.Score
				}
			}

			if result.MaxScore != actualMaxScore {
				t.Errorf("MaxScore mismatch: expected %.6f, got %.6f", actualMaxScore, result.MaxScore)
			}
			if result.MinScore != actualMinScore {
				t.Errorf("MinScore mismatch: expected %.6f, got %.6f", actualMinScore, result.MinScore)
			}
		}
	})

	t.Run("DifferentLambdaValues", func(t *testing.T) {
		lambdaValues := []float64{0.0, 0.3, 0.5, 0.7, 1.0}

		for _, lambda := range lambdaValues {
			t.Run(fmt.Sprintf("Lambda=%.1f", lambda), func(t *testing.T) {
				opts := &types.MMRSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              5,
					FetchK:         15,
					LambdaMult:     lambda,
					VectorUsing:    "dense", // Specify vector name for named vector collections
				}

				result, err := env.Store.SearchMMR(ctx, opts)
				if err != nil {
					t.Fatalf("SearchMMR with lambda=%.1f failed: %v", lambda, err)
				}

				if len(result.Documents) == 0 {
					t.Errorf("No documents returned for lambda=%.1f", lambda)
				}

				t.Logf("Lambda=%.1f returned %d documents with max score %.6f",
					lambda, len(result.Documents), result.MaxScore)
			})
		}
	})

	t.Run("DifferentFetchKValues", func(t *testing.T) {
		fetchKValues := []int{5, 10, 20, 50}

		for _, fetchK := range fetchKValues {
			t.Run(fmt.Sprintf("FetchK=%d", fetchK), func(t *testing.T) {
				opts := &types.MMRSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              5,
					FetchK:         fetchK,
					LambdaMult:     0.5,
					VectorUsing:    "dense", // Specify vector name for named vector collections
				}

				result, err := env.Store.SearchMMR(ctx, opts)
				if err != nil {
					t.Fatalf("SearchMMR with FetchK=%d failed: %v", fetchK, err)
				}

				expectedK := min(5, len(testDataSet.Documents))
				if len(result.Documents) > expectedK {
					t.Errorf("Expected at most %d documents, got %d for FetchK=%d",
						expectedK, len(result.Documents), fetchK)
				}

				t.Logf("FetchK=%d returned %d documents", fetchK, len(result.Documents))
			})
		}
	})

	t.Run("CompareMMRWithSimilaritySearch", func(t *testing.T) {
		// Run MMR search
		mmrOpts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			FetchK:         15,
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		mmrResult, err := env.Store.SearchMMR(ctx, mmrOpts)
		if err != nil {
			t.Fatalf("SearchMMR failed: %v", err)
		}

		// Run regular similarity search
		simOpts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		simResult, err := env.Store.SearchSimilar(ctx, simOpts)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}

		// Compare results
		t.Logf("MMR returned %d documents, Similarity returned %d documents",
			len(mmrResult.Documents), len(simResult.Documents))

		// MMR might return different documents due to diversity optimization
		mmrIDs := make(map[string]bool)
		for _, doc := range mmrResult.Documents {
			mmrIDs[doc.Document.ID] = true
		}

		simIDs := make(map[string]bool)
		for _, doc := range simResult.Documents {
			simIDs[doc.Document.ID] = true
		}

		// Count overlap
		overlap := 0
		for id := range mmrIDs {
			if simIDs[id] {
				overlap++
			}
		}

		t.Logf("Document overlap between MMR and similarity search: %d/%d",
			overlap, len(mmrResult.Documents))
	})
}

// TestSearchMMR_ErrorScenarios tests error conditions
func TestSearchMMR_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)

	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	tests := []struct {
		name     string
		opts     *types.MMRSearchOptions
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "NilOptions",
			opts:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "MMR search options cannot be nil")
			},
		},
		{
			name: "EmptyCollectionName",
			opts: &types.MMRSearchOptions{
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
			opts: &types.MMRSearchOptions{
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
			opts: &types.MMRSearchOptions{
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
			opts: &types.MMRSearchOptions{
				CollectionName: "nonexistent_collection_12345",
				QueryVector:    queryVector,
				K:              5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "failed to fetch MMR candidates")
			},
		},
		{
			name: "VeryHighMinScore",
			opts: &types.MMRSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
				MinScore:       0.99999, // Very high threshold
				VectorUsing:    "dense", // Specify vector name for named vector collections
			},
			wantErr: false, // Should not error, but might return no results
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchMMR(ctx, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchMMR() expected error, got nil")
				} else if tt.errCheck != nil && !tt.errCheck(err) {
					t.Errorf("SearchMMR() error = %v, error check failed", err)
				}
			} else {
				if err != nil {
					t.Errorf("SearchMMR() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("SearchMMR() returned nil result")
				}
			}
		})
	}
}

// TestSearchMMR_EdgeCases tests edge cases
func TestSearchMMR_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
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

	t.Run("ZeroK", func(t *testing.T) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              0,
			FetchK:         10,
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR with K=0 failed: %v", err)
		}

		// Should return some default number of results or no results
		t.Logf("K=0 returned %d documents", len(result.Documents))
	})

	t.Run("KLargerThanFetchK", func(t *testing.T) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              20,
			FetchK:         10, // Smaller than K
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR with K > FetchK failed: %v", err)
		}

		// Should return at most FetchK documents
		if len(result.Documents) > 10 {
			t.Errorf("Expected at most 10 documents (FetchK), got %d", len(result.Documents))
		}
	})

	t.Run("ZeroFetchK", func(t *testing.T) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			FetchK:         0, // Should use default
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR with FetchK=0 failed: %v", err)
		}

		// Should still return results using default FetchK
		t.Logf("FetchK=0 returned %d documents", len(result.Documents))
	})

	t.Run("NegativeLambda", func(t *testing.T) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			FetchK:         15,
			LambdaMult:     -0.1,    // Negative lambda
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR with negative lambda failed: %v", err)
		}

		// Should still work, might prioritize diversity over similarity
		t.Logf("Negative lambda returned %d documents", len(result.Documents))
	})

	t.Run("LambdaGreaterThanOne", func(t *testing.T) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			FetchK:         15,
			LambdaMult:     1.5,     // Greater than 1
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR with lambda > 1 failed: %v", err)
		}

		// Should still work, might prioritize similarity over diversity
		t.Logf("Lambda > 1 returned %d documents", len(result.Documents))
	})

	t.Run("ZeroVector", func(t *testing.T) {
		zeroVector := make([]float64, len(queryVector))

		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    zeroVector,
			K:              5,
			FetchK:         15,
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR with zero vector failed: %v", err)
		}

		// Should still return results, just with different scores
		t.Logf("Zero vector returned %d documents", len(result.Documents))
	})

	t.Run("SingleDocumentDataset", func(t *testing.T) {
		// This test would require a dataset with only one document
		// For now, we simulate by using K=1 and FetchK=1
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              1,
			FetchK:         1,
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR with K=1, FetchK=1 failed: %v", err)
		}

		if len(result.Documents) > 1 {
			t.Errorf("Expected at most 1 document, got %d", len(result.Documents))
		}
	})

	t.Run("VeryShortTimeout", func(t *testing.T) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			FetchK:         15,
			LambdaMult:     0.5,
			Timeout:        1,       // 1 millisecond
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		// This might timeout, which is acceptable
		if err != nil {
			if stringContains(err.Error(), "timeout") || stringContains(err.Error(), "context deadline exceeded") {
				t.Log("Very short timeout caused timeout error (acceptable)")
				return
			}
			t.Fatalf("SearchMMR with very short timeout failed with unexpected error: %v", err)
		}

		// If it didn't timeout, it should still return valid results
		if result != nil && len(result.Documents) > 0 {
			t.Log("Very short timeout completed successfully")
		}
	})
}

// TestSearchMMR_NotConnectedStore tests error when store is not connected
func TestSearchMMR_NotConnectedStore(t *testing.T) {
	store := NewStore()

	opts := &types.MMRSearchOptions{
		CollectionName: "test_collection",
		QueryVector:    []float64{1.0, 2.0, 3.0},
		K:              5,
		FetchK:         15,
		LambdaMult:     0.5,
	}

	result, err := store.SearchMMR(context.Background(), opts)

	if err == nil {
		t.Error("SearchMMR() should fail when store is not connected")
	}

	if !stringContains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}

	if result != nil {
		t.Error("SearchMMR() should return nil result when not connected")
	}
}

// TestSearchMMR_WithPagination tests MMR search with pagination
func TestSearchMMR_WithPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
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

	t.Run("PaginationTest", func(t *testing.T) {
		pageSize := 3
		totalPages := 2

		var allResults []*types.SearchResultItem

		for page := 1; page <= totalPages; page++ {
			opts := &types.MMRSearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              pageSize * totalPages,
				FetchK:         pageSize * totalPages * 2, // Enough candidates for MMR
				LambdaMult:     0.5,
				Page:           page,
				PageSize:       pageSize,
				IncludeTotal:   true,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			}

			result, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				t.Fatalf("Paginated MMR search (page %d) failed: %v", page, err)
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

		t.Logf("MMR pagination test completed: total results across pages: %d, unique IDs: %d",
			len(allResults), len(seenIDs))
	})
}

// TestSearchMMR_MultiLanguageData tests MMR search across different languages
func TestSearchMMR_MultiLanguageData(t *testing.T) {
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

	t.Run("SearchEnglishDataMMR", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(enDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from English test data")
		}

		opts := &types.MMRSearchOptions{
			CollectionName:  enDataSet.CollectionName,
			QueryVector:     queryVector,
			K:               5,
			FetchK:          15,
			LambdaMult:      0.5,
			IncludeMetadata: true,
			VectorUsing:     "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR on English data failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned for English MMR search")
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

	t.Run("SearchChineseDataMMR", func(t *testing.T) {
		queryVector := getQueryVectorFromDataSet(zhDataSet)
		if len(queryVector) == 0 {
			t.Skip("No dense query vector available from Chinese test data")
		}

		opts := &types.MMRSearchOptions{
			CollectionName:  zhDataSet.CollectionName,
			QueryVector:     queryVector,
			K:               5,
			FetchK:          15,
			LambdaMult:      0.5,
			IncludeMetadata: true,
			VectorUsing:     "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("SearchMMR on Chinese data failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned for Chinese MMR search")
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

	t.Run("CrossLanguageMMRQuery", func(t *testing.T) {
		// Use English vector to search Chinese collection with MMR
		enVector := getQueryVectorFromDataSet(enDataSet)
		if len(enVector) == 0 {
			t.Skip("No dense query vector available from English test data")
		}

		opts := &types.MMRSearchOptions{
			CollectionName:  zhDataSet.CollectionName,
			QueryVector:     enVector,
			K:               3,
			FetchK:          9,
			LambdaMult:      0.5,
			IncludeMetadata: true,
			VectorUsing:     "dense", // Specify vector name for named vector collections
		}

		result, err := env.Store.SearchMMR(ctx, opts)
		if err != nil {
			t.Fatalf("Cross-language MMR search failed: %v", err)
		}

		// Results should still be from Chinese collection
		for i, doc := range result.Documents {
			if lang, ok := doc.Document.Metadata["language"]; ok {
				if lang != "zh" {
					t.Errorf("Cross-language MMR search result %d should be Chinese, got: %v", i, lang)
				}
			}
		}
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// BenchmarkSearchMMR benchmarks the SearchMMR method
func BenchmarkSearchMMR(b *testing.B) {
	// Setup test environment
	env := getOrCreateSearchTestEnvironment(&testing.T{})
	testDataSet := getOrCreateTestDataSet(&testing.T{}, "en")

	if len(testDataSet.Documents) == 0 {
		b.Skip("No test documents available for benchmarking")
	}

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)

	if len(queryVector) == 0 {
		b.Skip("No dense query vector available from test data")
	}

	b.Run("BasicMMRSearch", func(b *testing.B) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			FetchK:         30,
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				b.Fatalf("SearchMMR failed: %v", err)
			}
		}
	})

	b.Run("MMRWithHighFetchK", func(b *testing.B) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			FetchK:         100,
			LambdaMult:     0.5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				b.Fatalf("SearchMMR with high FetchK failed: %v", err)
			}
		}
	})

	b.Run("MMRHighSimilarity", func(b *testing.B) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			FetchK:         30,
			LambdaMult:     0.9,     // High similarity weight
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				b.Fatalf("SearchMMR with high similarity failed: %v", err)
			}
		}
	})

	b.Run("MMRHighDiversity", func(b *testing.B) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			FetchK:         30,
			LambdaMult:     0.1,     // High diversity weight
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				b.Fatalf("SearchMMR with high diversity failed: %v", err)
			}
		}
	})

	b.Run("MMRWithPagination", func(b *testing.B) {
		opts := &types.MMRSearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              20,
			FetchK:         60,
			LambdaMult:     0.5,
			Page:           1,
			PageSize:       5,
			VectorUsing:    "dense", // Specify vector name for named vector collections
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				b.Fatalf("SearchMMR with pagination failed: %v", err)
			}
		}
	})
}

// =============================================================================
// Memory and Stress Tests
// =============================================================================

// TestSearchMMR_MemoryLeakDetection tests for memory leaks during MMR searches
func TestSearchMMR_MemoryLeakDetection(t *testing.T) {
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

	// Perform many MMR searches
	iterations := 300 // Fewer iterations than similarity search due to higher complexity
	searchesPerIteration := 5

	for i := 0; i < iterations; i++ {
		for j := 0; j < searchesPerIteration; j++ {
			opts := &types.MMRSearchOptions{
				CollectionName:  testDataSet.CollectionName,
				QueryVector:     queryVector,
				K:               10,
				FetchK:          30,
				LambdaMult:      0.5,
				IncludeVector:   j%2 == 0,
				IncludeMetadata: j%3 == 0,
				IncludeContent:  j%4 == 0,
				VectorUsing:     "dense", // Specify vector name for named vector collections
			}

			result, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				t.Fatalf("SearchMMR failed at iteration %d, search %d: %v", i, j, err)
			}

			if len(result.Documents) == 0 {
				t.Fatalf("No documents returned at iteration %d, search %d", i, j)
			}

			// Force result to go out of scope
			result = nil
		}

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

	t.Logf("MMR memory leak test completed:")
	t.Logf("  Operations: %d searches", iterations*searchesPerIteration)
	t.Logf("  Initial HeapAlloc: %d KB", initialStats.HeapAlloc/1024)
	t.Logf("  Final HeapAlloc: %d KB", finalStats.HeapAlloc/1024)
	t.Logf("  Heap Growth: %d KB", heapGrowth/1024)
	t.Logf("  Total Alloc Growth: %d KB", totalAllocGrowth/1024)
	t.Logf("  GC Runs: %d", finalStats.NumGC-initialStats.NumGC)

	// Check for excessive memory growth (MMR uses more memory than simple similarity search)
	// Allow up to 150MB heap growth and 1.5GB total allocation growth
	if heapGrowth > 150*1024*1024 {
		t.Errorf("Excessive heap growth: %d MB", heapGrowth/(1024*1024))
	}
	if totalAllocGrowth > 1536*1024*1024 {
		t.Errorf("Excessive total allocation growth: %d MB", totalAllocGrowth/(1024*1024))
	}
}

// TestSearchMMR_ConcurrentStress tests concurrent MMR search operations
func TestSearchMMR_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available for concurrent stress test")
	}

	ctx := context.Background()
	queryVector := getQueryVectorFromDataSet(testDataSet)
	if len(queryVector) == 0 {
		t.Skip("No dense query vector available from test data")
	}

	// Test parameters (smaller than similarity search due to MMR complexity)
	numGoroutines := 15
	operationsPerGoroutine := 30

	// Different test scenarios
	testScenarios := []struct {
		name string
		opts func(int) *types.MMRSearchOptions
	}{
		{
			name: "basic_mmr",
			opts: func(i int) *types.MMRSearchOptions {
				return &types.MMRSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              10,
					FetchK:         30,
					LambdaMult:     0.5,
					VectorUsing:    "dense", // Specify vector name for named vector collections
				}
			},
		},
		{
			name: "high_similarity_mmr",
			opts: func(i int) *types.MMRSearchOptions {
				return &types.MMRSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              8,
					FetchK:         24,
					LambdaMult:     0.9,     // High similarity weight
					VectorUsing:    "dense", // Specify vector name for named vector collections
				}
			},
		},
		{
			name: "high_diversity_mmr",
			opts: func(i int) *types.MMRSearchOptions {
				return &types.MMRSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              8,
					FetchK:         24,
					LambdaMult:     0.1,     // High diversity weight
					VectorUsing:    "dense", // Specify vector name for named vector collections
				}
			},
		},
		{
			name: "paginated_mmr",
			opts: func(i int) *types.MMRSearchOptions {
				return &types.MMRSearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              15,
					FetchK:         45,
					LambdaMult:     0.5,
					Page:           (i % 3) + 1,
					PageSize:       5,
					VectorUsing:    "dense", // Specify vector name for named vector collections
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

						result, err := env.Store.SearchMMR(ctx, opts)
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

			t.Logf("Concurrent MMR stress test (%s) completed:", scenario.name)
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

				// For MMR, documents might not be in strict score order due to diversity optimization
				// Just verify that we have valid scores (allow scores > 1 for cosine similarity)
				for j, doc := range result.Documents {
					if doc.Score < 0 {
						t.Errorf("Result %d, document %d has invalid negative score: %.6f", i, j, doc.Score)
					}
					// Note: Scores can be > 1.0 in some similarity metrics, so we don't check upper bound
				}
			}
		})
	}
}

// TestSearchMMR_ConcurrentWithDifferentCollections tests concurrent MMR access to different collections
func TestSearchMMR_ConcurrentWithDifferentCollections(t *testing.T) {
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
	enQueryVector := getQueryVectorFromDataSet(enDataSet)
	zhQueryVector := getQueryVectorFromDataSet(zhDataSet)

	if len(enQueryVector) == 0 || len(zhQueryVector) == 0 {
		t.Skip("No dense query vectors available from test data")
	}

	var wg sync.WaitGroup
	errors := make(chan error, 50)

	// Concurrent MMR searches on English collection
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			opts := &types.MMRSearchOptions{
				CollectionName: enDataSet.CollectionName,
				QueryVector:    enQueryVector,
				K:              5,
				FetchK:         15,
				LambdaMult:     0.5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			}

			result, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				errors <- fmt.Errorf("EN MMR search %d: %w", idx, err)
				return
			}

			if len(result.Documents) == 0 {
				errors <- fmt.Errorf("EN MMR search %d: no results", idx)
			}
		}(i)
	}

	// Concurrent MMR searches on Chinese collection
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			opts := &types.MMRSearchOptions{
				CollectionName: zhDataSet.CollectionName,
				QueryVector:    zhQueryVector,
				K:              5,
				FetchK:         15,
				LambdaMult:     0.5,
				VectorUsing:    "dense", // Specify vector name for named vector collections
			}

			result, err := env.Store.SearchMMR(ctx, opts)
			if err != nil {
				errors <- fmt.Errorf("ZH MMR search %d: %w", idx, err)
				return
			}

			if len(result.Documents) == 0 {
				errors <- fmt.Errorf("ZH MMR search %d: no results", idx)
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
		t.Errorf("Concurrent MMR collections test had %d errors:", len(errorList))
		for i, err := range errorList {
			if i >= 5 {
				t.Logf("... and %d more errors", len(errorList)-5)
				break
			}
			t.Logf("Error %d: %v", i+1, err)
		}
	}
}
