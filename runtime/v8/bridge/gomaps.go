package bridge

import (
	"sync"

	"github.com/google/uuid"
)

// GoMaps is a global registry for Go objects that need to be accessed from JavaScript
// This avoids the overhead of calling V8 functions and provides a unified way to manage object lifecycle
var goMaps = struct {
	sync.RWMutex
	objects map[string]interface{}
}{
	objects: make(map[string]interface{}),
}

// RegisterGoObject registers a Go object and returns a unique ID
// The object can later be retrieved using GetGoObject with this ID
// Remember to call ReleaseGoObject when the object is no longer needed
func RegisterGoObject(obj interface{}) string {
	id := uuid.NewString()
	goMaps.Lock()
	goMaps.objects[id] = obj
	goMaps.Unlock()
	return id
}

// GetGoObject retrieves a Go object by its ID
// Returns nil if the object is not found
func GetGoObject(id string) interface{} {
	goMaps.RLock()
	obj := goMaps.objects[id]
	goMaps.RUnlock()
	return obj
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
	_, exists := goMaps.objects[id]
	goMaps.RUnlock()
	return exists
}

// CountGoObjects returns the number of registered Go objects
// Useful for debugging and testing
func CountGoObjects() int {
	goMaps.RLock()
	count := len(goMaps.objects)
	goMaps.RUnlock()
	return count
}
