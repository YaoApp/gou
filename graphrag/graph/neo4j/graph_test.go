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
// Tests for graph.go methods
// =============================================================================

// TestCreateGraph tests the CreateGraph method
func TestCreateGraph(t *testing.T) {
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

		// Test creating a graph
		err := store.CreateGraph(ctx, "test_graph")
		assert.NoError(t, err)

		// Verify graph exists
		exists, err := store.GraphExists(ctx, "test_graph")
		assert.NoError(t, err)
		assert.False(t, exists) // Should be false until nodes are added

		// Clean up
		store.DropGraph(ctx, "test_graph")
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

		// Test creating a graph (database)
		err := store.CreateGraph(ctx, "testenterprise")
		assert.NoError(t, err)

		// Verify graph exists
		exists, err := store.GraphExists(ctx, "testenterprise")
		assert.NoError(t, err)
		assert.True(t, exists)

		// Clean up
		store.DropGraph(ctx, "testenterprise")
	})

	t.Run("InvalidGraphName", func(t *testing.T) {
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

		// Test invalid graph names
		invalidNames := []string{"", "test.graph", "test space", "test@graph", "test#graph", "test!graph"}
		for _, name := range invalidNames {
			err := store.CreateGraph(ctx, name)
			assert.Error(t, err, "Should fail for invalid name: %s", name)
		}
	})

	t.Run("NotConnected", func(t *testing.T) {
		store := NewStore()
		ctx := context.Background()

		err := store.CreateGraph(ctx, "test_graph")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

// TestDropGraph tests the DropGraph method
func TestDropGraph(t *testing.T) {
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

		// Create and drop a graph
		err := store.CreateGraph(ctx, "testdropgraph")
		assert.NoError(t, err)

		err = store.DropGraph(ctx, "testdropgraph")
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

		// Create and drop a graph
		err := store.CreateGraph(ctx, "testdrop")
		assert.NoError(t, err)

		err = store.DropGraph(ctx, "testdrop")
		assert.NoError(t, err)

		// Verify it's gone
		exists, err := store.GraphExists(ctx, "testdrop")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("DropDefaultDatabase", func(t *testing.T) {
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

		// Should not be able to drop default database
		err := store.DropGraph(ctx, DefaultDatabase)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot drop default database")
	})
}

// TestGraphExists tests the GraphExists method
func TestGraphExists(t *testing.T) {
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

		// Non-existent graph
		exists, err := store.GraphExists(ctx, "non_existent_graph")
		assert.NoError(t, err)
		assert.False(t, exists)
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

		// Default database should exist
		exists, err := store.GraphExists(ctx, DefaultDatabase)
		assert.NoError(t, err)
		assert.True(t, exists)
	})
}

// TestListGraphs tests the ListGraphs method
func TestListGraphs(t *testing.T) {
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

		// List graphs (should be empty initially)
		graphs, err := store.ListGraphs(ctx)
		assert.NoError(t, err)
		assert.IsType(t, []string{}, graphs)
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

		// List databases
		graphs, err := store.ListGraphs(ctx)
		assert.NoError(t, err)
		assert.Contains(t, graphs, DefaultDatabase) // Should contain default database
	})
}

// TestDescribeGraph tests the DescribeGraph method
func TestDescribeGraph(t *testing.T) {
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

		// Create a graph and describe it
		err := store.CreateGraph(ctx, "testdescribe")
		assert.NoError(t, err)

		stats, err := store.DescribeGraph(ctx, "testdescribe")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, stats.TotalNodes, int64(0))
		assert.GreaterOrEqual(t, stats.TotalRelationships, int64(0))
		assert.Equal(t, "label_based", stats.ExtraStats["storage_type"])

		// Clean up
		store.DropGraph(ctx, "testdescribe")
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

		// Describe default database
		stats, err := store.DescribeGraph(ctx, DefaultDatabase)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, stats.TotalNodes, int64(0))
		assert.GreaterOrEqual(t, stats.TotalRelationships, int64(0))
		assert.Equal(t, "separate_database", stats.ExtraStats["storage_type"])
	})
}

// TestHelperFunctions tests helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("isValidGraphName", func(t *testing.T) {
		validNames := []string{"test", "test_graph", "TestGraph", "test123", "Test_Graph_123", "test-graph", "test-123", "Test-Graph-123"}
		for _, name := range validNames {
			assert.True(t, isValidGraphName(name), "Should be valid: %s", name)
		}

		invalidNames := []string{"", "test.graph", "test space", "test@graph", "test#graph", "test!graph", "test+graph", "test*graph"}
		for _, name := range invalidNames {
			assert.False(t, isValidGraphName(name), "Should be invalid: %s", name)
		}
	})

	t.Run("GetGraphLabel", func(t *testing.T) {
		// Test with default prefix
		store := NewStore()
		label := store.GetGraphLabel("test_graph")
		assert.Equal(t, "__Graph_test_graph", label)

		// Test with custom prefix from config
		store.config.DriverConfig = map[string]interface{}{
			"graph_label_prefix": "MyApp_",
		}
		label = store.GetGraphLabel("test_graph")
		assert.Equal(t, "MyApp_test_graph", label)
	})

	t.Run("GetGraphDatabase", func(t *testing.T) {
		// Community edition
		store := NewStore()
		db := store.GetGraphDatabase("test_graph")
		assert.Equal(t, DefaultDatabase, db)

		// Separate database mode
		store.SetUseSeparateDatabase(true)
		db = store.GetGraphDatabase("test_graph")
		assert.Equal(t, "test_graph", db)
	})

	t.Run("ConfigurablePrefixes", func(t *testing.T) {
		store := NewStore()

		// Test default prefixes
		assert.Equal(t, DefaultGraphLabelPrefix, store.getGraphLabelPrefix())
		assert.Equal(t, DefaultGraphNamespaceProperty, store.getGraphNamespaceProperty())

		// Test custom prefixes from config
		store.config.DriverConfig = map[string]interface{}{
			"graph_label_prefix":       "CustomApp_",
			"graph_namespace_property": "__custom_namespace",
		}
		assert.Equal(t, "CustomApp_", store.getGraphLabelPrefix())
		assert.Equal(t, "__custom_namespace", store.getGraphNamespaceProperty())

		// Test empty prefixes should use defaults
		store.config.DriverConfig = map[string]interface{}{
			"graph_label_prefix":       "",
			"graph_namespace_property": "",
		}
		assert.Equal(t, DefaultGraphLabelPrefix, store.getGraphLabelPrefix())
		assert.Equal(t, DefaultGraphNamespaceProperty, store.getGraphNamespaceProperty())

		// Test nil config should use defaults
		store.config.DriverConfig = nil
		assert.Equal(t, DefaultGraphLabelPrefix, store.getGraphLabelPrefix())
		assert.Equal(t, DefaultGraphNamespaceProperty, store.getGraphNamespaceProperty())
	})
}

// ===== Stress Tests =====

func TestStressGraphOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Use light stress config
	stressConfig := LightStressConfig()

	t.Logf("Starting graph operations stress test: %d workers, %d operations per worker",
		stressConfig.NumWorkers, stressConfig.OperationsPerWorker)

	// Capture initial state
	initialGoroutines := captureGoroutineState()
	initialMemory := captureMemoryStats()

	// Define the test operation
	operation := func(ctx context.Context) error {
		store := NewStore()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			return err
		}
		defer store.Close()

		graphName := fmt.Sprintf("stress_test_%d", time.Now().UnixNano())

		// Create graph
		err = store.CreateGraph(ctx, graphName)
		if err != nil {
			return err
		}

		// Check existence
		exists, err := store.GraphExists(ctx, graphName)
		if err != nil {
			return err
		}
		if !exists {
			// For community edition, this is expected until nodes are added
		}

		// List graphs
		_, err = store.ListGraphs(ctx)
		if err != nil {
			return err
		}

		// Describe graph
		_, err = store.DescribeGraph(ctx, graphName)
		if err != nil {
			return err
		}

		// Drop graph
		return store.DropGraph(ctx, graphName)
	}

	// Run stress test
	result := runStressTest(stressConfig, operation)

	t.Logf("Graph stress test completed: %d total operations, %d errors, %.2f%% success rate, duration: %v",
		result.TotalOperations, result.ErrorCount, result.SuccessRate, result.Duration)

	// Verify success rate
	assert.GreaterOrEqual(t, result.SuccessRate, stressConfig.MinSuccessRate,
		"Success rate %.2f%% is below minimum %.2f%%", result.SuccessRate, stressConfig.MinSuccessRate)

	// Check for leaks
	time.Sleep(2 * time.Second)
	finalGoroutines := captureGoroutineState()
	leaked, _ := analyzeGoroutineChanges(initialGoroutines, finalGoroutines)

	var appLeaked []GoroutineInfo
	for _, g := range leaked {
		if !g.IsSystem {
			appLeaked = append(appLeaked, g)
		}
	}

	if len(appLeaked) > 0 {
		t.Logf("Potential goroutine leaks in graph stress test (%d):", len(appLeaked))
		for _, g := range appLeaked {
			t.Logf("  Goroutine %d [%s]: %s", g.ID, g.State, g.Function)
		}
	}

	// Check memory growth
	finalMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(initialMemory, finalMemory)

	t.Logf("Graph stress test memory growth: Alloc=%d, HeapAlloc=%d, Sys=%d",
		memGrowth.AllocGrowth, memGrowth.HeapAllocGrowth, memGrowth.SysGrowth)
}

func TestConcurrentGraphOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	const numGoroutines = 10
	const operationsPerGoroutine = 5

	t.Logf("Starting concurrent graph operations test: %d goroutines, %d operations each",
		numGoroutines, operationsPerGoroutine)

	// Capture initial state
	initialGoroutines := captureGoroutineState()
	initialMemory := captureMemoryStats()

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Start concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				func() {
					store := NewStore()

					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					err := store.Connect(ctx, storeConfig)
					if err != nil {
						errors <- fmt.Errorf("worker %d, op %d: Connect failed: %w", workerID, j, err)
						return
					}
					defer store.Close()

					graphName := fmt.Sprintf("concurrent_test_%d_%d", workerID, j)

					// Create graph
					err = store.CreateGraph(ctx, graphName)
					if err != nil {
						errors <- fmt.Errorf("worker %d, op %d: CreateGraph failed: %w", workerID, j, err)
						return
					}

					// List graphs
					_, err = store.ListGraphs(ctx)
					if err != nil {
						errors <- fmt.Errorf("worker %d, op %d: ListGraphs failed: %w", workerID, j, err)
						return
					}

					// Drop graph
					err = store.DropGraph(ctx, graphName)
					if err != nil {
						errors <- fmt.Errorf("worker %d, op %d: DropGraph failed: %w", workerID, j, err)
						return
					}
				}()
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(errors)

	// Count errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent graph test error: %v", err)
		errorCount++
	}

	totalOperations := numGoroutines * operationsPerGoroutine
	successRate := float64(totalOperations-errorCount) / float64(totalOperations) * 100

	t.Logf("Concurrent graph test completed: %d total operations, %d errors, %.2f%% success rate",
		totalOperations, errorCount, successRate)

	// Verify success rate
	assert.GreaterOrEqual(t, successRate, 90.0,
		"Success rate %.2f%% is below minimum 90%%", successRate)

	// Check for leaks
	time.Sleep(2 * time.Second)
	finalGoroutines := captureGoroutineState()
	leaked, _ := analyzeGoroutineChanges(initialGoroutines, finalGoroutines)

	var appLeaked []GoroutineInfo
	for _, g := range leaked {
		if !g.IsSystem {
			appLeaked = append(appLeaked, g)
		}
	}

	if len(appLeaked) > 0 {
		t.Logf("Potential goroutine leaks after concurrent graph test (%d):", len(appLeaked))
		for _, g := range appLeaked {
			t.Logf("  Goroutine %d [%s]: %s", g.ID, g.State, g.Function)
		}
	}

	// Check memory growth
	finalMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(initialMemory, finalMemory)

	t.Logf("Concurrent graph test memory growth: Alloc=%d, HeapAlloc=%d, Sys=%d",
		memGrowth.AllocGrowth, memGrowth.HeapAllocGrowth, memGrowth.SysGrowth)
}

func TestGraphMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Capture baseline
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	baselineMemory := captureMemoryStats()
	t.Logf("Graph test baseline memory: Alloc=%d, HeapAlloc=%d, Sys=%d",
		baselineMemory.Alloc, baselineMemory.HeapAlloc, baselineMemory.Sys)

	// Perform multiple graph operation cycles
	const cycles = 50
	for i := 0; i < cycles; i++ {
		func() {
			store := NewStore()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := store.Connect(ctx, storeConfig)
			if err != nil {
				t.Skipf("Failed to connect in cycle %d: %v", i, err)
			}
			defer store.Close()

			graphName := fmt.Sprintf("memory_test_%d", i)

			// Full graph lifecycle
			err = store.CreateGraph(ctx, graphName)
			assert.NoError(t, err)

			_, err = store.GraphExists(ctx, graphName)
			assert.NoError(t, err)

			_, err = store.ListGraphs(ctx)
			assert.NoError(t, err)

			_, err = store.DescribeGraph(ctx, graphName)
			assert.NoError(t, err)

			err = store.DropGraph(ctx, graphName)
			assert.NoError(t, err)
		}()

		// Force GC every 10 cycles
		if i%10 == 9 {
			runtime.GC()
		}
	}

	// Final measurement
	runtime.GC()
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	finalMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(baselineMemory, finalMemory)

	t.Logf("Graph test final memory: Alloc=%d, HeapAlloc=%d, Sys=%d",
		finalMemory.Alloc, finalMemory.HeapAlloc, finalMemory.Sys)
	t.Logf("Graph test memory growth: Alloc=%d, HeapAlloc=%d, Sys=%d, GC cycles=%d",
		memGrowth.AllocGrowth, memGrowth.HeapAllocGrowth, memGrowth.SysGrowth, memGrowth.NumGCDiff)

	// Check for excessive memory growth
	const maxAllowedGrowth = 3 * 1024 * 1024 // 3MB
	if memGrowth.HeapAllocGrowth > maxAllowedGrowth {
		t.Errorf("Excessive memory growth in graph operations: %d bytes (max allowed: %d)",
			memGrowth.HeapAllocGrowth, maxAllowedGrowth)
	}
}

func TestGraphGoroutineLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Capture baseline
	baselineGoroutines := captureGoroutineState()
	t.Logf("Graph test baseline goroutines: %d total", len(baselineGoroutines))

	// Perform multiple graph operation cycles
	const cycles = 30
	for i := 0; i < cycles; i++ {
		func() {
			store := NewStore()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := store.Connect(ctx, storeConfig)
			if err != nil {
				t.Skipf("Failed to connect in cycle %d: %v", i, err)
			}
			defer store.Close()

			graphName := fmt.Sprintf("goroutine_test_%d", i)

			// Full graph lifecycle
			store.CreateGraph(ctx, graphName)
			store.GraphExists(ctx, graphName)
			store.ListGraphs(ctx)
			store.DescribeGraph(ctx, graphName)
			store.DropGraph(ctx, graphName)
		}()
	}

	// Allow cleanup
	time.Sleep(2 * time.Second)
	runtime.GC()

	// Capture final state
	finalGoroutines := captureGoroutineState()
	t.Logf("Graph test final goroutines: %d total", len(finalGoroutines))

	// Analyze changes
	leaked, cleaned := analyzeGoroutineChanges(baselineGoroutines, finalGoroutines)

	t.Logf("Graph test goroutine changes: %d leaked, %d cleaned", len(leaked), len(cleaned))

	// Filter out system goroutines
	var appLeaked []GoroutineInfo
	for _, g := range leaked {
		if !g.IsSystem {
			appLeaked = append(appLeaked, g)
		}
	}

	if len(appLeaked) > 0 {
		t.Logf("Application goroutine leaks in graph operations (%d):", len(appLeaked))
		for _, g := range appLeaked {
			t.Logf("  Goroutine %d [%s]: %s", g.ID, g.State, g.Function)
			if len(g.Stack) > 0 {
				t.Logf("    Stack: %s", strings.Split(g.Stack, "\n")[0])
			}
		}

		// Fail if too many leaks
		if len(appLeaked) > 3 {
			t.Errorf("Too many application goroutine leaks in graph operations: %d (threshold: 3)", len(appLeaked))
		}
	}
}

func TestCustomPrefixIntegration(t *testing.T) {
	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	store := NewStore()
	ctx := context.Background()

	// Configure with custom prefixes
	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username":                 config.User,
			"password":                 config.Password,
			"graph_label_prefix":       "TestApp_",
			"graph_namespace_property": "__test_namespace",
		},
	}

	connectWithRetry(ctx, t, store, storeConfig)
	defer store.Close()

	// Test that custom prefixes are used
	assert.Equal(t, "TestApp_", store.getGraphLabelPrefix())
	assert.Equal(t, "__test_namespace", store.getGraphNamespaceProperty())

	// Test graph operations with custom prefixes
	graphName := "custom_prefix_test"

	// Create graph
	err := store.CreateGraph(ctx, graphName)
	assert.NoError(t, err)

	// Verify GetGraphLabel uses custom prefix
	expectedLabel := "TestApp_" + graphName
	actualLabel := store.GetGraphLabel(graphName)
	assert.Equal(t, expectedLabel, actualLabel)

	// Test describe graph to ensure custom prefixes work in statistics
	stats, err := store.DescribeGraph(ctx, graphName)
	assert.NoError(t, err)
	assert.Equal(t, "label_based", stats.ExtraStats["storage_type"])
	assert.Equal(t, expectedLabel, stats.ExtraStats["__graph_label"])

	// Clean up
	err = store.DropGraph(ctx, graphName)
	assert.NoError(t, err)
}
