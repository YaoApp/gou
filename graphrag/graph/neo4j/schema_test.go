package neo4j

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/graphrag/types"
)

// =============================================================================
// Tests for schema.go methods
// =============================================================================

// TestGetSchema tests the GetSchema method
func TestGetSchema(t *testing.T) {
	t.Run("CommunityEdition", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Create test graph and add some nodes/relationships
		graphName := "test_schema_graph"
		err := store.CreateGraph(ctx, graphName)
		assert.NoError(t, err)
		defer store.DropGraph(ctx, graphName)

		// Add test data using utility functions
		testNodes := CreateTestNodes(10)
		_, err = store.AddNodes(ctx, &types.AddNodesOptions{
			GraphName: graphName,
			Nodes:     testNodes,
		})
		assert.NoError(t, err)

		testRelationships := CreateTestRelationships(5)
		_, err = store.AddRelationships(ctx, &types.AddRelationshipsOptions{
			GraphName:     graphName,
			Relationships: testRelationships,
		})
		assert.NoError(t, err)

		// Get schema
		schema, err := store.GetSchema(ctx, graphName)
		assert.NoError(t, err)
		assert.NotNil(t, schema)

		// Verify schema structure
		assert.NotNil(t, schema.NodeLabels)
		assert.NotNil(t, schema.RelationshipTypes)
		assert.NotNil(t, schema.NodeProperties)
		assert.NotNil(t, schema.RelProperties)
		assert.NotNil(t, schema.Statistics)
		assert.NotNil(t, schema.Constraints)
		assert.NotNil(t, schema.Indexes)

		// Should contain our test node labels
		found := false
		for _, label := range schema.NodeLabels {
			if label == "TestNode" || label == "Entity" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should contain TestNode or Entity label")

		// Verify statistics
		assert.True(t, schema.Statistics.TotalNodes > 0, "Should have nodes")
		assert.True(t, schema.Statistics.TotalRelationships >= 0, "Should have relationship count")
	})

	t.Run("EnterpriseEdition", func(t *testing.T) {
		config := getEnterpriseTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username":              config.User,
				"password":              config.Password,
				"use_separate_database": true,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Create test graph
		graphName := "testschemaent"
		err := store.CreateGraph(ctx, graphName)
		assert.NoError(t, err)
		defer store.DropGraph(ctx, graphName)

		// Add test data
		testNodes := CreateTestNodes(5)
		_, err = store.AddNodes(ctx, &types.AddNodesOptions{
			GraphName: graphName,
			Nodes:     testNodes,
		})
		assert.NoError(t, err)

		// Get schema
		schema, err := store.GetSchema(ctx, graphName)
		assert.NoError(t, err)
		assert.NotNil(t, schema)

		// Verify schema structure
		assert.NotNil(t, schema.NodeLabels)
		assert.NotNil(t, schema.Statistics)
		assert.True(t, schema.Statistics.TotalNodes > 0)
	})

	t.Run("NonExistentGraph", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Try to get schema for non-existent graph
		_, err := store.GetSchema(ctx, "nonexistent_graph")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("InvalidInput", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Test empty graph name
		_, err := store.GetSchema(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("NotConnected", func(t *testing.T) {
		store := NewStore()
		ctx := context.Background()

		_, err := store.GetSchema(ctx, "test_graph")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

// TestCreateIndex tests the CreateIndex method
func TestCreateIndex(t *testing.T) {
	t.Run("CommunityEdition", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Create test graph
		graphName := "test_index_graph"
		err := store.CreateGraph(ctx, graphName)
		assert.NoError(t, err)
		defer store.DropGraph(ctx, graphName)

		// Add some test nodes first - required for index operations
		testNodes := CreateTestNodes(10)
		_, err = store.AddNodes(ctx, &types.AddNodesOptions{
			GraphName: graphName,
			Nodes:     testNodes,
		})
		assert.NoError(t, err)

		// Test BTREE index on node
		btreeOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       "test_btree_idx",
			Target:     "NODE",
			Labels:     []string{"TestLabel"},
			Properties: []string{"name"},
			IndexType:  "BTREE",
		}
		err = store.CreateIndex(ctx, btreeOpts)
		assert.NoError(t, err)

		// Clean up index
		dropOpts := &types.DropIndexOptions{
			GraphName: graphName,
			Name:      "test_btree_idx",
		}
		err = store.DropIndex(ctx, dropOpts)
		assert.NoError(t, err)

		// Test FULLTEXT index
		fulltextOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       "test_fulltext_idx",
			Target:     "NODE",
			Labels:     []string{"TestLabel"},
			Properties: []string{"content"},
			IndexType:  "FULLTEXT",
		}
		err = store.CreateIndex(ctx, fulltextOpts)
		assert.NoError(t, err)

		// Clean up
		dropOpts.Name = "test_fulltext_idx"
		err = store.DropIndex(ctx, dropOpts)
		assert.NoError(t, err)

		// Test relationship index
		relOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       "test_rel_idx",
			Target:     "RELATIONSHIP",
			Labels:     []string{"KNOWS"},
			Properties: []string{"since"},
			IndexType:  "BTREE",
		}
		err = store.CreateIndex(ctx, relOpts)
		assert.NoError(t, err)

		// Clean up
		dropOpts.Name = "test_rel_idx"
		err = store.DropIndex(ctx, dropOpts)
		assert.NoError(t, err)
	})

	t.Run("EnterpriseEdition", func(t *testing.T) {
		config := getEnterpriseTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username":              config.User,
				"password":              config.Password,
				"use_separate_database": true,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Create test graph
		graphName := "testindexent"
		err := store.CreateGraph(ctx, graphName)
		assert.NoError(t, err)
		defer store.DropGraph(ctx, graphName)

		// Add some test nodes first - required for index operations
		testNodes := CreateTestNodes(10)
		_, err = store.AddNodes(ctx, &types.AddNodesOptions{
			GraphName: graphName,
			Nodes:     testNodes,
		})
		assert.NoError(t, err)

		// Test index creation
		indexOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       "test_ent_idx",
			Target:     "NODE",
			Labels:     []string{"Person"},
			Properties: []string{"email"},
			IndexType:  "BTREE",
		}
		err = store.CreateIndex(ctx, indexOpts)
		assert.NoError(t, err)

		// Test vector index if supported
		vectorOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       "test_vector_idx",
			Target:     "NODE",
			Labels:     []string{"Document"},
			Properties: []string{"embedding"},
			IndexType:  "VECTOR",
			Config: map[string]interface{}{
				"dimension":  128,
				"similarity": "cosine",
			},
		}
		err = store.CreateIndex(ctx, vectorOpts)
		// Vector indexes might not be supported in all Neo4j versions
		if err != nil && !strings.Contains(err.Error(), "Unknown index type") {
			assert.NoError(t, err)
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Test nil options
		err := store.CreateIndex(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")

		// Test empty graph name
		opts := &types.CreateIndexOptions{
			GraphName:  "",
			Properties: []string{"name"},
		}
		err = store.CreateIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")

		// Test empty properties
		opts.GraphName = "test"
		opts.Properties = []string{}
		err = store.CreateIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("NonExistentGraph", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		opts := &types.CreateIndexOptions{
			GraphName:  "nonexistent_graph",
			Target:     "NODE",
			Labels:     []string{"Test"},
			Properties: []string{"name"},
		}
		err := store.CreateIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("NotConnected", func(t *testing.T) {
		store := NewStore()
		ctx := context.Background()

		opts := &types.CreateIndexOptions{
			GraphName:  "test",
			Properties: []string{"name"},
		}
		err := store.CreateIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

// TestDropIndex tests the DropIndex method
func TestDropIndex(t *testing.T) {
	t.Run("CommunityEdition", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Create test graph
		graphName := "test_drop_index_graph"
		err := store.CreateGraph(ctx, graphName)
		assert.NoError(t, err)
		defer store.DropGraph(ctx, graphName)

		// Add some test nodes first - required for index operations
		testNodes := CreateTestNodes(10)
		_, err = store.AddNodes(ctx, &types.AddNodesOptions{
			GraphName: graphName,
			Nodes:     testNodes,
		})
		assert.NoError(t, err)

		// Create an index to drop
		createOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       "test_drop_idx",
			Target:     "NODE",
			Labels:     []string{"TestLabel"},
			Properties: []string{"name"},
			IndexType:  "BTREE",
		}
		err = store.CreateIndex(ctx, createOpts)
		assert.NoError(t, err)

		// Drop the index
		dropOpts := &types.DropIndexOptions{
			GraphName: graphName,
			Name:      "test_drop_idx",
		}
		err = store.DropIndex(ctx, dropOpts)
		assert.NoError(t, err)

		// Try to drop again - should fail
		err = store.DropIndex(ctx, dropOpts)
		assert.Error(t, err)

		// Test with IfExists flag
		dropOpts.IfExists = true
		err = store.DropIndex(ctx, dropOpts)
		assert.NoError(t, err) // Should succeed silently
	})

	t.Run("EnterpriseEdition", func(t *testing.T) {
		config := getEnterpriseTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username":              config.User,
				"password":              config.Password,
				"use_separate_database": true,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Create test graph
		graphName := "testdropidxent"
		err := store.CreateGraph(ctx, graphName)
		assert.NoError(t, err)
		defer store.DropGraph(ctx, graphName)

		// Add some test nodes first - required for index operations
		testNodes := CreateTestNodes(10)
		_, err = store.AddNodes(ctx, &types.AddNodesOptions{
			GraphName: graphName,
			Nodes:     testNodes,
		})
		assert.NoError(t, err)

		// Create and drop index
		createOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       "test_ent_drop_idx",
			Target:     "NODE",
			Labels:     []string{"Person"},
			Properties: []string{"id"},
		}
		err = store.CreateIndex(ctx, createOpts)
		assert.NoError(t, err)

		dropOpts := &types.DropIndexOptions{
			GraphName: graphName,
			Name:      "test_ent_drop_idx",
		}
		err = store.DropIndex(ctx, dropOpts)
		assert.NoError(t, err)
	})

	t.Run("InvalidInput", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Test nil options
		err := store.DropIndex(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")

		// Test empty graph name
		opts := &types.DropIndexOptions{
			GraphName: "",
			Name:      "test",
		}
		err = store.DropIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")

		// Test empty index name
		opts.GraphName = "test"
		opts.Name = ""
		err = store.DropIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("NonExistentGraph", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Test non-existent graph without IfExists
		opts := &types.DropIndexOptions{
			GraphName: "nonexistent_graph",
			Name:      "test_idx",
		}
		err := store.DropIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")

		// Test with IfExists
		opts.IfExists = true
		err = store.DropIndex(ctx, opts)
		assert.NoError(t, err) // Should succeed silently
	})

	t.Run("NotConnected", func(t *testing.T) {
		store := NewStore()
		ctx := context.Background()

		opts := &types.DropIndexOptions{
			GraphName: "test",
			Name:      "test_idx",
		}
		err := store.DropIndex(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

// =============================================================================
// Stress Tests
// =============================================================================

// TestStressSchemaOperations tests schema operations under stress
func TestStressSchemaOperations(t *testing.T) {
	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	store := NewStore()
	ctx := context.Background()

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	connectWithRetry(ctx, t, store, storeConfig)
	defer store.Close()

	// Create test graph
	graphName := "stress_schema_graph"
	err := store.CreateGraph(ctx, graphName)
	assert.NoError(t, err)
	defer store.DropGraph(ctx, graphName)

	// Add some test data using test utilities
	testNodes := CreateTestNodes(100)
	_, err = store.AddNodes(ctx, &types.AddNodesOptions{
		GraphName: graphName,
		Nodes:     testNodes,
	})
	assert.NoError(t, err)

	testRelationships := CreateTestRelationships(50)
	_, err = store.AddRelationships(ctx, &types.AddRelationshipsOptions{
		GraphName:     graphName,
		Relationships: testRelationships,
	})
	assert.NoError(t, err)

	// Use light stress configuration
	stressConfig := LightStressConfig()

	t.Run("GetSchemaStress", func(t *testing.T) {
		operation := func(ctx context.Context) error {
			schema, err := store.GetSchema(ctx, graphName)
			if err != nil {
				return err
			}
			if schema == nil {
				return fmt.Errorf("schema is nil")
			}
			return nil
		}

		result := runStressTest(stressConfig, operation)

		if result.SuccessRate < stressConfig.MinSuccessRate {
			t.Errorf("Stress test failed: success rate %.2f%% < minimum %.2f%%",
				result.SuccessRate, stressConfig.MinSuccessRate)
		}

		t.Logf("GetSchema stress test: %d operations, %.2f%% success rate, %d errors, %v duration",
			result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)
	})

	t.Run("IndexOperationsStress", func(t *testing.T) {
		operation := func(ctx context.Context) error {
			// Generate unique index name for this operation
			indexName := fmt.Sprintf("stress_idx_%d_%d", time.Now().UnixNano(), runtime.NumGoroutine())

			// Create index with IfNotExists to handle race conditions
			createOpts := &types.CreateIndexOptions{
				GraphName:   graphName,
				Name:        indexName,
				Target:      "NODE",
				Labels:      []string{"TestNode"},
				Properties:  []string{"name"},
				IndexType:   "BTREE",
				IfNotExists: true, // Handle race conditions gracefully
			}
			err := store.CreateIndex(ctx, createOpts)
			if err != nil {
				return fmt.Errorf("create index error: %w", err)
			}

			// Get schema to verify index was created
			schema, err := store.GetSchema(ctx, graphName)
			if err != nil {
				return fmt.Errorf("get schema error: %w", err)
			}
			if schema == nil {
				return fmt.Errorf("schema is nil")
			}

			// Drop index with IfExists to handle race conditions
			dropOpts := &types.DropIndexOptions{
				GraphName: graphName,
				Name:      indexName,
				IfExists:  true, // Handle race conditions gracefully
			}
			err = store.DropIndex(ctx, dropOpts)
			if err != nil {
				return fmt.Errorf("drop index error: %w", err)
			}

			return nil
		}

		result := runStressTest(stressConfig, operation)

		if result.SuccessRate < stressConfig.MinSuccessRate {
			t.Errorf("Stress test failed: success rate %.2f%% < minimum %.2f%%",
				result.SuccessRate, stressConfig.MinSuccessRate)
		}

		t.Logf("Index operations stress test: %d operations, %.2f%% success rate, %d errors, %v duration",
			result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)
	})
}

// =============================================================================
// Memory Leak Detection Tests
// =============================================================================

// TestSchemaMemoryLeakDetection tests for memory leaks in schema operations
func TestSchemaMemoryLeakDetection(t *testing.T) {
	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	// Capture initial memory stats
	initialStats := captureMemoryStats()

	store := NewStore()
	ctx := context.Background()

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	connectWithRetry(ctx, t, store, storeConfig)
	defer store.Close()

	// Create test graph
	graphName := "memory_test_schema"
	err := store.CreateGraph(ctx, graphName)
	assert.NoError(t, err)
	defer store.DropGraph(ctx, graphName)

	// Add test data
	testNodes := CreateTestNodes(1000)
	_, err = store.AddNodes(ctx, &types.AddNodesOptions{
		GraphName: graphName,
		Nodes:     testNodes,
	})
	assert.NoError(t, err)

	// Perform many schema operations (read-only to avoid index conflicts)
	iterations := 1000
	for i := 0; i < iterations; i++ {
		// Get schema multiple times - this is sufficient for memory leak testing
		schema, err := store.GetSchema(ctx, graphName)
		assert.NoError(t, err)
		assert.NotNil(t, schema)

		// Verify schema structure to exercise memory allocation
		assert.NotNil(t, schema.NodeLabels)
		// RelationshipTypes may be nil if no relationships exist
		assert.NotNil(t, schema.NodeProperties)
		// RelProperties may be nil if no relationships exist
		assert.NotNil(t, schema.Statistics)
		// Constraints and Indexes may be empty arrays but not nil
		assert.NotNil(t, schema.Constraints)
		assert.NotNil(t, schema.Indexes)

		// Access some data to ensure full traversal
		if schema.Statistics != nil {
			_ = schema.Statistics.TotalNodes
			_ = schema.Statistics.TotalRelationships
		}

		// Force garbage collection periodically
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Capture final memory stats
	finalStats := captureMemoryStats()

	// Calculate memory growth
	growth := calculateMemoryGrowth(initialStats, finalStats)

	// Allow for reasonable memory growth (50MB threshold)
	maxAllowedGrowth := int64(50 * 1024 * 1024)
	if growth.HeapAllocGrowth > maxAllowedGrowth {
		t.Errorf("Potential memory leak detected: heap memory grew by %d bytes (threshold: %d bytes)",
			growth.HeapAllocGrowth, maxAllowedGrowth)
	}

	t.Logf("Memory growth: heap=%d bytes, alloc=%d bytes, sys=%d bytes (threshold: %d bytes)",
		growth.HeapAllocGrowth, growth.AllocGrowth, growth.SysGrowth, maxAllowedGrowth)
}

// =============================================================================
// Goroutine Leak Detection Tests
// =============================================================================

// TestSchemaGoroutineLeakDetection tests for goroutine leaks in schema operations
func TestSchemaGoroutineLeakDetection(t *testing.T) {
	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	// Capture initial goroutine state
	initialGoroutines := captureGoroutineState()

	func() {
		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)
		defer store.Close()

		// Create test graph
		graphName := "goroutine_test_schema"
		err := store.CreateGraph(ctx, graphName)
		assert.NoError(t, err)
		defer store.DropGraph(ctx, graphName)

		// Add some test data
		testNodes := CreateTestNodes(100)
		_, err = store.AddNodes(ctx, &types.AddNodesOptions{
			GraphName: graphName,
			Nodes:     testNodes,
		})
		assert.NoError(t, err)

		// Perform many concurrent schema operations with real-world race conditions
		var wg sync.WaitGroup
		numWorkers := 30          // Moderate concurrency for realistic test
		operationsPerWorker := 10 // Mix of operations per worker

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < operationsPerWorker; j++ {
					// Mix different operations to test real-world scenarios
					switch j % 3 {
					case 0:
						// Get schema operation (most common)
						schema, err := store.GetSchema(ctx, graphName)
						assert.NoError(t, err)
						assert.NotNil(t, schema)
						assert.NotNil(t, schema.NodeLabels)
						assert.NotNil(t, schema.Statistics)
						assert.True(t, schema.Statistics.TotalNodes > 0)

					case 1:
						// Create index operation with race condition handling
						indexName := fmt.Sprintf("goroutine_idx_%d_%d_%d", workerID, j, time.Now().UnixNano())
						createOpts := &types.CreateIndexOptions{
							GraphName:   graphName,
							Name:        indexName,
							Target:      "NODE",
							Labels:      []string{"TestNode"},
							Properties:  []string{"name"},
							IndexType:   "BTREE",
							IfNotExists: true, // Handle race conditions gracefully
						}
						err := store.CreateIndex(ctx, createOpts)
						assert.NoError(t, err)

						// Immediately drop it to clean up
						dropOpts := &types.DropIndexOptions{
							GraphName: graphName,
							Name:      indexName,
							IfExists:  true, // Handle race conditions gracefully
						}
						err = store.DropIndex(ctx, dropOpts)
						assert.NoError(t, err)

					case 2:
						// Another schema read to stress test
						schema, err := store.GetSchema(ctx, graphName)
						assert.NoError(t, err)
						assert.NotNil(t, schema)
						// Access schema components to exercise memory allocation
						_ = len(schema.NodeLabels)
						_ = len(schema.Constraints)
						_ = len(schema.Indexes)
					}

					// Small delay to simulate real usage and allow scheduling
					time.Sleep(2 * time.Millisecond)
					runtime.Gosched()
				}
			}(i)
		}

		wg.Wait()
	}()

	// Allow time for cleanup
	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Capture final goroutine state
	finalGoroutines := captureGoroutineState()

	// Analyze goroutine changes
	leaked, _ := analyzeGoroutineChanges(initialGoroutines, finalGoroutines)

	// Filter out system goroutines
	applicationLeaks := make([]GoroutineInfo, 0)
	for _, g := range leaked {
		if !g.IsSystem {
			applicationLeaks = append(applicationLeaks, g)
		}
	}

	// Check for significant goroutine leaks
	if len(applicationLeaks) > 5 {
		t.Errorf("Potential goroutine leak detected: %d application goroutines may have leaked", len(applicationLeaks))
		t.Logf("Initial goroutines: %d, Final goroutines: %d", len(initialGoroutines), len(finalGoroutines))
		if len(applicationLeaks) > 0 {
			t.Logf("First few leaked goroutine stacks:")
			for i, g := range applicationLeaks {
				if i < 3 { // Limit output
					t.Logf("Goroutine %d: %s\nStack: %s", g.ID, g.Function, g.Stack)
				}
			}
		}
	}

	t.Logf("Goroutine changes: initial=%d, final=%d, potentially leaked=%d",
		len(initialGoroutines), len(finalGoroutines), len(applicationLeaks))
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// connectWithRetryBench is a wrapper for benchmarks that converts *testing.B to interface{}
func connectWithRetryBench(ctx context.Context, b *testing.B, store *Store, config types.GraphStoreConfig) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := store.Connect(ctx, config)
		if err == nil {
			return
		}
		if i == maxRetries-1 {
			b.Fatalf("Failed to connect after %d retries: %v", maxRetries, err)
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
}

// BenchmarkGetSchema benchmarks GetSchema method
func BenchmarkGetSchema(b *testing.B) {
	config := getTestConfig()
	if config == nil {
		b.Skip("NEO4J_TEST_URL environment variable not set")
	}

	store := NewStore()
	ctx := context.Background()

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	connectWithRetryBench(ctx, b, store, storeConfig)
	defer store.Close()

	// Create test graph with data
	graphName := "bench_schema_graph"
	err := store.CreateGraph(ctx, graphName)
	if err != nil {
		b.Fatalf("Failed to create test graph: %v", err)
	}
	defer store.DropGraph(ctx, graphName)

	// Add test data
	testNodes := CreateTestNodes(1000)
	_, err = store.AddNodes(ctx, &types.AddNodesOptions{
		GraphName: graphName,
		Nodes:     testNodes,
	})
	if err != nil {
		b.Fatalf("Failed to add test nodes: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.GetSchema(ctx, graphName)
		if err != nil {
			b.Fatalf("GetSchema failed: %v", err)
		}
	}
}

// BenchmarkCreateIndex benchmarks CreateIndex method
func BenchmarkCreateIndex(b *testing.B) {
	config := getEnterpriseTestConfig()
	if config == nil {
		b.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
	}

	store := NewStore()
	ctx := context.Background()

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	connectWithRetryBench(ctx, b, store, storeConfig)
	defer store.Close()

	// Set to use separate database mode (Enterprise feature)
	store.SetUseSeparateDatabase(true)

	// Step 1: Delete all existing graphs
	existingGraphs, err := store.ListGraphs(ctx)
	if err == nil {
		for _, graphName := range existingGraphs {
			store.DropGraph(ctx, graphName) // Ignore errors
		}
	}

	// Step 2: Create a single test graph using separate database mode
	graphName := "bench-create-index-graph"
	err = store.CreateGraph(ctx, graphName)
	if err != nil {
		b.Fatalf("Failed to create test graph: %v", err)
	}
	defer store.DropGraph(ctx, graphName)

	// Step 3: Add test data
	testNodes := CreateTestNodes(20)
	for _, node := range testNodes {
		node.Labels = []string{"BenchLabel"}
		if node.Properties == nil {
			node.Properties = make(map[string]interface{})
		}
		node.Properties["name"] = fmt.Sprintf("bench_node_%s", node.ID)
		node.Properties["category"] = "test"
	}
	_, err = store.AddNodes(ctx, &types.AddNodesOptions{
		GraphName: graphName,
		Nodes:     testNodes,
	})
	if err != nil {
		b.Fatalf("Failed to add test nodes: %v", err)
	}

	b.ResetTimer()

	// Step 4: Benchmark index creation
	for i := 0; i < b.N; i++ {
		indexName := fmt.Sprintf("benchCreateIdx%d%d", time.Now().UnixNano(), i)
		createOpts := &types.CreateIndexOptions{
			GraphName:  graphName,
			Name:       indexName,
			Target:     "NODE",
			Labels:     []string{"BenchLabel"},
			Properties: []string{"name"},
		}

		err := store.CreateIndex(ctx, createOpts)
		if err != nil {
			b.Fatalf("CreateIndex failed at iteration %d: %v", i, err)
		}

		// Clean up immediately
		dropOpts := &types.DropIndexOptions{
			GraphName: graphName,
			Name:      indexName,
			IfExists:  true,
		}
		store.DropIndex(ctx, dropOpts) // Ignore cleanup errors
	}
}

// BenchmarkDropIndex benchmarks DropIndex method
func BenchmarkDropIndex(b *testing.B) {
	config := getEnterpriseTestConfig()
	if config == nil {
		b.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
	}

	store := NewStore()
	ctx := context.Background()

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	connectWithRetryBench(ctx, b, store, storeConfig)
	defer store.Close()

	// Set to use separate database mode (Enterprise feature)
	store.SetUseSeparateDatabase(true)

	// Step 1: Delete all existing graphs
	existingGraphs, err := store.ListGraphs(ctx)
	if err == nil {
		for _, graphName := range existingGraphs {
			store.DropGraph(ctx, graphName) // Ignore errors
		}
	}

	// Step 2: Create a single test graph using separate database mode
	graphName := "bench-drop-index-graph"
	err = store.CreateGraph(ctx, graphName)
	if err != nil {
		b.Fatalf("Failed to create test graph: %v", err)
	}
	defer store.DropGraph(ctx, graphName)

	// Step 3: Add test data
	testNodes := CreateTestNodes(20)
	for _, node := range testNodes {
		node.Labels = []string{"DropLabel"}
		if node.Properties == nil {
			node.Properties = make(map[string]interface{})
		}
		node.Properties["name"] = fmt.Sprintf("drop_node_%s", node.ID)
		node.Properties["category"] = "test"
	}
	_, err = store.AddNodes(ctx, &types.AddNodesOptions{
		GraphName: graphName,
		Nodes:     testNodes,
	})
	if err != nil {
		b.Fatalf("Failed to add test nodes: %v", err)
	}

	// Step 4: Pre-create all indexes that will be dropped (not measured)
	uniqueTimestamp := time.Now().UnixNano()
	indexNames := make([]string, b.N)

	for i := 0; i < b.N; i++ {
		// Use timestamp + iteration for guaranteed uniqueness
		indexName := fmt.Sprintf("dropIdx%d%d", uniqueTimestamp, i)
		indexNames[i] = indexName

		createOpts := &types.CreateIndexOptions{
			GraphName:   graphName,
			Name:        indexName,
			Target:      "NODE",
			Labels:      []string{"DropLabel"},
			Properties:  []string{"name"},
			IfNotExists: true,
		}

		err := store.CreateIndex(ctx, createOpts)
		if err != nil {
			b.Fatalf("Failed to create index %d during setup: %v", i, err)
		}
	}

	// Wait for all indexes to be fully created
	time.Sleep(time.Millisecond * 300)

	// Step 5: Reset timer and measure drop operations
	b.ResetTimer()

	// Step 6: Benchmark index deletion
	for i := 0; i < b.N; i++ {
		dropOpts := &types.DropIndexOptions{
			GraphName: graphName,
			Name:      indexNames[i],
			IfExists:  true,
		}

		err := store.DropIndex(ctx, dropOpts)
		if err != nil {
			b.Fatalf("DropIndex failed at iteration %d: %v", i, err)
		}
	}
}
