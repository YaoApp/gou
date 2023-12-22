package v8

import "github.com/yaoapp/gou/runtime/v8/store"

var runtimeOption = &Option{}

// Start v8 runtime
func Start(option *Option) error {
	option.Validate()
	runtimeOption = option
	chIsoReady = make(chan *Isolate, option.MaxSize)
	isoReady = make(chan *store.Isolate, option.MaxSize)
	for i := 0; i < option.MinSize; i++ {
		_, err := NewIsolate()
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop v8 runtime
func Stop() {
	// chIsoReady = make(chan *Isolate, runtimeOption.MaxSize)
	close(isoReady)
	close(chIsoReady)
	// Remove iso
	isolates.Range(func(iso *Isolate) bool {
		isolates.Remove(iso)
		return true
	})
}
