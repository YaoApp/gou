package v8

import (
	"github.com/yaoapp/gou/runtime/v8/store"
	"github.com/yaoapp/kun/utils"
)

var runtimeOption = &Option{}

// Start v8 runtime
func Start(option *Option) error {
	option.Validate()
	runtimeOption = option
	utils.Dump(runtimeOption)
	initialize()
	return nil
}

// Stop v8 runtime
func Stop() {
	close(isoReady)
	store.Isolates.Range(func(iso store.IStore) bool {
		key := iso.Key()
		store.CleanIsolateCache(key)
		store.Isolates.Remove(key)
		return true
	})
}
