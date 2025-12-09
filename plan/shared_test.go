package plan

import (
	"sync"
	"testing"
	"time"
)

// =============================================================================
// MemorySharedSpace Tests
// =============================================================================

func TestNewMemorySharedSpace(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		if shared == nil {
			t.Fatal("NewMemorySharedSpace returned nil")
		}

		if shared.data == nil {
			t.Error("data map is nil")
		}

		if shared.subscribers == nil {
			t.Error("subscribers map is nil")
		}
	})
}

func TestMemorySharedSpace_SetAndGet(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Test setting and getting a string value
		err := shared.Set("key1", "value1")
		if err != nil {
			t.Errorf("Failed to set key1: %v", err)
		}

		val, err := shared.Get("key1")
		if err != nil {
			t.Errorf("Failed to get key1: %v", err)
		}
		if val != "value1" {
			t.Errorf("Expected 'value1', got %v", val)
		}

		// Test setting and getting an int value
		err = shared.Set("key2", 42)
		if err != nil {
			t.Errorf("Failed to set key2: %v", err)
		}

		val, err = shared.Get("key2")
		if err != nil {
			t.Errorf("Failed to get key2: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}

		// Test setting and getting a complex value
		complexVal := map[string]interface{}{
			"nested": "data",
			"count":  100,
		}
		err = shared.Set("key3", complexVal)
		if err != nil {
			t.Errorf("Failed to set key3: %v", err)
		}

		val, err = shared.Get("key3")
		if err != nil {
			t.Errorf("Failed to get key3: %v", err)
		}
		if valMap, ok := val.(map[string]interface{}); !ok {
			t.Errorf("Expected map, got %T", val)
		} else if valMap["nested"] != "data" {
			t.Errorf("Expected nested='data', got %v", valMap["nested"])
		}

		// Test overwriting existing value
		err = shared.Set("key1", "new_value")
		if err != nil {
			t.Errorf("Failed to overwrite key1: %v", err)
		}

		val, err = shared.Get("key1")
		if err != nil {
			t.Errorf("Failed to get overwritten key1: %v", err)
		}
		if val != "new_value" {
			t.Errorf("Expected 'new_value', got %v", val)
		}
	})
}

func TestMemorySharedSpace_GetNonExistent(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Test getting a non-existent key
		val, err := shared.Get("non_existent")
		if err == nil {
			t.Error("Expected error for non-existent key, got nil")
		}
		if val != nil {
			t.Errorf("Expected nil value for non-existent key, got %v", val)
		}
	})
}

func TestMemorySharedSpace_Delete(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set a value
		err := shared.Set("key1", "value1")
		if err != nil {
			t.Errorf("Failed to set key1: %v", err)
		}

		// Verify it exists
		val, err := shared.Get("key1")
		if err != nil {
			t.Errorf("Failed to get key1: %v", err)
		}
		if val != "value1" {
			t.Errorf("Expected 'value1', got %v", val)
		}

		// Delete the key
		err = shared.Delete("key1")
		if err != nil {
			t.Errorf("Failed to delete key1: %v", err)
		}

		// Verify it's gone
		val, err = shared.Get("key1")
		if err == nil {
			t.Error("Expected error after deletion, got nil")
		}
		if val != nil {
			t.Errorf("Expected nil after deletion, got %v", val)
		}

		// Delete non-existent key should not error
		err = shared.Delete("non_existent")
		if err != nil {
			t.Errorf("Delete non-existent key should not error: %v", err)
		}
	})
}

func TestMemorySharedSpace_Clear(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set multiple values
		shared.Set("key1", "value1")
		shared.Set("key2", "value2")
		shared.Set("key3", "value3")

		// Clear all
		err := shared.Clear()
		if err != nil {
			t.Errorf("Failed to clear: %v", err)
		}

		// Verify all keys are gone
		for _, key := range []string{"key1", "key2", "key3"} {
			val, err := shared.Get(key)
			if err == nil {
				t.Errorf("Expected error for %s after clear, got nil", key)
			}
			if val != nil {
				t.Errorf("Expected nil for %s after clear, got %v", key, val)
			}
		}
	})
}

func TestMemorySharedSpace_ClearNotify(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set up subscriber to track notifications
		notifications := make(chan string, 10)

		shared.Subscribe("key1", func(key string, value interface{}) {
			if value == nil {
				notifications <- key + "_cleared"
			}
		})
		shared.Subscribe("key2", func(key string, value interface{}) {
			if value == nil {
				notifications <- key + "_cleared"
			}
		})

		// Set values
		shared.Set("key1", "value1")
		shared.Set("key2", "value2")

		// Drain any set notifications
		time.Sleep(10 * time.Millisecond)
		for len(notifications) > 0 {
			<-notifications
		}

		// Clear with notification
		err := shared.ClearNotify()
		if err != nil {
			t.Errorf("Failed to clear with notify: %v", err)
		}

		// Verify notifications were sent
		receivedNotifications := make(map[string]bool)
		timeout := time.After(100 * time.Millisecond)
	collectLoop:
		for {
			select {
			case notif := <-notifications:
				receivedNotifications[notif] = true
				if len(receivedNotifications) >= 2 {
					break collectLoop
				}
			case <-timeout:
				break collectLoop
			}
		}

		if !receivedNotifications["key1_cleared"] {
			t.Error("Did not receive clear notification for key1")
		}
		if !receivedNotifications["key2_cleared"] {
			t.Error("Did not receive clear notification for key2")
		}

		// Verify all keys are gone
		for _, key := range []string{"key1", "key2"} {
			val, err := shared.Get(key)
			if err == nil {
				t.Errorf("Expected error for %s after clear, got nil", key)
			}
			if val != nil {
				t.Errorf("Expected nil for %s after clear, got %v", key, val)
			}
		}

		// Cleanup
		shared.Unsubscribe("key1")
		shared.Unsubscribe("key2")
	})
}

func TestMemorySharedSpace_Subscribe(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		notifications := make(chan interface{}, 10)

		// Subscribe to key changes
		err := shared.Subscribe("test-key", func(key string, value interface{}) {
			notifications <- value
		})
		if err != nil {
			t.Errorf("Failed to subscribe: %v", err)
		}

		// Set value should trigger notification
		shared.Set("test-key", "first_value")

		select {
		case val := <-notifications:
			if val != "first_value" {
				t.Errorf("Expected 'first_value', got %v", val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive notification for set")
		}

		// Update value should trigger notification
		shared.Set("test-key", "second_value")

		select {
		case val := <-notifications:
			if val != "second_value" {
				t.Errorf("Expected 'second_value', got %v", val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive notification for update")
		}

		// Delete should trigger notification with nil
		shared.Delete("test-key")

		select {
		case val := <-notifications:
			if val != nil {
				t.Errorf("Expected nil for delete notification, got %v", val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive notification for delete")
		}

		// Cleanup
		shared.Unsubscribe("test-key")
	})
}

func TestMemorySharedSpace_MultipleSubscribers(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		notifications1 := make(chan interface{}, 5)
		notifications2 := make(chan interface{}, 5)

		// Subscribe multiple callbacks to same key
		shared.Subscribe("multi-key", func(key string, value interface{}) {
			notifications1 <- value
		})
		shared.Subscribe("multi-key", func(key string, value interface{}) {
			notifications2 <- value
		})

		// Set value
		shared.Set("multi-key", "test_value")

		// Both subscribers should receive notification
		select {
		case val := <-notifications1:
			if val != "test_value" {
				t.Errorf("Subscriber 1: expected 'test_value', got %v", val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Subscriber 1 did not receive notification")
		}

		select {
		case val := <-notifications2:
			if val != "test_value" {
				t.Errorf("Subscriber 2: expected 'test_value', got %v", val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Subscriber 2 did not receive notification")
		}

		// Cleanup
		shared.Unsubscribe("multi-key")
	})
}

func TestMemorySharedSpace_Unsubscribe(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		notifications := make(chan interface{}, 5)

		// Subscribe
		shared.Subscribe("unsub-key", func(key string, value interface{}) {
			notifications <- value
		})

		// Set value - should notify
		shared.Set("unsub-key", "value1")

		select {
		case <-notifications:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive notification before unsubscribe")
		}

		// Unsubscribe
		err := shared.Unsubscribe("unsub-key")
		if err != nil {
			t.Errorf("Failed to unsubscribe: %v", err)
		}

		// Set value again - should NOT notify
		shared.Set("unsub-key", "value2")

		select {
		case val := <-notifications:
			t.Errorf("Received unexpected notification after unsubscribe: %v", val)
		case <-time.After(50 * time.Millisecond):
			// Expected - no notification
		}

		// Unsubscribe non-existent key should not error
		err = shared.Unsubscribe("non_existent")
		if err != nil {
			t.Errorf("Unsubscribe non-existent key should not error: %v", err)
		}
	})
}

func TestMemorySharedSpace_Snapshot(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set multiple values
		shared.Set("key1", "value1")
		shared.Set("key2", 42)
		shared.Set("key3", map[string]interface{}{"nested": "data"})

		// Take snapshot
		snapshot := shared.Snapshot()

		// Verify snapshot contains all values
		if snapshot["key1"] != "value1" {
			t.Errorf("Snapshot key1: expected 'value1', got %v", snapshot["key1"])
		}
		if snapshot["key2"] != 42 {
			t.Errorf("Snapshot key2: expected 42, got %v", snapshot["key2"])
		}
		if nestedMap, ok := snapshot["key3"].(map[string]interface{}); !ok {
			t.Errorf("Snapshot key3: expected map, got %T", snapshot["key3"])
		} else if nestedMap["nested"] != "data" {
			t.Errorf("Snapshot key3.nested: expected 'data', got %v", nestedMap["nested"])
		}

		// Verify snapshot is a copy (modifying snapshot doesn't affect original)
		snapshot["key1"] = "modified"
		val, _ := shared.Get("key1")
		if val != "value1" {
			t.Error("Modifying snapshot affected original data")
		}

		// Verify original change doesn't affect snapshot
		shared.Set("key1", "new_value")
		if snapshot["key1"] != "modified" {
			t.Error("Modifying original affected snapshot")
		}
	})
}

func TestMemorySharedSpace_SnapshotEmpty(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Snapshot of empty space
		snapshot := shared.Snapshot()

		if snapshot == nil {
			t.Error("Snapshot should not be nil for empty space")
		}
		if len(snapshot) != 0 {
			t.Errorf("Snapshot should be empty, got %d items", len(snapshot))
		}
	})
}

func TestMemorySharedSpace_Restore(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set initial value
		shared.Set("existing", "original")

		// Restore from snapshot
		snapshot := map[string]interface{}{
			"key1": "restored_value1",
			"key2": 123,
			"key3": []string{"a", "b", "c"},
		}

		err := shared.Restore(snapshot)
		if err != nil {
			t.Errorf("Failed to restore: %v", err)
		}

		// Verify restored values
		val, err := shared.Get("key1")
		if err != nil {
			t.Errorf("Failed to get key1: %v", err)
		}
		if val != "restored_value1" {
			t.Errorf("key1: expected 'restored_value1', got %v", val)
		}

		val, err = shared.Get("key2")
		if err != nil {
			t.Errorf("Failed to get key2: %v", err)
		}
		if val != 123 {
			t.Errorf("key2: expected 123, got %v", val)
		}

		val, err = shared.Get("key3")
		if err != nil {
			t.Errorf("Failed to get key3: %v", err)
		}
		if arr, ok := val.([]string); !ok {
			t.Errorf("key3: expected []string, got %T", val)
		} else if len(arr) != 3 || arr[0] != "a" {
			t.Errorf("key3: unexpected value %v", arr)
		}

		// Verify existing value is preserved
		val, err = shared.Get("existing")
		if err != nil {
			t.Errorf("Failed to get existing: %v", err)
		}
		if val != "original" {
			t.Errorf("existing: expected 'original', got %v", val)
		}
	})
}

func TestMemorySharedSpace_RestoreNil(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set initial value
		shared.Set("key1", "value1")

		// Restore nil should not error and not affect existing data
		err := shared.Restore(nil)
		if err != nil {
			t.Errorf("Restore nil should not error: %v", err)
		}

		// Verify existing value is preserved
		val, err := shared.Get("key1")
		if err != nil {
			t.Errorf("Failed to get key1: %v", err)
		}
		if val != "value1" {
			t.Errorf("key1: expected 'value1', got %v", val)
		}
	})
}

func TestMemorySharedSpace_RestoreNotifiesSubscribers(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		notifications := make(chan interface{}, 10)

		// Subscribe before restore
		shared.Subscribe("key1", func(key string, value interface{}) {
			notifications <- value
		})

		// Restore
		snapshot := map[string]interface{}{
			"key1": "restored_value",
		}

		err := shared.Restore(snapshot)
		if err != nil {
			t.Errorf("Failed to restore: %v", err)
		}

		// Verify notification was sent
		select {
		case val := <-notifications:
			if val != "restored_value" {
				t.Errorf("Expected 'restored_value', got %v", val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive notification for restored value")
		}

		// Cleanup
		shared.Unsubscribe("key1")
	})
}

func TestMemorySharedSpace_RestoreOverwrite(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set initial values
		shared.Set("key1", "original1")
		shared.Set("key2", "original2")

		// Restore with overlapping keys
		snapshot := map[string]interface{}{
			"key1": "overwritten",
			"key3": "new_value",
		}

		err := shared.Restore(snapshot)
		if err != nil {
			t.Errorf("Failed to restore: %v", err)
		}

		// key1 should be overwritten
		val, _ := shared.Get("key1")
		if val != "overwritten" {
			t.Errorf("key1: expected 'overwritten', got %v", val)
		}

		// key2 should be preserved
		val, _ = shared.Get("key2")
		if val != "original2" {
			t.Errorf("key2: expected 'original2', got %v", val)
		}

		// key3 should be added
		val, _ = shared.Get("key3")
		if val != "new_value" {
			t.Errorf("key3: expected 'new_value', got %v", val)
		}
	})
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestMemorySharedSpace_ConcurrentSetGet(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		var wg sync.WaitGroup
		numGoroutines := 100
		numOperations := 100

		// Concurrent writes
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := "key" + string(rune('A'+id%26))
					shared.Set(key, id*numOperations+j)
				}
			}(i)
		}
		wg.Wait()

		// Concurrent reads
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := "key" + string(rune('A'+id%26))
					shared.Get(key)
				}
			}(i)
		}
		wg.Wait()

		// If we get here without deadlock or panic, the test passes
	})
}

func TestMemorySharedSpace_ConcurrentSnapshot(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Pre-populate
		for i := 0; i < 100; i++ {
			shared.Set("key"+string(rune('A'+i%26)), i)
		}

		var wg sync.WaitGroup
		numGoroutines := 50

		// Concurrent snapshots while writing
		wg.Add(numGoroutines * 2)

		// Writers
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					shared.Set("key"+string(rune('A'+id%26)), id*100+j)
				}
			}(i)
		}

		// Snapshot readers
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					snapshot := shared.Snapshot()
					if snapshot == nil {
						t.Error("Snapshot returned nil")
					}
				}
			}()
		}

		wg.Wait()
	})
}

func TestMemorySharedSpace_ConcurrentRestore(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		var wg sync.WaitGroup
		numGoroutines := 20

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				snapshot := map[string]interface{}{
					"key1": id,
					"key2": "value" + string(rune('A'+id%26)),
				}
				err := shared.Restore(snapshot)
				if err != nil {
					t.Errorf("Restore failed: %v", err)
				}
			}(i)
		}

		wg.Wait()

		// Verify data is consistent (some value should exist)
		val, err := shared.Get("key1")
		if err != nil {
			t.Errorf("Failed to get key1: %v", err)
		}
		if val == nil {
			t.Error("key1 should have a value")
		}
	})
}

func TestMemorySharedSpace_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		var wg sync.WaitGroup
		numGoroutines := 50

		wg.Add(numGoroutines * 3)

		// Subscribers
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				key := "key" + string(rune('A'+id%10))
				shared.Subscribe(key, func(k string, v interface{}) {})
			}(i)
		}

		// Unsubscribers
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				key := "key" + string(rune('A'+id%10))
				shared.Unsubscribe(key)
			}(i)
		}

		// Writers (to trigger notifications)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				key := "key" + string(rune('A'+id%10))
				shared.Set(key, id)
			}(i)
		}

		wg.Wait()
	})
}

// =============================================================================
// Space Interface Compliance Test
// =============================================================================

func TestMemorySharedSpace_ImplementsSpaceInterface(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		var _ Space = (*MemorySharedSpace)(nil)

		// Create instance and use through interface
		var space Space = NewMemorySharedSpace()

		// Test all interface methods
		err := space.Set("key", "value")
		if err != nil {
			t.Errorf("Set failed: %v", err)
		}

		val, err := space.Get("key")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if val != "value" {
			t.Errorf("Expected 'value', got %v", val)
		}

		err = space.Delete("key")
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		err = space.Clear()
		if err != nil {
			t.Errorf("Clear failed: %v", err)
		}

		err = space.ClearNotify()
		if err != nil {
			t.Errorf("ClearNotify failed: %v", err)
		}

		err = space.Subscribe("key", func(k string, v interface{}) {})
		if err != nil {
			t.Errorf("Subscribe failed: %v", err)
		}

		err = space.Unsubscribe("key")
		if err != nil {
			t.Errorf("Unsubscribe failed: %v", err)
		}

		snapshot := space.Snapshot()
		if snapshot == nil {
			t.Error("Snapshot returned nil")
		}

		err = space.Restore(map[string]interface{}{"key": "value"})
		if err != nil {
			t.Errorf("Restore failed: %v", err)
		}
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestMemorySharedSpace_NilValue(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Set nil value
		err := shared.Set("nil_key", nil)
		if err != nil {
			t.Errorf("Failed to set nil value: %v", err)
		}

		// Get nil value - should succeed (key exists)
		val, err := shared.Get("nil_key")
		if err != nil {
			t.Errorf("Failed to get nil value: %v", err)
		}
		if val != nil {
			t.Errorf("Expected nil, got %v", val)
		}
	})
}

func TestMemorySharedSpace_EmptyStringKey(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Empty string key should work
		err := shared.Set("", "empty_key_value")
		if err != nil {
			t.Errorf("Failed to set empty key: %v", err)
		}

		val, err := shared.Get("")
		if err != nil {
			t.Errorf("Failed to get empty key: %v", err)
		}
		if val != "empty_key_value" {
			t.Errorf("Expected 'empty_key_value', got %v", val)
		}
	})
}

func TestMemorySharedSpace_LargeData(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		// Create large data
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		err := shared.Set("large_key", largeData)
		if err != nil {
			t.Errorf("Failed to set large data: %v", err)
		}

		val, err := shared.Get("large_key")
		if err != nil {
			t.Errorf("Failed to get large data: %v", err)
		}

		if valBytes, ok := val.([]byte); !ok {
			t.Errorf("Expected []byte, got %T", val)
		} else if len(valBytes) != len(largeData) {
			t.Errorf("Data size mismatch: expected %d, got %d", len(largeData), len(valBytes))
		}
	})
}

func TestMemorySharedSpace_ManyKeys(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()

		numKeys := 10000

		// Set many keys
		for i := 0; i < numKeys; i++ {
			key := "key_" + string(rune(i))
			err := shared.Set(key, i)
			if err != nil {
				t.Errorf("Failed to set key %d: %v", i, err)
			}
		}

		// Snapshot should contain all keys
		snapshot := shared.Snapshot()
		if len(snapshot) != numKeys {
			t.Errorf("Snapshot should have %d keys, got %d", numKeys, len(snapshot))
		}

		// Clear all
		err := shared.Clear()
		if err != nil {
			t.Errorf("Failed to clear: %v", err)
		}

		// Verify all cleared
		snapshot = shared.Snapshot()
		if len(snapshot) != 0 {
			t.Errorf("After clear, snapshot should be empty, got %d keys", len(snapshot))
		}
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMemorySharedSpace_Set(b *testing.B) {
	shared := NewMemorySharedSpace()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		shared.Set("key", i)
	}
}

func BenchmarkMemorySharedSpace_Get(b *testing.B) {
	shared := NewMemorySharedSpace()
	shared.Set("key", "value")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		shared.Get("key")
	}
}

func BenchmarkMemorySharedSpace_Snapshot(b *testing.B) {
	shared := NewMemorySharedSpace()
	for i := 0; i < 100; i++ {
		shared.Set("key"+string(rune(i)), i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		shared.Snapshot()
	}
}

func BenchmarkMemorySharedSpace_Restore(b *testing.B) {
	shared := NewMemorySharedSpace()
	snapshot := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		snapshot["key"+string(rune(i))] = i
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		shared.Restore(snapshot)
	}
}

func BenchmarkMemorySharedSpace_ConcurrentAccess(b *testing.B) {
	shared := NewMemorySharedSpace()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				shared.Set("key", i)
			} else {
				shared.Get("key")
			}
			i++
		}
	})
}
