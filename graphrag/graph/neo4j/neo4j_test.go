package neo4j

import (
	"runtime"
	"testing"
	"time"
)

// Note: Connection-related test helpers will be added when connection.go is implemented

// =============================================================================
// Tests for neo4j.go methods
// =============================================================================

// TestNewStore tests creating a new store instance
func TestNewStore(t *testing.T) {
	store := NewStore()

	// Verify store is not nil
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	// Verify initial state
	if store.connected {
		t.Error("New store should not be connected initially")
	}

	if store.UseSeparateDatabase() {
		t.Error("New store should not use separate database by default")
	}

	// Verify default config
	config := store.GetConfig()
	if config.StoreType != "" {
		t.Errorf("Expected empty store type, got %s", config.StoreType)
	}

	if config.DatabaseURL != "" {
		t.Errorf("Expected empty database URL, got %s", config.DatabaseURL)
	}
}

// TestUseSeparateDatabase tests UseSeparateDatabase and SetUseSeparateDatabase methods
func TestUseSeparateDatabase(t *testing.T) {
	store := NewStore()

	// Should default to false
	if store.UseSeparateDatabase() {
		t.Error("New store should not use separate database by default")
	}

	// Test setting separate database flag
	store.SetUseSeparateDatabase(true)
	if !store.UseSeparateDatabase() {
		t.Error("Store should use separate database after setting flag to true")
	}

	// Test unsetting separate database flag
	store.SetUseSeparateDatabase(false)
	if store.UseSeparateDatabase() {
		t.Error("Store should not use separate database after setting flag to false")
	}
}

// TestGetConfig tests GetConfig method
func TestGetConfig(t *testing.T) {
	store := NewStore()
	config := store.GetConfig()

	// Should return zero values for unconnected store
	if config.StoreType != "" {
		t.Errorf("Expected empty store type, got %s", config.StoreType)
	}
	if config.DatabaseURL != "" {
		t.Errorf("Expected empty database URL, got %s", config.DatabaseURL)
	}
	if config.DefaultGraphName != "" {
		t.Errorf("Expected empty default graph name, got %s", config.DefaultGraphName)
	}
	if config.DriverConfig != nil {
		t.Errorf("Expected nil driver config, got %v", config.DriverConfig)
	}
}

// TestStoreConcurrency tests concurrent access to store methods
func TestStoreConcurrency(t *testing.T) {
	store := NewStore()

	numGoroutines := 10
	numOperations := 100

	// Test concurrent UseSeparateDatabase calls
	t.Run("ConcurrentUseSeparateDatabase", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numOperations; j++ {
					_ = store.UseSeparateDatabase()
					runtime.Gosched()
				}
			}()
		}
	})

	// Test concurrent GetConfig calls
	t.Run("ConcurrentGetConfig", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numOperations; j++ {
					_ = store.GetConfig()
					runtime.Gosched()
				}
			}()
		}
	})

	// Test concurrent SetUseSeparateDatabase calls
	t.Run("ConcurrentSetUseSeparateDatabase", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					store.SetUseSeparateDatabase(id%2 == 0)
					runtime.Gosched()
				}
			}(i)
		}
	})

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)
}

// TestGetDriver tests GetDriver method
func TestGetDriver(t *testing.T) {
	store := NewStore()

	// Should return nil when not connected
	driver := store.GetDriver()
	if driver != nil {
		t.Error("GetDriver() should return nil for unconnected store")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

// BenchmarkNewStore benchmarks store creation
func BenchmarkNewStore(b *testing.B) {
	for i := 0; i < b.N; i++ {
		store := NewStore()
		_ = store
	}
}

// BenchmarkUseSeparateDatabase benchmarks UseSeparateDatabase method
func BenchmarkUseSeparateDatabase(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.UseSeparateDatabase()
	}
}

// BenchmarkGetConfig benchmarks GetConfig method
func BenchmarkGetConfig(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.GetConfig()
	}
}

// BenchmarkSetUseSeparateDatabase benchmarks SetUseSeparateDatabase method
func BenchmarkSetUseSeparateDatabase(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.SetUseSeparateDatabase(i%2 == 0)
	}
}

// BenchmarkUseSeparateDatabaseConcurrent benchmarks concurrent UseSeparateDatabase calls
func BenchmarkUseSeparateDatabaseConcurrent(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.UseSeparateDatabase()
		}
	})
}

// BenchmarkGetConfigConcurrent benchmarks concurrent GetConfig calls
func BenchmarkGetConfigConcurrent(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.GetConfig()
		}
	})
}

// TODO: TestEnterpriseFeatures will be added when connection.go is implemented
