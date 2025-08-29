package neo4j

import (
	"context"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

const (
	testSaveGraphName = "test_save_graph"
	testSaveTimeout   = 30 * time.Second
)

// TestSaveExtractionResults_Basic tests basic SaveExtractionResults functionality
func TestSaveExtractionResults_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testSaveTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testSaveGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testSaveGraphName)
	}()

	// Test with empty results list
	_, err = store.SaveExtractionResults(ctx, testSaveGraphName, []*types.ExtractionResult{})
	if err != nil {
		t.Fatalf("SaveExtractionResults with empty list failed: %v", err)
	}

	// Create test extraction results
	extractionResults := createTestExtractionResults()

	// Save extraction results
	saveResponse, err := store.SaveExtractionResults(ctx, testSaveGraphName, extractionResults)
	if err != nil {
		t.Fatalf("SaveExtractionResults failed: %v", err)
	}

	// Verify response contains expected data
	if saveResponse.EntitiesCount != 4 {
		t.Errorf("Expected 4 entities in response, got %d", saveResponse.EntitiesCount)
	}
	if saveResponse.RelationshipsCount != 3 {
		t.Errorf("Expected 3 relationships in response, got %d", saveResponse.RelationshipsCount)
	}
	if saveResponse.ProcessedCount != 1 {
		t.Errorf("Expected 1 processed result, got %d", saveResponse.ProcessedCount)
	}

	// Verify entities were created
	getNodesOpts := &types.GetNodesOptions{
		GraphName:         testSaveGraphName,
		IncludeProperties: true,
		IncludeMetadata:   true,
		Limit:             50,
	}
	nodes, err := store.GetNodes(ctx, getNodesOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes after save: %v", err)
	}

	// Should have 4 unique entities (Alice, Bob, Company A, Company B)
	if len(nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(nodes))
	}

	// Verify relationships were created
	getRelsOpts := &types.GetRelationshipsOptions{
		GraphName:         testSaveGraphName,
		IncludeProperties: true,
		IncludeMetadata:   true,
		Limit:             50,
	}
	rels, err := store.GetRelationships(ctx, getRelsOpts)
	if err != nil {
		t.Fatalf("Failed to get relationships after save: %v", err)
	}

	// Should have 3 relationships
	if len(rels) != 3 {
		t.Errorf("Expected 3 relationships, got %d", len(rels))
	}

	// Verify node properties contain source information
	for _, node := range nodes {
		if node.Properties == nil {
			t.Errorf("Node %s has nil properties", node.ID)
			continue
		}

		// Check for name property
		if name, ok := node.Properties["name"]; !ok {
			t.Errorf("Node %s missing name property", node.ID)
		} else if name == "" {
			t.Errorf("Node %s has empty name", node.ID)
		}

		// Check for source documents
		if sourceDocs, ok := node.Properties["source_documents"]; ok {
			if docs, ok := sourceDocs.([]interface{}); ok && len(docs) > 0 {
				t.Logf("Node %s has source documents: %v", node.ID, docs)
			}
		}
	}
}

// TestSaveExtractionResults_Deduplication tests entity deduplication
func TestSaveExtractionResults_Deduplication(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testSaveTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testSaveGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testSaveGraphName)
	}()

	// First extraction - save initial entities
	firstResults := []*types.ExtractionResult{
		{
			Usage: types.ExtractionUsage{TotalTokens: 100},
			Model: "test-model",
			Nodes: []types.Node{
				{
					ID:              "alice_1",
					Name:            "Alice Smith",
					Type:            "Person",
					Labels:          []string{"Person", "Employee"},
					Description:     "Software engineer",
					Confidence:      0.9,
					SourceDocuments: []string{"doc1.txt"},
					SourceChunks:    []string{"chunk1"},
				},
			},
			Relationships: []types.Relationship{},
		},
	}

	_, err = store.SaveExtractionResults(ctx, testSaveGraphName, firstResults)
	if err != nil {
		t.Fatalf("First SaveExtractionResults failed: %v", err)
	}

	// Second extraction - same entity with different ID and additional sources
	secondResults := []*types.ExtractionResult{
		{
			Usage: types.ExtractionUsage{TotalTokens: 120},
			Model: "test-model",
			Nodes: []types.Node{
				{
					ID:              "alice_2", // Different ID but same name/type
					Name:            "Alice Smith",
					Type:            "Person",
					Labels:          []string{"Person", "Developer"},
					Description:     "Senior software engineer",
					Confidence:      0.95,
					SourceDocuments: []string{"doc2.txt"},
					SourceChunks:    []string{"chunk2"},
				},
			},
			Relationships: []types.Relationship{},
		},
	}

	_, err = store.SaveExtractionResults(ctx, testSaveGraphName, secondResults)
	if err != nil {
		t.Fatalf("Second SaveExtractionResults failed: %v", err)
	}

	// Verify only one entity exists (deduplication worked)
	getNodesOpts := &types.GetNodesOptions{
		GraphName:         testSaveGraphName,
		IncludeProperties: true,
		Limit:             50,
	}
	nodes, err := store.GetNodes(ctx, getNodesOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes after deduplication test: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 deduplicated node, got %d", len(nodes))
	}

	// Verify the entity has merged source documents and chunks
	if len(nodes) > 0 {
		node := nodes[0]
		sourceDocs := store.getStringSliceFromProperty(node.Properties, "source_documents")
		sourceChunks := store.getStringSliceFromProperty(node.Properties, "source_chunks")

		if len(sourceDocs) != 2 {
			t.Errorf("Expected 2 merged source documents, got %d: %v", len(sourceDocs), sourceDocs)
		}

		if len(sourceChunks) != 2 {
			t.Errorf("Expected 2 merged source chunks, got %d: %v", len(sourceChunks), sourceChunks)
		}

		// Check that both doc1.txt and doc2.txt are present
		docMap := make(map[string]bool)
		for _, doc := range sourceDocs {
			docMap[doc] = true
		}
		if !docMap["doc1.txt"] || !docMap["doc2.txt"] {
			t.Errorf("Missing expected documents in merged sources: %v", sourceDocs)
		}
	}
}

// TestSaveExtractionResults_RelationshipMapping tests relationship entity ID mapping
func TestSaveExtractionResults_RelationshipMapping(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testSaveTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testSaveGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testSaveGraphName)
	}()

	// Pre-create an entity
	preEntity := []*types.ExtractionResult{
		{
			Nodes: []types.Node{
				{
					ID:              "existing_alice",
					Name:            "Alice Smith",
					Type:            "Person",
					SourceDocuments: []string{"existing_doc.txt"},
				},
			},
		},
	}

	_, err = store.SaveExtractionResults(ctx, testSaveGraphName, preEntity)
	if err != nil {
		t.Fatalf("Failed to save pre-existing entity: %v", err)
	}

	// Now save extraction with relationship pointing to the same entity
	extractionWithRel := []*types.ExtractionResult{
		{
			Nodes: []types.Node{
				{
					ID:   "new_alice", // Different ID but same name/type - should be deduplicated
					Name: "Alice Smith",
					Type: "Person",
				},
				{
					ID:   "company_x",
					Name: "Company X",
					Type: "Organization",
				},
			},
			Relationships: []types.Relationship{
				{
					ID:        "rel_1",
					Type:      "WORKS_FOR",
					StartNode: "new_alice", // This should be mapped to existing entity
					EndNode:   "company_x",
				},
			},
		},
	}

	_, err = store.SaveExtractionResults(ctx, testSaveGraphName, extractionWithRel)
	if err != nil {
		t.Fatalf("Failed to save extraction with relationships: %v", err)
	}

	// Verify entities
	getNodesOpts := &types.GetNodesOptions{
		GraphName:         testSaveGraphName,
		IncludeProperties: true,
		Limit:             50,
	}
	nodes, err := store.GetNodes(ctx, getNodesOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	// Should have 2 entities (Alice deduplicated, Company X new)
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes after relationship mapping test, got %d", len(nodes))
	}

	// Verify relationships
	getRelsOpts := &types.GetRelationshipsOptions{
		GraphName:         testSaveGraphName,
		IncludeProperties: true,
		Limit:             50,
	}
	rels, err := store.GetRelationships(ctx, getRelsOpts)
	if err != nil {
		t.Fatalf("Failed to get relationships: %v", err)
	}

	if len(rels) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(rels))
	}

	// Verify the relationship points to the correct (deduplicated) entity
	if len(rels) > 0 {
		rel := rels[0]
		if rel.Type != "WORKS_FOR" {
			t.Errorf("Expected relationship type WORKS_FOR, got %s", rel.Type)
		}

		// Find Alice's actual ID after deduplication
		var aliceID, companyID string
		for _, node := range nodes {
			if name, ok := node.Properties["name"]; ok {
				if name == "Alice Smith" {
					aliceID = node.ID
				} else if name == "Company X" {
					companyID = node.ID
				}
			}
		}

		if aliceID == "" || companyID == "" {
			t.Fatal("Could not find Alice or Company X nodes")
		}

		// Verify relationship uses correct IDs
		if rel.StartNode != aliceID {
			t.Errorf("Expected relationship start node %s, got %s", aliceID, rel.StartNode)
		}
		if rel.EndNode != companyID {
			t.Errorf("Expected relationship end node %s, got %s", companyID, rel.EndNode)
		}
	}
}

// TestSaveExtractionResults_ErrorHandling tests error scenarios
func TestSaveExtractionResults_ErrorHandling(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testSaveTimeout)
	defer cancel()

	// Test with nil results
	_, err := store.SaveExtractionResults(ctx, "test", nil)
	if err != nil {
		t.Errorf("SaveExtractionResults with nil should not error, got: %v", err)
	}

	// Test with results containing nil result
	resultsWithNil := []*types.ExtractionResult{nil}
	_, err = store.SaveExtractionResults(ctx, "test", resultsWithNil)
	if err != nil {
		t.Errorf("SaveExtractionResults with nil result should not error, got: %v", err)
	}
}

// TestSaveExtractionResults_Disconnected tests behavior when store is disconnected
func TestSaveExtractionResults_Disconnected(t *testing.T) {
	store := NewStore()

	ctx, cancel := context.WithTimeout(context.Background(), testSaveTimeout)
	defer cancel()

	extractionResults := createTestExtractionResults()

	_, err := store.SaveExtractionResults(ctx, "test", extractionResults)
	if err == nil {
		t.Error("Expected error when saving to disconnected store")
	}
}

// Helper function to create test extraction results
func createTestExtractionResults() []*types.ExtractionResult {
	return []*types.ExtractionResult{
		{
			Usage: types.ExtractionUsage{
				TotalTokens:  500,
				PromptTokens: 100,
				TotalTexts:   2,
			},
			Model: "gpt-4",
			Nodes: []types.Node{
				{
					ID:              "alice_001",
					Name:            "Alice Smith",
					Type:            "Person",
					Labels:          []string{"Person", "Employee"},
					Properties:      map[string]interface{}{"department": "Engineering"},
					Description:     "Senior Software Engineer",
					Confidence:      0.95,
					EmbeddingVector: []float64{0.1, 0.2, 0.3},
					SourceDocuments: []string{"doc1.txt", "doc2.txt"},
					SourceChunks:    []string{"chunk1", "chunk2"},
					CreatedAt:       time.Now().Unix(),
					Version:         1,
				},
				{
					ID:              "bob_002",
					Name:            "Bob Johnson",
					Type:            "Person",
					Labels:          []string{"Person", "Manager"},
					Properties:      map[string]interface{}{"department": "Sales"},
					Description:     "Sales Manager",
					Confidence:      0.90,
					SourceDocuments: []string{"doc1.txt"},
					SourceChunks:    []string{"chunk3"},
					CreatedAt:       time.Now().Unix(),
					Version:         1,
				},
				{
					ID:              "company_a_003",
					Name:            "Company A",
					Type:            "Organization",
					Labels:          []string{"Organization", "Company"},
					Properties:      map[string]interface{}{"industry": "Technology"},
					Description:     "Technology company",
					Confidence:      0.85,
					SourceDocuments: []string{"doc2.txt"},
					SourceChunks:    []string{"chunk4"},
					CreatedAt:       time.Now().Unix(),
					Version:         1,
				},
				{
					ID:              "company_b_004",
					Name:            "Company B",
					Type:            "Organization",
					Labels:          []string{"Organization", "Company"},
					Properties:      map[string]interface{}{"industry": "Finance"},
					Description:     "Financial services company",
					Confidence:      0.88,
					SourceDocuments: []string{"doc3.txt"},
					SourceChunks:    []string{"chunk5"},
					CreatedAt:       time.Now().Unix(),
					Version:         1,
				},
			},
			Relationships: []types.Relationship{
				{
					ID:              "rel_001",
					Type:            "WORKS_FOR",
					StartNode:       "alice_001",
					EndNode:         "company_a_003",
					Properties:      map[string]interface{}{"since": "2020-01-01"},
					Description:     "Employment relationship",
					Confidence:      0.92,
					Weight:          0.8,
					SourceDocuments: []string{"doc1.txt"},
					SourceChunks:    []string{"chunk1"},
					CreatedAt:       time.Now().Unix(),
					Version:         1,
				},
				{
					ID:              "rel_002",
					Type:            "WORKS_FOR",
					StartNode:       "bob_002",
					EndNode:         "company_b_004",
					Properties:      map[string]interface{}{"since": "2019-06-01"},
					Description:     "Management relationship",
					Confidence:      0.89,
					Weight:          0.9,
					SourceDocuments: []string{"doc1.txt"},
					SourceChunks:    []string{"chunk3"},
					CreatedAt:       time.Now().Unix(),
					Version:         1,
				},
				{
					ID:              "rel_003",
					Type:            "KNOWS",
					StartNode:       "alice_001",
					EndNode:         "bob_002",
					Properties:      map[string]interface{}{"relationship": "colleague"},
					Description:     "Professional acquaintance",
					Confidence:      0.75,
					Weight:          0.6,
					SourceDocuments: []string{"doc2.txt"},
					SourceChunks:    []string{"chunk2"},
					CreatedAt:       time.Now().Unix(),
					Version:         1,
				},
			},
		},
	}
}

// Helper functions reused from existing test files

// setupTestStore creates a test store for save testing
func setupSaveTestStore(t *testing.T) *Store {
	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	store := NewStore()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username":              config.User,
			"password":              config.Password,
			"use_separate_database": false,
		},
	}

	connectWithRetry(ctx, t, store, storeConfig)
	return store
}

// cleanupSaveTestStore cleans up test store resources for save testing
func cleanupSaveTestStore(t *testing.T, store *Store) {
	if store != nil && store.IsConnected() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Clean up test graphs including the default graph used by SaveExtractionResults
		testGraphs := []string{testSaveGraphName, "default"}
		for _, graph := range testGraphs {
			exists, err := store.GraphExists(ctx, graph)
			if err == nil && exists {
				_ = store.DropGraph(ctx, graph)
			}
		}

		err := store.Disconnect(ctx)
		if err != nil {
			t.Logf("Warning: Failed to disconnect test store: %v", err)
		}
	}
}
