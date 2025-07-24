package qdrant

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestGetStats tests the GetStats function
func TestGetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stats tests in short mode")
	}

	tests := []struct {
		name           string
		language       string
		collectionName string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Valid English collection",
			language:       "en",
			collectionName: "", // Will be set from test data
			expectError:    false,
		},
		{
			name:           "Valid Chinese collection",
			language:       "zh",
			collectionName: "", // Will be set from test data
			expectError:    false,
		},
		{
			name:           "Empty collection name",
			language:       "en",
			collectionName: "",
			expectError:    true,
			errorContains:  "collection name cannot be empty",
		},
		{
			name:           "Non-existent collection",
			language:       "en",
			collectionName: "non_existent_collection_12345",
			expectError:    true,
			errorContains:  "failed to get collection info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get test environment
			env := getOrCreateSearchTestEnvironment(t)

			// Handle empty collection name test case
			if tt.name == "Empty collection name" {
				ctx := context.Background()
				_, err := env.Store.GetStats(ctx, "")

				if !tt.expectError {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if err == nil {
					t.Error("Expected error for empty collection name, got nil")
					return
				}

				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// For non-existent collection test
			if tt.name == "Non-existent collection" {
				ctx := context.Background()
				_, err := env.Store.GetStats(ctx, tt.collectionName)

				if !tt.expectError {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if err == nil {
					t.Error("Expected error for non-existent collection, got nil")
					return
				}

				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// Get or create test data set
			dataSet := getOrCreateTestDataSet(t, tt.language)
			collectionName := dataSet.CollectionName

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Get stats
			stats, err := env.Store.GetStats(ctx, collectionName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if stats == nil {
				t.Error("Expected stats, got nil")
				return
			}

			// Validate stats fields
			if stats.TotalVectors < 0 {
				t.Errorf("Expected non-negative TotalVectors, got: %d", stats.TotalVectors)
			}

			if stats.Dimension <= 0 {
				t.Errorf("Expected positive Dimension, got: %d", stats.Dimension)
			}

			if stats.IndexType == "" {
				t.Error("Expected non-empty IndexType")
			}

			if stats.DistanceMetric == "" {
				t.Error("Expected non-empty DistanceMetric")
			}

			if stats.IndexSize < 0 {
				t.Errorf("Expected non-negative IndexSize, got: %d", stats.IndexSize)
			}

			if stats.MemoryUsage < 0 {
				t.Errorf("Expected non-negative MemoryUsage, got: %d", stats.MemoryUsage)
			}

			if stats.ExtraStats == nil {
				t.Error("Expected ExtraStats to be initialized")
			}

			t.Logf("Stats for %s: TotalVectors=%d, Dimension=%d, IndexType=%s, DistanceMetric=%s, IndexSize=%d, MemoryUsage=%d",
				collectionName, stats.TotalVectors, stats.Dimension, stats.IndexType, stats.DistanceMetric, stats.IndexSize, stats.MemoryUsage)
		})
	}
}

// TestGetSearchEngineStats tests the GetSearchEngineStats function
func TestGetSearchEngineStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping search engine stats tests in short mode")
	}

	tests := []struct {
		name           string
		language       string
		collectionName string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Valid English collection",
			language:       "en",
			collectionName: "", // Will be set from test data
			expectError:    false,
		},
		{
			name:           "Valid Chinese collection",
			language:       "zh",
			collectionName: "", // Will be set from test data
			expectError:    false,
		},
		{
			name:           "Empty collection name",
			language:       "en",
			collectionName: "",
			expectError:    true,
			errorContains:  "collection name cannot be empty",
		},
		{
			name:           "Non-existent collection",
			language:       "en",
			collectionName: "non_existent_search_collection_12345",
			expectError:    true,
			errorContains:  "failed to get collection info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get test environment
			env := getOrCreateSearchTestEnvironment(t)

			// Handle empty collection name test case
			if tt.name == "Empty collection name" {
				ctx := context.Background()
				_, err := env.Store.GetSearchEngineStats(ctx, "")

				if !tt.expectError {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if err == nil {
					t.Error("Expected error for empty collection name, got nil")
					return
				}

				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// For non-existent collection test
			if tt.name == "Non-existent collection" {
				ctx := context.Background()
				_, err := env.Store.GetSearchEngineStats(ctx, tt.collectionName)

				if !tt.expectError {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if err == nil {
					t.Error("Expected error for non-existent collection, got nil")
					return
				}

				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// Get or create test data set
			dataSet := getOrCreateTestDataSet(t, tt.language)
			collectionName := dataSet.CollectionName

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Get search engine stats
			stats, err := env.Store.GetSearchEngineStats(ctx, collectionName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if stats == nil {
				t.Error("Expected search engine stats, got nil")
				return
			}

			// Validate search engine stats fields
			if stats.TotalQueries < 0 {
				t.Errorf("Expected non-negative TotalQueries, got: %d", stats.TotalQueries)
			}

			if stats.AverageQueryTime < 0 {
				t.Errorf("Expected non-negative AverageQueryTime, got: %f", stats.AverageQueryTime)
			}

			if stats.CacheHitRate < 0 || stats.CacheHitRate > 1 {
				t.Errorf("Expected CacheHitRate between 0 and 1, got: %f", stats.CacheHitRate)
			}

			if stats.ErrorRate < 0 || stats.ErrorRate > 1 {
				t.Errorf("Expected ErrorRate between 0 and 1, got: %f", stats.ErrorRate)
			}

			if stats.IndexSize < 0 {
				t.Errorf("Expected non-negative IndexSize, got: %d", stats.IndexSize)
			}

			if stats.DocumentCount < 0 {
				t.Errorf("Expected non-negative DocumentCount, got: %d", stats.DocumentCount)
			}

			if stats.PopularQueries == nil {
				t.Error("Expected PopularQueries to be initialized")
			}

			if stats.SlowQueries == nil {
				t.Error("Expected SlowQueries to be initialized")
			}

			t.Logf("Search Engine Stats for %s: DocumentCount=%d, IndexSize=%d, AverageQueryTime=%f, TotalQueries=%d",
				collectionName, stats.DocumentCount, stats.IndexSize, stats.AverageQueryTime, stats.TotalQueries)
		})
	}
}

// TestOptimize tests the Optimize function
func TestOptimize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping optimize tests in short mode")
	}

	tests := []struct {
		name           string
		language       string
		collectionName string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Valid English collection",
			language:       "en",
			collectionName: "", // Will be set from test data
			expectError:    false,
		},
		{
			name:           "Valid Chinese collection",
			language:       "zh",
			collectionName: "", // Will be set from test data
			expectError:    false,
		},
		{
			name:           "Empty collection name",
			language:       "en",
			collectionName: "",
			expectError:    true,
			errorContains:  "collection name cannot be empty",
		},
		{
			name:           "Non-existent collection",
			language:       "en",
			collectionName: "non_existent_optimize_collection_12345",
			expectError:    true,
			errorContains:  "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get test environment
			env := getOrCreateSearchTestEnvironment(t)

			// Handle empty collection name test case
			if tt.name == "Empty collection name" {
				ctx := context.Background()
				err := env.Store.Optimize(ctx, "")

				if !tt.expectError {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if err == nil {
					t.Error("Expected error for empty collection name, got nil")
					return
				}

				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// For non-existent collection test
			if tt.name == "Non-existent collection" {
				ctx := context.Background()
				err := env.Store.Optimize(ctx, tt.collectionName)

				if !tt.expectError {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if err == nil {
					t.Error("Expected error for non-existent collection, got nil")
					return
				}

				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// Get or create test data set
			dataSet := getOrCreateTestDataSet(t, tt.language)
			collectionName := dataSet.CollectionName

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Optimize collection
			err := env.Store.Optimize(ctx, collectionName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			t.Logf("Successfully optimized collection: %s", collectionName)
		})
	}
}

// TestDisconnectedStore tests behavior when store is disconnected
func TestStatsWithDisconnectedStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping disconnected store tests in short mode")
	}

	// Create a new store without connecting
	store := NewStore()
	ctx := context.Background()

	// Test GetStats with disconnected store
	t.Run("GetStats with disconnected store", func(t *testing.T) {
		_, err := store.GetStats(ctx, "test_collection")
		if err == nil {
			t.Error("Expected error for disconnected store, got nil")
		}
		if !containsString(err.Error(), "not connected") {
			t.Errorf("Expected error to contain 'not connected', got: %v", err)
		}
	})

	// Test GetSearchEngineStats with disconnected store
	t.Run("GetSearchEngineStats with disconnected store", func(t *testing.T) {
		_, err := store.GetSearchEngineStats(ctx, "test_collection")
		if err == nil {
			t.Error("Expected error for disconnected store, got nil")
		}
		if !containsString(err.Error(), "not connected") {
			t.Errorf("Expected error to contain 'not connected', got: %v", err)
		}
	})

	// Test Optimize with disconnected store
	t.Run("Optimize with disconnected store", func(t *testing.T) {
		err := store.Optimize(ctx, "test_collection")
		if err == nil {
			t.Error("Expected error for disconnected store, got nil")
		}
		if !containsString(err.Error(), "not connected") {
			t.Errorf("Expected error to contain 'not connected', got: %v", err)
		}
	})
}

// TestStatsConcurrency tests concurrent access to stats functions
func TestStatsConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	// Get test environment
	env := getOrCreateSearchTestEnvironment(t)
	dataSet := getOrCreateTestDataSet(t, "en")
	collectionName := dataSet.CollectionName

	numGoroutines := 10
	numOperations := 5

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*numOperations)

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

				switch j % 3 {
				case 0:
					// Test GetStats
					_, err := env.Store.GetStats(ctx, collectionName)
					if err != nil {
						errChan <- err
					}
				case 1:
					// Test GetSearchEngineStats
					_, err := env.Store.GetSearchEngineStats(ctx, collectionName)
					if err != nil {
						errChan <- err
					}
				case 2:
					// Test Optimize (less frequently to avoid conflicts)
					if j == 2 { // Only run optimize once per goroutine
						err := env.Store.Optimize(ctx, collectionName)
						if err != nil {
							errChan <- err
						}
					}
				}

				cancel()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent operations failed with %d errors. First error: %v", len(errors), errors[0])
	}

	t.Logf("Successfully completed %d concurrent operations", numGoroutines*numOperations)
}

// TestStatsMemoryLeak tests for memory leaks in stats functions
func TestStatsMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak tests in short mode")
	}

	// Get test environment
	env := getOrCreateSearchTestEnvironment(t)
	dataSet := getOrCreateTestDataSet(t, "en")
	collectionName := dataSet.CollectionName

	// Get initial memory stats
	var initialMemStats, finalMemStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMemStats)

	// Run operations multiple times
	iterations := 100
	for i := 0; i < iterations; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		// GetStats
		_, err := env.Store.GetStats(ctx, collectionName)
		if err != nil {
			t.Errorf("GetStats failed at iteration %d: %v", i, err)
		}

		// GetSearchEngineStats
		_, err = env.Store.GetSearchEngineStats(ctx, collectionName)
		if err != nil {
			t.Errorf("GetSearchEngineStats failed at iteration %d: %v", i, err)
		}

		cancel()

		// Force garbage collection every 10 iterations
		if i%10 == 0 {
			runtime.GC()
		}
	}

	// Get final memory stats
	runtime.GC()
	runtime.ReadMemStats(&finalMemStats)

	// Check memory growth
	initialMem := initialMemStats.Alloc
	finalMem := finalMemStats.Alloc

	// Calculate growth safely to avoid overflow
	var memGrowth int64
	if finalMem >= initialMem {
		memGrowth = int64(finalMem - initialMem)
	} else {
		memGrowth = -int64(initialMem - finalMem)
	}

	// Allow some memory growth but flag if excessive (more than 10MB)
	const maxAllowedGrowth = 10 * 1024 * 1024 // 10MB
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Potential memory leak detected: memory grew by %d bytes (%.2f MB) over %d iterations",
			memGrowth, float64(memGrowth)/(1024*1024), iterations)
	}

	t.Logf("Memory usage: initial=%d bytes, final=%d bytes, growth=%d bytes (%.2f MB)",
		initialMem, finalMem, memGrowth, float64(memGrowth)/(1024*1024))
}

// BenchmarkGetStats benchmarks the GetStats function
func BenchmarkGetStats(b *testing.B) {
	// Skip if no Qdrant available
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Setup
	env := getOrCreateSearchTestEnvironment(&testing.T{})
	dataSet := getOrCreateTestDataSet(&testing.T{}, "en")
	collectionName := dataSet.CollectionName

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := env.Store.GetStats(ctx, collectionName)
			if err != nil {
				b.Errorf("GetStats failed: %v", err)
			}
		}
	})
}

// BenchmarkGetSearchEngineStats benchmarks the GetSearchEngineStats function
func BenchmarkGetSearchEngineStats(b *testing.B) {
	// Skip if no Qdrant available
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Setup
	env := getOrCreateSearchTestEnvironment(&testing.T{})
	dataSet := getOrCreateTestDataSet(&testing.T{}, "en")
	collectionName := dataSet.CollectionName

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := env.Store.GetSearchEngineStats(ctx, collectionName)
			if err != nil {
				b.Errorf("GetSearchEngineStats failed: %v", err)
			}
		}
	})
}

// TestStatsEdgeCases tests edge cases for better coverage
func TestStatsEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping edge case tests in short mode")
	}

	// Get test environment
	env := getOrCreateSearchTestEnvironment(t)
	dataSet := getOrCreateTestDataSet(t, "en")
	collectionName := dataSet.CollectionName

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test GetStats with context cancellation
	t.Run("GetStats with cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := env.Store.GetStats(cancelledCtx, collectionName)
		if err == nil {
			t.Log("GetStats with cancelled context succeeded (this is okay)")
		} else {
			t.Logf("GetStats with cancelled context failed as expected: %v", err)
		}
	})

	// Test GetSearchEngineStats with context cancellation
	t.Run("GetSearchEngineStats with cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := env.Store.GetSearchEngineStats(cancelledCtx, collectionName)
		if err == nil {
			t.Log("GetSearchEngineStats with cancelled context succeeded (this is okay)")
		} else {
			t.Logf("GetSearchEngineStats with cancelled context failed as expected: %v", err)
		}
	})

	// Test Optimize with context cancellation
	t.Run("Optimize with cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := env.Store.Optimize(cancelledCtx, collectionName)
		if err == nil {
			t.Log("Optimize with cancelled context succeeded (this is okay)")
		} else {
			t.Logf("Optimize with cancelled context failed as expected: %v", err)
		}
	})

	// Test GetStats with empty collection (0 vectors)
	t.Run("GetStats with empty collection", func(t *testing.T) {
		// Create an empty collection for testing
		emptyCollectionName := fmt.Sprintf("test_empty_%d", time.Now().UnixNano())

		// Create collection config
		collectionConfig := types.CreateCollectionOptions{
			CollectionName: emptyCollectionName,
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
		}

		// Create empty collection
		err := env.Store.CreateCollection(ctx, &collectionConfig)
		if err != nil {
			t.Skipf("Failed to create empty collection: %v", err)
		}

		// Cleanup
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			env.Store.DropCollection(cleanupCtx, emptyCollectionName)
		}()

		// Get stats for empty collection
		stats, err := env.Store.GetStats(ctx, emptyCollectionName)
		if err != nil {
			t.Errorf("Failed to get stats for empty collection: %v", err)
			return
		}

		if stats.TotalVectors != 0 {
			t.Errorf("Expected 0 vectors in empty collection, got %d", stats.TotalVectors)
		}

		if stats.IndexSize != 0 {
			t.Errorf("Expected 0 index size for empty collection, got %d", stats.IndexSize)
		}
	})

	// Test GetSearchEngineStats with empty collection
	t.Run("GetSearchEngineStats with empty collection", func(t *testing.T) {
		// Create an empty collection for testing
		emptyCollectionName := fmt.Sprintf("test_empty_search_%d", time.Now().UnixNano())

		// Create collection config
		collectionConfig := types.CreateCollectionOptions{
			CollectionName: emptyCollectionName,
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
		}

		// Create empty collection
		err := env.Store.CreateCollection(ctx, &collectionConfig)
		if err != nil {
			t.Skipf("Failed to create empty collection: %v", err)
		}

		// Cleanup
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			env.Store.DropCollection(cleanupCtx, emptyCollectionName)
		}()

		// Get search engine stats for empty collection
		stats, err := env.Store.GetSearchEngineStats(ctx, emptyCollectionName)
		if err != nil {
			t.Errorf("Failed to get search engine stats for empty collection: %v", err)
			return
		}

		if stats.DocumentCount != 0 {
			t.Errorf("Expected 0 documents in empty collection, got %d", stats.DocumentCount)
		}

		if stats.IndexSize != 0 {
			t.Errorf("Expected 0 index size for empty collection, got %d", stats.IndexSize)
		}
	})
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
