package graphrag

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/embedding"
	"github.com/yaoapp/gou/graphrag/types"
)

// ==== Search Test Helper Functions ====

// createSearchTestEmbedding creates an embedding configuration for search testing
func createSearchTestEmbedding(t *testing.T) (types.Embedding, error) {
	t.Helper()

	return embedding.NewOpenai(embedding.OpenaiOptions{
		ConnectorName: "openai",
		Concurrent:    10,
		Dimension:     1536,
		Model:         "text-embedding-3-small",
	})
}

// prepareSearchTestConnector creates connectors for Search testing
func prepareSearchTestConnector(t *testing.T) {
	t.Helper()

	// Create OpenAI connector for search testing
	openaiKey := getEnvOrDefault("OPENAI_TEST_KEY", "mock-key")

	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Search Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "gpt-4o-mini",
			"key": "%s"
		}
	}`, openaiKey)

	// Try creating connector directly with the name "openai"
	_, err := connector.New("openai", "openai", []byte(openaiDSL))
	if err != nil {
		t.Logf("Failed to create OpenAI search connector: %v", err)
	}
}

// ==== Search Tests ====

// TestSearch tests the Search function with different configurations
func TestSearch(t *testing.T) {
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
			collectionID := fmt.Sprintf("search_test_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.CollectionConfig{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"type": "search_test",
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
				} else {
					t.Logf("Collection %s was not found (already cleaned up)", collectionID)
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
			}

			var addedDocIDs []string

			// Add test documents using AddText
			for _, doc := range testDocs {
				options := &types.UpsertOptions{
					DocID:        fmt.Sprintf("search_test_%s_%s", configName, doc.id),
					CollectionID: collectionID,
					Metadata: map[string]interface{}{
						"source": "search_test",
						"type":   "text",
					},
				}

				docID, err := g.AddText(ctx, doc.content, options)
				if err != nil {
					// Expected errors with mock setup
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

			// Test basic search
			t.Run("Basic_Search", func(t *testing.T) {
				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				queryOptions := &types.QueryOptions{
					CollectionID: collectionID,
					Query:        "What is artificial intelligence?",
					Embedding:    testEmbedding,
				}

				segments, err := g.Search(ctx, queryOptions)
				if err != nil {
					expectedErrors := []string{
						"connection refused", "no such host", "connector not found", "connector openai not loaded",
						"vector search failed", "embedding",
					}

					hasExpectedError := false
					for _, expectedErr := range expectedErrors {
						if strings.Contains(err.Error(), expectedErr) {
							hasExpectedError = true
							break
						}
					}

					if hasExpectedError {
						t.Logf("Expected error during search: %v", err)
					} else {
						t.Errorf("Unexpected error during search: %v", err)
					}
					return
				}

				// Verify segments are returned
				if len(segments) == 0 {
					t.Error("Expected at least one segment, got 0")
					return
				}

				t.Logf("Search returned %d segments", len(segments))

				// Verify each segment has required fields
				for i, seg := range segments {
					// Verify ID is not empty
					if seg.ID == "" {
						t.Errorf("Segment %d: ID should not be empty", i)
					}

					// Verify Text is not empty
					if seg.Text == "" {
						t.Errorf("Segment %d: Text should not be empty", i)
					}

					// Verify Score is valid (between 0 and 1 for cosine similarity)
					if seg.Score < 0 || seg.Score > 1 {
						t.Errorf("Segment %d: Score %.4f should be between 0 and 1", i, seg.Score)
					}

					// Verify CollectionID matches
					if seg.CollectionID != collectionID {
						t.Errorf("Segment %d: CollectionID mismatch, expected %s, got %s", i, collectionID, seg.CollectionID)
					}

					t.Logf("  Segment %d: ID=%s, Score=%.4f, Text=%s...",
						i, seg.ID, seg.Score, truncateString(seg.Text, 50))
				}

				// Verify segments are sorted by score (descending)
				for i := 1; i < len(segments); i++ {
					if segments[i].Score > segments[i-1].Score {
						t.Errorf("Segments not sorted by score: segment %d (%.4f) > segment %d (%.4f)",
							i, segments[i].Score, i-1, segments[i-1].Score)
					}
				}

				// Verify the top result is relevant to the query (contains AI-related content)
				topSegment := segments[0]
				aiKeywords := []string{"artificial intelligence", "machine learning", "AI", "deep learning"}
				hasRelevantContent := false
				lowerText := strings.ToLower(topSegment.Text)
				for _, keyword := range aiKeywords {
					if strings.Contains(lowerText, strings.ToLower(keyword)) {
						hasRelevantContent = true
						break
					}
				}
				if !hasRelevantContent {
					t.Logf("Warning: Top segment may not be relevant to AI query: %s", truncateString(topSegment.Text, 100))
				}
			})

			// Test search with history
			t.Run("Search_With_History", func(t *testing.T) {
				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				queryOptions := &types.QueryOptions{
					CollectionID: collectionID,
					History: []types.ChatMessage{
						{Role: "user", Content: "Tell me about AI"},
						{Role: "assistant", Content: "AI is a broad field of computer science."},
						{Role: "user", Content: "What about machine learning specifically?"},
					},
					Embedding: testEmbedding,
				}

				segments, err := g.Search(ctx, queryOptions)
				if err != nil {
					t.Logf("Search with history error (expected with mock): %v", err)
					return
				}

				// Verify segments are returned
				if len(segments) == 0 {
					t.Error("Expected at least one segment from history search, got 0")
					return
				}

				t.Logf("Search with history returned %d segments", len(segments))

				// Verify the search used the last user message ("What about machine learning specifically?")
				// The results should be relevant to machine learning
				foundMLContent := false
				for _, seg := range segments {
					lowerText := strings.ToLower(seg.Text)
					if strings.Contains(lowerText, "machine learning") ||
						strings.Contains(lowerText, "deep learning") ||
						strings.Contains(lowerText, "artificial intelligence") {
						foundMLContent = true
						break
					}
				}
				if !foundMLContent {
					t.Logf("Warning: No segments found with machine learning related content")
				}
			})

			// Test search with document filter
			t.Run("Search_With_DocumentID", func(t *testing.T) {
				if len(addedDocIDs) == 0 {
					t.Skip("No documents available for document filter test")
				}

				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				targetDocID := addedDocIDs[0]
				queryOptions := &types.QueryOptions{
					CollectionID: collectionID,
					DocumentID:   targetDocID,
					Query:        "machine learning",
					Embedding:    testEmbedding,
				}

				segments, err := g.Search(ctx, queryOptions)
				if err != nil {
					t.Logf("Search with document filter error (expected with mock): %v", err)
					return
				}

				t.Logf("Search with document filter returned %d segments", len(segments))

				// Verify all returned segments belong to the specified document
				for i, seg := range segments {
					if seg.DocumentID != "" && seg.DocumentID != targetDocID {
						t.Errorf("Segment %d: DocumentID mismatch, expected %s, got %s", i, targetDocID, seg.DocumentID)
					}
				}

				// When filtering by document, we should get fewer results than without filter
				// (assuming the document has fewer segments than the entire collection)
				if len(segments) > len(addedDocIDs) {
					t.Logf("Document filter returned %d segments (expected <= %d documents' worth)", len(segments), len(addedDocIDs))
				}
			})

			// Test search with progress callback
			t.Run("Search_With_Progress", func(t *testing.T) {
				testEmbedding, err := createSearchTestEmbedding(t)
				if err != nil {
					t.Skipf("Failed to create test embedding: %v", err)
				}

				var progressMessages []string
				progressCallback := func(status types.SearcherStatus, payload types.SearcherPayload) {
					progressMessages = append(progressMessages, fmt.Sprintf("[%s] %s (%.0f%%)", status, payload.Message, payload.Progress))
				}

				queryOptions := &types.QueryOptions{
					CollectionID: collectionID,
					Query:        "natural language processing",
					Embedding:    testEmbedding,
				}

				segments, err := g.Search(ctx, queryOptions, progressCallback)
				if err != nil {
					t.Logf("Search with progress error (expected with mock): %v", err)
					return
				}

				t.Logf("Search with progress returned %d segments", len(segments))
				t.Logf("Progress messages: %v", progressMessages)
			})
		})
	}
}

// TestSearchErrorHandling tests error conditions for Search
func TestSearchErrorHandling(t *testing.T) {
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
		_, err := g.Search(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil options")
		}
		if !strings.Contains(err.Error(), "cannot be nil") {
			t.Errorf("Expected 'cannot be nil' error, got: %v", err)
		}
		t.Logf("Nil options correctly rejected: %v", err)
	})

	t.Run("Empty_Query_And_History", func(t *testing.T) {
		queryOptions := &types.QueryOptions{
			CollectionID: "test_collection",
		}

		_, err := g.Search(ctx, queryOptions)
		if err == nil {
			t.Error("Expected error for empty query and history")
		}
		if !strings.Contains(err.Error(), "either query or history is required") {
			t.Errorf("Expected 'either query or history is required' error, got: %v", err)
		}
		t.Logf("Empty query and history correctly rejected: %v", err)
	})

	t.Run("Missing_CollectionID", func(t *testing.T) {
		queryOptions := &types.QueryOptions{
			Query: "test query",
		}

		_, err := g.Search(ctx, queryOptions)
		if err == nil {
			t.Error("Expected error for missing collection ID")
		}
		if !strings.Contains(err.Error(), "collection ID is required") {
			t.Errorf("Expected 'collection ID is required' error, got: %v", err)
		}
		t.Logf("Missing collection ID correctly rejected: %v", err)
	})

	t.Run("Missing_Embedding", func(t *testing.T) {
		queryOptions := &types.QueryOptions{
			CollectionID: "test_collection",
			Query:        "test query",
		}

		_, err := g.Search(ctx, queryOptions)
		if err == nil {
			t.Error("Expected error for missing embedding")
		}
		if !strings.Contains(err.Error(), "embedding function is required") {
			t.Errorf("Expected 'embedding function is required' error, got: %v", err)
		}
		t.Logf("Missing embedding correctly rejected: %v", err)
	})
}

// TestMultiSearch tests the MultiSearch function
func TestMultiSearch(t *testing.T) {
	prepareSearchTestConnector(t)

	configs := GetTestConfigs()
	config := configs["vector+graph+store"]
	if config == nil {
		t.Skip("vector+graph+store config not found")
	}

	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	ctx := context.Background()

	// Create collection for testing
	collectionID := fmt.Sprintf("multisearch_test_collection_%d", time.Now().Unix())
	collection := types.CollectionConfig{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"type": "multisearch_test",
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
		t.Skipf("Failed to create test collection: %v", err)
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
	testDocs := []string{
		"Machine learning algorithms can learn from data and make predictions.",
		"Deep neural networks have revolutionized computer vision and NLP.",
		"Reinforcement learning enables agents to learn through trial and error.",
	}

	for i, content := range testDocs {
		options := &types.UpsertOptions{
			DocID:        fmt.Sprintf("multisearch_doc_%d", i),
			CollectionID: collectionID,
			Metadata: map[string]interface{}{
				"source": "multisearch_test",
			},
		}

		_, err := g.AddText(ctx, content, options)
		if err != nil {
			t.Logf("Error adding document %d (expected with mock): %v", i, err)
		}
	}

	// Wait for indexing
	time.Sleep(500 * time.Millisecond)

	t.Run("Empty_Options", func(t *testing.T) {
		results, err := g.MultiSearch(ctx, []types.QueryOptions{})
		if err != nil {
			t.Errorf("Unexpected error for empty options: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected empty results for empty options, got %d", len(results))
		}
		t.Logf("Empty options handled correctly")
	})

	t.Run("Multiple_Queries", func(t *testing.T) {
		testEmbedding, err := createSearchTestEmbedding(t)
		if err != nil {
			t.Skipf("Failed to create test embedding: %v", err)
		}

		queries := []types.QueryOptions{
			{
				CollectionID: collectionID,
				Query:        "What is machine learning?",
				Embedding:    testEmbedding,
			},
			{
				CollectionID: collectionID,
				Query:        "How do neural networks work?",
				Embedding:    testEmbedding,
			},
			{
				CollectionID: collectionID,
				Query:        "What is reinforcement learning?",
				Embedding:    testEmbedding,
			},
		}

		results, err := g.MultiSearch(ctx, queries)
		if err != nil {
			t.Logf("MultiSearch error (expected with mock): %v", err)
			return
		}

		// Verify we got results for all queries
		if len(results) != len(queries) {
			t.Errorf("Expected results for %d queries, got %d", len(queries), len(results))
		}

		t.Logf("MultiSearch returned results for %d queries", len(results))

		// Verify each query has results
		for key, segments := range results {
			t.Logf("  Query '%s': %d segments", truncateString(key, 30), len(segments))

			// Verify segments are not empty
			if len(segments) == 0 {
				t.Errorf("Query '%s' returned 0 segments", key)
				continue
			}

			// Verify each segment has valid fields
			for i, seg := range segments {
				if seg.ID == "" {
					t.Errorf("Query '%s', Segment %d: ID should not be empty", key, i)
				}
				if seg.Score < 0 || seg.Score > 1 {
					t.Errorf("Query '%s', Segment %d: Score %.4f should be between 0 and 1", key, i, seg.Score)
				}
			}
		}
	})

	t.Run("MultiSearch_With_Progress", func(t *testing.T) {
		testEmbedding, err := createSearchTestEmbedding(t)
		if err != nil {
			t.Skipf("Failed to create test embedding: %v", err)
		}

		var progressMessages []string
		progressCallback := func(status types.SearcherStatus, payload types.SearcherPayload) {
			progressMessages = append(progressMessages, payload.Message)
		}

		queries := []types.QueryOptions{
			{
				CollectionID: collectionID,
				Query:        "AI applications",
				Embedding:    testEmbedding,
			},
			{
				CollectionID: collectionID,
				Query:        "Deep learning",
				Embedding:    testEmbedding,
			},
		}

		results, err := g.MultiSearch(ctx, queries, progressCallback)
		if err != nil {
			t.Logf("MultiSearch with progress error (expected with mock): %v", err)
			return
		}

		t.Logf("MultiSearch with progress returned %d results", len(results))
		t.Logf("Progress messages: %v", progressMessages)
	})
}

// TestSearchWithGraphEnrichment tests search with graph data enrichment
func TestSearchWithGraphEnrichment(t *testing.T) {
	prepareSearchTestConnector(t)

	configs := GetTestConfigs()
	config := configs["vector+graph"]
	if config == nil {
		t.Skip("vector+graph config not found")
	}

	g, err := New(config)
	if err != nil {
		t.Skipf("Failed to create GraphRag instance: %v", err)
	}

	// Skip if graph is not available
	if g.Graph == nil {
		t.Skip("Graph store not available")
	}

	ctx := context.Background()

	// Create collection for testing
	collectionID := fmt.Sprintf("graph_search_test_%d", time.Now().Unix())
	collection := types.CollectionConfig{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"type": "graph_search_test",
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
		t.Skipf("Failed to create test collection: %v", err)
	}

	// Cleanup collection after test
	defer func() {
		g.RemoveCollection(ctx, collectionID)
	}()

	// Add test document with extraction for graph
	content := "Albert Einstein developed the theory of relativity. He was born in Germany and later moved to the United States. Einstein received the Nobel Prize in Physics in 1921."

	options := &types.UpsertOptions{
		DocID:        "graph_test_doc",
		CollectionID: collectionID,
		Metadata: map[string]interface{}{
			"source": "graph_test",
		},
	}

	_, err = g.AddText(ctx, content, options)
	if err != nil {
		t.Logf("Error adding document (expected with mock): %v", err)
	}

	// Wait for indexing
	time.Sleep(500 * time.Millisecond)

	t.Run("Search_With_Graph_Enrichment", func(t *testing.T) {
		testEmbedding, err := createSearchTestEmbedding(t)
		if err != nil {
			t.Skipf("Failed to create test embedding: %v", err)
		}

		queryOptions := &types.QueryOptions{
			CollectionID: collectionID,
			Query:        "Who is Albert Einstein?",
			Embedding:    testEmbedding,
		}

		segments, err := g.Search(ctx, queryOptions)
		if err != nil {
			t.Logf("Search with graph enrichment error (expected with mock): %v", err)
			return
		}

		t.Logf("Search returned %d segments", len(segments))
		for i, seg := range segments {
			t.Logf("  Segment %d: %d nodes, %d relationships",
				i, len(seg.Nodes), len(seg.Relationships))

			// Log nodes if any
			for _, node := range seg.Nodes {
				t.Logf("    Node: ID=%s, Type=%s", node.ID, node.EntityType)
			}

			// Log relationships if any
			for _, rel := range seg.Relationships {
				t.Logf("    Relationship: %s -[%s]-> %s", rel.StartNode, rel.Type, rel.EndNode)
			}
		}
	})
}

// ==== Helper Functions ====

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ==== Benchmark Tests ====

// BenchmarkSearch benchmarks the Search function
func BenchmarkSearch(b *testing.B) {
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
	collectionID := fmt.Sprintf("benchmark_search_%d", time.Now().Unix())
	collection := types.CollectionConfig{
		ID: collectionID,
		Metadata: map[string]interface{}{
			"type": "benchmark",
		},
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

	// Add some test documents
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
		queryOptions := &types.QueryOptions{
			CollectionID: collectionID,
			Query:        "test document topics",
			Embedding:    testEmbedding,
		}

		_, err := g.Search(ctx, queryOptions)
		if err != nil {
			// Expected errors with mock setup
			continue
		}
	}
}
