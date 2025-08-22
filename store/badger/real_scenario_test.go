package badger

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRealScenario tests the original problem scenario
func TestRealScenario(t *testing.T) {
	// Simulate the original path that was causing issues
	testPath := "./data/stores/oauth/client"
	defer os.RemoveAll("./data")

	// This should work without "Cannot acquire directory lock" error
	store1, err := New(testPath)
	if err != nil {
		t.Fatalf("Failed to create first store: %v", err)
	}

	// This would previously fail with "Cannot acquire directory lock"
	// but should now work with connection pooling
	store2, err := New(testPath)
	if err != nil {
		t.Fatalf("Failed to create second store (this would previously fail): %v", err)
	}

	// Test that both stores are actually the same instance
	if store1.path != store2.path {
		t.Errorf("Expected same path, got %s and %s", store1.path, store2.path)
	}

	// Test functionality
	err = store1.Set("oauth_client", map[string]interface{}{
		"client_id":     "test_client",
		"client_secret": "test_secret",
		"redirect_uri":  "http://localhost:8080/callback",
	}, 0)
	if err != nil {
		t.Errorf("Failed to set oauth client: %v", err)
	}

	// Read from second store
	client, ok := store2.Get("oauth_client")
	if !ok {
		t.Error("Failed to get oauth client from second store")
	}

	clientMap, ok := client.(map[string]interface{})
	if !ok {
		t.Error("OAuth client is not a map")
	} else {
		if clientMap["client_id"] != "test_client" {
			t.Errorf("Expected client_id 'test_client', got %v", clientMap["client_id"])
		}
	}

	// Check connection pool status
	info := GetConnectionInfo()
	t.Logf("Connection pool info: %+v", info)

	if len(info) != 1 {
		t.Errorf("Expected 1 connection in pool, got %d", len(info))
	}

	for path, refCount := range info {
		if refCount != 2 {
			t.Errorf("Expected reference count 2 for %s, got %d", path, refCount)
		}
		absTestPath, _ := filepath.Abs(testPath)
		if path != absTestPath {
			t.Errorf("Expected path %s, got %s", absTestPath, path)
		}
	}

	// Close stores
	err = store1.Close()
	if err != nil {
		t.Errorf("Failed to close store1: %v", err)
	}

	err = store2.Close()
	if err != nil {
		t.Errorf("Failed to close store2: %v", err)
	}

	// Verify pool is cleaned up
	info = GetConnectionInfo()
	if len(info) != 0 {
		t.Errorf("Expected empty pool after closing all connections, got %+v", info)
	}
}

// TestMultiplePathsInPool tests that different paths create different connections
func TestMultiplePathsInPool(t *testing.T) {
	defer os.RemoveAll("./test_multi")

	path1 := "./test_multi/db1"
	path2 := "./test_multi/db2"

	store1, err := New(path1)
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	store2, err := New(path2)
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	// Should have 2 different connections
	info := GetConnectionInfo()
	if len(info) != 2 {
		t.Errorf("Expected 2 connections in pool, got %d", len(info))
	}

	// Each should have refCount = 1
	for path, refCount := range info {
		if refCount != 1 {
			t.Errorf("Expected reference count 1 for %s, got %d", path, refCount)
		}
	}

	// Test that they're independent
	store1.Set("key", "value1", 0)
	store2.Set("key", "value2", 0)

	val1, _ := store1.Get("key")
	val2, _ := store2.Get("key")

	if val1 == val2 {
		t.Error("Expected different values from different databases")
	}

	// Clean up
	store1.Close()
	store2.Close()
}
