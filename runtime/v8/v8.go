package v8

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/store"
)

var runtimeOption = &Option{}

// Start v8 runtime
func Start(option *Option) error {
	option.Validate()
	runtimeOption = option
	initialize()
	// Note: Periodic GC for goMaps is automatically started on first RegisterGoObject call
	// No explicit initialization needed here
	return nil
}

// Stop v8 runtime
func Stop() {
	if isoReady != nil {
		close(isoReady)
	}
	isoReady = nil
	store.Isolates.Range(func(iso store.IStore) bool {
		key := iso.Key()
		store.CleanIsolateCache(key)
		store.Isolates.Remove(key)
		return true
	})
	release()

	// Stop periodic garbage collection for goMaps
	bridge.StopPeriodicGC()
}
