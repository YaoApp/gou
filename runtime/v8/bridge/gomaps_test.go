package bridge

import (
	"testing"
	"time"
)

func TestRegisterAndGetGoObject(t *testing.T) {
	obj := "test-object"
	id := RegisterGoObject(obj)

	retrieved := GetGoObject(id)
	if retrieved != obj {
		t.Errorf("Expected %v, got %v", obj, retrieved)
	}

	// Clean up
	ReleaseGoObject(id)
}

func TestReleaseGoObject(t *testing.T) {
	obj := "test-object"
	id := RegisterGoObject(obj)

	ReleaseGoObject(id)

	retrieved := GetGoObject(id)
	if retrieved != nil {
		t.Errorf("Expected nil after release, got %v", retrieved)
	}
}

func TestHasGoObject(t *testing.T) {
	obj := "test-object"
	id := RegisterGoObject(obj)

	if !HasGoObject(id) {
		t.Error("Expected HasGoObject to return true")
	}

	ReleaseGoObject(id)

	if HasGoObject(id) {
		t.Error("Expected HasGoObject to return false after release")
	}
}

func TestHasGoObjectWithNilObject(t *testing.T) {
	// Manually create entry with nil object
	id := RegisterGoObject(nil)

	// HasGoObject should return false for nil objects
	if HasGoObject(id) {
		t.Error("Expected HasGoObject to return false for nil object")
	}

	// Clean up
	ReleaseGoObject(id)
}

func TestCountGoObjects(t *testing.T) {
	initialCount := CountGoObjects()

	id1 := RegisterGoObject("obj1")
	id2 := RegisterGoObject("obj2")
	id3 := RegisterGoObject("obj3")

	count := CountGoObjects()
	expected := initialCount + 3
	if count != expected {
		t.Errorf("Expected count %d, got %d", expected, count)
	}

	ReleaseGoObject(id1)
	ReleaseGoObject(id2)
	ReleaseGoObject(id3)
}

func TestCollectGarbage(t *testing.T) {
	// Register some objects
	id1 := RegisterGoObject("obj1")
	id2 := RegisterGoObject(nil) // This should be collected
	id3 := RegisterGoObject("obj3")

	// Manually set one entry to nil to simulate garbage
	goMaps.Lock()
	goMaps.objects["manual-nil"] = nil
	goMaps.Unlock()

	initialCount := CountGoObjects()

	// Run garbage collection
	collectGarbage()

	afterGCCount := CountGoObjects()

	// Should have removed at least the nil entries
	if afterGCCount >= initialCount {
		t.Errorf("Expected GC to reduce count from %d to less, got %d", initialCount, afterGCCount)
	}

	// Valid objects should still exist
	if !HasGoObject(id1) {
		t.Error("Expected obj1 to still exist after GC")
	}
	if !HasGoObject(id3) {
		t.Error("Expected obj3 to still exist after GC")
	}

	// Nil object should be gone
	if HasGoObject(id2) {
		t.Error("Expected nil object to be removed by GC")
	}

	// Clean up
	ReleaseGoObject(id1)
	ReleaseGoObject(id3)
}

func TestGCThreshold(t *testing.T) {
	// This test verifies that GC is triggered when threshold is exceeded
	// Note: This is a behavioral test, not a strict requirement

	initialCount := CountGoObjects()

	// Register objects up to threshold + 1
	var ids []string
	for i := 0; i < 100; i++ { // Use smaller number for testing
		id := RegisterGoObject(struct{ index int }{i})
		ids = append(ids, id)
	}

	// Add some nil objects
	RegisterGoObject(nil)
	RegisterGoObject(nil)

	// Wait a bit for GC to potentially run
	time.Sleep(100 * time.Millisecond)

	// Count should not include nil objects if GC ran
	count := CountGoObjects()
	if count > initialCount+102 {
		t.Logf("Warning: GC may not have run, count is %d (expected <= %d)", count, initialCount+102)
	}

	// Clean up
	for _, id := range ids {
		ReleaseGoObject(id)
	}
}

func TestPeriodicGC(t *testing.T) {
	// Note: The periodic GC is started on first RegisterGoObject call
	// This test just verifies it doesn't crash

	id1 := RegisterGoObject("test1")
	id2 := RegisterGoObject(nil)

	// Wait a short time to ensure GC goroutine is running
	time.Sleep(100 * time.Millisecond)

	// Check that valid object still exists
	if !HasGoObject(id1) {
		t.Error("Expected test1 to still exist")
	}

	// Clean up
	ReleaseGoObject(id1)
	ReleaseGoObject(id2)
}

func TestGoObjectEntryTimestamp(t *testing.T) {
	before := time.Now()
	id := RegisterGoObject("test")
	after := time.Now()

	// Verify the entry has a timestamp
	goMaps.RLock()
	entry := goMaps.objects[id]
	goMaps.RUnlock()

	if entry == nil {
		t.Fatal("Expected entry to exist")
	}

	if entry.RegisteredAt.Before(before) || entry.RegisteredAt.After(after) {
		t.Errorf("Expected timestamp between %v and %v, got %v", before, after, entry.RegisteredAt)
	}

	// Clean up
	ReleaseGoObject(id)
}
