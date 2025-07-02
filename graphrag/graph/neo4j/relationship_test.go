package neo4j

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

const (
	testRelGraphName = "test_relationships_graph"
	testRelTimeout   = 30 * time.Second
)

// ===== AddRelationships Tests =====

// TestAddRelationships_Basic tests basic AddRelationships functionality
func TestAddRelationships_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Create test graph and add some nodes first
	err := store.CreateGraph(ctx, testRelGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testRelGraphName)
	}()

	// Test with empty relationships list
	opts := &types.AddRelationshipsOptions{
		GraphName:     testRelGraphName,
		Relationships: []*types.GraphRelationship{},
	}
	relIDs, err := store.AddRelationships(ctx, opts)
	if err != nil {
		t.Fatalf("AddRelationships with empty list failed: %v", err)
	}
	if len(relIDs) != 0 {
		t.Errorf("Expected 0 relationship IDs, got %d", len(relIDs))
	}

	// Test with single relationship
	testRels := CreateTestRelationships(1)
	opts = &types.AddRelationshipsOptions{
		GraphName:     testRelGraphName,
		Relationships: testRels,
		CreateNodes:   true,
	}
	relIDs, err = store.AddRelationships(ctx, opts)
	if err != nil {
		t.Fatalf("AddRelationships with single relationship failed: %v", err)
	}
	if len(relIDs) != 1 {
		t.Errorf("Expected 1 relationship ID, got %d", len(relIDs))
	}
	if relIDs[0] != testRels[0].ID {
		t.Errorf("Expected relationship ID %s, got %s", testRels[0].ID, relIDs[0])
	}
}

// TestGetRelationships_Basic tests basic GetRelationships functionality
func TestGetRelationships_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Create test graph
	err := store.CreateGraph(ctx, testRelGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testRelGraphName)
	}()

	// Add test relationships first
	testRels := CreateTestRelationships(5)
	addOpts := &types.AddRelationshipsOptions{
		GraphName:     testRelGraphName,
		Relationships: testRels,
		CreateNodes:   true,
	}
	relIDs, err := store.AddRelationships(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test relationships: %v", err)
	}
	if len(relIDs) != 5 {
		t.Fatalf("Expected 5 relationship IDs, got %d", len(relIDs))
	}

	// Test get all relationships
	getOpts := &types.GetRelationshipsOptions{
		GraphName:         testRelGraphName,
		IncludeProperties: true,
		IncludeMetadata:   true,
		Limit:             10,
	}
	rels, err := store.GetRelationships(ctx, getOpts)
	if err != nil {
		t.Fatalf("GetRelationships failed: %v", err)
	}
	if len(rels) != 5 {
		t.Errorf("Expected 5 relationships, got %d", len(rels))
	}

	// Verify relationship content
	for i, rel := range rels {
		if rel.ID == "" {
			t.Errorf("Relationship %d has empty ID", i)
		}
		if rel.Type == "" {
			t.Errorf("Relationship %d has empty type", i)
		}
		if rel.StartNode == "" || rel.EndNode == "" {
			t.Errorf("Relationship %d has empty start or end node", i)
		}
		if rel.Properties == nil {
			t.Errorf("Relationship %d has nil properties", i)
		}
	}
}

// TestDeleteRelationships_Basic tests basic DeleteRelationships functionality
func TestDeleteRelationships_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Create test graph and add relationships
	err := store.CreateGraph(ctx, testRelGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testRelGraphName)
	}()

	testRels := CreateTestRelationships(5)
	addOpts := &types.AddRelationshipsOptions{
		GraphName:     testRelGraphName,
		Relationships: testRels,
		CreateNodes:   true,
	}
	relIDs, err := store.AddRelationships(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test relationships: %v", err)
	}

	// Delete specific relationships by IDs
	deleteOpts := &types.DeleteRelationshipsOptions{
		GraphName: testRelGraphName,
		IDs:       []string{relIDs[0], relIDs[2]},
	}
	err = store.DeleteRelationships(ctx, deleteOpts)
	if err != nil {
		t.Fatalf("DeleteRelationships failed: %v", err)
	}

	// Verify relationships were deleted
	getOpts := &types.GetRelationshipsOptions{
		GraphName:         testRelGraphName,
		IncludeProperties: true,
	}
	remainingRels, err := store.GetRelationships(ctx, getOpts)
	if err != nil {
		t.Fatalf("Failed to get remaining relationships: %v", err)
	}
	if len(remainingRels) != 3 {
		t.Errorf("Expected 3 remaining relationships, got %d", len(remainingRels))
	}

	// Verify deleted relationships are not in results
	for _, rel := range remainingRels {
		if rel.ID == relIDs[0] || rel.ID == relIDs[2] {
			t.Errorf("Deleted relationship %s still exists", rel.ID)
		}
	}
}

// TestAddRelationships_LabelBasedMode tests label-based storage mode
func TestAddRelationships_LabelBasedMode(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	// Force label-based mode
	store.SetUseSeparateDatabase(false)
	store.SetIsEnterpriseEdition(false)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Create test graph and add relationships
	err := store.CreateGraph(ctx, testRelGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testRelGraphName)
	}()

	testRels := CreateTestRelationships(3)
	opts := &types.AddRelationshipsOptions{
		GraphName:     testRelGraphName,
		Relationships: testRels,
		CreateNodes:   true,
	}

	relIDs, err := store.AddRelationships(ctx, opts)
	if err != nil {
		t.Fatalf("AddRelationships in label-based mode failed: %v", err)
	}
	if len(relIDs) != 3 {
		t.Errorf("Expected 3 relationship IDs, got %d", len(relIDs))
	}
}

// TestAddRelationships_SeparateDatabaseMode tests separate database mode
func TestAddRelationships_SeparateDatabaseMode(t *testing.T) {
	if !hasEnterpriseConnection() {
		t.Skip("Skipping enterprise-only test: NEO4J_TEST_ENTERPRISE_URL not set")
	}

	store := setupEnterpriseTestStore(t)
	defer cleanupTestStore(t, store)

	// Force separate database mode
	store.SetUseSeparateDatabase(true)
	store.SetIsEnterpriseEdition(true)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Use enterprise-compatible graph name
	enterpriseGraphName := "testrelsgraph"

	// Create test graph and add relationships
	err := store.CreateGraph(ctx, enterpriseGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, enterpriseGraphName)
	}()

	testRels := CreateTestRelationships(3)
	opts := &types.AddRelationshipsOptions{
		GraphName:     enterpriseGraphName,
		Relationships: testRels,
		CreateNodes:   true,
	}

	relIDs, err := store.AddRelationships(ctx, opts)
	if err != nil {
		t.Fatalf("AddRelationships in separate database mode failed: %v", err)
	}
	if len(relIDs) != 3 {
		t.Errorf("Expected 3 relationship IDs, got %d", len(relIDs))
	}
}

// TestAddRelationships_ConcurrentStress tests concurrent AddRelationships operations
func TestAddRelationships_ConcurrentStress(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Memory and goroutine tracking
	initialGoroutines := runtime.NumGoroutine()
	defer func() {
		// Check for goroutine leaks
		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines+5 { // Allow some buffer
			t.Errorf("Potential goroutine leak: started with %d, ended with %d", initialGoroutines, finalGoroutines)
		}
	}()

	// Create test graph
	err := store.CreateGraph(ctx, testRelGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testRelGraphName)
	}()

	// Memory tracking
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Stress test parameters
	numWorkers := 10
	operationsPerWorker := 5
	totalOperations := numWorkers * operationsPerWorker

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	start := time.Now()

	// Create concurrent workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// Create unique relationships for each operation
				rels := []*types.GraphRelationship{
					{
						ID:        fmt.Sprintf("stress_rel_%d_%d", workerID, j),
						Type:      "STRESS_TEST",
						StartNode: fmt.Sprintf("stress_node_%d_%d_start", workerID, j),
						EndNode:   fmt.Sprintf("stress_node_%d_%d_end", workerID, j),
						Properties: map[string]interface{}{
							"worker_id":  workerID,
							"operation":  j,
							"created_at": time.Now().Unix(),
							"stress":     true,
						},
						Description: fmt.Sprintf("Stress test relationship %d-%d", workerID, j),
						Confidence:  rand.Float64(),
						Weight:      rand.Float64(),
						CreatedAt:   time.Now(),
						Version:     1,
					},
				}

				opts := &types.AddRelationshipsOptions{
					GraphName:     testRelGraphName,
					Relationships: rels,
					CreateNodes:   true,
					BatchSize:     1,
				}

				_, err := store.AddRelationships(ctx, opts)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Logf("Worker %d operation %d failed: %v", workerID, j, err)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Memory tracking after operations
	runtime.GC()
	runtime.ReadMemStats(&m2)
	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)

	// Calculate success rate
	successRate := float64(successCount) / float64(totalOperations) * 100

	t.Logf("AddRelationships Stress Test Results:")
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Success rate: %.2f%%", successRate)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Operations/sec: %.2f", float64(totalOperations)/elapsed.Seconds())
	t.Logf("  Heap growth: %d bytes", heapGrowth)
	t.Logf("  Memory per operation: %d bytes", heapGrowth/int64(totalOperations))

	// Verify high success rate
	if successRate < 95.0 {
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}

	// Verify reasonable memory usage (less than 5KB per operation for relationships)
	if heapGrowth > int64(totalOperations)*5120 {
		t.Errorf("Memory usage too high: %d bytes (expected < %d bytes)", heapGrowth, int64(totalOperations)*5120)
	}

	// Verify reasonable performance (operations should complete in reasonable time)
	if elapsed > 30*time.Second {
		t.Errorf("Operations took too long: %v (expected < 30s)", elapsed)
	}
}

// TestGetRelationships_ConcurrentStress tests concurrent GetRelationships operations
func TestGetRelationships_ConcurrentStress(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Create test graph and add relationships
	err := store.CreateGraph(ctx, testRelGraphName)
	if err != nil {
		t.Fatalf("Failed to create test graph: %v", err)
	}
	defer func() {
		_ = store.DropGraph(ctx, testRelGraphName)
	}()

	// Pre-populate with test relationships
	testRels := CreateTestRelationships(50)
	addOpts := &types.AddRelationshipsOptions{
		GraphName:     testRelGraphName,
		Relationships: testRels,
		CreateNodes:   true,
	}
	_, err = store.AddRelationships(ctx, addOpts)
	if err != nil {
		t.Fatalf("Failed to add test relationships: %v", err)
	}

	// Memory tracking
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Stress test parameters
	numWorkers := 10
	operationsPerWorker := 5
	totalOperations := numWorkers * operationsPerWorker

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	start := time.Now()

	// Create concurrent workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				opts := &types.GetRelationshipsOptions{
					GraphName:         testRelGraphName,
					IncludeProperties: true,
					IncludeMetadata:   true,
					Limit:             25,
				}

				// Vary the query parameters
				switch j % 3 {
				case 0:
					// Get all relationships
				case 1:
					// Get by type
					opts.Types = []string{"RELATED_TO"}
				case 2:
					// Get by filter
					opts.Filter = map[string]interface{}{"source": "test"}
				}

				_, err := store.GetRelationships(ctx, opts)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Logf("Worker %d operation %d failed: %v", workerID, j, err)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Memory tracking after operations
	runtime.GC()
	runtime.ReadMemStats(&m2)
	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)

	// Calculate success rate
	successRate := float64(successCount) / float64(totalOperations) * 100

	t.Logf("GetRelationships Stress Test Results:")
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Success rate: %.2f%%", successRate)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Operations/sec: %.2f", float64(totalOperations)/elapsed.Seconds())
	t.Logf("  Heap growth: %d bytes", heapGrowth)

	// Verify high success rate
	if successRate < 95.0 {
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}

	// Verify reasonable memory usage
	if heapGrowth > 200*1024 { // 200KB limit for read operations
		t.Errorf("Memory usage too high: %d bytes (expected < 200KB)", heapGrowth)
	}
}

// TestAddGetDeleteRelationships_Disconnected tests behavior when store is disconnected
func TestAddGetDeleteRelationships_Disconnected(t *testing.T) {
	store := NewStore()

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	testRels := CreateTestRelationships(1)

	// Test AddRelationships when disconnected
	addOpts := &types.AddRelationshipsOptions{
		GraphName:     testRelGraphName,
		Relationships: testRels,
	}
	_, err := store.AddRelationships(ctx, addOpts)
	if err == nil {
		t.Error("Expected error when adding relationships to disconnected store")
	}

	// Test GetRelationships when disconnected
	getOpts := &types.GetRelationshipsOptions{
		GraphName: testRelGraphName,
	}
	_, err = store.GetRelationships(ctx, getOpts)
	if err == nil {
		t.Error("Expected error when getting relationships from disconnected store")
	}

	// Test DeleteRelationships when disconnected
	deleteOpts := &types.DeleteRelationshipsOptions{
		GraphName: testRelGraphName,
		IDs:       []string{"test"},
	}
	err = store.DeleteRelationships(ctx, deleteOpts)
	if err == nil {
		t.Error("Expected error when deleting relationships from disconnected store")
	}
}

// TestRelationships_ErrorHandling tests error scenarios
func TestRelationships_ErrorHandling(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(t, store)

	ctx, cancel := context.WithTimeout(context.Background(), testRelTimeout)
	defer cancel()

	// Test AddRelationships with nil options
	_, err := store.AddRelationships(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// Test GetRelationships with nil options
	_, err = store.GetRelationships(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// Test DeleteRelationships with nil options
	err = store.DeleteRelationships(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// Test with empty graph name
	addOpts := &types.AddRelationshipsOptions{
		GraphName:     "",
		Relationships: CreateTestRelationships(1),
	}
	_, err = store.AddRelationships(ctx, addOpts)
	if err == nil {
		t.Error("Expected error for empty graph name")
	}

	// Test with invalid graph name
	addOpts.GraphName = "invalid-graph-name!"
	_, err = store.AddRelationships(ctx, addOpts)
	if err == nil {
		t.Error("Expected error for invalid graph name")
	}
}
