package qdrant

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestConfig holds test configuration
type TestConfig struct {
	Host     string
	Port     string
	APIKey   string
	Username string
	Password string
}

// TestEnvironment holds test environment data
type TestEnvironment struct {
	Store           *Store
	Config          types.VectorStoreConfig
	TestCollections []string
}

// getTestConfig returns test configuration from environment variables
func getTestConfig() *TestConfig {
	return &TestConfig{
		Host:     getEnvOrDefault("QDRANT_TEST_HOST", "localhost"),
		Port:     getEnvOrDefault("QDRANT_TEST_PORT", "6334"),
		APIKey:   os.Getenv("QDRANT_TEST_API_KEY"),
		Username: os.Getenv("QDRANT_TEST_USERNAME"),
		Password: os.Getenv("QDRANT_TEST_PASSWORD"),
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupTestEnvironment creates a clean test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	config := getTestConfig()

	store := NewStore()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: fmt.Sprintf("test_collection_%d", time.Now().UnixNano()),
		M:              16,
		EfConstruction: 100,
		Timeout:        30,
		ExtraParams: map[string]interface{}{
			"host": config.Host,
			"port": config.Port,
		},
	}

	if config.APIKey != "" {
		storeConfig.ExtraParams["api_key"] = config.APIKey
	}
	if config.Username != "" {
		storeConfig.ExtraParams["username"] = config.Username
	}
	if config.Password != "" {
		storeConfig.ExtraParams["password"] = config.Password
	}

	err := store.Connect(ctx, storeConfig)
	if err != nil {
		t.Skipf("Failed to connect to Qdrant server: %v", err)
	}

	env := &TestEnvironment{
		Store:           store,
		Config:          storeConfig,
		TestCollections: make([]string, 0),
	}

	return env
}

// cleanupTestEnvironment cleans up all test data and closes connections
func cleanupTestEnvironment(env *TestEnvironment, t *testing.T) {
	if env == nil {
		return
	}

	// Close the store connection
	if env.Store != nil {
		env.Store.Close()
	}
}

// withTestEnvironment is a helper function that sets up and cleans up test environment
func withTestEnvironment(t *testing.T, testFunc func(*TestEnvironment)) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	testFunc(env)
}

// =============================================================================
// Tests for qdrant.go methods (100% coverage)
// =============================================================================

// TestNewStore tests creating a new store instance
func TestNewStore(t *testing.T) {
	store := NewStore()

	// Verify store is not nil
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	// Verify initial state
	if store.client != nil {
		t.Error("New store should have nil client initially")
	}

	if store.conn != nil {
		t.Error("New store should have nil connection initially")
	}

	if store.connected {
		t.Error("New store should not be connected initially")
	}

	// Verify default config
	config := store.GetConfig()
	if config.Dimension != 0 {
		t.Errorf("Expected dimension 0, got %d", config.Dimension)
	}

	if config.CollectionName != "" {
		t.Errorf("Expected empty collection name, got %s", config.CollectionName)
	}
}

// TestGetClient tests GetClient method
func TestGetClient(t *testing.T) {
	t.Run("UnconnectedStore", func(t *testing.T) {
		store := NewStore()
		client := store.GetClient()
		if client != nil {
			t.Error("GetClient() should return nil for unconnected store")
		}
	})

	t.Run("ConnectedStore", func(t *testing.T) {
		withTestEnvironment(t, func(env *TestEnvironment) {
			client := env.Store.GetClient()
			if client == nil {
				t.Error("GetClient() should return non-nil client when connected")
			}
		})
	})
}

// TestGetConfig tests GetConfig method
func TestGetConfig(t *testing.T) {
	t.Run("UnconnectedStore", func(t *testing.T) {
		store := NewStore()
		config := store.GetConfig()

		// Should return zero values for unconnected store
		if config.Dimension != 0 {
			t.Errorf("Expected dimension 0, got %d", config.Dimension)
		}
		if config.CollectionName != "" {
			t.Errorf("Expected empty collection name, got %s", config.CollectionName)
		}
		if config.Distance != "" {
			t.Errorf("Expected empty distance, got %s", config.Distance)
		}
		if config.IndexType != "" {
			t.Errorf("Expected empty index type, got %s", config.IndexType)
		}
		if config.ExtraParams != nil {
			t.Errorf("Expected nil extra params, got %v", config.ExtraParams)
		}
	})

	t.Run("ConnectedStore", func(t *testing.T) {
		withTestEnvironment(t, func(env *TestEnvironment) {
			config := env.Store.GetConfig()

			// Should return the configured values
			if config.Dimension != env.Config.Dimension {
				t.Errorf("Expected dimension %d, got %d", env.Config.Dimension, config.Dimension)
			}
			if config.Distance != env.Config.Distance {
				t.Errorf("Expected distance %v, got %v", env.Config.Distance, config.Distance)
			}
			if config.IndexType != env.Config.IndexType {
				t.Errorf("Expected index type %v, got %v", env.Config.IndexType, config.IndexType)
			}
			if config.CollectionName != env.Config.CollectionName {
				t.Errorf("Expected collection name %s, got %s", env.Config.CollectionName, config.CollectionName)
			}
			if config.M != env.Config.M {
				t.Errorf("Expected M %d, got %d", env.Config.M, config.M)
			}
			if config.EfConstruction != env.Config.EfConstruction {
				t.Errorf("Expected EfConstruction %d, got %d", env.Config.EfConstruction, config.EfConstruction)
			}
			if config.Timeout != env.Config.Timeout {
				t.Errorf("Expected Timeout %d, got %d", env.Config.Timeout, config.Timeout)
			}

			// Check ExtraParams
			if config.ExtraParams == nil {
				t.Error("Expected non-nil extra params")
			} else {
				if host, ok := config.ExtraParams["host"].(string); !ok || host == "" {
					t.Error("Expected host in extra params")
				}
				if port, ok := config.ExtraParams["port"].(string); !ok || port == "" {
					t.Error("Expected port in extra params")
				}
			}
		})
	})
}

// TestStoreConcurrency tests concurrent access to Store methods
func TestStoreConcurrency(t *testing.T) {
	store := NewStore()

	const numGoroutines = 50
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	// Test concurrent GetClient calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				client := store.GetClient()
				_ = client // Use the result to prevent optimization
			}
		}()
	}

	// Test concurrent GetConfig calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				config := store.GetConfig()
				_ = config // Use the result to prevent optimization
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines*2; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations to complete")
		}
	}
}

// =============================================================================
// Performance Benchmarks
// =============================================================================

// BenchmarkNewStore benchmarks creating new store instances
func BenchmarkNewStore(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := NewStore()
		_ = store
	}
}

// BenchmarkGetClient benchmarks GetClient method
func BenchmarkGetClient(b *testing.B) {
	store := NewStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := store.GetClient()
		_ = client
	}
}

// BenchmarkGetConfig benchmarks GetConfig method
func BenchmarkGetConfig(b *testing.B) {
	store := NewStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := store.GetConfig()
		_ = config
	}
}

// BenchmarkGetClientConcurrent benchmarks concurrent GetClient calls
func BenchmarkGetClientConcurrent(b *testing.B) {
	store := NewStore()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := store.GetClient()
			_ = client
		}
	})
}

// BenchmarkGetConfigConcurrent benchmarks concurrent GetConfig calls
func BenchmarkGetConfigConcurrent(b *testing.B) {
	store := NewStore()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			config := store.GetConfig()
			_ = config
		}
	})
}

// BenchmarkStoreOperations benchmarks all store operations together
func BenchmarkStoreOperations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := NewStore()
		client := store.GetClient()
		config := store.GetConfig()
		_, _ = client, config
	}
}

// =============================================================================
// Memory Leak Tests
// =============================================================================

// TestMemoryLeakDetection tests for memory leaks in store operations
func TestMemoryLeakDetection(t *testing.T) {
	// Skip this test if not in verbose mode or specific flag is not set
	if !testing.Verbose() && os.Getenv("RUN_MEMORY_TESTS") == "" {
		t.Skip("Skipping memory leak test (set RUN_MEMORY_TESTS=1 or use -v to run)")
	}

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform operations that might leak memory
	for i := 0; i < 10000; i++ {
		store := NewStore()

		// Test GetClient multiple times
		for j := 0; j < 10; j++ {
			client := store.GetClient()
			_ = client
		}

		// Test GetConfig multiple times
		for j := 0; j < 10; j++ {
			config := store.GetConfig()
			_ = config
		}

		// Force garbage collection periodically
		if i%1000 == 0 {
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

	t.Logf("Memory stats:")
	t.Logf("  Alloc growth: %d bytes", memGrowth)
	t.Logf("  Heap growth: %d bytes", heapGrowth)
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)

	// Allow some memory growth, but not excessive
	maxAllowedGrowth := int64(1024 * 1024) // 1MB threshold
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (threshold: %d bytes)", memGrowth, maxAllowedGrowth)
	}

	if heapGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: heap grew by %d bytes (threshold: %d bytes)", heapGrowth, maxAllowedGrowth)
	}
}

// TestMemoryLeakWithConnection tests memory leaks with actual connections
func TestMemoryLeakWithConnection(t *testing.T) {
	// Skip this test if not in verbose mode or specific flag is not set
	if !testing.Verbose() && os.Getenv("RUN_MEMORY_TESTS") == "" {
		t.Skip("Skipping memory leak test (set RUN_MEMORY_TESTS=1 or use -v to run)")
	}

	config := getTestConfig()
	storeConfig := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: "test_memory_leak",
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

	// Perform operations that might leak memory with connections
	for i := 0; i < 100; i++ {
		func() {
			store := NewStore()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := store.Connect(ctx, storeConfig)
			if err != nil {
				// Skip if connection fails, but don't fail the test
				return
			}

			// Test methods on connected store
			client := store.GetClient()
			config := store.GetConfig()
			_, _ = client, config

			// Clean up
			store.Close()
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

	t.Logf("Memory stats with connections:")
	t.Logf("  Alloc growth: %d bytes", memGrowth)
	t.Logf("  Heap growth: %d bytes", heapGrowth)
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)

	// Allow more memory growth for connection tests
	maxAllowedGrowth := int64(5 * 1024 * 1024) // 5MB threshold
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (threshold: %d bytes)", memGrowth, maxAllowedGrowth)
	}

	if heapGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: heap grew by %d bytes (threshold: %d bytes)", heapGrowth, maxAllowedGrowth)
	}
}
