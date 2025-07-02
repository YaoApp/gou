package neo4j

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

const (
	testGraphName = "test_nodes_graph"
	testTimeout   = 30 * time.Second
)

// TestAddNodes_Basic tests basic AddNodes functionality
func TestAddNodes_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Test with empty nodes list
	opts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     []*types.GraphNode{},
	}
	nodeIDs, err := store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("AddNodes with empty list failed: %v", err)
	}
	if len(nodeIDs) != 0 {
		t.Errorf("Expected 0 node IDs, got %d", len(nodeIDs))
	}

	// Test with single node
	testNodes := CreateTestNodes(1)
	opts = &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	nodeIDs, err = store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("AddNodes with single node failed: %v", err)
	}
	if len(nodeIDs) != 1 {
		t.Errorf("Expected 1 node ID, got %d", len(nodeIDs))
	}
	if nodeIDs[0] != testNodes[0].ID {
		t.Errorf("Expected node ID %s, got %s", testNodes[0].ID, nodeIDs[0])
	}
}

// TestAddNodes_BatchSize tests batch size functionality
func TestAddNodes_BatchSize(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Test with batch size smaller than total nodes
	testNodes := CreateTestNodes(10)
	opts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
		BatchSize: 3,
	}
	nodeIDs, err := store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("AddNodes with batch size failed: %v", err)
	}
	if len(nodeIDs) != 10 {
		t.Errorf("Expected 10 node IDs, got %d", len(nodeIDs))
	}
}

// TestAddNodes_Upsert tests upsert functionality
func TestAddNodes_Upsert(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// First insertion
	testNodes := CreateTestNodes(2)
	opts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
		Upsert:    false,
	}
	nodeIDs, err := store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("First AddNodes failed: %v", err)
	}
	if len(nodeIDs) != 2 {
		t.Errorf("Expected 2 node IDs, got %d", len(nodeIDs))
	}

	// Try to insert same nodes again without upsert (should fail or create duplicates)
	// Update node properties for upsert test
	testNodes[0].Properties["updated"] = true
	testNodes[0].Description = "Updated description"

	opts.Upsert = true
	nodeIDs, err = store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("Upsert AddNodes failed: %v", err)
	}
	if len(nodeIDs) != 2 {
		t.Errorf("Expected 2 node IDs for upsert, got %d", len(nodeIDs))
	}
}

// TestAddNodes_WithTimeout tests timeout functionality
func TestAddNodes_WithTimeout(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	testNodes := CreateTestNodes(5)
	opts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
		Timeout:   1, // 1 second timeout
	}

	start := time.Now()
	nodeIDs, err := store.AddNodes(ctx, opts)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("AddNodes with timeout failed: %v", err)
	}
	if len(nodeIDs) != 5 {
		t.Errorf("Expected 5 node IDs, got %d", len(nodeIDs))
	}

	// Operation should complete within timeout + buffer
	if elapsed > 2*time.Second {
		t.Errorf("Operation took too long: %v", elapsed)
	}
}

// TestAddNodes_LabelBasedMode tests label-based storage mode (community edition)
func TestAddNodes_LabelBasedMode(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	// Force label-based mode
	store.SetUseSeparateDatabase(false)
	store.SetIsEnterpriseEdition(false)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	testNodes := CreateTestNodes(3)
	opts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}

	nodeIDs, err := store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("AddNodes in label-based mode failed: %v", err)
	}
	if len(nodeIDs) != 3 {
		t.Errorf("Expected 3 node IDs, got %d", len(nodeIDs))
	}
}

// TestAddNodes_SeparateDatabaseMode tests separate database mode (enterprise edition)
func TestAddNodes_SeparateDatabaseMode(t *testing.T) {
	if !hasEnterpriseConnection() {
		t.Skip("Skipping enterprise-only test: NEO4J_TEST_ENTERPRISE_URL not set")
	}

	store := setupEnterpriseTestStore(t)
	defer cleanupTestStore(t, store)

	// Force separate database mode
	store.SetUseSeparateDatabase(true)
	store.SetIsEnterpriseEdition(true)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Use a database-compatible name for enterprise mode (no underscores)
	enterpriseGraphName := "testnodesgraph"

	// Create test graph
	err := store.CreateGraph(ctx, enterpriseGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, enterpriseGraphName)
	}()

	testNodes := CreateTestNodes(3)
	opts := &types.AddNodesOptions{
		GraphName: enterpriseGraphName,
		Nodes:     testNodes,
	}

	nodeIDs, err := store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("AddNodes in separate database mode failed: %v", err)
	}
	if len(nodeIDs) != 3 {
		t.Errorf("Expected 3 node IDs, got %d", len(nodeIDs))
	}
}

// TestAddNodes_RealData tests with real test data from semantic files
func TestAddNodes_RealData(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Load Chinese test data
	zhNodes, _, err := LoadTestDataset("zh")
	if err != nil {
		t.Skipf("Skipping real data test: %v", err)
	}

	if len(zhNodes) == 0 {
		t.Skip("No Chinese test data available")
	}

	// Test with subset of real data
	maxNodes := 20
	if len(zhNodes) > maxNodes {
		zhNodes = zhNodes[:maxNodes]
	}

	opts := &types.AddNodesOptions{
		GraphName: testGraphName + "_zh",
		Nodes:     zhNodes,
		BatchSize: 5,
	}

	// Create graph for Chinese data
	err = store.CreateGraph(ctx, testGraphName+"_zh")
	if err != nil {
		t.Fatalf("Failed to create Chinese test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName+"_zh")
	}()

	nodeIDs, err := store.AddNodes(ctx, opts)
	if err != nil {
		t.Fatalf("AddNodes with real Chinese data failed: %v", err)
	}

	if len(nodeIDs) != len(zhNodes) {
		t.Errorf("Expected %d node IDs, got %d", len(zhNodes), len(nodeIDs))
	}

	// Load English test data if available
	enNodes, _, err := LoadTestDataset("en")
	if err == nil && len(enNodes) > 0 {
		if len(enNodes) > maxNodes {
			enNodes = enNodes[:maxNodes]
		}

		opts = &types.AddNodesOptions{
			GraphName: testGraphName + "_en",
			Nodes:     enNodes,
			BatchSize: 10,
		}

		// Create graph for English data
		err = store.CreateGraph(ctx, testGraphName+"_en")
		if err != nil {
			t.Fatalf("Failed to create English test graph: %v", err)
		}
		defer func() {
			_ = store.DropGraph(ctx, testGraphName+"_en")
		}()

		nodeIDs, err = store.AddNodes(ctx, opts)
		if err != nil {
			t.Fatalf("AddNodes with real English data failed: %v", err)
		}

		if len(nodeIDs) != len(enNodes) {
			t.Errorf("Expected %d node IDs for English data, got %d", len(enNodes), len(nodeIDs))
		}
	}
}

// TestAddNodes_ConcurrentStress tests concurrent operations and stress
func TestAddNodes_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), 2*testTimeout)
	defer cancel()

	// Capture initial state for leak detection
	beforeGoroutines := captureGoroutineState()
	beforeMemory := captureMemoryStats()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Stress test configuration
	config := LightStressConfig()
	if os.Getenv("STRESS_TEST_FULL") == "true" {
		config = DefaultStressConfig()
	}

	// Run stress test
	operation := func(ctx context.Context) error {
		testNodes := CreateTestNodes(5)
		opts := &types.AddNodesOptions{
			GraphName: testGraphName,
			Nodes:     testNodes,
			BatchSize: 2,
			Upsert:    true, // Use upsert to avoid conflicts
		}

		_, err := store.AddNodes(ctx, opts)
		return err
	}

	result := runStressTest(config, operation)

	// Check results
	if result.SuccessRate < config.MinSuccessRate {
		t.Errorf("Stress test success rate %.2f%% is below minimum %.2f%%",
			result.SuccessRate, config.MinSuccessRate)
	}

	t.Logf("Stress test completed: %d operations, %.2f%% success rate, %d errors, duration: %v",
		result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)

	// Allow some time for cleanup
	time.Sleep(2 * time.Second)

	// Check for goroutine leaks
	afterGoroutines := captureGoroutineState()
	leaked, cleaned := analyzeGoroutineChanges(beforeGoroutines, afterGoroutines)

	if len(leaked) > 0 {
		t.Logf("Potential goroutine leaks detected: %d new goroutines", len(leaked))
		for _, g := range leaked {
			if !g.IsSystem {
				t.Errorf("Application goroutine leak detected: ID=%d, State=%s, Function=%s",
					g.ID, g.State, g.Function)
			}
		}
	}

	if len(cleaned) > 0 {
		t.Logf("Goroutines cleaned up: %d", len(cleaned))
	}

	// Check for memory leaks
	afterMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(beforeMemory, afterMemory)

	// Allow for some memory growth during tests, but not excessive
	maxAllowedGrowth := int64(50 * 1024 * 1024) // 50MB
	if memGrowth.HeapAllocGrowth > maxAllowedGrowth {
		t.Errorf("Excessive memory growth detected: %d bytes heap allocation growth",
			memGrowth.HeapAllocGrowth)
	}

	t.Logf("Memory growth: Heap=%d bytes, Total=%d bytes, GC cycles=%d",
		memGrowth.HeapAllocGrowth, memGrowth.AllocGrowth, memGrowth.NumGCDiff)
}

// TestAddNodes_ErrorHandling tests error scenarios
func TestAddNodes_ErrorHandling(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test with nil options
	_, err := store.AddNodes(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// Test with empty graph name
	opts := &types.AddNodesOptions{
		GraphName: "",
		Nodes:     CreateTestNodes(1),
	}
	_, err = store.AddNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error for empty graph name")
	}

	// Test with invalid graph name
	opts.GraphName = "invalid-graph-name"
	_, err = store.AddNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error for invalid graph name")
	}

	// Test with node missing ID
	opts.GraphName = testGraphName
	opts.Nodes = []*types.GraphNode{
		{
			Labels:     []string{"Test"},
			Properties: map[string]interface{}{"name": "test"},
		},
	}

	err = store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	_, err = store.AddNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error for node missing ID")
	}
}

// TestAddNodes_Disconnected tests behavior when store is disconnected
func TestAddNodes_Disconnected(t *testing.T) {
	store := NewStore()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	opts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     CreateTestNodes(1),
	}

	_, err := store.AddNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error when store is not connected")
	}
}

// TestAddNodes_HighConcurrency tests heavy concurrent load
func TestAddNodes_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")
	}

	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), 3*testTimeout)
	defer cancel()

	// Capture initial state
	beforeGoroutines := captureGoroutineState()
	beforeMemory := captureMemoryStats()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// High concurrency configuration
	numWorkers := 20
	operationsPerWorker := 10
	nodesBatch := 5

	var wg sync.WaitGroup
	errChan := make(chan error, numWorkers*operationsPerWorker)
	successCount := int64(0)
	totalOps := int64(0)

	t.Logf("Starting high concurrency test: %d workers, %d ops/worker, %d nodes/batch",
		numWorkers, operationsPerWorker, nodesBatch)

	startTime := time.Now()

	// Launch concurrent workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				atomic.AddInt64(&totalOps, 1)

				// Create unique nodes for each operation to avoid conflicts
				testNodes := make([]*types.GraphNode, nodesBatch)
				for k := 0; k < nodesBatch; k++ {
					testNodes[k] = &types.GraphNode{
						ID:     fmt.Sprintf("concurrent_worker_%d_op_%d_node_%d", workerID, j, k),
						Labels: []string{"ConcurrentTest", fmt.Sprintf("Worker%d", workerID)},
						Properties: map[string]interface{}{
							"worker_id":    workerID,
							"operation_id": j,
							"node_index":   k,
							"timestamp":    time.Now().Unix(),
						},
						Confidence: 0.95,
						CreatedAt:  time.Now(),
						Version:    1,
					}
				}

				opts := &types.AddNodesOptions{
					GraphName: testGraphName,
					Nodes:     testNodes,
					BatchSize: 3,
					Upsert:    false, // Use create to ensure we're testing conflicts
					Timeout:   30,    // 30 second timeout per operation
				}

				_, err := store.AddNodes(ctx, opts)
				if err != nil {
					errChan <- fmt.Errorf("worker %d, operation %d: %w", workerID, j, err)
				} else {
					atomic.AddInt64(&successCount, 1)
				}

				// Small random delay to increase timing variations
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(errChan)
	duration := time.Since(startTime)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	finalSuccessCount := atomic.LoadInt64(&successCount)
	finalTotalOps := atomic.LoadInt64(&totalOps)
	successRate := float64(finalSuccessCount) / float64(finalTotalOps) * 100

	t.Logf("High concurrency test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total operations: %d", finalTotalOps)
	t.Logf("  Successful operations: %d", finalSuccessCount)
	t.Logf("  Success rate: %.2f%%", successRate)
	t.Logf("  Errors: %d", len(errors))

	// Log first few errors for diagnosis
	if len(errors) > 0 {
		t.Logf("Sample errors:")
		for i, err := range errors {
			if i >= 5 {
				break
			}
			t.Logf("  %v", err)
		}
	}

	// We expect some level of success in high concurrency
	minSuccessRate := 80.0
	if successRate < minSuccessRate {
		t.Errorf("Success rate %.2f%% is below minimum %.2f%%", successRate, minSuccessRate)
	}

	// Allow time for cleanup and connection settling
	time.Sleep(3 * time.Second)

	// Comprehensive goroutine leak detection
	afterGoroutines := captureGoroutineState()
	leaked, cleaned := analyzeGoroutineChanges(beforeGoroutines, afterGoroutines)

	if len(leaked) > 0 {
		t.Logf("Goroutine analysis after high concurrency test:")
		t.Logf("  Leaked goroutines: %d", len(leaked))
		t.Logf("  Cleaned goroutines: %d", len(cleaned))

		// Be more strict about application goroutine leaks
		appLeaks := 0
		for _, g := range leaked {
			if !g.IsSystem {
				t.Errorf("Application goroutine leak: ID=%d, State=%s, Function=%s",
					g.ID, g.State, g.Function)
				t.Errorf("  Stack trace: %s", g.Stack)
				appLeaks++
			}
		}

		if appLeaks > 0 {
			t.Errorf("Found %d application goroutine leaks", appLeaks)
		}
	}

	// Comprehensive memory leak detection
	afterMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(beforeMemory, afterMemory)

	t.Logf("Memory analysis after high concurrency test:")
	t.Logf("  Heap allocation growth: %d bytes", memGrowth.HeapAllocGrowth)
	t.Logf("  Total allocation growth: %d bytes", memGrowth.AllocGrowth)
	t.Logf("  System memory growth: %d bytes", memGrowth.SysGrowth)
	t.Logf("  GC cycles: %d", memGrowth.NumGCDiff)

	// Allow for some memory growth but flag excessive growth
	maxHeapGrowth := int64(100 * 1024 * 1024) // 100MB
	if memGrowth.HeapAllocGrowth > maxHeapGrowth {
		t.Errorf("Excessive heap memory growth: %d bytes (max allowed: %d bytes)",
			memGrowth.HeapAllocGrowth, maxHeapGrowth)
	}

	// Check for memory efficiency
	avgMemoryPerOp := memGrowth.HeapAllocGrowth / finalTotalOps
	maxMemoryPerOp := int64(1024 * 1024) // 1MB per operation
	if avgMemoryPerOp > maxMemoryPerOp {
		t.Errorf("Memory usage per operation too high: %d bytes/op (max: %d bytes/op)",
			avgMemoryPerOp, maxMemoryPerOp)
	}

	t.Logf("Memory efficiency: %d bytes per operation", avgMemoryPerOp)
}

// TestAddNodes_MemoryLeakDetection focused memory leak test
func TestAddNodes_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), 2*testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Baseline memory measurement
	runtime.GC()
	runtime.GC() // Double GC to ensure clean state
	time.Sleep(500 * time.Millisecond)
	baselineMemory := captureMemoryStats()

	// Run multiple iterations to check for memory leaks
	iterations := 50
	nodesPerIteration := 20

	t.Logf("Running memory leak detection: %d iterations, %d nodes per iteration",
		iterations, nodesPerIteration)

	var memorySnapshots []MemoryStats
	memorySnapshots = append(memorySnapshots, baselineMemory)

	for i := 0; i < iterations; i++ {
		// Create and add nodes
		testNodes := CreateTestNodes(nodesPerIteration)

		// Use unique IDs to avoid conflicts
		for j, node := range testNodes {
			node.ID = fmt.Sprintf("leak_test_iter_%d_node_%d", i, j)
		}

		opts := &types.AddNodesOptions{
			GraphName: testGraphName,
			Nodes:     testNodes,
			BatchSize: 5,
			Upsert:    true,
		}

		_, err := store.AddNodes(ctx, opts)
		if err != nil {
			t.Fatalf("AddNodes failed at iteration %d: %v", i, err)
		}

		// Force GC every 10 iterations and capture memory
		if i%10 == 9 {
			runtime.GC()
			runtime.GC()
			time.Sleep(100 * time.Millisecond)
			snapshot := captureMemoryStats()
			memorySnapshots = append(memorySnapshots, snapshot)

			t.Logf("Iteration %d: Heap=%d bytes, Total=%d bytes",
				i+1, snapshot.HeapAlloc, snapshot.TotalAlloc)
		}
	}

	// Final memory measurement
	runtime.GC()
	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	finalMemory := captureMemoryStats()
	memorySnapshots = append(memorySnapshots, finalMemory)

	// Analyze memory growth trend
	totalGrowth := calculateMemoryGrowth(baselineMemory, finalMemory)

	t.Logf("Memory leak analysis:")
	t.Logf("  Baseline heap: %d bytes", baselineMemory.HeapAlloc)
	t.Logf("  Final heap: %d bytes", finalMemory.HeapAlloc)
	t.Logf("  Total heap growth: %d bytes", totalGrowth.HeapAllocGrowth)
	t.Logf("  Total allocations growth: %d bytes", totalGrowth.TotalAllocGrowth)
	t.Logf("  GC cycles: %d", totalGrowth.NumGCDiff)

	// Check for linear memory growth (potential leak)
	if len(memorySnapshots) >= 3 {
		// Check if memory consistently grows between snapshots
		consecutiveGrowth := 0
		for i := 1; i < len(memorySnapshots); i++ {
			if memorySnapshots[i].HeapAlloc > memorySnapshots[i-1].HeapAlloc {
				consecutiveGrowth++
			}
		}

		growthRate := float64(consecutiveGrowth) / float64(len(memorySnapshots)-1)
		t.Logf("  Memory growth rate: %.2f%% of measurements", growthRate*100)

		// Only flag potential memory leak if:
		// 1. Memory grows in >95% of measurements (almost always growing)
		// 2. AND absolute growth is significant (>10MB)
		if growthRate > 0.95 && totalGrowth.HeapAllocGrowth > 10*1024*1024 {
			t.Errorf("Potential memory leak detected: memory grew in %.2f%% of measurements with %d bytes growth",
				growthRate*100, totalGrowth.HeapAllocGrowth)
		} else if growthRate > 0.95 {
			t.Logf("High growth rate (%.2f%%) but low absolute growth (%d bytes) - likely normal operation",
				growthRate*100, totalGrowth.HeapAllocGrowth)
		}
	}

	// Check absolute memory growth limits
	maxAcceptableGrowth := int64(50 * 1024 * 1024) // 50MB
	if totalGrowth.HeapAllocGrowth > maxAcceptableGrowth {
		t.Errorf("Excessive memory growth: %d bytes (max acceptable: %d bytes)",
			totalGrowth.HeapAllocGrowth, maxAcceptableGrowth)
	}

	// Memory efficiency check
	totalNodes := iterations * nodesPerIteration
	avgMemoryPerNode := totalGrowth.HeapAllocGrowth / int64(totalNodes)
	maxMemoryPerNode := int64(10 * 1024) // 10KB per node

	if avgMemoryPerNode > maxMemoryPerNode {
		t.Errorf("Memory usage per node too high: %d bytes/node (max: %d bytes/node)",
			avgMemoryPerNode, maxMemoryPerNode)
	}

	t.Logf("  Memory efficiency: %d bytes per node", avgMemoryPerNode)
}

// TestAddNodes_GoroutineLeakDetection focused goroutine leak test
func TestAddNodes_GoroutineLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), 2*testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Baseline goroutine measurement
	time.Sleep(500 * time.Millisecond) // Allow stabilization
	baselineGoroutines := captureGoroutineState()

	t.Logf("Baseline goroutines: %d total, %d application",
		len(baselineGoroutines), countApplicationGoroutines(baselineGoroutines))

	// Run operations that might create goroutines
	iterations := 30
	nodesPerIteration := 15

	for i := 0; i < iterations; i++ {
		// Create nodes with timeouts to potentially trigger goroutine creation
		testNodes := CreateTestNodes(nodesPerIteration)

		for j, node := range testNodes {
			node.ID = fmt.Sprintf("goroutine_test_iter_%d_node_%d", i, j)
		}

		opts := &types.AddNodesOptions{
			GraphName: testGraphName,
			Nodes:     testNodes,
			BatchSize: 7,
			Timeout:   10, // Short timeout to potentially create timeout goroutines
		}

		_, err := store.AddNodes(ctx, opts)
		if err != nil {
			t.Fatalf("AddNodes failed at iteration %d: %v", i, err)
		}

		// Periodically check goroutine count
		if i%10 == 9 {
			time.Sleep(200 * time.Millisecond) // Allow operations to complete
			currentGoroutines := captureGoroutineState()
			appGoroutines := countApplicationGoroutines(currentGoroutines)

			t.Logf("Iteration %d: %d total goroutines, %d application",
				i+1, len(currentGoroutines), appGoroutines)
		}
	}

	// Allow all operations to complete
	time.Sleep(2 * time.Second)

	// Final goroutine measurement
	finalGoroutines := captureGoroutineState()
	leaked, cleaned := analyzeGoroutineChanges(baselineGoroutines, finalGoroutines)

	t.Logf("Goroutine leak analysis:")
	t.Logf("  Baseline: %d goroutines", len(baselineGoroutines))
	t.Logf("  Final: %d goroutines", len(finalGoroutines))
	t.Logf("  Leaked: %d goroutines", len(leaked))
	t.Logf("  Cleaned: %d goroutines", len(cleaned))

	// Analyze leaked goroutines
	applicationLeaks := 0
	systemLeaks := 0

	for _, g := range leaked {
		if g.IsSystem {
			systemLeaks++
			t.Logf("System goroutine (acceptable): ID=%d, State=%s, Function=%s",
				g.ID, g.State, g.Function)
		} else {
			applicationLeaks++
			t.Errorf("Application goroutine leak: ID=%d, State=%s, Function=%s",
				g.ID, g.State, g.Function)
			t.Errorf("  Stack: %s", g.Stack)
		}
	}

	if applicationLeaks > 0 {
		t.Errorf("Detected %d application goroutine leaks", applicationLeaks)
	}

	// Check for excessive total goroutine growth
	totalGrowth := len(finalGoroutines) - len(baselineGoroutines)
	maxAcceptableGrowth := 5 // Allow for some system goroutines

	if totalGrowth > maxAcceptableGrowth {
		t.Errorf("Excessive goroutine growth: %d new goroutines (max acceptable: %d)",
			totalGrowth, maxAcceptableGrowth)
	}

	t.Logf("Goroutine efficiency: %d total growth, %d application leaks", totalGrowth, applicationLeaks)
}

// countApplicationGoroutines counts non-system goroutines
func countApplicationGoroutines(goroutines []GoroutineInfo) int {
	count := 0
	for _, g := range goroutines {
		if !g.IsSystem {
			count++
		}
	}
	return count
}

// Helper functions using existing utilities

// setupTestStore creates a test store for node testing
func setupTestStore(t *testing.T) *Store {
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

// setupEnterpriseTestStore creates an enterprise test store for node testing
func setupEnterpriseTestStore(t *testing.T) *Store {
	config := getEnterpriseTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
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
			"use_separate_database": true,
		},
	}

	connectWithRetry(ctx, t, store, storeConfig)
	return store
}

// cleanupTestStore cleans up test store resources for node testing
func cleanupTestStore(t *testing.T, store *Store) {
	if store != nil && store.IsConnected() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Try to clean up any test graphs
		graphs, err := store.ListGraphs(ctx)
		if err == nil {
			for _, graph := range graphs {
				if strings.Contains(graph, "test_") {
					_ = store.DropGraph(ctx, graph)
				}
			}
		}

		err = store.Disconnect(ctx)
		if err != nil {
			t.Logf("Warning: Failed to disconnect test store: %v", err)
		}
	}
}

// hasEnterpriseConnection checks if enterprise connection is available
func hasEnterpriseConnection() bool {
	return os.Getenv("NEO4J_TEST_ENTERPRISE_URL") != ""
}

// ===== GetNodes Tests =====

// TestGetNodes_Basic tests basic GetNodes functionality
func TestGetNodes_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Add some test nodes first
	testNodes := CreateTestNodes(5)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	nodeIDs, err := store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}
	if len(nodeIDs) != 5 {
		t.Fatalf("Expected 5 node IDs, got %d", len(nodeIDs))
	}

	// Test get all nodes
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		IncludeProperties: true,
		IncludeMetadata:   true,
		Limit:             10,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes failed: %v", err)
	}
	if len(nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(nodes))
	}

	// Verify node content
	for i, node := range nodes {
		if node.ID == "" {
			t.Errorf("Node %d has empty ID", i)
		}
		if len(node.Labels) == 0 {
			t.Errorf("Node %d has no labels", i)
		}
		if node.Properties == nil {
			t.Errorf("Node %d has nil properties", i)
		}
	}
}

// TestGetNodes_ByIDs tests retrieving nodes by specific IDs
func TestGetNodes_ByIDs(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	testNodes := CreateTestNodes(10)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	nodeIDs, err := store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Get specific nodes by IDs
	targetIDs := []string{nodeIDs[0], nodeIDs[2], nodeIDs[4]}
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		IDs:               targetIDs,
		IncludeProperties: true,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes by IDs failed: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(nodes))
	}

	// Verify retrieved nodes match requested IDs
	retrievedIDs := make(map[string]bool)
	for _, node := range nodes {
		retrievedIDs[node.ID] = true
	}
	for _, targetID := range targetIDs {
		if !retrievedIDs[targetID] {
			t.Errorf("Expected node ID %s not found in results", targetID)
		}
	}
}

// TestGetNodes_ByLabels tests retrieving nodes by labels
func TestGetNodes_ByLabels(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Create nodes with different labels
	nodesA := []*types.GraphNode{
		{ID: "a1", Labels: []string{"TypeA", "Entity"}, Properties: map[string]interface{}{"type": "A"}},
		{ID: "a2", Labels: []string{"TypeA", "Entity"}, Properties: map[string]interface{}{"type": "A"}},
	}
	nodesB := []*types.GraphNode{
		{ID: "b1", Labels: []string{"TypeB", "Entity"}, Properties: map[string]interface{}{"type": "B"}},
		{ID: "b2", Labels: []string{"TypeB", "Entity"}, Properties: map[string]interface{}{"type": "B"}},
	}

	// Add nodes
	allNodes := append(nodesA, nodesB...)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     allNodes,
	}
	_, err = store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Get nodes with TypeA label
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		Labels:            []string{"TypeA"},
		IncludeProperties: true,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes by labels failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("Expected 2 TypeA nodes, got %d", len(nodes))
	}

	// Verify all retrieved nodes have TypeA label
	for _, node := range nodes {
		hasTypeA := false
		for _, label := range node.Labels {
			if label == "TypeA" {
				hasTypeA = true
				break
			}
		}
		if !hasTypeA {
			t.Errorf("Node %s does not have TypeA label", node.ID)
		}
	}
}

// TestGetNodes_ByFilter tests retrieving nodes by property filters
func TestGetNodes_ByFilter(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Create nodes with different properties
	nodes := []*types.GraphNode{
		{ID: "user1", Labels: []string{"User"}, Properties: map[string]interface{}{"status": "active", "age": 25}},
		{ID: "user2", Labels: []string{"User"}, Properties: map[string]interface{}{"status": "active", "age": 30}},
		{ID: "user3", Labels: []string{"User"}, Properties: map[string]interface{}{"status": "inactive", "age": 35}},
	}

	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     nodes,
	}
	_, err = store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Get active users
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		Filter:            map[string]interface{}{"status": "active"},
		IncludeProperties: true,
	}
	activeUsers, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes by filter failed: %v", err)
	}
	if len(activeUsers) != 2 {
		t.Errorf("Expected 2 active users, got %d", len(activeUsers))
	}

	// Verify all retrieved nodes have active status
	for _, user := range activeUsers {
		if status, ok := user.Properties["status"]; !ok || status != "active" {
			t.Errorf("User %s does not have active status", user.ID)
		}
	}
}

// TestGetNodes_EmptyGraph tests retrieving from empty graph
func TestGetNodes_EmptyGraph(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create empty test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Try to get nodes from empty graph
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		IncludeProperties: true,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes from empty graph failed: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes from empty graph, got %d", len(nodes))
	}
}

// TestGetNodes_NonExistentGraph tests retrieving from non-existent graph
func TestGetNodes_NonExistentGraph(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Try to get nodes from non-existent graph
	getOpts := &types.GetNodesOptions{
		GraphName:         "non_existent_graph",
		IncludeProperties: true,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes from non-existent graph failed: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes from non-existent graph, got %d", len(nodes))
	}
}

// TestGetNodes_ErrorHandling tests error scenarios
func TestGetNodes_ErrorHandling(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test with nil options
	_, err := store.GetNodes(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// Test with empty graph name
	opts := &types.GetNodesOptions{
		GraphName: "",
	}
	_, err = store.GetNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error for empty graph name")
	}

	// Test with invalid graph name
	opts.GraphName = "invalid-graph-name!"
	_, err = store.GetNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error for invalid graph name")
	}
}

// TestGetNodes_LabelBasedMode tests label-based storage mode
func TestGetNodes_LabelBasedMode(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	// Force label-based mode
	store.SetUseSeparateDatabase(false)
	store.SetIsEnterpriseEdition(false)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	testNodes := CreateTestNodes(3)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	_, err = store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Get nodes in label-based mode
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		IncludeProperties: true,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes in label-based mode failed: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes in label-based mode, got %d", len(nodes))
	}
}

// TestGetNodes_SeparateDatabaseMode tests separate database mode
func TestGetNodes_SeparateDatabaseMode(t *testing.T) {
	if !hasEnterpriseConnection() {
		t.Skip("Skipping enterprise-only test: NEO4J_TEST_ENTERPRISE_URL not set")
	}

	store := setupEnterpriseTestStore(t)
	defer cleanupTestStore(t, store)

	// Force separate database mode
	store.SetUseSeparateDatabase(true)
	store.SetIsEnterpriseEdition(true)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Use enterprise-compatible graph name
	enterpriseGraphName := "testgetgraph"

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, enterpriseGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, enterpriseGraphName)
	}()

	testNodes := CreateTestNodes(3)
	addOpts := &types.AddNodesOptions{
		GraphName: enterpriseGraphName,
		Nodes:     testNodes,
	}
	_, err = store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Get nodes in separate database mode
	getOpts := &types.GetNodesOptions{
		GraphName:         enterpriseGraphName,
		IncludeProperties: true,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes in separate database mode failed: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes in separate database mode, got %d", len(nodes))
	}
}

// TestGetNodes_ConcurrentStress tests concurrent GetNodes operations
func TestGetNodes_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), 2*testTimeout)
	defer cancel()

	// Capture initial state for leak detection
	beforeGoroutines := captureGoroutineState()
	beforeMemory := captureMemoryStats()

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Add test data
	testNodes := CreateTestNodes(50)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	_, err = store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Stress test configuration
	config := LightStressConfig()

	// Run stress test
	operation := func(ctx context.Context) error {
		getOpts := &types.GetNodesOptions{
			GraphName:         testGraphName,
			IncludeProperties: true,
			Limit:             20,
		}
		_, err := store.GetNodes(ctx, getOpts)
		return err
	}

	result := runStressTest(config, operation)

	// Check results
	if result.SuccessRate < config.MinSuccessRate {
		t.Errorf("GetNodes stress test success rate %.2f%% is below minimum %.2f%%",
			result.SuccessRate, config.MinSuccessRate)
	}

	t.Logf("GetNodes stress test completed: %d operations, %.2f%% success rate, %d errors, duration: %v",
		result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)

	// Allow time for cleanup
	time.Sleep(time.Second)

	// Check for leaks
	afterGoroutines := captureGoroutineState()
	leaked, _ := analyzeGoroutineChanges(beforeGoroutines, afterGoroutines)

	if len(leaked) > 0 {
		for _, g := range leaked {
			if !g.IsSystem {
				t.Errorf("GetNodes goroutine leak detected: ID=%d, State=%s, Function=%s",
					g.ID, g.State, g.Function)
			}
		}
	}

	afterMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(beforeMemory, afterMemory)

	// Allow for some memory growth, but not excessive
	maxAllowedGrowth := int64(20 * 1024 * 1024) // 20MB
	if memGrowth.HeapAllocGrowth > maxAllowedGrowth {
		t.Errorf("GetNodes excessive memory growth detected: %d bytes heap allocation growth",
			memGrowth.HeapAllocGrowth)
	}

	t.Logf("GetNodes memory growth: Heap=%d bytes, Total=%d bytes, GC cycles=%d",
		memGrowth.HeapAllocGrowth, memGrowth.AllocGrowth, memGrowth.NumGCDiff)
}

// ===== DeleteNodes Tests =====

// TestDeleteNodes_Basic tests basic DeleteNodes functionality
func TestDeleteNodes_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	testNodes := CreateTestNodes(5)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	nodeIDs, err := store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Verify nodes exist
	getOpts := &types.GetNodesOptions{
		GraphName: testGraphName,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}
	if len(nodes) != 5 {
		t.Fatalf("Expected 5 nodes before deletion, got %d", len(nodes))
	}

	// Delete specific nodes by IDs
	deleteIDs := []string{nodeIDs[0], nodeIDs[2]}
	delOpts := &types.DeleteNodesOptions{
		GraphName: testGraphName,
		IDs:       deleteIDs,
	}
	err = store.DeleteNodes(ctx, delOpts)
	if err != nil {
		t.Fatalf("DeleteNodes failed: %v", err)
	}

	// Verify nodes were deleted
	nodes, err = store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes after deletion: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes after deletion, got %d", len(nodes))
	}

	// Verify specific nodes were deleted
	remainingIDs := make(map[string]bool)
	for _, node := range nodes {
		remainingIDs[node.ID] = true
	}
	for _, deletedID := range deleteIDs {
		if remainingIDs[deletedID] {
			t.Errorf("Node %s should have been deleted but still exists", deletedID)
		}
	}
}

// TestDeleteNodes_ByFilter tests deleting nodes by filter
func TestDeleteNodes_ByFilter(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Create nodes with different properties
	nodes := []*types.GraphNode{
		{ID: "active1", Labels: []string{"User"}, Properties: map[string]interface{}{"status": "active"}},
		{ID: "active2", Labels: []string{"User"}, Properties: map[string]interface{}{"status": "active"}},
		{ID: "inactive1", Labels: []string{"User"}, Properties: map[string]interface{}{"status": "inactive"}},
		{ID: "inactive2", Labels: []string{"User"}, Properties: map[string]interface{}{"status": "inactive"}},
	}

	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     nodes,
	}
	_, err = store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Delete inactive users by filter
	delOpts := &types.DeleteNodesOptions{
		GraphName: testGraphName,
		Filter:    map[string]interface{}{"status": "inactive"},
	}
	err = store.DeleteNodes(ctx, delOpts)
	if err != nil {
		t.Fatalf("DeleteNodes by filter failed: %v", err)
	}

	// Verify only active users remain
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		IncludeProperties: true,
	}
	remainingNodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("Failed to get remaining nodes: %v", err)
	}
	if len(remainingNodes) != 2 {
		t.Errorf("Expected 2 remaining nodes, got %d", len(remainingNodes))
	}

	// Verify all remaining nodes are active
	for _, node := range remainingNodes {
		if status, ok := node.Properties["status"]; !ok || status != "active" {
			t.Errorf("Remaining node %s should be active but has status %v", node.ID, status)
		}
	}
}

// TestDeleteNodes_DryRun tests dry run functionality
func TestDeleteNodes_DryRun(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph and add nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	testNodes := CreateTestNodes(5)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	nodeIDs, err := store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Perform dry run
	delOpts := &types.DeleteNodesOptions{
		GraphName: testGraphName,
		IDs:       []string{nodeIDs[0], nodeIDs[1]},
		DryRun:    true,
	}
	err = store.DeleteNodes(ctx, delOpts)
	if err == nil {
		t.Error("Expected dry run to return an informational error")
	}
	if !strings.Contains(err.Error(), "dry run") {
		t.Errorf("Expected dry run error message, got: %v", err)
	}

	// Verify no nodes were actually deleted
	getOpts := &types.GetNodesOptions{
		GraphName: testGraphName,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes after dry run: %v", err)
	}
	if len(nodes) != 5 {
		t.Errorf("Expected 5 nodes after dry run (no actual deletion), got %d", len(nodes))
	}
}

// TestDeleteNodes_ErrorHandling tests error scenarios
func TestDeleteNodes_ErrorHandling(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test with nil options
	err := store.DeleteNodes(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// Test with empty graph name
	opts := &types.DeleteNodesOptions{
		GraphName: "",
	}
	err = store.DeleteNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error for empty graph name")
	}

	// Test with invalid graph name
	opts.GraphName = "invalid-graph-name!"
	err = store.DeleteNodes(ctx, opts)
	if err == nil {
		t.Error("Expected error for invalid graph name")
	}

	// Test without IDs or Filter (should prevent accidental deletion of all nodes)
	// Note: This validation now happens before graph existence check
	emptyOpts := &types.DeleteNodesOptions{
		GraphName: testGraphName,
		IDs:       nil,
		Filter:    nil,
	}
	err = store.DeleteNodes(ctx, emptyOpts)
	if err == nil {
		t.Error("Expected error when neither IDs nor Filter is specified")
	}
}

// TestDeleteNodes_LabelBasedMode tests deletion in label-based mode
func TestDeleteNodes_LabelBasedMode(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	// Force label-based mode
	store.SetUseSeparateDatabase(false)
	store.SetIsEnterpriseEdition(false)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph and add/delete nodes
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	testNodes := CreateTestNodes(5)
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     testNodes,
	}
	nodeIDs, err := store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Delete nodes in label-based mode
	delOpts := &types.DeleteNodesOptions{
		GraphName: testGraphName,
		IDs:       []string{nodeIDs[0], nodeIDs[1]},
	}
	err = store.DeleteNodes(ctx, delOpts)
	if err != nil {
		t.Fatalf("DeleteNodes in label-based mode failed: %v", err)
	}

	// Verify deletion
	getOpts := &types.GetNodesOptions{
		GraphName: testGraphName,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes after deletion: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes after deletion in label-based mode, got %d", len(nodes))
	}
}

// TestDeleteNodes_SeparateDatabaseMode tests deletion in separate database mode
func TestDeleteNodes_SeparateDatabaseMode(t *testing.T) {
	if !hasEnterpriseConnection() {
		t.Skip("Skipping enterprise-only test: NEO4J_TEST_ENTERPRISE_URL not set")
	}

	store := setupEnterpriseTestStore(t)
	defer cleanupTestStore(t, store)

	// Force separate database mode
	store.SetUseSeparateDatabase(true)
	store.SetIsEnterpriseEdition(true)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Use enterprise-compatible graph name
	enterpriseGraphName := "testdelgraph"

	// Create test graph and add/delete nodes
	err := store.CreateGraph(ctx, enterpriseGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, enterpriseGraphName)
	}()

	testNodes := CreateTestNodes(5)
	addOpts := &types.AddNodesOptions{
		GraphName: enterpriseGraphName,
		Nodes:     testNodes,
	}
	nodeIDs, err := store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test nodes: %v", err)
	}

	// Delete nodes in separate database mode
	delOpts := &types.DeleteNodesOptions{
		GraphName: enterpriseGraphName,
		IDs:       []string{nodeIDs[0], nodeIDs[1]},
	}
	err = store.DeleteNodes(ctx, delOpts)
	if err != nil {
		t.Fatalf("DeleteNodes in separate database mode failed: %v", err)
	}

	// Verify deletion
	getOpts := &types.GetNodesOptions{
		GraphName: enterpriseGraphName,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("Failed to get nodes after deletion: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes after deletion in separate database mode, got %d", len(nodes))
	}
}

// TestDeleteNodes_ConcurrentStress tests concurrent DeleteNodes operations
func TestDeleteNodes_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), 2*testTimeout)
	defer cancel()

	// Capture initial state for leak detection
	beforeGoroutines := captureGoroutineState()
	beforeMemory := captureMemoryStats()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Stress test configuration
	config := LightStressConfig()
	config.NumWorkers = 5 // Reduce workers for deletion to avoid conflicts

	// Run stress test
	operation := func(ctx context.Context) error {
		// Create unique nodes for each operation to avoid conflicts
		testNodes := make([]*types.GraphNode, 3)
		for i := 0; i < 3; i++ {
			testNodes[i] = &types.GraphNode{
				ID:     fmt.Sprintf("stress_del_%d_%d", time.Now().UnixNano(), i),
				Labels: []string{"StressTest"},
				Properties: map[string]interface{}{
					"created": time.Now().Unix(),
				},
				CreatedAt: time.Now(),
				Version:   1,
			}
		}

		// Add nodes
		addOpts := &types.AddNodesOptions{
			GraphName: testGraphName,
			Nodes:     testNodes,
		}
		nodeIDs, err := store.AddNodes(ctx, addOpts)
		if err != nil {
			return err
		}

		// Delete some nodes
		if len(nodeIDs) > 1 {
			delOpts := &types.DeleteNodesOptions{
				GraphName: testGraphName,
				IDs:       []string{nodeIDs[0]},
			}
			err = store.DeleteNodes(ctx, delOpts)
			if err != nil {
				return err
			}
		}

		return nil
	}

	result := runStressTest(config, operation)

	// Check results
	if result.SuccessRate < config.MinSuccessRate {
		t.Errorf("DeleteNodes stress test success rate %.2f%% is below minimum %.2f%%",
			result.SuccessRate, config.MinSuccessRate)
	}

	t.Logf("DeleteNodes stress test completed: %d operations, %.2f%% success rate, %d errors, duration: %v",
		result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)

	// Allow time for cleanup
	time.Sleep(2 * time.Second)

	// Check for leaks
	afterGoroutines := captureGoroutineState()
	leaked, _ := analyzeGoroutineChanges(beforeGoroutines, afterGoroutines)

	if len(leaked) > 0 {
		for _, g := range leaked {
			if !g.IsSystem {
				t.Errorf("DeleteNodes goroutine leak detected: ID=%d, State=%s, Function=%s",
					g.ID, g.State, g.Function)
			}
		}
	}

	afterMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(beforeMemory, afterMemory)

	// Allow for some memory growth, but not excessive
	maxAllowedGrowth := int64(30 * 1024 * 1024) // 30MB
	if memGrowth.HeapAllocGrowth > maxAllowedGrowth {
		t.Errorf("DeleteNodes excessive memory growth detected: %d bytes heap allocation growth",
			memGrowth.HeapAllocGrowth)
	}

	t.Logf("DeleteNodes memory growth: Heap=%d bytes, Total=%d bytes, GC cycles=%d",
		memGrowth.HeapAllocGrowth, memGrowth.AllocGrowth, memGrowth.NumGCDiff)
}

// TestGetDeleteNodes_RealData tests GetNodes and DeleteNodes with real test data
func TestGetDeleteNodes_RealData(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testGraphName)
	}()

	// Load Chinese test data
	zhNodes, _, err := LoadTestDataset("zh")
	if err != nil {
		t.Skipf("Skipping real data test: %v", err)
	}

	if len(zhNodes) == 0 {
		t.Skip("No Chinese test data available")
	}

	// Test with subset of real data
	maxNodes := 15
	if len(zhNodes) > maxNodes {
		zhNodes = zhNodes[:maxNodes]
	}

	// Add real nodes
	addOpts := &types.AddNodesOptions{
		GraphName: testGraphName,
		Nodes:     zhNodes,
	}
	nodeIDs, err := store.AddNodes(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add real test nodes: %v", err)
	}

	if len(nodeIDs) != len(zhNodes) {
		t.Fatalf("Expected %d node IDs, got %d", len(zhNodes), len(nodeIDs))
	}

	// Test GetNodes with real data
	getOpts := &types.GetNodesOptions{
		GraphName:         testGraphName,
		IncludeProperties: true,
		IncludeMetadata:   true,
	}
	nodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes with real data failed: %v", err)
	}
	if len(nodes) != len(zhNodes) {
		t.Errorf("Expected %d nodes, got %d", len(zhNodes), len(nodes))
	}

	// Test filtering by entity type if available
	if len(nodes) > 0 && nodes[0].EntityType != "" {
		filterOpts := &types.GetNodesOptions{
			GraphName: testGraphName,
			Filter:    map[string]interface{}{"entity_type": nodes[0].EntityType},
		}
		filteredNodes, err := store.GetNodes(ctx, filterOpts)
		if err != nil {
			t.Fatalf("GetNodes with entity type filter failed: %v", err)
		}
		if len(filteredNodes) == 0 {
			t.Error("Expected at least one node with entity type filter")
		}
	}

	// Test DeleteNodes with half of the real data
	deleteCount := len(nodeIDs) / 2
	deleteIDs := nodeIDs[:deleteCount]

	delOpts := &types.DeleteNodesOptions{
		GraphName: testGraphName,
		IDs:       deleteIDs,
	}
	err = store.DeleteNodes(ctx, delOpts)
	if err != nil {
		t.Fatalf("DeleteNodes with real data failed: %v", err)
	}

	// Verify deletion
	remainingNodes, err := store.GetNodes(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetNodes after deletion failed: %v", err)
	}

	expectedRemaining := len(nodeIDs) - deleteCount
	if len(remainingNodes) != expectedRemaining {
		t.Errorf("Expected %d remaining nodes, got %d", expectedRemaining, len(remainingNodes))
	}

	// Verify that deleted nodes are not in remaining nodes
	remainingIDs := make(map[string]bool)
	for _, node := range remainingNodes {
		remainingIDs[node.ID] = true
	}
	for _, deletedID := range deleteIDs {
		if remainingIDs[deletedID] {
			t.Errorf("Deleted node %s should not exist in remaining nodes", deletedID)
		}
	}
}

// TestGetDeleteNodes_Disconnected tests behavior when store is disconnected
func TestGetDeleteNodes_Disconnected(t *testing.T) {
	store := NewStore()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test GetNodes when disconnected
	getOpts := &types.GetNodesOptions{
		GraphName: testGraphName,
	}
	_, err := store.GetNodes(ctx, getOpts)
	if err == nil {
		t.Error("Expected error when GetNodes called on disconnected store")
	}

	// Test DeleteNodes when disconnected
	delOpts := &types.DeleteNodesOptions{
		GraphName: testGraphName,
		IDs:       []string{"test_id"},
	}
	err = store.DeleteNodes(ctx, delOpts)
	if err == nil {
		t.Error("Expected error when DeleteNodes called on disconnected store")
	}
}
