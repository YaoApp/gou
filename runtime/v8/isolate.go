package v8

import (
	"fmt"
	"sync"
	"time"

	atobT "github.com/yaoapp/gou/runtime/v8/functions/atob"
	btoaT "github.com/yaoapp/gou/runtime/v8/functions/btoa"
	langT "github.com/yaoapp/gou/runtime/v8/functions/lang"
	processT "github.com/yaoapp/gou/runtime/v8/functions/process"
	studioT "github.com/yaoapp/gou/runtime/v8/functions/studio"
	exceptionT "github.com/yaoapp/gou/runtime/v8/objects/exception"
	fsT "github.com/yaoapp/gou/runtime/v8/objects/fs"
	httpT "github.com/yaoapp/gou/runtime/v8/objects/http"
	jobT "github.com/yaoapp/gou/runtime/v8/objects/job"
	logT "github.com/yaoapp/gou/runtime/v8/objects/log"
	queryT "github.com/yaoapp/gou/runtime/v8/objects/query"
	storeT "github.com/yaoapp/gou/runtime/v8/objects/store"
	timeT "github.com/yaoapp/gou/runtime/v8/objects/time"
	websocketT "github.com/yaoapp/gou/runtime/v8/objects/websocket"
	"github.com/yaoapp/gou/runtime/v8/store"

	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

var isolates = &Isolates{Data: &sync.Map{}, Len: 0}
var contextCache = map[*Isolate]map[*Script]*Context{}
var isoReady chan *store.Isolate

var chIsoReady chan *Isolate
var newIsolateLock = &sync.RWMutex{}

var chCtxReady chan *Context
var newContextLock = &sync.RWMutex{}

// initialize create a new Isolate
// in performance mode, the minSize isolates will be created
func initialize() {

	v8go.YaoInit(uint(runtimeOption.HeapSizeLimit / 1024 / 1024))

	// Make a global Isolate
	// makeGlobalIsolate()

	if runtimeOption.Mode == "performance" {
		dispatcher = NewDispatcher(runtimeOption.MinSize, runtimeOption.MaxSize)
		dispatcher.Start()
	}

	isoReady = make(chan *store.Isolate, runtimeOption.MaxSize)
	store.Isolates = store.New()
	log.Trace(
		"[V8] VM is initializing  MinSize=%d MaxSize=%d HeapLimit=%d",
		runtimeOption.MinSize, runtimeOption.MaxSize, runtimeOption.HeapSizeLimit,
	)
	if runtimeOption.Mode == "performance" {
		for store.Isolates.Len() < runtimeOption.MinSize {
			addIsolate()
		}
	}
}

func release() {
	v8go.YaoDispose()
	if runtimeOption.Mode == "performance" {
		dispatcher.Stop()
	}
}

// addIsolate create a new and add to the isolates
func addIsolate() (*store.Isolate, error) {

	if store.Isolates.Len() >= runtimeOption.MaxSize {
		log.Warn("[V8] The maximum number of v8 vm has been reached (%d)", runtimeOption.MaxSize)
		return nil, fmt.Errorf("The maximum number of v8 vm has been reached (%d)", runtimeOption.MaxSize)
	}

	iso := makeIsolate()
	if runtimeOption.Precompile {
		precompile(iso)
	}

	store.Isolates.Add(iso)
	// store.MakeIsolateCache(iso.Key())
	isoReady <- iso
	log.Trace("[V8] VM %s is ready (%d)", iso.Key(), len(isoReady))
	return iso, nil
}

// replaceIsolate
// remove a isolate
// create a new one append to the isolates if the isolates is less than minSize
func replaceIsolate(iso *store.Isolate) {
	removeIsolate(iso)
	if store.Isolates.Len() < runtimeOption.MinSize {
		addIsolate()
	}
}

// removeIsolate remove a isolate
func removeIsolate(iso *store.Isolate) {
	key := iso.Key()
	// store.CleanIsolateCache(key)
	// store.Isolates.Remove(key)
	iso.Dispose()
	log.Trace("[V8] VM %s is removed", key)
}

// precompile compile the loaded scirpts
// it cost too much time and memory to compile all scripts
// ignore the error
func precompile(iso *store.Isolate) {
	return
}

// MakeTemplate make a new template
func MakeTemplate(iso *v8go.Isolate) *v8go.ObjectTemplate {

	template := v8go.NewObjectTemplate(iso)
	template.Set("log", logT.New().ExportObject(iso))
	template.Set("time", timeT.New().ExportObject(iso))
	template.Set("http", httpT.New(runtimeOption.DataRoot).ExportObject(iso))

	// set functions
	template.Set("Exception", exceptionT.New().ExportFunction(iso))
	template.Set("FS", fsT.New().ExportFunction(iso))
	template.Set("Job", jobT.New().ExportFunction(iso))
	template.Set("Store", storeT.New().ExportFunction(iso))
	template.Set("Query", queryT.New().ExportFunction(iso))
	template.Set("WebSocket", websocketT.New().ExportFunction(iso))
	template.Set("$L", langT.ExportFunction(iso))
	template.Set("Process", processT.ExportFunction(iso))
	template.Set("Studio", studioT.ExportFunction(iso))
	template.Set("Require", Require(iso))

	// Window object (std functions)
	template.Set("atob", atobT.ExportFunction(iso))
	template.Set("btoa", btoaT.ExportFunction(iso))
	return template
}

func makeGlobalIsolate() {
	iso := v8go.YaoNewIsolate()
	iso.AsGlobal()
}

func makeIsolate() *store.Isolate {
	// iso, err := v8go.YaoNewIsolateFromGlobal()
	// if err != nil {
	// 	log.Error("[V8] Create isolate failed: %s", err.Error())
	// 	return nil
	// }

	iso := v8go.YaoNewIsolate()
	return &store.Isolate{
		Isolate:  iso,
		Template: MakeTemplate(iso),
		Status:   IsoReady,
	}
}

// SelectIsoPerformance one ready isolate
func SelectIsoPerformance(timeout time.Duration) (*store.Isolate, error) {

	// make a timer
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case iso := <-isoReady:
		Lock(iso)
		return iso, nil

	case <-timer.C:
		log.Error("[V8] Select isolate timeout %v", timeout)
		return nil, fmt.Errorf("Select isolate timeout %v", timeout)
	}

}

// SelectIsoStandard one ready isolate ( the max size is 2 )
func SelectIsoStandard(timeout time.Duration) (*store.Isolate, error) {

	go func() {
		// Create a new isolate
		iso := makeIsolate()
		isoReady <- iso
	}()

	// make a timer
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		log.Error("[V8] Select isolate timeout %v", timeout)
		return nil, fmt.Errorf("Select isolate timeout %v", timeout)

	case iso := <-isoReady:
		return iso, nil
	}
}

// Lock the isolate
func Lock(iso *store.Isolate) {
	iso.Lock()
}

// Unlock the isolate
// Recycle the isolate if the isolate is not health
func Unlock(iso *store.Isolate) {

	health := iso.Health(runtimeOption.HeapSizeRelease, runtimeOption.HeapAvailableSize)
	available := len(isoReady)
	log.Trace("[V8] VM %s is health %v available %d", iso.Key(), health, available)

	// add the isolate if the available isolates are less than min size
	if available < runtimeOption.MinSize {
		defer addIsolate()
	}

	// remove the isolate if the available isolates are more than min size
	if available > runtimeOption.MinSize {
		go removeIsolate(iso)
		return
	}

	// unlock the isolate if the isolate is health
	if health {
		iso.Unlock()
		isoReady <- iso
		return
	}

	// remove the isolate if the isolate is not health
	// then create a new one
	go replaceIsolate(iso)

}
