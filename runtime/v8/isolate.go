package v8

import (
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

var isolates = &Isolates{Data: &sync.Map{}, Len: 0}
var chIsoReady chan *Isolate
var newIsolateLock = &sync.RWMutex{}

// NewIsolate create a new Isolate
func NewIsolate() (*Isolate, error) {

	newIsolateLock.Lock()
	defer newIsolateLock.Unlock()

	if isolates.Len >= runtimeOption.MaxSize {
		log.Warn("[V8] The maximum number of v8 vm has been reached (%d)", runtimeOption.MaxSize)
		return nil, fmt.Errorf("The maximum number of v8 vm has been reached (%d)", runtimeOption.MaxSize)
	}

	iso := v8go.NewIsolate()

	// add yao javascript apis

	// create instance
	new := &Isolate{Isolate: iso, status: IsoReady, contexts: map[*Script]*v8go.Context{}}

	// Compile Scirpts
	if runtimeOption.Precompile {
		for _, script := range Scripts {
			timeout := script.Timeout
			if timeout == 0 {
				timeout = time.Millisecond * 100
			}
			script.Compile(new, timeout)
		}

		for _, script := range RootScripts {
			timeout := script.Timeout
			if timeout == 0 {
				timeout = time.Millisecond * 100
			}
			script.Compile(new, timeout)
		}
	}

	isolates.Add(new)
	return new, nil
}

// SelectIso one ready isolate
func SelectIso(timeout time.Duration) (*Isolate, error) {

	// Create a new isolate
	if len(chIsoReady) == 0 {
		go NewIsolate()
	}

	select {
	case iso := <-chIsoReady:
		iso.Lock()
		return iso, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("Select isolate timeout %v", timeout)
	}
}

// Resize set the maxSize
func (list *Isolates) Resize(minSize, maxSize int) error {
	if maxSize > 100 {
		log.Warn("[V8] the maximum value of maxSize is 100")
		maxSize = 100
	}

	// Remove iso
	isolates.Range(func(iso *Isolate) bool {
		isolates.Remove(iso)
		return true
	})

	runtimeOption.MinSize = minSize
	runtimeOption.MaxSize = maxSize
	runtimeOption.Validate()
	chIsoReady = make(chan *Isolate, runtimeOption.MaxSize)
	for i := 0; i < runtimeOption.MinSize; i++ {
		_, err := NewIsolate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Add a isolate
func (list *Isolates) Add(iso *Isolate) {
	list.Data.Store(iso, true)
	list.Len = list.Len + 1
	go func() { chIsoReady <- iso }()
}

// Remove a isolate
func (list *Isolates) Remove(iso *Isolate) {
	iso.Isolate.Dispose()
	list.Data.Delete(iso)
	list.Len = list.Len - 1
}

// Range traverse isolates
func (list *Isolates) Range(callback func(iso *Isolate) bool) {
	list.Data.Range(func(key, value any) bool {
		return callback(key.(*Isolate))
	})
}

// Lock the isolate
func (iso *Isolate) Lock() error {
	iso.status = IsoBusy
	return nil
}

// Unlock the isolate
func (iso *Isolate) Unlock() error {

	if iso.health() {
		iso.status = IsoReady
		chIsoReady <- iso
		return nil
	}

	// Remove the iso and create new one
	go func() {
		log.Info("[V8] VM %p will be removed", iso)
		isolates.Remove(iso)
		NewIsolate()
	}()

	return nil
}

// Locked check if the isolate is locked
func (iso Isolate) Locked() bool {
	return iso.status == IsoBusy
}

func (iso *Isolate) health() bool {

	// {
	// 	"ExternalMemory": 0,
	// 	"HeapSizeLimit": 1518338048,
	// 	"MallocedMemory": 16484,
	// 	"NumberOfDetachedContexts": 0,
	// 	"NumberOfNativeContexts": 3,
	// 	"PeakMallocedMemory": 24576,
	// 	"TotalAvailableSize": 1518051356,
	// 	"TotalHeapSize": 1261568,
	// 	"TotalHeapSizeExecutable": 262144,
	// 	"TotalPhysicalSize": 499164,
	// 	"UsedHeapSize": 713616
	// }

	stat := iso.GetHeapStatistics()
	if stat.TotalHeapSize > runtimeOption.HeapSizeRelease {
		return false
	}

	if stat.TotalAvailableSize < runtimeOption.HeapAvailableSize { // 500M
		return false
	}

	return true
}
