package qdrant

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/rag/driver"
	"github.com/yaoapp/gou/rag/driver/openai"
)

func getTestConfig(t *testing.T) Config {
	host := os.Getenv("QDRANT_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	portStr := os.Getenv("QDRANT_TEST_PORT")
	if portStr == "" {
		portStr = "6334"
	}

	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		t.Fatalf("Invalid QDRANT_TEST_PORT: %v", err)
	}

	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set")
	}

	vectorizer, err := openai.New(openai.Config{
		APIKey: openaiKey,
		Model:  "text-embedding-ada-002",
	})
	if err != nil {
		t.Fatalf("Failed to create vectorizer: %v", err)
	}

	return Config{
		Host:       host,
		Port:       uint32(port),
		Vectorizer: vectorizer,
	}
}

// TestBasicOperations tests basic CRUD operations
func TestBasicOperations(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	// Create engine
	engine, err := NewEngine(config)
	assert.NoError(t, err)
	defer engine.Close()

	// Test index operations
	indexName := "test_index"
	err = engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
	assert.NoError(t, err)

	// Test HasIndex
	exists, err := engine.HasIndex(ctx, indexName)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = engine.HasIndex(ctx, "non_existent_index")
	assert.NoError(t, err)
	assert.False(t, exists)

	// List indexes
	indexes, err := engine.ListIndexes(ctx)
	assert.NoError(t, err)
	assert.Contains(t, indexes, indexName)

	// Test document operations
	doc := &driver.Document{
		DocID:    "test-doc-123",
		Content:  "This is a test document for Qdrant vector search.",
		Metadata: map[string]interface{}{"type": "test", "version": 1.0},
	}

	// Test HasDocument before indexing
	exists, err = engine.HasDocument(ctx, indexName, doc.DocID)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Index document
	err = engine.IndexDoc(ctx, indexName, doc)
	assert.NoError(t, err)

	// Test HasDocument after indexing
	exists, err = engine.HasDocument(ctx, indexName, doc.DocID)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Get document
	retrieved, err := engine.GetDocument(ctx, indexName, doc.DocID)
	assert.NoError(t, err)
	assert.Equal(t, doc.Content, retrieved.Content)
	assert.Equal(t, doc.Metadata["type"], retrieved.Metadata["type"])
	assert.Equal(t, doc.Metadata["version"], retrieved.Metadata["version"])

	// Search
	searchOpts := driver.VectorSearchOptions{
		QueryText: "test document",
		TopK:      5,
	}
	results, err := engine.Search(ctx, indexName, nil, searchOpts)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
	assert.Equal(t, doc.DocID, results[0].DocID)

	// Delete document
	err = engine.DeleteDoc(ctx, indexName, doc.DocID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = engine.GetDocument(ctx, indexName, doc.DocID)
	assert.Error(t, err)

	// Cleanup
	err = engine.DeleteIndex(ctx, indexName)
	assert.NoError(t, err)
}

// TestBatchOperations tests batch indexing and deletion
func TestBatchOperations(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	engine, err := NewEngine(config)
	assert.NoError(t, err)
	defer engine.Close()

	indexName := "test_batch_index"
	err = engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
	assert.NoError(t, err)
	defer engine.DeleteIndex(ctx, indexName)

	// Prepare batch documents
	docs := make([]*driver.Document, 10)
	docIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		docID := fmt.Sprintf("test-doc-%d", i)
		docs[i] = &driver.Document{
			DocID:    docID,
			Content:  fmt.Sprintf("This is test document %d", i),
			Metadata: map[string]interface{}{"index": i},
		}
		docIDs[i] = docID
	}

	// Test batch indexing
	taskID, err := engine.IndexBatch(ctx, indexName, docs)
	if err != nil {
		t.Fatalf("Failed to batch index documents: %v", err)
	}
	assert.NotEmpty(t, taskID)

	// Wait for indexing to complete and verify with retries
	var taskInfo *driver.TaskInfo
	for i := 0; i < 10; i++ {
		taskInfo, err = engine.GetTaskInfo(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get task info: %v", err)
		}
		if taskInfo.Status == driver.StatusComplete {
			break
		}
		time.Sleep(time.Second)
	}
	assert.Equal(t, driver.StatusComplete, taskInfo.Status)

	// Wait a bit more to ensure documents are fully indexed
	time.Sleep(2 * time.Second)

	// Verify documents were indexed
	for _, docID := range docIDs {
		doc, err := engine.GetDocument(ctx, indexName, docID)
		if err != nil {
			t.Fatalf("Failed to get document %s: %v", docID, err)
		}
		assert.NotNil(t, doc)
		assert.Equal(t, docID, doc.DocID)
	}

	// Test batch deletion
	deleteTaskID, err := engine.DeleteBatch(ctx, indexName, docIDs)
	if err != nil {
		t.Fatalf("Failed to batch delete documents: %v", err)
	}
	assert.NotEmpty(t, deleteTaskID)

	// Wait for deletion to complete and verify with retries
	var deleteTaskInfo *driver.TaskInfo
	for i := 0; i < 10; i++ {
		deleteTaskInfo, err = engine.GetTaskInfo(ctx, deleteTaskID)
		if err != nil {
			t.Fatalf("Failed to get delete task info: %v", err)
		}
		if deleteTaskInfo.Status == driver.StatusComplete {
			break
		}
		time.Sleep(time.Second)
	}
	assert.Equal(t, driver.StatusComplete, deleteTaskInfo.Status)

	// Wait a bit more to ensure documents are fully deleted
	time.Sleep(2 * time.Second)

	// Verify documents were deleted
	for _, docID := range docIDs {
		_, err := engine.GetDocument(ctx, indexName, docID)
		assert.Error(t, err, "Document %s should be deleted", docID)
	}
}

// TestTaskManagement tests task management operations
func TestTaskManagement(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	engine, err := NewEngine(config)
	assert.NoError(t, err)
	defer engine.Close()

	indexName := "test_task_index"
	err = engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
	assert.NoError(t, err)
	defer engine.DeleteIndex(ctx, indexName)

	// Create a batch operation to get a task ID
	docs := make([]*driver.Document, 5)
	for i := 0; i < 5; i++ {
		docID := fmt.Sprintf("test-doc-%d", i)
		docs[i] = &driver.Document{
			DocID:   docID,
			Content: fmt.Sprintf("Test document for task management %d", i),
		}
	}

	taskID, err := engine.IndexBatch(ctx, indexName, docs)
	if err != nil {
		t.Fatalf("Failed to batch index documents: %v", err)
	}
	assert.NotEmpty(t, taskID)

	// Test GetTaskInfo with retries
	var taskInfo *driver.TaskInfo
	var getTaskErr error
	for i := 0; i < 5; i++ {
		taskInfo, getTaskErr = engine.GetTaskInfo(ctx, taskID)
		if getTaskErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	assert.NoError(t, getTaskErr)
	assert.Equal(t, taskID, taskInfo.TaskID)
	assert.NotZero(t, taskInfo.Created)

	// Test ListTasks
	tasks, err := engine.ListTasks(ctx, indexName)
	assert.NoError(t, err)
	// Note: Since Qdrant doesn't provide direct task listing, this will be empty
	assert.Empty(t, tasks)

	// Test CancelTask
	err = engine.CancelTask(ctx, taskID)
	assert.NoError(t, err)
}

// TestResourceLeaks tests for memory and goroutine leaks
func TestResourceLeaks(t *testing.T) {
	// Record initial state after forcing GC
	runtime.GC()
	time.Sleep(time.Second)
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	numGoroutineStart := runtime.NumGoroutine()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	config := getTestConfig(t)

	// Create a single engine instance
	engine, err := NewEngine(config)
	assert.NoError(t, err)

	// Create cleanup function with proper waiting
	cleanup := func() {
		// Cancel context first
		cancel()

		// Wait for any pending operations
		time.Sleep(2 * time.Second)

		// Close engine
		assert.NoError(t, engine.Close())

		// Force garbage collection and wait
		runtime.GC()
		time.Sleep(time.Second)
		runtime.GC()
		time.Sleep(time.Second)
	}
	defer cleanup()

	// Run operations in a controlled manner
	for i := 0; i < 2; i++ { // Further reduce iterations
		func() {
			indexName := fmt.Sprintf("test_leak_index_%d", i)
			err = engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
			assert.NoError(t, err)

			// Ensure index is deleted even if test fails
			defer func() {
				// Wait before deleting index
				time.Sleep(time.Second)
				assert.NoError(t, engine.DeleteIndex(ctx, indexName))
				time.Sleep(time.Second) // Wait after deletion
			}()

			// Use proper UUID format
			docID := fmt.Sprintf("test-doc-%d", i)
			doc := &driver.Document{
				DocID:   docID,
				Content: "Test document for leak detection",
			}

			// Index document and wait
			err = engine.IndexDoc(ctx, indexName, doc)
			assert.NoError(t, err)
			time.Sleep(time.Second)

			// Search operations
			searchOpts := driver.VectorSearchOptions{
				QueryText: "test document",
				TopK:      5,
			}
			_, err = engine.Search(ctx, indexName, nil, searchOpts)
			assert.NoError(t, err)
			time.Sleep(time.Second)
		}()

		// Force garbage collection after each iteration
		runtime.GC()
		time.Sleep(time.Second)
	}

	// Close engine and wait for cleanup
	cleanup()

	// Force garbage collection multiple times with longer waits
	for i := 0; i < 5; i++ {
		runtime.GC()
		time.Sleep(time.Second)
	}

	// Check final state
	runtime.ReadMemStats(&m2)
	numGoroutineEnd := runtime.NumGoroutine()

	// Check for goroutine leaks with a smaller delta
	delta := 1 // Reduce allowed delta to minimum
	if numGoroutineEnd > numGoroutineStart+delta {
		t.Errorf("Goroutine leak detected: started with %d, ended with %d (delta: %d, allowed: %d)",
			numGoroutineStart, numGoroutineEnd, numGoroutineEnd-numGoroutineStart, delta)
	}

	// Check for significant memory leaks with a reasonable threshold
	memoryGrowth := int64(m2.Alloc) - int64(m1.Alloc)
	maxGrowth := int64(2 * 1024 * 1024) // Reduce to 2MB
	if memoryGrowth > maxGrowth {
		t.Errorf("Memory leak detected: growth %d bytes exceeds threshold of %d bytes",
			memoryGrowth, maxGrowth)
	}
}

// TestConcurrentOperations tests concurrent access to the engine
func TestConcurrentOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := getTestConfig(t)

	engine, err := NewEngine(config)
	assert.NoError(t, err)

	// Ensure index name is unique
	indexName := fmt.Sprintf("test_concurrent_index_%d", time.Now().UnixNano())
	err = engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
	assert.NoError(t, err)

	// Use sync.Once to ensure cleanup happens only once
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			// Wait for all operations to complete
			time.Sleep(2 * time.Second)

			// Delete index first
			if err := engine.DeleteIndex(ctx, indexName); err != nil {
				t.Logf("Failed to delete index: %v", err)
			}
			// Wait for deletion to complete
			time.Sleep(time.Second)
			// Then close the engine
			if err := engine.Close(); err != nil {
				t.Logf("Failed to close engine: %v", err)
			}
			// Force GC
			runtime.GC()
			time.Sleep(time.Second)
		})
	}
	defer cleanup()

	// Number of concurrent operations
	numOps := 5 // Reduce number of concurrent operations
	var wg sync.WaitGroup
	wg.Add(numOps * 2) // For both indexing and searching

	// Channel to collect errors
	errCh := make(chan error, numOps*2)

	// Concurrent indexing
	for i := 0; i < numOps; i++ {
		go func(i int) {
			defer wg.Done()
			docID := fmt.Sprintf("test-doc-concurrent-%d", i)
			doc := &driver.Document{
				DocID:    docID,
				Content:  fmt.Sprintf("Concurrent test document %d", i),
				Metadata: map[string]interface{}{"index": i},
			}
			if err := engine.IndexDoc(ctx, indexName, doc); err != nil {
				select {
				case errCh <- fmt.Errorf("indexing error: %w", err):
				default:
					t.Logf("Error channel full: %v", err)
				}
				return
			}
		}(i)
	}

	// Wait a bit for indexing to complete before searching
	time.Sleep(2 * time.Second)

	// Concurrent searching
	for i := 0; i < numOps; i++ {
		go func(i int) {
			defer wg.Done()
			searchOpts := driver.VectorSearchOptions{
				QueryText: fmt.Sprintf("test document %d", i),
				TopK:      5,
			}
			if _, err := engine.Search(ctx, indexName, nil, searchOpts); err != nil {
				select {
				case errCh <- fmt.Errorf("search error: %w", err):
				default:
					t.Logf("Error channel full: %v", err)
				}
				return
			}
		}(i)
	}

	// Wait for all operations to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All operations completed
	case <-ctx.Done():
		t.Fatal("Timeout waiting for concurrent operations")
	}

	// Close error channel after all operations are done
	close(errCh)

	// Check for errors
	for err := range errCh {
		assert.NoError(t, err)
	}
}

// TestQdrantEngineErrors tests error handling
func TestQdrantEngineErrors(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	// Create engine
	engine, err := NewEngine(config)
	assert.NoError(t, err)

	// Ensure index name is unique
	indexName := fmt.Sprintf("test_error_index_%d", time.Now().UnixNano())
	err = engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
	assert.NoError(t, err)

	// Use defer to ensure proper cleanup order
	defer func() {
		// Delete index first
		if err := engine.DeleteIndex(ctx, indexName); err != nil {
			t.Logf("Failed to delete index: %v", err)
		}
		// Wait for deletion to complete
		time.Sleep(time.Second)
		// Finally close the engine
		if err := engine.Close(); err != nil {
			t.Logf("Failed to close engine: %v", err)
		}
	}()

	// Test non-existent index with valid UUID format
	_, err = engine.GetDocument(ctx, "non_existent_index", "00000000-0000-0000-0000-000000000001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collection doesn't exist")

	// Test non-existent document with valid UUID format
	_, err = engine.GetDocument(ctx, indexName, "00000000-0000-0000-0000-000000000999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document not found")

	// Test invalid task ID format
	_, err = engine.GetTaskInfo(ctx, "invalid-task-id")
	assert.Error(t, err)
	assert.Equal(t, "invalid task ID format", err.Error())

	// Test batch operations with empty input
	_, err = engine.IndexBatch(ctx, indexName, []*driver.Document{})
	assert.Error(t, err)
	assert.Equal(t, "empty document batch", err.Error())

	_, err = engine.DeleteBatch(ctx, indexName, []string{})
	assert.Error(t, err)
	assert.Equal(t, "empty batch", err.Error())

	// Test operations with nil context
	searchOpts := driver.VectorSearchOptions{
		QueryText: "test",
		TopK:      5,
	}
	_, err = engine.Search(nil, indexName, nil, searchOpts)
	assert.Error(t, err)
	assert.Equal(t, "nil context", err.Error())

	// Test operations with canceled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = engine.Search(cancelCtx, indexName, nil, searchOpts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Test invalid vector dimension
	invalidDoc := &driver.Document{
		DocID:      "test-doc-invalid",
		Content:    "Test document",
		Embeddings: []float32{0.1, 0.2}, // Invalid dimension
	}
	err = engine.IndexDoc(ctx, indexName, invalidDoc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dimension mismatch")

	// Create a new engine instance to test operations after engine is closed
	closedEngine, err := NewEngine(config)
	assert.NoError(t, err)
	err = closedEngine.Close()
	assert.NoError(t, err)

	// Test operations after engine is closed
	_, err = closedEngine.Search(ctx, indexName, nil, searchOpts)
	assert.Error(t, err)
	assert.Equal(t, "engine is closed", err.Error())

	// Test empty batch delete
	_, err = engine.DeleteBatch(ctx, indexName, []string{})
	assert.Error(t, err)
	assert.Equal(t, "empty batch", err.Error())
}

// TestGetMetadata tests the GetMetadata functionality
func TestGetMetadata(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	engine, err := NewEngine(config)
	assert.NoError(t, err)
	defer engine.Close()

	indexName := fmt.Sprintf("test_metadata_index_%d", time.Now().UnixNano())
	err = engine.CreateIndex(ctx, driver.IndexConfig{Name: indexName})
	assert.NoError(t, err)
	defer engine.DeleteIndex(ctx, indexName)

	// Test document with metadata
	doc := &driver.Document{
		DocID:   "test-doc-metadata",
		Content: "Test document with metadata",
		Metadata: map[string]interface{}{
			"type":    "test",
			"version": 1.0,
			"tags":    []string{"test", "metadata"},
			"nested": map[string]interface{}{
				"key": "value",
			},
		},
	}

	// Index the document
	err = engine.IndexDoc(ctx, indexName, doc)
	assert.NoError(t, err)

	// Test GetMetadata
	metadata, err := engine.GetMetadata(ctx, indexName, doc.DocID)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "test", metadata["type"])
	assert.Equal(t, 1.0, metadata["version"])

	// Test GetMetadata with non-existent document
	_, err = engine.GetMetadata(ctx, indexName, "non-existent-doc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document not found")

	// Test GetMetadata with non-existent collection
	_, err = engine.GetMetadata(ctx, "non-existent-index", doc.DocID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collection doesn't exist")

	// Test GetMetadata with nil context
	_, err = engine.GetMetadata(nil, indexName, doc.DocID)
	assert.Error(t, err)
	assert.Equal(t, "nil context", err.Error())

	// Test GetMetadata after engine is closed
	closedEngine, err := NewEngine(config)
	assert.NoError(t, err)
	err = closedEngine.Close()
	assert.NoError(t, err)
	_, err = closedEngine.GetMetadata(ctx, indexName, doc.DocID)
	assert.Error(t, err)
	assert.Equal(t, "engine is closed", err.Error())
}
