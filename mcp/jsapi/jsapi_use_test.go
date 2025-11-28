package jsapi_test

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
)

// ============================================================================
// Basic Use() Tests
// ============================================================================

// TestMCPWithUse tests using MCP with the Use() function for automatic cleanup
func TestMCPWithUse(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			return Use(MCP, "dsl", (client) => {
				const tools = client.ListTools();
				return {
					id: client.id,
					toolCount: tools.tools.length,
					hasRelease: typeof client.Release === 'function'
				};
			});
		}`)

	assert.NoError(t, err)
	result, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "dsl", result["id"])
	assert.True(t, result["toolCount"].(float64) > 0)
	assert.Equal(t, true, result["hasRelease"])
}

// TestMCPWithUseNested tests nested Use() calls with multiple MCP clients
func TestMCPWithUseNested(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			return Use(MCP, "dsl", (dslClient) => {
				return Use(MCP, "customer", (customerClient) => {
					const tools = dslClient.ListTools();
					const resources = customerClient.ListResources();
					return {
						dslId: dslClient.id,
						customerId: customerClient.id,
						toolCount: tools.tools.length,
						resourceCount: resources.resources.length
					};
				});
			});
		}`)

	assert.NoError(t, err)
	result, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "dsl", result["dslId"])
	assert.Equal(t, "customer", result["customerId"])
	assert.True(t, result["toolCount"].(float64) > 0)
	assert.True(t, result["resourceCount"].(float64) > 0)
}

// TestMCPWithUseError tests that Use() properly propagates errors
func TestMCPWithUseError(t *testing.T) {
	// Errors thrown inside Use() should propagate to v8.Call()
	_, err := v8.Call(v8.CallOptions{}, `
		function test() {
			Use(MCP, "dsl", (client) => {
				throw new Error("Test error in Use callback");
			});
		}`)

	// The error should propagate
	if err == nil {
		t.Fatal("Expected error but got none - error should propagate from Use() callback")
	}
	assert.Contains(t, err.Error(), "Test error", "Error message should be preserved")
}

// TestMCPWithUseCallTool tests calling tools with Use()
func TestMCPWithUseCallTool(t *testing.T) {
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			return Use(MCP, "echo", (client) => {
				const result = client.CallTool("ping", { count: 1 });
				return {
					hasContent: result.content && result.content.length > 0,
					isSuccess: result.isError === false
				};
			});
		}`)

	assert.NoError(t, err)
	result, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, result["hasContent"])
}

// ============================================================================
// Memory Leak Tests
// ============================================================================

// TestUseMemoryLeak tests that Use() doesn't cause memory leaks
func TestUseMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	// Force GC and get baseline
	runtime.GC()
	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	// Execute many iterations with Use()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				return Use(MCP, "dsl", (client) => {
					const tools = client.ListTools();
					return { count: tools.tools.length };
				});
			}`)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Periodic GC
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force final GC and measure memory
	runtime.GC()
	runtime.GC()
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	finalObjects := bridge.CountGoObjects()
	t.Logf("Final Go objects: %d", finalObjects)

	// Calculate memory growth
	allocDiff := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(allocDiff) / float64(iterations)

	t.Logf("Memory Statistics (Use() with %d iterations):", iterations)
	t.Logf("  Baseline HeapAlloc:   %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc:      %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth:               %d bytes (%.2f MB)", allocDiff, float64(allocDiff)/1024/1024)
	t.Logf("  Growth/iteration:     %.2f bytes", growthPerIteration)
	t.Logf("  Go object delta:      %+d", finalObjects-initialObjects)
	t.Logf("  GC Runs:              %d", final.NumGC-baseline.NumGC)

	// Memory should be stable with Use()
	maxGrowthPerIteration := 10240.0 // 10KB per iteration
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
			growthPerIteration, maxGrowthPerIteration)
	}

	// Go objects should be cleaned up
	objectDelta := finalObjects - initialObjects
	if objectDelta > 10 {
		t.Errorf("Go object leak detected: %d objects not released", objectDelta)
	}
}

// TestUseMemoryLeakMultipleClients tests memory with multiple nested Use() calls
func TestUseMemoryLeakMultipleClients(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	// Force GC and get baseline
	runtime.GC()
	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	// Execute iterations with multiple nested clients
	iterations := 500
	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				return Use(MCP, "dsl", (dslClient) => {
					return Use(MCP, "customer", (customerClient) => {
						return Use(MCP, "echo", (echoClient) => {
							dslClient.ListTools();
							customerClient.ListResources();
							echoClient.CallTool("ping", { count: 1 });
							return { success: true };
						});
					});
				});
			}`)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Periodic GC
		if i%50 == 0 {
			runtime.GC()
		}
	}

	// Force final GC and measure memory
	runtime.GC()
	runtime.GC()
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	finalObjects := bridge.CountGoObjects()
	t.Logf("Final Go objects: %d", finalObjects)

	// Calculate memory growth
	allocDiff := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(allocDiff) / float64(iterations)

	t.Logf("Memory Statistics (3 nested clients, %d iterations):", iterations)
	t.Logf("  Baseline HeapAlloc:   %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc:      %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth:               %d bytes (%.2f MB)", allocDiff, float64(allocDiff)/1024/1024)
	t.Logf("  Growth/iteration:     %.2f bytes", growthPerIteration)
	t.Logf("  Go object delta:      %+d", finalObjects-initialObjects)
	t.Logf("  GC Runs:              %d", final.NumGC-baseline.NumGC)

	// Allow more overhead for multiple clients
	maxGrowthPerIteration := 15360.0 // 15KB per iteration
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
			growthPerIteration, maxGrowthPerIteration)
	}

	// Go objects should be cleaned up
	objectDelta := finalObjects - initialObjects
	if objectDelta > 10 {
		t.Errorf("Go object leak detected: %d objects not released", objectDelta)
	}
}

// ============================================================================
// Stress Tests
// ============================================================================

// TestUseStressConcurrent tests concurrent Use() calls under stress
func TestUseStressConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	concurrency := 50
	iterationsPerGoroutine := 20

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterationsPerGoroutine)
	successes := make(chan bool, concurrency*iterationsPerGoroutine)

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				_, err := v8.Call(v8.CallOptions{}, `
					function test() {
						return Use(MCP, "echo", (client) => {
							const tools = client.ListTools();
							const result = client.CallTool("ping", { count: 1 });
							return { 
								toolCount: tools.tools.length,
								hasContent: result.content && result.content.length > 0
							};
						});
					}`)

				if err != nil {
					errors <- err
				} else {
					successes <- true
				}
			}
		}(i)
	}

	// Wait for all goroutines
	wg.Wait()
	close(errors)
	close(successes)

	// Count results
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Errorf("Concurrent call failed: %v", err)
	}

	successCount := 0
	for range successes {
		successCount++
	}

	totalOps := concurrency * iterationsPerGoroutine
	t.Logf("Concurrent stress test completed:")
	t.Logf("  Total operations: %d", totalOps)
	t.Logf("  Successful:       %d", successCount)
	t.Logf("  Failed:           %d", errorCount)

	// Cleanup Go objects
	runtime.GC()
	runtime.GC()
	finalObjects := bridge.CountGoObjects()
	t.Logf("Final Go objects: %d (delta: %+d)", finalObjects, finalObjects-initialObjects)

	assert.Equal(t, totalOps, successCount, "All operations should succeed")

	// Check for Go object leaks
	objectDelta := finalObjects - initialObjects
	if objectDelta > 20 {
		t.Errorf("Go object leak detected: %d objects not released", objectDelta)
	}
}

// TestUseRapidCreateDestroy tests rapid creation and destruction with Use()
func TestUseRapidCreateDestroy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rapid test in short mode")
	}

	initialObjects := bridge.CountGoObjects()
	t.Logf("Initial Go objects: %d", initialObjects)

	iterations := 1000
	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test() {
				return Use(MCP, "dsl", (client) => {
					return { id: client.id };
				});
			}`)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}
	}

	// Cleanup
	runtime.GC()
	runtime.GC()
	finalObjects := bridge.CountGoObjects()

	t.Logf("Rapid create/destroy test completed:")
	t.Logf("  Iterations:    %d", iterations)
	t.Logf("  Final objects: %d (delta: %+d)", finalObjects, finalObjects-initialObjects)

	// Check for Go object leaks
	objectDelta := finalObjects - initialObjects
	if objectDelta > 10 {
		t.Errorf("Go object leak detected: %d objects not released", objectDelta)
	}
}

// ============================================================================
// Comparison Tests: Use() vs try-finally vs No cleanup
// ============================================================================

// TestComparisonUseVsManual compares Use() with manual try-finally cleanup
func TestComparisonUseVsManual(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	iterations := 100

	// Test 1: With Use()
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	obj1 := bridge.CountGoObjects()

	for i := 0; i < iterations; i++ {
		v8.Call(v8.CallOptions{}, `
			function test() {
				return Use(MCP, "dsl", (client) => {
					return client.ListTools();
				});
			}`)
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	obj2 := bridge.CountGoObjects()

	useGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	useObjDelta := obj2 - obj1

	// Test 2: With manual try-finally
	runtime.GC()
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)
	obj3 := bridge.CountGoObjects()

	for i := 0; i < iterations; i++ {
		v8.Call(v8.CallOptions{}, `
			function test() {
				const client = new MCP("dsl");
				try {
					return client.ListTools();
				} finally {
					client.Release();
				}
			}`)
	}

	runtime.GC()
	var m4 runtime.MemStats
	runtime.ReadMemStats(&m4)
	obj4 := bridge.CountGoObjects()

	manualGrowth := int64(m4.HeapAlloc) - int64(m3.HeapAlloc)
	manualObjDelta := obj4 - obj3

	// Report comparison
	t.Logf("Comparison Results (%d iterations):", iterations)
	t.Logf("  Use() method:")
	t.Logf("    Memory growth:  %d bytes (%.2f MB)", useGrowth, float64(useGrowth)/1024/1024)
	t.Logf("    Object delta:   %+d", useObjDelta)
	t.Logf("  try-finally method:")
	t.Logf("    Memory growth:  %d bytes (%.2f MB)", manualGrowth, float64(manualGrowth)/1024/1024)
	t.Logf("    Object delta:   %+d", manualObjDelta)

	// Both should have similar cleanup characteristics
	assert.True(t, useObjDelta <= 10, "Use() should cleanup Go objects")
	assert.True(t, manualObjDelta <= 10, "try-finally should cleanup Go objects")
}

// ============================================================================
// Benchmarks
// ============================================================================

// BenchmarkUseVsManual compares performance of Use() vs manual cleanup
func BenchmarkUseVsManual(b *testing.B) {
	b.Run("Use", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v8.Call(v8.CallOptions{}, `
				function test() {
					return Use(MCP, "dsl", (client) => {
						return client.ListTools();
					});
				}`)
		}
	})

	b.Run("Manual", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v8.Call(v8.CallOptions{}, `
				function test() {
					const client = new MCP("dsl");
					try {
						return client.ListTools();
					} finally {
						client.Release();
					}
				}`)
		}
	})
}
