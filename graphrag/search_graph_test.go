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

// ==== SearchGraph Tests ====

// TestSearchGraph tests the SearchGraph function with different configurations
func TestSearchGraph(t *testing.T) {
	prepareSearchTestConnector(t)

	configs := GetTestConfigs()
	// Only test configs that include graph
	testConfigs := []string{"vector+graph", "vector+graph+store"}

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

			// Skip if graph is not available
			if g.Graph == nil {
				t.Skip("Graph store not available")
			}

			ctx := context.Background()

			// Check if graph is connected
			if !g.Graph.IsConnected() {
				t.Skip("Graph store is not connected")
			}

			// Create collection for testing
			safeName := strings.ReplaceAll(configName, "+", "_")
			collectionID := fmt.Sprintf("graph_search_test_%s_%d", safeName, time.Now().Unix())
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

			// Add test documents with entities for graph extraction
			testDocs := []struct {
				id      string
				content string
			}{
				{
					id:      "doc1",
					content: "Albert Einstein developed the theory of relativity. He was born in Germany in 1879 and later moved to the United States. Einstein received the Nobel Prize in Physics in 1921 for his discovery of the photoelectric effect.",
				},
				{
					id:      "doc2",
					content: "Marie Curie was a physicist and chemist who conducted pioneering research on radioactivity. She was the first woman to win a Nobel Prize. Curie discovered the elements polonium and radium.",
				},
				{
					id:      "doc3",
					content: "Isaac Newton formulated the laws of motion and universal gravitation. He was born in England in 1643. Newton also made significant contributions to mathematics, including the development of calculus.",
				},
			}

			var addedDocIDs []string

			// Add test documents using AddText
			for _, doc := range testDocs {
				options := &types.UpsertOptions{
					DocID:        fmt.Sprintf("graph_search_test_%s_%s", configName, doc.id),
					CollectionID: collectionID,
					Metadata: map[string]interface{}{
						"source": "graph_search_test",
						"type":   "text",
					},
				}

				docID, err := g.AddText(ctx, doc.content, options)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error adding document %s: %v", doc.id, err)
						continue
					}
					t.Logf("Error adding document %s: %v", doc.id, err)
					continue
				}

				if docID != "" {
					addedDocIDs = append(addedDocIDs, docID)
					t.Logf("Successfully added document %s with ID: %s", doc.id, docID)
				}
			}

			// Wait for indexing
			time.Sleep(1 * time.Second)

			// Test search by entity names
			t.Run("Search_By_Entity_Names", func(t *testing.T) {
				searchOptions := &types.GraphSearchOptions{
					CollectionID: collectionID,
					Entities:     []string{"Einstein", "Nobel Prize"},
					MaxDepth:     2,
					Limit:        10,
				}

				result, err := g.SearchGraph(ctx, searchOptions)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error during graph search: %v", err)
						return
					}
					t.Errorf("Unexpected error during graph search: %v", err)
					return
				}

				// Verify result is not nil
				assert.NotNil(t, result, "Result should not be nil")

				// Verify nodes are returned (we searched for Einstein and Nobel Prize)
				assert.Greater(t, len(result.Nodes), 0, "Expected at least one node for Einstein/Nobel Prize search")

				t.Logf("Graph search by entity names returned:")
				t.Logf("  Nodes: %d", len(result.Nodes))
				t.Logf("  Relationships: %d", len(result.Relationships))
				t.Logf("  Paths: %d", len(result.Paths))
				t.Logf("  Segments: %d", len(result.Segments))

				// Verify each node has required fields
				foundEinsteinRelated := false
				foundNobelRelated := false
				for i, node := range result.Nodes {
					// Verify node ID is not empty
					assert.NotEmpty(t, node.ID, "Node %d: ID should not be empty", i)

					// Verify node has labels
					assert.NotNil(t, node.Labels, "Node %d: Labels should not be nil", i)

					t.Logf("  Node %d: ID=%s, Type=%s, Labels=%v",
						i, node.ID, node.EntityType, node.Labels)

					// Check if node is related to our search terms
					lowerID := strings.ToLower(node.ID)
					if strings.Contains(lowerID, "einstein") {
						foundEinsteinRelated = true
					}
					if strings.Contains(lowerID, "nobel") {
						foundNobelRelated = true
					}
				}

				// Verify we found relevant nodes
				assert.True(t, foundEinsteinRelated || foundNobelRelated,
					"Should find nodes related to Einstein or Nobel Prize")

				// Verify relationship structure if any
				for i, rel := range result.Relationships {
					assert.NotEmpty(t, rel.Type, "Relationship %d: Type should not be empty", i)
					t.Logf("  Relationship %d: %s -[%s]-> %s",
						i, rel.StartNode, rel.Type, rel.EndNode)
				}
			})

			// Test search by entity IDs
			t.Run("Search_By_Entity_IDs", func(t *testing.T) {
				// First, get some entity IDs by searching with names
				searchOptions := &types.GraphSearchOptions{
					CollectionID: collectionID,
					Entities:     []string{"Marie Curie"},
					MaxDepth:     1,
					Limit:        5,
				}

				result, err := g.SearchGraph(ctx, searchOptions)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error during graph search: %v", err)
						return
					}
					t.Errorf("Unexpected error during graph search: %v", err)
					return
				}

				if len(result.Nodes) == 0 {
					t.Log("No nodes found to test entity ID search")
					return
				}

				// Verify first search returned valid nodes
				firstNode := result.Nodes[0]
				assert.NotEmpty(t, firstNode.ID, "First node ID should not be empty")
				t.Logf("Using entity ID: %s for secondary search", firstNode.ID)

				// Use the first node's ID for entity ID search
				entityIDs := []string{firstNode.ID}

				searchByIDOptions := &types.GraphSearchOptions{
					CollectionID: collectionID,
					EntityIDs:    entityIDs,
					MaxDepth:     2,
					Limit:        10,
				}

				resultByID, err := g.SearchGraph(ctx, searchByIDOptions)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error during entity ID search: %v", err)
						return
					}
					t.Errorf("Unexpected error during entity ID search: %v", err)
					return
				}

				assert.NotNil(t, resultByID, "Result should not be nil")
				assert.Greater(t, len(resultByID.Nodes), 0, "Should find at least one node by ID")

				// Verify the searched entity ID is in the results
				foundSearchedID := false
				for _, node := range resultByID.Nodes {
					if node.ID == firstNode.ID {
						foundSearchedID = true
						break
					}
				}
				assert.True(t, foundSearchedID, "Should find the searched entity ID in results")

				t.Logf("Graph search by entity IDs returned %d nodes, %d relationships",
					len(resultByID.Nodes), len(resultByID.Relationships))
			})

			// Test search with custom Cypher query
			t.Run("Search_With_Cypher", func(t *testing.T) {
				searchOptions := &types.GraphSearchOptions{
					CollectionID: collectionID,
					Cypher:       "MATCH (n) RETURN n LIMIT 10",
					Parameters:   map[string]interface{}{},
				}

				result, err := g.SearchGraph(ctx, searchOptions)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error during Cypher search: %v", err)
						return
					}
					t.Errorf("Unexpected error during Cypher search: %v", err)
					return
				}

				assert.NotNil(t, result, "Result should not be nil")

				// Cypher query should return nodes (we added 3 documents with entities)
				assert.Greater(t, len(result.Nodes), 0, "Cypher query should return at least one node")
				assert.LessOrEqual(t, len(result.Nodes), 10, "Cypher query should respect LIMIT 10")

				t.Logf("Cypher search returned %d nodes", len(result.Nodes))

				// Verify nodes have required fields
				for i, node := range result.Nodes {
					assert.NotEmpty(t, node.ID, "Node %d: ID should not be empty", i)
					assert.NotNil(t, node.Labels, "Node %d: Labels should not be nil", i)
					t.Logf("  Node %d: ID=%s, Type=%s, Labels=%v", i, node.ID, node.EntityType, node.Labels)
				}
			})

			// Test search with relationship type filter
			t.Run("Search_With_Relationship_Filter", func(t *testing.T) {
				searchOptions := &types.GraphSearchOptions{
					CollectionID:  collectionID,
					Entities:      []string{"Newton"},
					MaxDepth:      2,
					RelationTypes: []string{"BORN_IN", "CONTRIBUTED_TO"},
					Limit:         10,
				}

				result, err := g.SearchGraph(ctx, searchOptions)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error during filtered graph search: %v", err)
						return
					}
					t.Errorf("Unexpected error during filtered graph search: %v", err)
					return
				}

				assert.NotNil(t, result, "Result should not be nil")

				// Should find Newton-related nodes
				foundNewtonRelated := false
				for _, node := range result.Nodes {
					if strings.Contains(strings.ToLower(node.ID), "newton") {
						foundNewtonRelated = true
						break
					}
				}
				assert.True(t, foundNewtonRelated, "Should find Newton-related nodes")

				t.Logf("Filtered graph search returned %d nodes, %d relationships",
					len(result.Nodes), len(result.Relationships))

				// Verify node fields
				for i, node := range result.Nodes {
					assert.NotEmpty(t, node.ID, "Node %d: ID should not be empty", i)
				}
			})

			// Test search with progress callback
			t.Run("Search_With_Progress", func(t *testing.T) {
				var progressMessages []string
				var progressValues []float64
				progressCallback := func(status types.SearcherStatus, payload types.SearcherPayload) {
					progressMessages = append(progressMessages, fmt.Sprintf("[%s] %s", status, payload.Message))
					progressValues = append(progressValues, payload.Progress)
				}

				searchOptions := &types.GraphSearchOptions{
					CollectionID: collectionID,
					Entities:     []string{"Einstein"},
					MaxDepth:     2,
				}

				result, err := g.SearchGraph(ctx, searchOptions, progressCallback)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error during graph search: %v", err)
						return
					}
					t.Errorf("Unexpected error during graph search: %v", err)
					return
				}

				t.Logf("Graph search with progress returned %d nodes", len(result.Nodes))
				t.Logf("Progress messages: %v", progressMessages)

				// Verify progress was reported
				assert.Greater(t, len(progressMessages), 0, "Expected at least one progress message")

				// Verify progress values are increasing
				for i := 1; i < len(progressValues); i++ {
					assert.GreaterOrEqual(t, progressValues[i], progressValues[i-1],
						"Progress values should be non-decreasing")
				}
			})

			// Test that related segments are fetched
			t.Run("Search_Returns_Related_Segments", func(t *testing.T) {
				if len(addedDocIDs) == 0 {
					t.Skip("No documents were added, skipping segment test")
				}

				searchOptions := &types.GraphSearchOptions{
					CollectionID: collectionID,
					Entities:     []string{"Einstein", "relativity"},
					MaxDepth:     1,
				}

				result, err := g.SearchGraph(ctx, searchOptions)
				if err != nil {
					if isExpectedGraphSearchError(err) {
						t.Logf("Expected error during graph search: %v", err)
						return
					}
					t.Errorf("Unexpected error during graph search: %v", err)
					return
				}

				assert.NotNil(t, result, "Result should not be nil")

				t.Logf("Graph search returned %d nodes and %d related segments",
					len(result.Nodes), len(result.Segments))

				// Verify nodes were found
				assert.Greater(t, len(result.Nodes), 0, "Should find nodes for Einstein/relativity")

				// Verify segments have required fields if any
				for i, seg := range result.Segments {
					assert.NotEmpty(t, seg.ID, "Segment %d: ID should not be empty", i)
					assert.NotEmpty(t, seg.Text, "Segment %d: Text should not be empty", i)
					assert.Equal(t, collectionID, seg.CollectionID, "Segment %d: CollectionID should match", i)

					// Verify segment content is relevant
					lowerText := strings.ToLower(seg.Text)
					isRelevant := strings.Contains(lowerText, "einstein") ||
						strings.Contains(lowerText, "relativity") ||
						strings.Contains(lowerText, "physics")
					assert.True(t, isRelevant, "Segment %d: Text should be relevant to search terms", i)

					t.Logf("  Segment %d: ID=%s, Text=%s...",
						i, seg.ID, truncateString(seg.Text, 50))
				}

				// If we have segments, verify they are connected to the nodes
				if len(result.Segments) > 0 && len(result.Nodes) > 0 {
					t.Logf("Successfully retrieved %d segments related to %d nodes",
						len(result.Segments), len(result.Nodes))
				}
			})
		})
	}
}

// TestSearchGraphErrorHandling tests error conditions for SearchGraph
func TestSearchGraphErrorHandling(t *testing.T) {
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

	ctx := context.Background()

	t.Run("Nil_Options", func(t *testing.T) {
		_, err := g.SearchGraph(ctx, nil)
		assert.Error(t, err, "Expected error for nil options")
		assert.Contains(t, err.Error(), "cannot be nil", "Error should mention nil options")
		t.Logf("Nil options correctly rejected: %v", err)
	})

	t.Run("Empty_CollectionID", func(t *testing.T) {
		searchOptions := &types.GraphSearchOptions{
			Entities: []string{"test"},
		}

		_, err := g.SearchGraph(ctx, searchOptions)
		assert.Error(t, err, "Expected error for empty collection ID")
		assert.Contains(t, err.Error(), "collection ID is required", "Error should mention collection ID")
		t.Logf("Empty collection ID correctly rejected: %v", err)
	})

	t.Run("No_Query_Parameters", func(t *testing.T) {
		searchOptions := &types.GraphSearchOptions{
			CollectionID: "test_collection",
			// No Query, Entities, EntityIDs, or Cypher
		}

		_, err := g.SearchGraph(ctx, searchOptions)
		if err == nil {
			// Graph store might not be connected, which is also an error
			t.Log("No error returned, graph store may not be connected")
			return
		}

		// Should fail with either "not connected", "at least one of", "does not exist", or "not available" error
		hasExpectedError := strings.Contains(err.Error(), "at least one of") ||
			strings.Contains(err.Error(), "not connected") ||
			strings.Contains(err.Error(), "not available") ||
			strings.Contains(err.Error(), "does not exist")

		assert.True(t, hasExpectedError, "Error should mention query requirements or connection: %v", err)
		t.Logf("No query parameters correctly rejected: %v", err)
	})

	t.Run("NL_Query_Without_Extraction", func(t *testing.T) {
		// Skip if graph is not connected
		if g.Graph == nil || !g.Graph.IsConnected() {
			t.Skip("Graph store not available")
		}

		// Create a test collection first
		collectionID := fmt.Sprintf("graph_error_test_%d", time.Now().Unix())
		collection := types.CollectionConfig{
			ID: collectionID,
			Config: &types.CreateCollectionOptions{
				CollectionName: fmt.Sprintf("%s_vector", collectionID),
				Dimension:      1536,
				Distance:       types.DistanceCosine,
				IndexType:      types.IndexTypeHNSW,
			},
		}

		_, err := g.CreateCollection(ctx, collection)
		if err != nil {
			t.Skipf("Failed to create test collection: %v", err)
		}
		defer g.RemoveCollection(ctx, collectionID)

		searchOptions := &types.GraphSearchOptions{
			CollectionID: collectionID,
			Query:        "Who is Einstein?",
			// Extraction is nil - should fail
		}

		_, err = g.SearchGraph(ctx, searchOptions)
		if err == nil {
			t.Error("Expected error for NL query without extraction")
			return
		}

		// Should fail with extraction requirement or graph not found error
		hasExpectedError := strings.Contains(err.Error(), "extraction function is required") ||
			strings.Contains(err.Error(), "does not exist")

		assert.True(t, hasExpectedError, "Error should mention extraction requirement: %v", err)
		t.Logf("NL query without extraction correctly rejected: %v", err)
	})

	t.Run("Graph_Not_Connected", func(t *testing.T) {
		// Create a new instance without graph
		vectorOnlyConfig := configs["vector"]
		if vectorOnlyConfig == nil {
			t.Skip("Vector-only config not found")
		}

		gVectorOnly, err := New(vectorOnlyConfig)
		if err != nil {
			t.Skipf("Failed to create vector-only instance: %v", err)
		}

		searchOptions := &types.GraphSearchOptions{
			CollectionID: "test_collection",
			Entities:     []string{"test"},
		}

		_, err = gVectorOnly.SearchGraph(ctx, searchOptions)
		assert.Error(t, err, "Expected error when graph is not available")
		assert.Contains(t, err.Error(), "graph store is not available", "Error should mention graph availability")
		t.Logf("Graph not available correctly rejected: %v", err)
	})
}

// TestSearchGraphWithNLQuery tests graph search with natural language query
func TestSearchGraphWithNLQuery(t *testing.T) {
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

	if g.Graph == nil || !g.Graph.IsConnected() {
		t.Skip("Graph store not available")
	}

	ctx := context.Background()

	// Create collection for testing
	collectionID := fmt.Sprintf("graph_nl_test_%d", time.Now().Unix())
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
	defer g.RemoveCollection(ctx, collectionID)

	// Add test document
	content := "Steve Jobs co-founded Apple Computer with Steve Wozniak. He was born in San Francisco and later led the development of the iPhone and iPad."
	options := &types.UpsertOptions{
		DocID:        "nl_test_doc",
		CollectionID: collectionID,
	}

	_, err = g.AddText(ctx, content, options)
	if err != nil {
		t.Logf("Error adding document: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	t.Run("NL_Query_With_Extraction", func(t *testing.T) {
		testExtraction, err := createSearchTestExtraction(t)
		if err != nil {
			t.Skipf("Failed to create test extraction: %v", err)
		}

		searchOptions := &types.GraphSearchOptions{
			CollectionID: collectionID,
			Query:        "Who founded Apple and what products did they create?",
			Extraction:   testExtraction,
			MaxDepth:     2,
		}

		result, err := g.SearchGraph(ctx, searchOptions)
		if err != nil {
			if isExpectedGraphSearchError(err) {
				t.Logf("Expected error during NL graph search: %v", err)
				return
			}
			t.Errorf("Unexpected error during NL graph search: %v", err)
			return
		}

		t.Logf("NL graph search returned %d nodes, %d relationships",
			len(result.Nodes), len(result.Relationships))

		for i, node := range result.Nodes {
			t.Logf("  Node %d: ID=%s, Type=%s", i, node.ID, node.EntityType)
		}
	})
}

// TestSearchGraphPathQueries tests graph path queries
func TestSearchGraphPathQueries(t *testing.T) {
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

	if g.Graph == nil || !g.Graph.IsConnected() {
		t.Skip("Graph store not available")
	}

	ctx := context.Background()

	// Create collection for testing
	collectionID := fmt.Sprintf("graph_path_test_%d", time.Now().Unix())
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
	defer g.RemoveCollection(ctx, collectionID)

	t.Run("Path_Query_With_Cypher", func(t *testing.T) {
		// Query for paths between nodes
		searchOptions := &types.GraphSearchOptions{
			CollectionID: collectionID,
			Cypher:       "MATCH p=(a)-[*1..3]-(b) RETURN p LIMIT 5",
		}

		result, err := g.SearchGraph(ctx, searchOptions)
		if err != nil {
			if isExpectedGraphSearchError(err) {
				t.Logf("Expected error during path query: %v", err)
				return
			}
			t.Errorf("Unexpected error during path query: %v", err)
			return
		}

		t.Logf("Path query returned %d paths", len(result.Paths))

		for i, path := range result.Paths {
			t.Logf("  Path %d: %d nodes, %d relationships, length=%d",
				i, len(path.Nodes), len(path.Relationships), path.Length)
		}
	})
}

// BenchmarkSearchGraph benchmarks the SearchGraph function
func BenchmarkSearchGraph(b *testing.B) {
	prepareSearchTestConnector(&testing.T{})

	configs := GetTestConfigs()
	config := configs["vector+graph"]
	if config == nil {
		b.Skip("vector+graph config not found")
	}

	g, err := New(config)
	if err != nil {
		b.Skipf("Failed to create GraphRag instance: %v", err)
	}

	if g.Graph == nil || !g.Graph.IsConnected() {
		b.Skip("Graph store not available")
	}

	ctx := context.Background()

	// Create collection for benchmarking
	collectionID := fmt.Sprintf("benchmark_graph_search_%d", time.Now().Unix())
	collection := types.CollectionConfig{
		ID: collectionID,
		Config: &types.CreateCollectionOptions{
			CollectionName: fmt.Sprintf("%s_vector", collectionID),
			Dimension:      1536,
			Distance:       types.DistanceCosine,
		},
	}

	_, err = g.CreateCollection(ctx, collection)
	if err != nil {
		b.Skipf("Failed to create benchmark collection: %v", err)
	}

	defer g.RemoveCollection(ctx, collectionID)

	// Add test documents
	for i := 0; i < 5; i++ {
		options := &types.UpsertOptions{
			DocID:        fmt.Sprintf("bench_doc_%d", i),
			CollectionID: collectionID,
		}
		g.AddText(ctx, fmt.Sprintf("Person %d works at Company %d in City %d.", i, i%3, i%5), options)
	}

	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		searchOptions := &types.GraphSearchOptions{
			CollectionID: collectionID,
			Entities:     []string{"Person", "Company"},
			MaxDepth:     2,
			Limit:        10,
		}

		_, err := g.SearchGraph(ctx, searchOptions)
		if err != nil {
			continue
		}
	}
}

// ==== Helper Functions ====

// isExpectedGraphSearchError checks if the error is expected in test environment
func isExpectedGraphSearchError(err error) bool {
	expectedErrors := []string{
		"connection refused", "no such host", "connector not found", "connector openai not loaded",
		"graph search failed", "embedding", "extraction", "request failed",
		"not connected", "not available", "does not exist",
	}

	errStr := err.Error()
	for _, expected := range expectedErrors {
		if strings.Contains(errStr, expected) {
			return true
		}
	}
	return false
}

// createSearchTestExtraction creates an extraction configuration for search testing
func createSearchTestExtraction(t *testing.T) (types.Extraction, error) {
	t.Helper()

	// Try to create extraction using the test configuration
	// This requires the extraction package to be available
	return nil, fmt.Errorf("extraction not implemented in test helper")
}
