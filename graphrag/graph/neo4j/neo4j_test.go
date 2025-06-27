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

	if store.Enterprise() {
		t.Error("New store should not be enterprise by default")
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

// TestEnterprise tests Enterprise and SetEnterprise methods
func TestEnterprise(t *testing.T) {
	store := NewStore()

	// Should default to false
	if store.Enterprise() {
		t.Error("New store should not be enterprise by default")
	}

	// Test setting enterprise flag
	store.SetEnterprise(true)
	if !store.Enterprise() {
		t.Error("Store should be enterprise after setting flag to true")
	}

	// Test unsetting enterprise flag
	store.SetEnterprise(false)
	if store.Enterprise() {
		t.Error("Store should not be enterprise after setting flag to false")
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

	// Test concurrent Enterprise calls
	t.Run("ConcurrentEnterprise", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numOperations; j++ {
					_ = store.Enterprise()
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

	// Test concurrent SetEnterprise calls
	t.Run("ConcurrentSetEnterprise", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					store.SetEnterprise(id%2 == 0)
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

// BenchmarkEnterprise benchmarks Enterprise method
func BenchmarkEnterprise(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.Enterprise()
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

// BenchmarkSetEnterprise benchmarks SetEnterprise method
func BenchmarkSetEnterprise(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.SetEnterprise(i%2 == 0)
	}
}

// BenchmarkEnterpriseConcurrent benchmarks concurrent Enterprise calls
func BenchmarkEnterpriseConcurrent(b *testing.B) {
	store := NewStore()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.Enterprise()
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
