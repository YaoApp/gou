package bridge

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// goObjectEntry represents a registered Go object with metadata
type goObjectEntry struct {
	Object       interface{}
	RegisteredAt time.Time
}

// GoMaps is a global registry for Go objects that need to be accessed from JavaScript
// This avoids the overhead of calling V8 functions and provides a unified way to manage object lifecycle
var goMaps = struct {
	sync.RWMutex
	objects map[string]*goObjectEntry
}{
	objects: make(map[string]*goObjectEntry),
}

const (
	// gcThreshold is the maximum number of objects before triggering GC check
	gcThreshold = 1000
	// gcCheckInterval is the interval for periodic GC checks
	gcCheckInterval = 5 * time.Minute
)

var (
	gcOnce     sync.Once
	gcStopChan chan struct{}
)

// RegisterGoObject registers a Go object and returns a unique ID
// The object can later be retrieved using GetGoObject with this ID
// Remember to call ReleaseGoObject when the object is no longer needed
func RegisterGoObject(obj interface{}) string {
	id := uuid.NewString()

	goMaps.Lock()
	goMaps.objects[id] = &goObjectEntry{
		Object:       obj,
		RegisteredAt: time.Now(),
	}
	count := len(goMaps.objects)
	goMaps.Unlock()

	// Start periodic GC goroutine on first registration
	gcOnce.Do(func() {
		gcStopChan = make(chan struct{})
		go periodicGC()
	})

	// Trigger GC check if threshold exceeded
	if count > gcThreshold {
		go collectGarbage()
	}

	return id
}

// GetGoObject retrieves a Go object by its ID
// Returns nil if the object is not found
func GetGoObject(id string) interface{} {
	goMaps.RLock()
	entry := goMaps.objects[id]
	goMaps.RUnlock()

	if entry == nil {
		return nil
	}
	return entry.Object
}

// ReleaseGoObject removes a Go object from the registry
// This should be called when the JavaScript object is garbage collected or released
func ReleaseGoObject(id string) {
	goMaps.Lock()
	delete(goMaps.objects, id)
	goMaps.Unlock()
}

// HasGoObject checks if a Go object exists in the registry
func HasGoObject(id string) bool {
	goMaps.RLock()
	entry, exists := goMaps.objects[id]
	goMaps.RUnlock()
	return exists && entry != nil && entry.Object != nil
}

// CountGoObjects returns the number of registered Go objects
// Useful for debugging and testing
func CountGoObjects() int {
	goMaps.RLock()
	count := len(goMaps.objects)
	goMaps.RUnlock()
	return count
}

// collectGarbage scans the registry and removes entries with nil objects
func collectGarbage() {
	goMaps.Lock()
	defer goMaps.Unlock()

	var toDelete []string
	for id, entry := range goMaps.objects {
		// Check if object is nil or entry itself is nil
		if entry == nil || entry.Object == nil {
			toDelete = append(toDelete, id)
		}
	}

	// Delete nil entries
	for _, id := range toDelete {
		delete(goMaps.objects, id)
	}
}

// periodicGC runs periodic garbage collection checks
func periodicGC() {
	ticker := time.NewTicker(gcCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			collectGarbage()
		case <-gcStopChan:
			return
		}
	}
}

// StopPeriodicGC stops the periodic garbage collection
// Useful for graceful shutdown or testing
func StopPeriodicGC() {
	if gcStopChan != nil {
		select {
		case <-gcStopChan:
			// Already closed
		default:
			close(gcStopChan)
		}
	}
}
