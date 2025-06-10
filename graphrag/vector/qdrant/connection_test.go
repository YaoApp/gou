package qdrant

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// =============================================================================
// Tests for connection.go methods (100% coverage)
// =============================================================================

// TestConnect tests the Connect method
func TestConnect(t *testing.T) {
	t.Run("SuccessfulConnection", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_connection",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		// Verify connection state
		if !store.IsConnected() {
			t.Error("Store should be connected after successful Connect()")
		}

		client := store.GetClient()
		if client == nil {
			t.Error("Client should not be nil after successful connection")
		}

		// Verify config is stored
		storedConfig := store.GetConfig()
		if storedConfig.Dimension != storeConfig.Dimension {
			t.Errorf("Expected dimension %d, got %d", storeConfig.Dimension, storedConfig.Dimension)
		}
	})

	t.Run("ConnectionWithAPIKey", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_connection_apikey",
			ExtraParams: map[string]interface{}{
				"host":    config.Host,
				"port":    config.Port,
				"api_key": "test-api-key",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		// This might fail if API key is invalid, but we test the code path
		if err == nil {
			if !store.IsConnected() {
				t.Error("Store should be connected after successful Connect() with API key")
			}
		} else {
			t.Logf("Connection with API key failed as expected: %v", err)
		}
	})

	t.Run("ConnectionWithStringPort", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_connection_string_port",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port, // String port
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		if !store.IsConnected() {
			t.Error("Store should be connected after successful Connect() with string port")
		}
	})

	t.Run("ConnectionWithIntPort", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()
		defer store.Close()

		port := 6334
		if config.Port != "" {
			if p, err := strconv.Atoi(config.Port); err == nil {
				port = p
			}
		}

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_connection_int_port",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": port, // Int port
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		if !store.IsConnected() {
			t.Error("Store should be connected after successful Connect() with int port")
		}
	})

	t.Run("ConnectionWithDefaults", func(t *testing.T) {
		store := NewStore()
		defer store.Close()

		// Test with no ExtraParams (should use defaults)
		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_connection_defaults",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		// This will likely fail unless Qdrant is running on localhost:6334
		if err == nil {
			if !store.IsConnected() {
				t.Error("Store should be connected after successful Connect() with defaults")
			}
		} else {
			t.Logf("Connection with defaults failed as expected: %v", err)
		}
	})

	t.Run("AlreadyConnected", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_already_connected",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// First connection
		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		// Second connection (should return immediately)
		err = store.Connect(ctx, storeConfig)
		if err != nil {
			t.Errorf("Second Connect() should not fail: %v", err)
		}

		if !store.IsConnected() {
			t.Error("Store should remain connected after second Connect()")
		}
	})

	t.Run("ConnectionFailure", func(t *testing.T) {
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_connection_failure",
			ExtraParams: map[string]interface{}{
				"host": "invalid-host-that-does-not-exist",
				"port": "6334",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err == nil {
			t.Error("Expected connection to fail with invalid host")
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after failed Connect()")
		}

		client := store.GetClient()
		if client != nil {
			t.Error("Client should be nil after failed connection")
		}
	})

	t.Run("InvalidPortString", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_invalid_port",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": "invalid-port",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Should fall back to default port 6334
		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		if !store.IsConnected() {
			t.Error("Store should be connected after Connect() with invalid port string")
		}
	})
}

// TestDisconnect tests the Disconnect method
func TestDisconnect(t *testing.T) {
	t.Run("SuccessfulDisconnect", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_disconnect",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Connect first
		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		if !store.IsConnected() {
			t.Fatal("Store should be connected before testing disconnect")
		}

		// Disconnect
		err = store.Disconnect(ctx)
		if err != nil {
			t.Errorf("Disconnect() failed: %v", err)
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after Disconnect()")
		}

		client := store.GetClient()
		if client != nil {
			t.Error("Client should be nil after Disconnect()")
		}
	})

	t.Run("DisconnectNotConnected", func(t *testing.T) {
		store := NewStore()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Disconnect without connecting first
		err := store.Disconnect(ctx)
		if err != nil {
			t.Errorf("Disconnect() on unconnected store should not fail: %v", err)
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after Disconnect() on unconnected store")
		}
	})

	t.Run("DoubleDisconnect", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_double_disconnect",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Connect first
		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		// First disconnect
		err = store.Disconnect(ctx)
		if err != nil {
			t.Errorf("First Disconnect() failed: %v", err)
		}

		// Second disconnect (should not fail)
		err = store.Disconnect(ctx)
		if err != nil {
			t.Errorf("Second Disconnect() should not fail: %v", err)
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after double Disconnect()")
		}
	})
}

// TestIsConnected tests the IsConnected method
func TestIsConnected(t *testing.T) {
	t.Run("InitialState", func(t *testing.T) {
		store := NewStore()
		if store.IsConnected() {
			t.Error("New store should not be connected initially")
		}
	})

	t.Run("AfterConnect", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_is_connected",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		if !store.IsConnected() {
			t.Error("Store should be connected after successful Connect()")
		}
	})

	t.Run("AfterDisconnect", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_is_connected_after_disconnect",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		err = store.Disconnect(ctx)
		if err != nil {
			t.Fatalf("Failed to disconnect: %v", err)
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after Disconnect()")
		}
	})

	t.Run("AfterFailedConnect", func(t *testing.T) {
		store := NewStore()
		defer store.Close()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_is_connected_failed",
			ExtraParams: map[string]interface{}{
				"host": "invalid-host",
				"port": "6334",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err == nil {
			t.Skip("Expected connection to fail, but it succeeded")
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after failed Connect()")
		}
	})
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	t.Run("CloseConnectedStore", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_close",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		if !store.IsConnected() {
			t.Fatal("Store should be connected before testing Close()")
		}

		err = store.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after Close()")
		}
	})

	t.Run("CloseUnconnectedStore", func(t *testing.T) {
		store := NewStore()

		err := store.Close()
		if err != nil {
			t.Errorf("Close() on unconnected store should not fail: %v", err)
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after Close() on unconnected store")
		}
	})

	t.Run("DoubleClose", func(t *testing.T) {
		config := getTestConfig()
		store := NewStore()

		storeConfig := types.VectorStoreConfig{
			Dimension:      128,
			Distance:       types.DistanceCosine,
			IndexType:      types.IndexTypeHNSW,
			CollectionName: "test_double_close",
			ExtraParams: map[string]interface{}{
				"host": config.Host,
				"port": config.Port,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		// First close
		err = store.Close()
		if err != nil {
			t.Errorf("First Close() failed: %v", err)
		}

		// Second close (should not fail)
		err = store.Close()
		if err != nil {
			t.Errorf("Second Close() should not fail: %v", err)
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after double Close()")
		}
	})
}

// TestConnectionConcurrency tests concurrent connection operations
func TestConnectionConcurrency(t *testing.T) {
	config := getTestConfig()

	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "test_concurrency",
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	t.Run("ConcurrentConnect", func(t *testing.T) {
		store := NewStore()
		defer store.Close()

		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				err := store.Connect(ctx, storeConfig)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Check if any errors occurred
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Logf("Concurrent connect error: %v", err)
				errorCount++
			}
		}

		// At least one connection should succeed
		if store.IsConnected() {
			t.Logf("Store connected successfully with %d errors out of %d attempts", errorCount, numGoroutines)
		} else if errorCount == numGoroutines {
			t.Skip("All connection attempts failed - likely server not available")
		} else {
			t.Error("Store should be connected after concurrent Connect() attempts")
		}
	})

	t.Run("ConcurrentDisconnect", func(t *testing.T) {
		store := NewStore()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := store.Disconnect(ctx)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Check if any errors occurred
		for err := range errors {
			if err != nil {
				t.Errorf("Concurrent disconnect error: %v", err)
			}
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after concurrent Disconnect()")
		}
	})

	t.Run("ConcurrentIsConnected", func(t *testing.T) {
		store := NewStore()
		defer store.Close()

		const numGoroutines = 50
		const numOperations = 100
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					connected := store.IsConnected()
					_ = connected // Use the result to prevent optimization
				}
			}()
		}

		// Wait for all goroutines to complete with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent IsConnected() operations to complete")
		}
	})
}

// =============================================================================
// Performance Benchmarks
// =============================================================================

// BenchmarkConnect benchmarks the Connect method
func BenchmarkConnect(b *testing.B) {
	config := getTestConfig()
	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "bench_connect",
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := NewStore()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			b.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		store.Close()
		cancel()
	}
}

// BenchmarkDisconnect benchmarks the Disconnect method
func BenchmarkDisconnect(b *testing.B) {
	config := getTestConfig()
	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "bench_disconnect",
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create and connect for each iteration
		store := NewStore()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := store.Connect(ctx, storeConfig)
		cancel()
		if err != nil {
			b.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		// Benchmark the disconnect operation
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		err = store.Disconnect(ctx)
		cancel()
		if err != nil {
			b.Errorf("Disconnect failed: %v", err)
		}
	}
}

// BenchmarkIsConnected benchmarks the IsConnected method
func BenchmarkIsConnected(b *testing.B) {
	store := NewStore()
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connected := store.IsConnected()
		_ = connected
	}
}

// BenchmarkClose benchmarks the Close method
func BenchmarkClose(b *testing.B) {
	config := getTestConfig()
	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "bench_close",
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create and connect for each iteration
		store := NewStore()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := store.Connect(ctx, storeConfig)
		cancel()
		if err != nil {
			b.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		// Benchmark the close operation
		err = store.Close()
		if err != nil {
			b.Errorf("Close failed: %v", err)
		}
	}
}

// BenchmarkConnectionCycle benchmarks full connect-disconnect cycle
func BenchmarkConnectionCycle(b *testing.B) {
	config := getTestConfig()
	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "bench_cycle",
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := NewStore()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := store.Connect(ctx, storeConfig)
		if err != nil {
			cancel()
			b.Skipf("Failed to connect to Qdrant server: %v", err)
		}

		_ = store.IsConnected()

		err = store.Disconnect(ctx)
		if err != nil {
			cancel()
			b.Errorf("Disconnect failed: %v", err)
		}

		cancel()
	}
}

// =============================================================================
// Memory Leak Tests
// =============================================================================

// TestConnectionMemoryLeakDetection tests for memory leaks in connection operations
func TestConnectionMemoryLeakDetection(t *testing.T) {
	// Skip this test if not in verbose mode or specific flag is not set
	if !testing.Verbose() && os.Getenv("RUN_MEMORY_TESTS") == "" {
		t.Skip("Skipping memory leak test (set RUN_MEMORY_TESTS=1 or use -v to run)")
	}

	config := getTestConfig()
	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "test_connection_memory_leak",
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform operations that might leak memory
	for i := 0; i < 100; i++ {
		func() {
			store := NewStore()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := store.Connect(ctx, storeConfig)
			if err != nil {
				// Skip if connection fails, but don't fail the test
				return
			}

			// Test IsConnected multiple times
			for j := 0; j < 10; j++ {
				connected := store.IsConnected()
				_ = connected
			}

			// Disconnect
			err = store.Disconnect(ctx)
			if err != nil {
				t.Logf("Failed to disconnect in memory leak test: %v", err)
			}

			// Test IsConnected after disconnect
			for j := 0; j < 10; j++ {
				connected := store.IsConnected()
				_ = connected
			}

			// Close
			err = store.Close()
			if err != nil {
				t.Logf("Failed to close in memory leak test: %v", err)
			}
		}()

		// Force garbage collection periodically
		if i%10 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check memory growth (handle potential underflow)
	var memGrowth, heapGrowth int64
	if m2.Alloc >= m1.Alloc {
		memGrowth = int64(m2.Alloc - m1.Alloc)
	} else {
		memGrowth = -int64(m1.Alloc - m2.Alloc)
	}
	if m2.HeapAlloc >= m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = -int64(m1.HeapAlloc - m2.HeapAlloc)
	}

	t.Logf("Connection memory stats:")
	t.Logf("  Alloc growth: %d bytes", memGrowth)
	t.Logf("  Heap growth: %d bytes", heapGrowth)
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)

	// Allow some memory growth for connection operations
	maxAllowedGrowth := int64(10 * 1024 * 1024) // 10MB threshold
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (threshold: %d bytes)", memGrowth, maxAllowedGrowth)
	}

	if heapGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: heap grew by %d bytes (threshold: %d bytes)", heapGrowth, maxAllowedGrowth)
	}
}

// TestConcurrentConnectionMemoryLeak tests memory leaks with concurrent operations
func TestConcurrentConnectionMemoryLeak(t *testing.T) {
	// Skip this test if not in verbose mode or specific flag is not set
	if !testing.Verbose() && os.Getenv("RUN_MEMORY_TESTS") == "" {
		t.Skip("Skipping memory leak test (set RUN_MEMORY_TESTS=1 or use -v to run)")
	}

	config := getTestConfig()
	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "test_concurrent_memory_leak",
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform concurrent operations that might leak memory
	const numIterations = 50
	const numGoroutines = 5

	for i := 0; i < numIterations; i++ {
		var wg sync.WaitGroup

		for j := 0; j < numGoroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				store := NewStore()

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				err := store.Connect(ctx, storeConfig)
				if err != nil {
					// Skip if connection fails
					return
				}

				// Test concurrent IsConnected calls
				for k := 0; k < 5; k++ {
					connected := store.IsConnected()
					_ = connected
				}

				store.Close()
			}()
		}

		wg.Wait()

		// Force garbage collection periodically
		if i%10 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check memory growth (handle potential underflow)
	var memGrowth, heapGrowth int64
	if m2.Alloc >= m1.Alloc {
		memGrowth = int64(m2.Alloc - m1.Alloc)
	} else {
		memGrowth = -int64(m1.Alloc - m2.Alloc)
	}
	if m2.HeapAlloc >= m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = -int64(m1.HeapAlloc - m2.HeapAlloc)
	}

	t.Logf("Concurrent connection memory stats:")
	t.Logf("  Alloc growth: %d bytes", memGrowth)
	t.Logf("  Heap growth: %d bytes", heapGrowth)
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)

	// Allow more memory growth for concurrent operations
	maxAllowedGrowth := int64(15 * 1024 * 1024) // 15MB threshold
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (threshold: %d bytes)", memGrowth, maxAllowedGrowth)
	}

	if heapGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: heap grew by %d bytes (threshold: %d bytes)", heapGrowth, maxAllowedGrowth)
	}
}
