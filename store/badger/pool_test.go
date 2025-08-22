package badger

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestConnectionPool(t *testing.T) {
	// Use temporary directory for testing
	testPath := "./test_pool_db"
	defer os.RemoveAll(testPath)

	// Create first connection
	store1, err := New(testPath)
	if err != nil {
		t.Fatalf("Failed to create first store: %v", err)
	}

	// Create second connection to the same path
	store2, err := New(testPath)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}

	// Check connection info
	info := GetConnectionInfo()
	if len(info) != 1 {
		t.Errorf("Expected 1 connection in pool, got %d", len(info))
	}

	// Check reference count
	for path, refCount := range info {
		if refCount != 2 {
			t.Errorf("Expected reference count 2 for %s, got %d", path, refCount)
		}
	}

	// Test that both stores work
	err = store1.Set("test1", "value1", 0)
	if err != nil {
		t.Errorf("Failed to set value in store1: %v", err)
	}

	value, ok := store2.Get("test1")
	if !ok {
		t.Error("Failed to get value from store2")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// Close first store
	err = store1.Close()
	if err != nil {
		t.Errorf("Failed to close store1: %v", err)
	}

	// Check that connection is still alive (refCount = 1)
	info = GetConnectionInfo()
	if len(info) != 1 {
		t.Errorf("Expected 1 connection in pool after closing store1, got %d", len(info))
	}

	for path, refCount := range info {
		if refCount != 1 {
			t.Errorf("Expected reference count 1 for %s after closing store1, got %d", path, refCount)
		}
	}

	// Store2 should still work
	value, ok = store2.Get("test1")
	if !ok {
		t.Error("Store2 should still work after closing store1")
	}

	// Close second store
	err = store2.Close()
	if err != nil {
		t.Errorf("Failed to close store2: %v", err)
	}

	// Check that connection pool is empty
	info = GetConnectionInfo()
	if len(info) != 0 {
		t.Errorf("Expected empty connection pool after closing all stores, got %d connections", len(info))
	}
}

func TestConcurrentAccess(t *testing.T) {
	testPath := "./test_concurrent_db"
	defer os.RemoveAll(testPath)

	// Test concurrent access to the same database path
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			store, err := New(testPath)
			if err != nil {
				t.Errorf("Goroutine %d: Failed to create store: %v", id, err)
				return
			}
			defer store.Close()

			// Set and get some values
			key := fmt.Sprintf("key_%d", id)
			value := fmt.Sprintf("value_%d", id)

			err = store.Set(key, value, 0)
			if err != nil {
				t.Errorf("Goroutine %d: Failed to set value: %v", id, err)
				return
			}

			retrieved, ok := store.Get(key)
			if !ok {
				t.Errorf("Goroutine %d: Failed to get value", id)
				return
			}

			if retrieved != value {
				t.Errorf("Goroutine %d: Expected %s, got %v", id, value, retrieved)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify connection pool is empty after all connections are closed
	info := GetConnectionInfo()
	if len(info) != 0 {
		t.Errorf("Expected empty connection pool, got %d connections", len(info))
	}
}
