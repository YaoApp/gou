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

var isoMaxSize = 10
var isoInitSize = 2
var isoHeapSizeLimit uint64 = 1518338048 // 1.5G
var isoHeapSizeRelease uint64 = 52428800 //52428800 // 50M

// Setup Initialize the v8 virtual machines
func Setup(size int, maxSize int) error {

	// Initialize the channels
	isoMaxSize = maxSize
	isoInitSize = size
	chIsoReady = make(chan *Isolate, isoMaxSize)

	for i := 0; i < isoInitSize; i++ {
		_, err := NewIsolate()
		if err != nil {
			return err
		}
	}

	return nil
}

// NewIsolate create a new Isolate
func NewIsolate() (*Isolate, error) {
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()

	if isolates.Len >= isoMaxSize {
		log.Warn("[V8] The maximum number of v8 vm has been reached (%d)", isoMaxSize)
		return nil, fmt.Errorf("The maximum number of v8 vm has been reached (%d)", isoMaxSize)
	}

	log.Info("[V8] Add a new v8 vm")

	iso := v8go.NewIsolate()

	// add yao javascript apis

	// create instance
	new := &Isolate{Isolate: iso, status: IsoReady}

	// Compile Scirpts
	contexts[new] = map[string]*v8go.Context{}
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

// Add a isolate
func (list *Isolates) Add(iso *Isolate) {
	list.Data.Store(iso, true)
	list.Len = list.Len + 1
	chIsoReady <- iso
}

// Remove a isolate
func (list *Isolates) Remove(iso *Isolate) {
	iso.Dispose()
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

	// Remove the iso and create new
	go func() {
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
	if stat.TotalHeapSize > isoHeapSizeRelease {
		return false
	}

	if stat.TotalAvailableSize < 524288000 { // 500M
		return false
	}

	return true
}
