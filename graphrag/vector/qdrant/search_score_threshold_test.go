package qdrant

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

func TestSearchWithScoreThreshold_BasicFunctionality(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	tests := []struct {
		name           string
		scoreThreshold float64
		expectedMin    int // minimum number of results we expect
		expectedMax    int // maximum number of results we expect
	}{
		{
			name:           "Low Score Threshold",
			scoreThreshold: 0.3,
			expectedMin:    1,
			expectedMax:    len(testDataSet.Documents),
		},
		{
			name:           "Medium Score Threshold",
			scoreThreshold: 0.6,
			expectedMin:    0,
			expectedMax:    len(testDataSet.Documents),
		},
		{
			name:           "High Score Threshold",
			scoreThreshold: 0.9,
			expectedMin:    0,
			expectedMax:    5,
		},
		{
			name:           "Very High Score Threshold",
			scoreThreshold: 0.99,
			expectedMin:    0,
			expectedMax:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &types.ScoreThresholdOptions{
				CollectionName:  collectionName,
				QueryVector:     queryVector,
				ScoreThreshold:  tt.scoreThreshold,
				K:               10,
				IncludeVector:   true,
				IncludeMetadata: true,
				IncludeContent:  true,
				VectorUsing:     "dense",
			}

			result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
			if err != nil {
				t.Fatalf("SearchWithScoreThreshold failed: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			// Check result count is within expected range
			if len(result.Documents) < tt.expectedMin {
				t.Errorf("Expected at least %d results, got %d", tt.expectedMin, len(result.Documents))
			}
			if len(result.Documents) > tt.expectedMax {
				t.Errorf("Expected at most %d results, got %d", tt.expectedMax, len(result.Documents))
			}

			// Check all scores meet the threshold
			for _, doc := range result.Documents {
				if doc.Score < tt.scoreThreshold {
					t.Errorf("Score %f is below threshold %f", doc.Score, tt.scoreThreshold)
				}
			}

			// Check basic result properties
			if len(result.Documents) > 0 {
				if result.QueryTime < 0 {
					t.Error("QueryTime should be non-negative")
				}
				if result.MaxScore < result.MinScore {
					t.Errorf("MaxScore %f should be >= MinScore %f", result.MaxScore, result.MinScore)
				}
				if result.MaxScore < tt.scoreThreshold {
					t.Errorf("MaxScore %f should be >= threshold %f", result.MaxScore, tt.scoreThreshold)
				}
				if result.MinScore < tt.scoreThreshold {
					t.Errorf("MinScore %f should be >= threshold %f", result.MinScore, tt.scoreThreshold)
				}

				// Check that results include requested data
				for _, doc := range result.Documents {
					if doc.Document.ID == "" {
						t.Error("Document ID should not be empty")
					}
					if len(doc.Document.Vector) == 0 {
						t.Error("Document vector should not be empty")
					}
					if doc.Document.Metadata == nil {
						t.Error("Document metadata should not be nil")
					}
					if doc.Document.Content == "" {
						t.Error("Document content should not be empty")
					}
				}
			}
		})
	}
}

func TestSearchWithScoreThreshold_PaginationSupport(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	// Use a low threshold to ensure we get multiple results
	opts := &types.ScoreThresholdOptions{
		CollectionName:  collectionName,
		QueryVector:     queryVector,
		ScoreThreshold:  0.1,
		Page:            1,
		PageSize:        3,
		IncludeTotal:    true,
		IncludeMetadata: true,
		IncludeContent:  true,
		VectorUsing:     "dense",
	}

	result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
	if err != nil {
		t.Fatalf("SearchWithScoreThreshold failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}

	// Check pagination metadata
	if result.Page != 1 {
		t.Errorf("Expected page 1, got %d", result.Page)
	}
	if result.PageSize != 3 {
		t.Errorf("Expected page size 3, got %d", result.PageSize)
	}
	if len(result.Documents) > 3 {
		t.Errorf("Expected at most 3 documents, got %d", len(result.Documents))
	}

	if len(result.Documents) == 3 && result.HasNext {
		if result.NextPage != 2 {
			t.Errorf("Expected next page 2, got %d", result.NextPage)
		}
	}
	if result.HasPrevious {
		t.Error("First page should not have previous page")
	}

	// Test second page if available
	if result.HasNext {
		opts.Page = 2
		result2, err := env.Store.SearchWithScoreThreshold(ctx, opts)
		if err != nil {
			t.Fatalf("SearchWithScoreThreshold page 2 failed: %v", err)
		}
		if result2 == nil {
			t.Fatal("Result2 is nil")
		}

		if result2.Page != 2 {
			t.Errorf("Expected page 2, got %d", result2.Page)
		}
		if !result2.HasPrevious {
			t.Error("Second page should have previous page")
		}
		if result2.PreviousPage != 1 {
			t.Errorf("Expected previous page 1, got %d", result2.PreviousPage)
		}

		// Ensure no duplicate documents between pages
		docIDs1 := make(map[string]bool)
		for _, doc := range result.Documents {
			docIDs1[doc.Document.ID] = true
		}

		for _, doc := range result2.Documents {
			if docIDs1[doc.Document.ID] {
				t.Errorf("Document %s appears in both pages", doc.Document.ID)
			}
		}
	}
}

func TestSearchWithScoreThreshold_MetadataFiltering(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	// Test with metadata filter
	filter := map[string]interface{}{
		"mapping_is_leaf": true,
	}

	opts := &types.ScoreThresholdOptions{
		CollectionName:  collectionName,
		QueryVector:     queryVector,
		ScoreThreshold:  0.1,
		K:               10,
		Filter:          filter,
		IncludeMetadata: true,
		IncludeContent:  true,
		VectorUsing:     "dense",
	}

	result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
	if err != nil {
		t.Fatalf("SearchWithScoreThreshold failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}

	// Check that all results meet the metadata filter
	for _, doc := range result.Documents {
		if isLeaf, exists := doc.Document.Metadata["mapping_is_leaf"]; exists {
			if !isLeaf.(bool) {
				t.Error("Document should be a leaf node")
			}
		}
	}
}

func TestSearchWithScoreThreshold_FieldSelection(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	opts := &types.ScoreThresholdOptions{
		CollectionName:  collectionName,
		QueryVector:     queryVector,
		ScoreThreshold:  0.1,
		K:               5,
		Fields:          []string{"mapping_filename", "mapping_depth"},
		IncludeMetadata: true,
		VectorUsing:     "dense",
	}

	result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
	if err != nil {
		t.Fatalf("SearchWithScoreThreshold failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}

	// Check that metadata is included and documents have the expected structure
	for _, doc := range result.Documents {
		if doc.Document.Metadata == nil {
			t.Error("Document metadata should not be nil")
			continue
		}
		// Note: Specific field requirements depend on actual test data structure
		// For now, just verify that metadata exists and is not empty
		if len(doc.Document.Metadata) == 0 {
			t.Error("Document metadata should not be empty")
		}
	}
}

func TestSearchWithScoreThreshold_SearchParameters(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	tests := []struct {
		name        string
		efSearch    int
		numProbes   int
		approximate bool
	}{
		{
			name:        "Default Parameters",
			efSearch:    0,
			numProbes:   0,
			approximate: false,
		},
		{
			name:        "Custom EfSearch",
			efSearch:    128,
			numProbes:   0,
			approximate: false,
		},
		{
			name:        "Approximate Search",
			efSearch:    0,
			numProbes:   0,
			approximate: true,
		},
		{
			name:        "Combined Parameters",
			efSearch:    64,
			numProbes:   10,
			approximate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &types.ScoreThresholdOptions{
				CollectionName:  collectionName,
				QueryVector:     queryVector,
				ScoreThreshold:  0.1,
				K:               5,
				EfSearch:        tt.efSearch,
				NumProbes:       tt.numProbes,
				Approximate:     tt.approximate,
				IncludeMetadata: true,
				IncludeContent:  true,
				VectorUsing:     "dense",
			}

			result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
			if err != nil {
				t.Fatalf("SearchWithScoreThreshold failed: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			if result.QueryTime < 0 {
				t.Error("QueryTime should be non-negative")
			}
		})
	}
}

func TestSearchWithScoreThreshold_TimeoutHandling(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	tests := []struct {
		name        string
		timeout     int
		expectError bool
	}{
		{
			name:        "No Timeout",
			timeout:     0,
			expectError: false,
		},
		{
			name:        "Reasonable Timeout",
			timeout:     5000, // 5 seconds
			expectError: false,
		},
		{
			name:        "Very Short Timeout",
			timeout:     1,     // 1ms - might timeout
			expectError: false, // Don't require error since it depends on system performance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &types.ScoreThresholdOptions{
				CollectionName:  collectionName,
				QueryVector:     queryVector,
				ScoreThreshold:  0.1,
				K:               5,
				Timeout:         tt.timeout,
				IncludeMetadata: true,
				IncludeContent:  true,
				VectorUsing:     "dense",
			}

			result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
			if !tt.expectError {
				// For very short timeouts, we allow either success or timeout error
				if err != nil && tt.timeout == 1 {
					t.Logf("Short timeout test failed as expected: %v", err)
				} else {
					if err != nil {
						t.Fatalf("SearchWithScoreThreshold failed: %v", err)
					}
					if result == nil {
						t.Fatal("Result is nil")
					}
				}
			}
		})
	}
}

func TestSearchWithScoreThreshold_ErrorScenarios(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	tests := []struct {
		name        string
		opts        *types.ScoreThresholdOptions
		expectError string
	}{
		{
			name:        "Nil Options",
			opts:        nil,
			expectError: "score threshold search options cannot be nil",
		},
		{
			name: "Empty Collection Name",
			opts: &types.ScoreThresholdOptions{
				CollectionName: "",
				QueryVector:    queryVector,
				ScoreThreshold: 0.5,
			},
			expectError: "collection name is required",
		},
		{
			name: "Empty Query Vector",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    []float64{},
				ScoreThreshold: 0.5,
			},
			expectError: "query vector is required",
		},
		{
			name: "Nil Query Vector",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    nil,
				ScoreThreshold: 0.5,
			},
			expectError: "query vector is required",
		},
		{
			name: "Nonexistent Collection",
			opts: &types.ScoreThresholdOptions{
				CollectionName: "nonexistent_collection_12345",
				QueryVector:    queryVector,
				ScoreThreshold: 0.5,
				K:              5,
				VectorUsing:    "dense",
			},
			expectError: "doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchWithScoreThreshold(ctx, tt.opts)
			if err == nil {
				t.Errorf("Expected error containing '%s', but got no error", tt.expectError)
			} else if !stringContains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectError, err.Error())
			}
			if result != nil {
				t.Error("Expected nil result on error")
			}
		})
	}
}

func TestSearchWithScoreThreshold_EdgeCases(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	tests := []struct {
		name string
		opts *types.ScoreThresholdOptions
	}{
		{
			name: "Zero Score Threshold",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.0,
				K:              5,
				VectorUsing:    "dense",
			},
		},
		{
			name: "Perfect Score Threshold",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 1.0,
				K:              5,
				VectorUsing:    "dense",
			},
		},
		{
			name: "Very High Score Threshold",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.9999,
				K:              5,
				VectorUsing:    "dense",
			},
		},
		{
			name: "Zero K Value",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.5,
				K:              0,
				VectorUsing:    "dense",
			},
		},
		{
			name: "Very High K Value",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.1,
				K:              10000,
				VectorUsing:    "dense",
			},
		},
		{
			name: "High MaxResults Limit",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.1,
				K:              1000,
				MaxResults:     100,
				VectorUsing:    "dense",
			},
		},
		{
			name: "Pagination Beyond Available Data",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.1,
				Page:           100,
				PageSize:       10,
				VectorUsing:    "dense",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchWithScoreThreshold(ctx, tt.opts)
			if err != nil {
				t.Fatalf("SearchWithScoreThreshold failed: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			// Check that all results meet the score threshold
			for _, doc := range result.Documents {
				if doc.Score < tt.opts.ScoreThreshold {
					t.Errorf("Score %f is below threshold %f", doc.Score, tt.opts.ScoreThreshold)
				}
			}
		})
	}
}

func TestSearchWithScoreThreshold_VectorDimensionMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	ctx := context.Background()
	collectionName := testDataSet.CollectionName

	// Test with vector of wrong dimension
	wrongVector := make([]float64, 100) // Assuming the actual dimension is different
	for i := range wrongVector {
		wrongVector[i] = 0.1 * float64(i)
	}

	opts := &types.ScoreThresholdOptions{
		CollectionName: collectionName,
		QueryVector:    wrongVector,
		ScoreThreshold: 0.5,
		K:              5,
		VectorUsing:    "dense",
	}

	result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
	// The error might occur or the search might return empty results
	// depending on Qdrant's behavior with dimension mismatch
	if err != nil {
		if !stringContains(err.Error(), "failed to perform score threshold search") {
			t.Errorf("Expected error about search failure, got: %v", err)
		}
	} else {
		// If no error, result should be valid but might be empty
		if result == nil {
			t.Error("Result should not be nil")
		}
	}
}

func TestSearchWithScoreThreshold_UnconnectedStore(t *testing.T) {
	// Create a new store without connecting
	store := &Store{
		connected: false,
	}

	ctx := context.Background()
	opts := &types.ScoreThresholdOptions{
		CollectionName: "test",
		QueryVector:    []float64{0.1, 0.2, 0.3},
		ScoreThreshold: 0.5,
		K:              5,
		VectorUsing:    "dense",
	}

	result, err := store.SearchWithScoreThreshold(ctx, opts)
	if err == nil {
		t.Error("Expected error for unconnected store")
	} else if !stringContains(err.Error(), "not connected to Qdrant server") {
		t.Errorf("Expected error about not connected, got: %v", err)
	}
	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestSearchWithScoreThreshold_MultiLanguageData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with English data
	enDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(enDataSet.Documents) == 0 {
		t.Skip("No English test documents available")
	}

	ctx := context.Background()
	enQueryVector := getQueryVectorFromDataSet(enDataSet)
	if len(enQueryVector) == 0 {
		t.Skip("No dense query vector available from English test data")
	}

	enResult, err := env.Store.SearchWithScoreThreshold(ctx, &types.ScoreThresholdOptions{
		CollectionName:  enDataSet.CollectionName,
		QueryVector:     enQueryVector,
		ScoreThreshold:  0.1,
		K:               5,
		IncludeMetadata: true,
		IncludeContent:  true,
		VectorUsing:     "dense",
	})
	if err != nil {
		t.Fatalf("English SearchWithScoreThreshold failed: %v", err)
	}
	if enResult == nil {
		t.Fatal("English result is nil")
	}

	// Test with Chinese data
	zhDataSet := getOrCreateTestDataSet(t, "zh")
	if len(zhDataSet.Documents) == 0 {
		t.Skip("No Chinese test documents available")
	}

	zhQueryVector := getQueryVectorFromDataSet(zhDataSet)
	if len(zhQueryVector) == 0 {
		t.Skip("No dense query vector available from Chinese test data")
	}

	zhResult, err := env.Store.SearchWithScoreThreshold(ctx, &types.ScoreThresholdOptions{
		CollectionName:  zhDataSet.CollectionName,
		QueryVector:     zhQueryVector,
		ScoreThreshold:  0.1,
		K:               5,
		IncludeMetadata: true,
		IncludeContent:  true,
		VectorUsing:     "dense",
	})
	if err != nil {
		t.Fatalf("Chinese SearchWithScoreThreshold failed: %v", err)
	}
	if zhResult == nil {
		t.Fatal("Chinese result is nil")
	}

	// Both searches should return results with scores above threshold
	for _, doc := range enResult.Documents {
		if doc.Score < 0.1 {
			t.Errorf("English result score %f below threshold 0.1", doc.Score)
		}
	}
	for _, doc := range zhResult.Documents {
		if doc.Score < 0.1 {
			t.Errorf("Chinese result score %f below threshold 0.1", doc.Score)
		}
	}
}

func BenchmarkSearchWithScoreThreshold(b *testing.B) {
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
	collectionName := testDataSet.CollectionName

	benchmarks := []struct {
		name string
		opts *types.ScoreThresholdOptions
	}{
		{
			name: "BasicSearch",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.5,
				K:              10,
				VectorUsing:    "dense",
			},
		},
		{
			name: "SearchWithMetadata",
			opts: &types.ScoreThresholdOptions{
				CollectionName:  collectionName,
				QueryVector:     queryVector,
				ScoreThreshold:  0.5,
				K:               10,
				IncludeMetadata: true,
				IncludeContent:  true,
				VectorUsing:     "dense",
			},
		},
		{
			name: "SearchWithVector",
			opts: &types.ScoreThresholdOptions{
				CollectionName: collectionName,
				QueryVector:    queryVector,
				ScoreThreshold: 0.5,
				K:              10,
				IncludeVector:  true,
				VectorUsing:    "dense",
			},
		},
		{
			name: "SearchWithPagination",
			opts: &types.ScoreThresholdOptions{
				CollectionName:  collectionName,
				QueryVector:     queryVector,
				ScoreThreshold:  0.3,
				Page:            1,
				PageSize:        5,
				IncludeTotal:    true,
				IncludeMetadata: true,
				VectorUsing:     "dense",
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := env.Store.SearchWithScoreThreshold(ctx, bm.opts)
				if err != nil {
					b.Fatalf("SearchWithScoreThreshold failed: %v", err)
				}
				if result == nil {
					b.Fatal("Result is nil")
				}
			}
		})
	}
}

func TestSearchWithScoreThreshold_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
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
	collectionName := testDataSet.CollectionName

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform many search operations
	iterations := 1500
	t.Logf("Performing %d SearchWithScoreThreshold operations for memory leak detection", iterations)

	opts := &types.ScoreThresholdOptions{
		CollectionName:  collectionName,
		QueryVector:     queryVector,
		ScoreThreshold:  0.1,
		K:               10,
		IncludeVector:   true,
		IncludeMetadata: true,
		IncludeContent:  true,
		VectorUsing:     "dense",
	}

	for i := 0; i < iterations; i++ {
		result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
		if err != nil {
			t.Fatalf("SearchWithScoreThreshold failed at iteration %d: %v", i, err)
		}
		if result == nil {
			t.Fatalf("Result is nil at iteration %d", i)
		}

		// Vary the query slightly to prevent caching effects
		if i%100 == 0 {
			for j := range opts.QueryVector {
				opts.QueryVector[j] += 0.001 * float64(i%10)
			}
		}

		// Force GC periodically
		if i%500 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&m2)

	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("Memory stats after %d SearchWithScoreThreshold operations:", iterations)
	t.Logf("  Heap allocation growth: %d bytes (%.2f MB)", heapGrowth, float64(heapGrowth)/(1024*1024))
	t.Logf("  Initial heap alloc: %d bytes", m1.HeapAlloc)
	t.Logf("  Final heap alloc: %d bytes", m2.HeapAlloc)
	t.Logf("  GC runs: %d", m2.NumGC-m1.NumGC)

	// Allow some memory growth, but it shouldn't be excessive
	maxAllowedGrowth := int64(10 * 1024 * 1024) // 10MB
	if heapGrowth > maxAllowedGrowth {
		t.Errorf("Potential memory leak detected: heap grew by %d bytes (%.2f MB), max allowed: %d bytes (%.2f MB)",
			heapGrowth, float64(heapGrowth)/(1024*1024),
			maxAllowedGrowth, float64(maxAllowedGrowth)/(1024*1024))
	}
}

func TestSearchWithScoreThreshold_ConcurrentStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	ctx := context.Background()
	collectionName := testDataSet.CollectionName

	// Test parameters
	numGoroutines := 15
	operationsPerGoroutine := 25
	totalOperations := numGoroutines * operationsPerGoroutine

	t.Logf("Starting concurrent stress test: %d goroutines × %d operations = %d total operations",
		numGoroutines, operationsPerGoroutine, totalOperations)

	var wg sync.WaitGroup
	var successCount, errorCount int64
	results := make(chan *types.SearchResult, totalOperations)
	errors := make(chan error, totalOperations)

	startTime := time.Now()

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Use different query vectors for variety, but ensure we get dense vectors
			queryVector := getQueryVectorFromDataSet(testDataSet)
			if len(queryVector) == 0 {
				// If no dense vector available, use a dummy vector
				queryVector = make([]float64, 384) // Assuming 384-dimensional vectors
				for i := range queryVector {
					queryVector[i] = 0.1 * float64(i%10)
				}
			}
			baseThreshold := 0.1 + 0.1*float64(workerID%5) // Vary threshold

			for j := 0; j < operationsPerGoroutine; j++ {
				opts := &types.ScoreThresholdOptions{
					CollectionName:  collectionName,
					QueryVector:     queryVector,
					ScoreThreshold:  baseThreshold,
					K:               5 + j%10, // Vary K
					IncludeVector:   j%2 == 0,
					IncludeMetadata: j%3 == 0,
					IncludeContent:  j%4 == 0,
					Timeout:         1000 + j%2000, // Vary timeout
					VectorUsing:     "dense",       // Specify named vector for collections with sparse vector support
				}

				result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					select {
					case errors <- err:
					default:
					}
				} else {
					atomic.AddInt64(&successCount, 1)
					select {
					case results <- result:
					default:
					}

					// Validate result
					if result != nil {
						for _, doc := range result.Documents {
							if doc.Score < opts.ScoreThreshold {
								t.Errorf("Worker %d: Score %f below threshold %f",
									workerID, doc.Score, opts.ScoreThreshold)
							}
						}
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)
	close(errors)

	duration := time.Since(startTime)
	successRate := float64(successCount) / float64(totalOperations) * 100
	errorRate := float64(errorCount) / float64(totalOperations) * 100
	opsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("Concurrent stress test completed in %v", duration)
	t.Logf("Total operations: %d", totalOperations)
	t.Logf("Successful operations: %d (%.2f%%)", successCount, successRate)
	t.Logf("Failed operations: %d (%.2f%%)", errorCount, errorRate)
	t.Logf("Operations per second: %.2f", opsPerSecond)

	// Collect and report any errors
	var errorSamples []string
	for err := range errors {
		errorSamples = append(errorSamples, err.Error())
		if len(errorSamples) >= 5 {
			break
		}
	}

	if len(errorSamples) > 0 {
		t.Logf("Sample errors:")
		for i, errMsg := range errorSamples {
			t.Logf("  %d: %s", i+1, errMsg)
		}
	}

	// Validate results
	if successCount+errorCount != int64(totalOperations) {
		t.Errorf("Operations count mismatch: success=%d + error=%d != total=%d",
			successCount, errorCount, totalOperations)
	}
	if successRate <= 95.0 {
		t.Errorf("Success rate too low: %.2f%%, expected > 95%%", successRate)
	}
	if opsPerSecond <= 10.0 {
		t.Errorf("Operations per second too low: %.2f, expected > 10", opsPerSecond)
	}

	// Process some results to verify they're valid
	resultCount := 0
	for result := range results {
		if result == nil {
			t.Error("Result should not be nil")
		} else if result.QueryTime < 0 {
			t.Error("QueryTime should be non-negative")
		}

		resultCount++
		if resultCount >= 100 { // Sample first 100 results
			break
		}
	}

	t.Logf("Validation completed on %d sample results", resultCount)
}

// TestSearchWithScoreThreshold_NamedVectorSelection tests the named vector selection functionality
func TestSearchWithScoreThreshold_NamedVectorSelection(t *testing.T) {
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
	collectionName := testDataSet.CollectionName

	tests := []struct {
		name        string
		vectorUsing string
		expectError bool
	}{
		{
			name:        "Use Dense Vector Explicitly",
			vectorUsing: "dense",
			expectError: false,
		},
		{
			name:        "Use Sparse Vector (should fallback to dense)",
			vectorUsing: "sparse",
			expectError: false, // Should fallback to dense since we only have dense vectors in test data
		},
		{
			name:        "Use Non-existent Vector (should fallback to dense)",
			vectorUsing: "nonexistent",
			expectError: false, // Should fallback to dense
		},
		{
			name:        "Empty VectorUsing (should default to dense)",
			vectorUsing: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &types.ScoreThresholdOptions{
				CollectionName:  collectionName,
				QueryVector:     queryVector,
				ScoreThreshold:  0.1,
				K:               5,
				IncludeVector:   true,
				IncludeMetadata: true,
				IncludeContent:  true,
				VectorUsing:     tt.vectorUsing,
			}

			result, err := env.Store.SearchWithScoreThreshold(ctx, opts)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("SearchWithScoreThreshold failed: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			// Verify that results include vectors when requested
			for _, doc := range result.Documents {
				if len(doc.Document.Vector) == 0 {
					t.Error("Document vector should not be empty when IncludeVector is true")
				}
				if doc.Document.ID == "" {
					t.Error("Document ID should not be empty")
				}
				if doc.Document.Metadata == nil {
					t.Error("Document metadata should not be nil when IncludeMetadata is true")
				}
				if doc.Document.Content == "" {
					t.Error("Document content should not be empty when IncludeContent is true")
				}
			}

			t.Logf("Test '%s' with VectorUsing='%s' returned %d results",
				tt.name, tt.vectorUsing, len(result.Documents))
		})
	}
}
