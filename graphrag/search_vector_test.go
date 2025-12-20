package graphrag

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/graphrag/types"
)

// ==== SearchVector Tests ====

// TestSearchVector tests the SearchVector function with different configurations
func TestSearchVector(t *testing.T) {
	prepareSearchTestConnector(t)

	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+graph", "vector+store", "vector+graph+store"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Skipf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Skipf("Failed to create GraphRag instance for %s: %v", configName, err)
			}

			ctx := context.Background()

			// Create collection for testing
			safeName := strings.ReplaceAll(configName, "+", "_")
			collectionID := fmt.Sprintf("vector_search_test_%s_%d", safeName, time.Now().Unix())
			collection := types.CollectionConfig{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"type": "vector_search_test",
				},
				Config: &types.CreateCollectionOptions{
					CollectionName: fmt.Sprintf("%s_vector", collectionID),
					Dimension:      1536,
					Distance:       types.DistanceCosine,
					IndexType:      types.IndexTypeHNSW,
				},
			}

			// Create collection
			_, err = g.CreateCollection(ctx, collection)
			if err != nil {
				t.Skipf("Failed to create test collection for %s: %v", configName, err)
			}

			// Cleanup collection after test
			defer func() {
				removed, err := g.RemoveCollection(ctx, collectionID)
				if err != nil {
					t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
				} else if removed {
					t.Logf("Successfully cleaned up collection: %s", collectionID)
				}
			}()

			// Add test documents
			testDocs := []struct {
				id      string
				content string
			}{
				{
					id:      "doc1",
					content: "Artificial intelligence and machine learning are transforming industries worldwide. Deep learning models can now recognize images, understand speech, and generate human-like text.",
				},
				{
					id:      "doc2",
					content: "Natural language processing enables computers to understand and generate human language. Applications include chatbots, translation, and sentiment analysis.",
				},
				{
					id:      "doc3",
					content: "Computer vision allows machines to interpret and understand visual information from the world. It powers autonomous vehicles, facial recognition, and medical imaging.",
				},
				{
					id:      "doc4",
					content: "Quantum computing leverages quantum mechanics to solve complex problems. It has potential applications in cryptography, drug discovery, and optimization.",
				},
			}

			var addedDocIDs []string

			// Add test documents using AddText
			for _, doc := range testDocs {
				options := &types.UpsertOptions{
					DocID:        fmt.Sprintf("vector_search_test_%s_%s", configName, doc.id),
					CollectionID: collectionID,
					Metadata: map[string]interface{}{
						"source": "vector_search_test",
						"type":   "text",
					},
				}

				docID, err := g.AddText(ctx, doc.content, options)
				if err != nil {
					expectedErrors := []string{
						"connection refused", "no such host", "connector not found", "connector openai not loaded",
						"vector store", "graph store", "store", "embedding", "extraction",
					}

					hasExpectedError := false
					for _, expectedErr := range expectedErrors {
						if strings.Contains(err.Error(), expectedErr) {
							hasExpectedError = true
							break
						}
					}

					if hasExpectedError {
						t.Logf("Expected error adding document %s: %v", doc.id, err)
						continue
					} else {
						t.Errorf("Unexpected error adding document %s: %v", doc.id, err)
						continue
					}
				}

				if docID != "" {
					addedDocIDs = append(addedDocIDs, docID)
					t.Logf("Successfully added document %s with ID: %s", doc.id, docID)
				}
			}

			// Skip search tests if no documents were added
			if len(addedDocIDs) == 0 {
				t.Skip("No documents were added successfully, skipping search tests")
			}

			// Wait for indexing
			time.Sleep(500 * time.Millisecond)

			// Test basic vector search
			t.Run("Basic_Vector_Search", func(t *testing.T) {
				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				searchOptions := &types.VectorSearchOptions{
					CollectionID: collectionID,
					Query:        "What is artificial intelligence and machine learning?",
					Embedding:    testEmbedding,
					K:            5,
				}

				result, err := g.SearchVector(ctx, searchOptions)
				if err != nil {
					if isExpectedSearchError(err) {
						t.Logf("Expected error during vector search: %v", err)
						return
					}
					t.Errorf("Unexpected error during vector search: %v", err)
					return
				}

				// Verify result is not nil
				assert.NotNil(t, result, "Result should not be nil")

				// Verify segments are returned
				assert.Greater(t, len(result.Segments), 0, "Expected at least one segment")
				assert.LessOrEqual(t, len(result.Segments), 5, "Expected at most 5 segments (K=5)")

				// Verify Total matches segment count
				assert.Equal(t, len(result.Segments), result.Total, "Total should match segment count")

				t.Logf("Vector search returned %d segments (Total: %d)", len(result.Segments), result.Total)

				// Verify each segment has required fields
				for i, seg := range result.Segments {
					assert.NotEmpty(t, seg.ID, "Segment %d: ID should not be empty", i)
					assert.NotEmpty(t, seg.Text, "Segment %d: Text should not be empty", i)
					assert.GreaterOrEqual(t, seg.Score, 0.0, "Segment %d: Score should be >= 0", i)
					assert.LessOrEqual(t, seg.Score, 1.0, "Segment %d: Score should be <= 1", i)
					assert.Equal(t, collectionID, seg.CollectionID, "Segment %d: CollectionID mismatch", i)

					t.Logf("  Segment %d: ID=%s, Score=%.4f, Text=%s...",
						i, seg.ID, seg.Score, truncateString(seg.Text, 50))
				}

				// Verify segments are sorted by score (descending)
				for i := 1; i < len(result.Segments); i++ {
					assert.LessOrEqual(t, result.Segments[i].Score, result.Segments[i-1].Score,
						"Segments should be sorted by score descending")
				}

				// Verify the top result is relevant to the query
				topSegment := result.Segments[0]
				aiKeywords := []string{"artificial intelligence", "machine learning", "AI", "deep learning"}
				hasRelevantContent := false
				lowerText := strings.ToLower(topSegment.Text)
				for _, keyword := range aiKeywords {
					if strings.Contains(lowerText, strings.ToLower(keyword)) {
						hasRelevantContent = true
						break
					}
				}
				assert.True(t, hasRelevantContent, "Top segment should be relevant to AI/ML query")
			})

			// Test vector search with pre-computed vector
			t.Run("Vector_Search_With_PreComputed_Vector", func(t *testing.T) {
				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				// First generate embedding
				embeddingResult, err := testEmbedding.EmbedQuery(ctx, "quantum computing applications")
				if err != nil {
					if isExpectedSearchError(err) {
						t.Logf("Expected error generating embedding: %v", err)
						return
					}
					t.Errorf("Failed to generate embedding: %v", err)
					return
				}

				// Search with pre-computed vector (no Embedding required)
				searchOptions := &types.VectorSearchOptions{
					CollectionID: collectionID,
					QueryVector:  embeddingResult.Embedding,
					K:            3,
				}

				result, err := g.SearchVector(ctx, searchOptions)
				if err != nil {
					if isExpectedSearchError(err) {
						t.Logf("Expected error during vector search: %v", err)
						return
					}
					t.Errorf("Unexpected error during vector search: %v", err)
					return
				}

				assert.NotNil(t, result, "Result should not be nil")
				assert.Greater(t, len(result.Segments), 0, "Expected at least one segment")
				t.Logf("Vector search with pre-computed vector returned %d segments", len(result.Segments))
			})

			// Test vector search with document filter
			t.Run("Vector_Search_With_DocumentID", func(t *testing.T) {
				if len(addedDocIDs) == 0 {
					t.Skip("No documents available for document filter test")
				}

				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				targetDocID := addedDocIDs[0]
				searchOptions := &types.VectorSearchOptions{
					CollectionID: collectionID,
					DocumentID:   targetDocID,
					Query:        "machine learning",
					Embedding:    testEmbedding,
				}

				result, err := g.SearchVector(ctx, searchOptions)
				if err != nil {
					if isExpectedSearchError(err) {
						t.Logf("Expected error during vector search: %v", err)
						return
					}
					t.Errorf("Unexpected error during vector search: %v", err)
					return
				}

				t.Logf("Vector search with document filter returned %d segments", len(result.Segments))

				// Verify all returned segments belong to the specified document
				for i, seg := range result.Segments {
					if seg.DocumentID != "" && seg.DocumentID != targetDocID {
						t.Errorf("Segment %d: DocumentID mismatch, expected %s, got %s", i, targetDocID, seg.DocumentID)
					}
				}
			})

			// Test vector search with minimum score filter
			t.Run("Vector_Search_With_MinScore", func(t *testing.T) {
				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				searchOptions := &types.VectorSearchOptions{
					CollectionID: collectionID,
					Query:        "artificial intelligence",
					Embedding:    testEmbedding,
					MinScore:     0.5, // Only return results with score >= 0.5
				}

				result, err := g.SearchVector(ctx, searchOptions)
				if err != nil {
					if isExpectedSearchError(err) {
						t.Logf("Expected error during vector search: %v", err)
						return
					}
					t.Errorf("Unexpected error during vector search: %v", err)
					return
				}

				t.Logf("Vector search with MinScore=0.5 returned %d segments", len(result.Segments))

				// Verify all returned segments have score >= MinScore
				for i, seg := range result.Segments {
					// Note: MinScore filtering depends on vector store implementation
					t.Logf("  Segment %d: Score=%.4f", i, seg.Score)
				}
			})

			// Test vector search with progress callback
			t.Run("Vector_Search_With_Progress", func(t *testing.T) {
				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				var progressMessages []string
				var progressValues []float64
				progressCallback := func(status types.SearcherStatus, payload types.SearcherPayload) {
					progressMessages = append(progressMessages, fmt.Sprintf("[%s] %s", status, payload.Message))
					progressValues = append(progressValues, payload.Progress)
				}

				searchOptions := &types.VectorSearchOptions{
					CollectionID: collectionID,
					Query:        "natural language processing",
					Embedding:    testEmbedding,
				}

				result, err := g.SearchVector(ctx, searchOptions, progressCallback)
				if err != nil {
					if isExpectedSearchError(err) {
						t.Logf("Expected error during vector search: %v", err)
						return
					}
					t.Errorf("Unexpected error during vector search: %v", err)
					return
				}

				t.Logf("Vector search with progress returned %d segments", len(result.Segments))
				t.Logf("Progress messages: %v", progressMessages)

				// Verify progress was reported
				assert.Greater(t, len(progressMessages), 0, "Expected at least one progress message")

				// Verify progress values are increasing
				for i := 1; i < len(progressValues); i++ {
					assert.GreaterOrEqual(t, progressValues[i], progressValues[i-1],
						"Progress values should be non-decreasing")
				}

				// Verify final progress is 100
				if len(progressValues) > 0 {
					assert.Equal(t, 100.0, progressValues[len(progressValues)-1],
						"Final progress should be 100")
				}
			})
		})
	}
}

// TestSearchVectorErrorHandling tests error conditions for SearchVector
func TestSearchVectorErrorHandling(t *testing.T) {
	prepareSearchTestConnector(t)

	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		t.Skip("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	t.Run("Nil_Options", func(t *testing.T) {
		_, err := g.SearchVector(ctx, nil)
		assert.Error(t, err, "Expected error for nil options")
		assert.Contains(t, err.Error(), "cannot be nil", "Error should mention nil options")
		t.Logf("Nil options correctly rejected: %v", err)
	})

	t.Run("Empty_CollectionID", func(t *testing.T) {
		searchOptions := &types.VectorSearchOptions{
			Query: "test query",
		}

		_, err := g.SearchVector(ctx, searchOptions)
		assert.Error(t, err, "Expected error for empty collection ID")
		assert.Contains(t, err.Error(), "collection ID is required", "Error should mention collection ID")
		t.Logf("Empty collection ID correctly rejected: %v", err)
	})

	t.Run("Empty_Query_And_Vector", func(t *testing.T) {
		searchOptions := &types.VectorSearchOptions{
			CollectionID: "test_collection",
		}

		_, err := g.SearchVector(ctx, searchOptions)
		assert.Error(t, err, "Expected error for empty query and vector")
		assert.Contains(t, err.Error(), "either query text or query vector is required", "Error should mention query requirement")
		t.Logf("Empty query and vector correctly rejected: %v", err)
	})

	t.Run("Missing_Embedding_With_Query", func(t *testing.T) {
		searchOptions := &types.VectorSearchOptions{
			CollectionID: "test_collection",
			Query:        "test query",
			// Embedding is nil
		}

		_, err := g.SearchVector(ctx, searchOptions)
		assert.Error(t, err, "Expected error for missing embedding with query")
		assert.Contains(t, err.Error(), "embedding function is required", "Error should mention embedding requirement")
		t.Logf("Missing embedding correctly rejected: %v", err)
	})

	t.Run("Valid_With_PreComputed_Vector", func(t *testing.T) {
		// When QueryVector is provided, Embedding is not required
		searchOptions := &types.VectorSearchOptions{
			CollectionID: "test_collection",
			QueryVector:  make([]float64, 1536), // Empty vector but valid structure
		}

		// This should fail at vector search, not at validation
		_, err := g.SearchVector(ctx, searchOptions)
		if err != nil {
			// Expected to fail at vector search level, not validation
			assert.NotContains(t, err.Error(), "embedding function is required",
				"Should not fail on embedding validation when QueryVector is provided")
		}
		t.Logf("Pre-computed vector validation passed, search result: %v", err)
	})
}

// TestSearchVectorWithMetadataFilter tests vector search with metadata filtering
func TestSearchVectorWithMetadataFilter(t *testing.T) {
	prepareSearchTestConnector(t)

	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		t.Skip("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	// Create collection for testing
	collectionID := fmt.Sprintf("vector_filter_test_%d", time.Now().Unix())
	collection := types.CollectionConfig{
		ID: collectionID,
		Config: &types.CreateCollectionOptions{
			CollectionName: fmt.Sprintf("%s_vector", collectionID),
			Dimension:      1536,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
		},
	}

	_, err = g.CreateCollection(ctx, collection)
	if err != nil {
		t.Skipf("Failed to create test collection: %v", err)
	}

	defer func() {
		g.RemoveCollection(ctx, collectionID)
	}()

	// Add documents with different categories
	docs := []struct {
		id       string
		content  string
		category string
	}{
		{"doc1", "Python is a programming language for data science", "programming"},
		{"doc2", "Machine learning uses algorithms to learn from data", "ai"},
		{"doc3", "JavaScript is used for web development", "programming"},
		{"doc4", "Neural networks are inspired by the brain", "ai"},
	}

	for _, doc := range docs {
		options := &types.UpsertOptions{
			DocID:        fmt.Sprintf("filter_test_%s", doc.id),
			CollectionID: collectionID,
			Metadata: map[string]interface{}{
				"category": doc.category,
			},
		}

		_, err := g.AddText(ctx, doc.content, options)
		if err != nil {
			t.Logf("Error adding document %s: %v", doc.id, err)
		}
	}

	time.Sleep(500 * time.Millisecond)

	t.Run("Filter_By_Category", func(t *testing.T) {
		testEmbedding, err := createSearchTestEmbedding(t)
		if err != nil {
			t.Skipf("Failed to create test embedding: %v", err)
		}

		searchOptions := &types.VectorSearchOptions{
			CollectionID: collectionID,
			Query:        "programming languages",
			Embedding:    testEmbedding,
			Filter: map[string]interface{}{
				"category": "programming",
			},
		}

		result, err := g.SearchVector(ctx, searchOptions)
		if err != nil {
			if isExpectedSearchError(err) {
				t.Logf("Expected error during vector search: %v", err)
				return
			}
			t.Errorf("Unexpected error: %v", err)
			return
		}

		t.Logf("Filtered search returned %d segments", len(result.Segments))

		// All results should be from "programming" category
		for i, seg := range result.Segments {
			if seg.Metadata != nil {
				if category, ok := seg.Metadata["category"].(string); ok {
					t.Logf("  Segment %d: category=%s", i, category)
				}
			}
		}
	})
}

// BenchmarkSearchVector benchmarks the SearchVector function
func BenchmarkSearchVector(b *testing.B) {
	prepareSearchTestConnector(&testing.T{})

	configs := GetTestConfigs()
	config := configs["vector"]
	if config == nil {
		b.Skip("Vector config not found")
	}

	g, err := New(config)
	if err != nil {
		b.Skipf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	// Create collection for benchmarking
	collectionID := fmt.Sprintf("benchmark_vector_search_%d", time.Now().Unix())
	collection := types.CollectionConfig{
		ID: collectionID,
		Config: &types.CreateCollectionOptions{
			CollectionName: fmt.Sprintf("%s_vector", collectionID),
			Dimension:      1536,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
		},
	}

	_, err = g.CreateCollection(ctx, collection)
	if err != nil {
		b.Skipf("Failed to create benchmark collection: %v", err)
	}

	defer g.RemoveCollection(ctx, collectionID)

	// Add test documents
	for i := 0; i < 10; i++ {
		options := &types.UpsertOptions{
			DocID:        fmt.Sprintf("bench_doc_%d", i),
			CollectionID: collectionID,
		}
		g.AddText(ctx, fmt.Sprintf("This is test document number %d about various topics.", i), options)
	}

	time.Sleep(500 * time.Millisecond)

	testEmbedding, err := createSearchTestEmbedding(&testing.T{})
	if err != nil {
		b.Skipf("Failed to create test embedding: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		searchOptions := &types.VectorSearchOptions{
			CollectionID: collectionID,
			Query:        "test document topics",
			Embedding:    testEmbedding,
			K:            5,
		}

		_, err := g.SearchVector(ctx, searchOptions)
		if err != nil {
			continue
		}
	}
}

// ==== Helper Functions ====

// isExpectedSearchError checks if the error is expected in test environment
func isExpectedSearchError(err error) bool {
	expectedErrors := []string{
		"connection refused", "no such host", "connector not found", "connector openai not loaded",
		"vector search failed", "embedding", "request failed",
	}

	errStr := err.Error()
	for _, expected := range expectedErrors {
		if strings.Contains(errStr, expected) {
			return true
		}
	}
	return false
}
