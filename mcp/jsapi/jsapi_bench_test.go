package jsapi_test

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
)

// cleanupGoObjects forces cleanup of all registered Go objects
func cleanupGoObjects(t *testing.T) {
	// Force multiple rounds of GC to ensure JavaScript objects are collected
	// This triggers V8 finalizers which call __release on MCP objects
	for i := 0; i < 5; i++ {
		runtime.GC()
	}

	// Log remaining objects for debugging
	remaining := bridge.CountGoObjects()
	if remaining > 0 {
		t.Logf("Warning: %d Go objects still registered after cleanup", remaining)
	}
}

// TestStressConcurrent tests concurrent MCP client usage under stress
func TestStressConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Force GC before starting to ensure clean state
	runtime.GC()
	runtime.GC() // Call twice to ensure full collection

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	concurrency := 50
	iterationsPerGoroutine := 20

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterationsPerGoroutine)
	results := make(chan bool, concurrency*iterationsPerGoroutine)

	start := time.Now()

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				_, err := v8.Call(v8.CallOptions{}, `
					function test() {
						const client = new MCP("echo");
						try {
							// List tools
							const tools = client.ListTools();
							if (!tools || !Array.isArray(tools.tools)) {
								throw new Error("Invalid tools response");
							}
							
							// Call tool
							const result = client.CallTool("ping", { count: 1 });
							if (!result || !result.content) {
								throw new Error("Invalid tool call response");
							}
							
							// List resources (if any)
							const resources = client.ListResources();
							
							return { success: true };
						} catch (error) {
							return { success: false, error: error.message };
						} finally {
							client.Release(); // Always release resources
						}
					}`)

				if err != nil {
					errors <- fmt.Errorf("routine %d iteration %d failed: %v", routineID, j, err)
					return
				}

				results <- true
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)
	close(results)

	elapsed := time.Since(start)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	// Count successes
	successCount := 0
	for range results {
		successCount++
	}

	totalOps := concurrency * iterationsPerGoroutine
	t.Logf("Stress test completed:")
	t.Logf("  Total operations: %d", totalOps)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Time elapsed: %v", elapsed)
	t.Logf("  Operations/sec: %.2f", float64(totalOps)/elapsed.Seconds())

	assert.Equal(t, totalOps, successCount, "All operations should succeed")

	// Cleanup registered Go objects
	cleanupGoObjects(t)
	finalObjects := bridge.CountGoObjects()
	t.Logf("Final Go objects: %d (delta: %+d)", finalObjects, finalObjects-initialObjects)
}

// TestMemoryLeak tests for memory leaks in MCP client usage
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	// Force GC and get baseline memory
	runtime.GC()
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	iterations := 1000
	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				const client = new MCP("echo");
				try {
					// Perform various operations
					const tools = client.ListTools();
					const result = client.CallTool("ping", { count: 1 });
					const prompts = client.ListPrompts();
					
					return { success: true };
				} finally {
					client.Release(); // Manually release resources
				}
			}`)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Force GC every 100 iterations
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force final GC and measure memory
	runtime.GC()
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate memory increase
	allocDiff := m2.Alloc - m1.Alloc
	heapDiff := m2.HeapAlloc - m1.HeapAlloc

	t.Logf("Memory usage after %d iterations:", iterations)
	t.Logf("  Alloc diff: %d bytes (%.2f MB)", allocDiff, float64(allocDiff)/(1024*1024))
	t.Logf("  HeapAlloc diff: %d bytes (%.2f MB)", heapDiff, float64(heapDiff)/(1024*1024))
	t.Logf("  Total Alloc: %d bytes (%.2f MB)", m2.TotalAlloc, float64(m2.TotalAlloc)/(1024*1024))
	t.Logf("  Mallocs: %d", m2.Mallocs-m1.Mallocs)
	t.Logf("  Frees: %d", m2.Frees-m1.Frees)
	t.Logf("  GC runs: %d", m2.NumGC-m1.NumGC)

	// Memory increase should be reasonable (< 10MB for 1000 iterations)
	maxAllowedIncrease := int64(10 * 1024 * 1024) // 10MB
	if int64(allocDiff) > maxAllowedIncrease {
		t.Errorf("Memory leak detected: allocation increased by %.2f MB (max allowed: %.2f MB)",
			float64(allocDiff)/(1024*1024),
			float64(maxAllowedIncrease)/(1024*1024))
	}

	// Cleanup registered Go objects
	cleanupGoObjects(t)
	finalObjects := bridge.CountGoObjects()
	t.Logf("Final Go objects: %d (delta: %+d)", finalObjects, finalObjects-initialObjects)

	// Check for Go object leaks
	objectDelta := finalObjects - initialObjects
	if objectDelta > 10 {
		t.Errorf("Go object leak detected: %d objects not released", objectDelta)
	}
}

// TestMemoryLeakWithMultipleClients tests memory leaks with multiple client instances
func TestMemoryLeakWithMultipleClients(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	// Force GC multiple times and get baseline memory
	runtime.GC()
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	iterations := 500
	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				// Create multiple client instances
				const client1 = new MCP("echo");
				const client2 = new MCP("dsl");
				const client3 = new MCP("customer");
				
				try {
					// Perform operations on each
					client1.ListTools();
					client2.ListTools();
					client3.ListResources();
					
					return { success: true };
				} finally {
					// Release all clients
					client1.Release();
					client2.Release();
					client3.Release();
				}
			}`)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Force GC every 50 iterations
		if i%50 == 0 {
			runtime.GC()
		}
	}

	// Force final GC and measure memory
	runtime.GC()
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate memory increase
	allocDiff := m2.Alloc - m1.Alloc
	heapDiff := m2.HeapAlloc - m1.HeapAlloc

	t.Logf("Memory usage after %d iterations (3 clients each):", iterations)
	t.Logf("  Alloc diff: %d bytes (%.2f MB)", allocDiff, float64(allocDiff)/(1024*1024))
	t.Logf("  HeapAlloc diff: %d bytes (%.2f MB)", heapDiff, float64(heapDiff)/(1024*1024))
	t.Logf("  GC runs: %d", m2.NumGC-m1.NumGC)

	// Memory increase should be reasonable (< 15MB for 500 iterations with 3 clients each)
	maxAllowedIncrease := int64(15 * 1024 * 1024) // 15MB
	if int64(allocDiff) > maxAllowedIncrease {
		t.Errorf("Memory leak detected: allocation increased by %.2f MB (max allowed: %.2f MB)",
			float64(allocDiff)/(1024*1024),
			float64(maxAllowedIncrease)/(1024*1024))
	}

	// Cleanup registered Go objects
	cleanupGoObjects(t)
	finalObjects := bridge.CountGoObjects()
	t.Logf("Final Go objects: %d (delta: %+d)", finalObjects, finalObjects-initialObjects)

	// Check for Go object leaks (3 clients per iteration)
	objectDelta := finalObjects - initialObjects
	if objectDelta > 10 {
		t.Errorf("Go object leak detected: %d objects not released", objectDelta)
	}
}

// TestRapidCreateDestroy tests rapid creation and destruction of MCP clients
func TestRapidCreateDestroy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rapid create/destroy test in short mode")
	}

	// Force GC before starting to ensure clean state
	// Multiple GC calls after previous tests that may have used significant memory
	for i := 0; i < 5; i++ {
		runtime.GC()
	}

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	iterations := 1000 // Increased back - Isolate disposal fix should prevent crashes
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				const client = new MCP("echo");
				try {
					return { id: client.id };
				} finally {
					client.Release(); // Explicitly release
				}
			}`)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Periodic GC to prevent excessive buildup
		if i%100 == 0 && i > 0 {
			runtime.GC()
		}
	}

	elapsed := time.Since(start)

	t.Logf("Rapid create/destroy test completed:")
	t.Logf("  Total iterations: %d", iterations)
	t.Logf("  Time elapsed: %v", elapsed)
	t.Logf("  Operations/sec: %.2f", float64(iterations)/elapsed.Seconds())
	t.Logf("  Avg time per operation: %v", elapsed/time.Duration(iterations))

	// Cleanup registered Go objects
	cleanupGoObjects(t)
	finalObjects := bridge.CountGoObjects()
	t.Logf("Final Go objects: %d (delta: %+d)", finalObjects, finalObjects-initialObjects)

	// Check for Go object leaks
	objectDelta := finalObjects - initialObjects
	if objectDelta > 10 {
		t.Errorf("Go object leak detected: %d objects not released", objectDelta)
	}
}

// TestLongRunningOperation tests MCP client with long-running operations
func TestLongRunningOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	duration := 10 * time.Second
	tickInterval := 100 * time.Millisecond
	operations := 0

	start := time.Now()
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(duration)
		done <- true
	}()

	for {
		select {
		case <-done:
			elapsed := time.Since(start)
			t.Logf("Long-running test completed:")
			t.Logf("  Duration: %v", elapsed)
			t.Logf("  Total operations: %d", operations)
			t.Logf("  Operations/sec: %.2f", float64(operations)/elapsed.Seconds())
			return

		case <-ticker.C:
			_, err := v8.Call(v8.CallOptions{}, `
				function test() {
					const client = new MCP("echo");
					try {
						const result = client.CallTool("ping", { count: 1 });
						return { success: true };
					} finally {
						client.Release();
					}
				}`)

			if err != nil {
				t.Fatalf("Operation %d failed: %v", operations, err)
			}
			operations++
		}
	}
}

// BenchmarkMCPConstruction benchmarks MCP client construction
func BenchmarkMCPConstruction(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				const client = new MCP("echo");
				return { id: client.id };
			}`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMCPListTools benchmarks ListTools method
func BenchmarkMCPListTools(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				const client = new MCP("echo");
				const tools = client.ListTools();
				return { count: tools.tools.length };
			}`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMCPCallTool benchmarks CallTool method
func BenchmarkMCPCallTool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				const client = new MCP("echo");
				const result = client.CallTool("ping", { count: 1 });
				return { success: true };
			}`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMCPParallel benchmarks parallel MCP operations
func BenchmarkMCPParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := v8.Call(v8.CallOptions{}, `
				function test() {
					const client = new MCP("echo");
					const result = client.CallTool("ping", { count: 1 });
					return { success: true };
				}`)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
