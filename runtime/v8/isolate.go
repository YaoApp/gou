package v8

import (
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

var isolates = []*Isolate{}
var chIsoReady chan *Isolate
var isoMaxSize = 10
var isoInitSize = 2

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

	if len(isolates) >= isoMaxSize {
		log.Warn("[V8] The maximum number of v8 vm has been reached (%d)", isoMaxSize)
		return nil, fmt.Errorf("The maximum number of v8 vm has been reached (%d)", isoMaxSize)
	}

	log.Info("[V8] Add a new v8 vm")

	iso := v8go.NewIsolate()

	// add yao javascript apis

	// create instance
	new := &Isolate{Isolate: iso, status: IsoReady}
	isolates = append(isolates, new)
	chIsoReady <- new
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

// Lock the isolate
func (iso *Isolate) Lock() error {
	iso.status = IsoBusy
	return nil
}

// Unlock the isolate
func (iso *Isolate) Unlock() error {
	iso.status = IsoReady
	chIsoReady <- iso
	return nil
}

// Locked check if the isolate is locked
func (iso *Isolate) Locked() bool {
	return iso.status == IsoBusy
}

// free unused isolate
func (iso *Isolate) free() {
}
