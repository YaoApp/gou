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

	for _, script := range Scripts {
		script.compile(iso)
	}

	for _, script := range RootScripts {
		script.compile(iso)
	}

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

func makeIsolate() *store.Isolate {
	iso := v8go.NewIsolateHeapSize(int(runtimeOption.HeapSizeLimit / 1024 / 1024))
	new := &store.Isolate{
		Isolate:  iso,
		Template: MakeTemplate(iso),
		Status:   IsoReady,
	}
	return new
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

// *********************************************************************************************************************
// -- Warning: The following code is deprecated --
// *********************************************************************************************************************

// NewIsolate create a new Isolate
func NewIsolate() (*Isolate, error) {

	newIsolateLock.Lock()
	defer newIsolateLock.Unlock()

	if isolates.Len >= runtimeOption.MaxSize {
		log.Warn("[V8] The maximum number of v8 vm has been reached (%d)", runtimeOption.MaxSize)
		return nil, fmt.Errorf("The maximum number of v8 vm has been reached (%d)", runtimeOption.MaxSize)
	}

	new := newIsolate()
	isolates.Add(new)
	return new, nil
}

func newIsolate() *Isolate {

	iso := v8go.NewIsolate()
	template := MakeTemplate(iso)
	new := &Isolate{
		Isolate:  iso,
		template: template,
		status:   IsoReady,
	}

	if runtimeOption.Precompile {
		new.Precompile()
	}
	return new
}

// Precompile compile the loaded scirpts
func (iso *Isolate) Precompile() {

	for _, script := range Scripts {
		timeout := script.Timeout
		if timeout == 0 {
			timeout = time.Millisecond * time.Duration(runtimeOption.ContextTimeout)
		}
	}

	for _, script := range RootScripts {
		timeout := script.Timeout
		if timeout == 0 {
			timeout = time.Millisecond * 100
		}
	}
}

// SelectIso one ready isolate
func SelectIso(timeout time.Duration) (*Isolate, error) {

	// Create a new isolate
	if len(chIsoReady) == 0 {
		go NewIsolate()
	}

	// make a timer
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, fmt.Errorf("Select isolate timeout %v", timeout)

	case iso := <-chIsoReady:
		iso.Lock()
		return iso, nil
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
	list.Data.Store(iso.Key(), true)
	list.Len = list.Len + 1
	chIsoReady <- iso
}

// Remove a isolate
func (list *Isolates) Remove(iso *Isolate) {
	defer iso.Dispose()
	list.Data.Delete(iso.Key())
	list.Len = list.Len - 1
}

// Dispose the isolate
func (iso *Isolate) Dispose() {
	iso.Isolate.Dispose()
	iso.Isolate = nil
	iso.template = nil
	iso = nil
}

// Range traverse isolates
func (list *Isolates) Range(callback func(iso *Isolate) bool) {
	list.Data.Range(func(key, value any) bool {
		return callback(key.(*Isolate))
	})
}

// Key return the key of the isolate
func (iso *Isolate) Key() string {
	return fmt.Sprintf("%p", iso)
}

// Lock the isolate
func (iso *Isolate) Lock() error {
	iso.status = IsoBusy
	return nil
}

// Unlock the isolate
func (iso *Isolate) Unlock() error {

	if iso.health() && len(chIsoReady) <= runtimeOption.MinSize-1 { // the available isolates are less than min size
		iso.status = IsoReady
		chIsoReady <- iso
		return nil
	}

	// Remove the iso and create new one
	go func() {
		log.Info("[V8] VM %p will be removed", iso)
		isolates.Remove(iso)
		if len(chIsoReady) <= runtimeOption.MinSize-1 { // the available isolates are less than min size
			NewIsolate()
		}
	}()

	return nil
}

// Locked check if the isolate is locked
func (iso Isolate) Locked() bool {
	return iso.status == IsoBusy
}

// health check the isolate health
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

	if iso.Isolate == nil {
		return false
	}

	stat := iso.Isolate.GetHeapStatistics()
	if stat.TotalHeapSize > runtimeOption.HeapSizeRelease {
		return false
	}

	if stat.TotalAvailableSize < runtimeOption.HeapAvailableSize { // 500M
		return false
	}

	return true
}

// MakeContext make a new context
func (iso *Isolate) MakeContext(script *Script) (*v8go.Context, error) {
	newContext := v8go.NewContext(iso.Isolate, iso.template)
	instance, err := iso.Isolate.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		newContext.Close()
		return nil, err
	}

	_, err = instance.Run(newContext)
	if err != nil {
		newContext.Close()
		return nil, err
	}

	return newContext, nil
}
