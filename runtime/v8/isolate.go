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

	new := newIsolate()
	isolates.Add(new)
	return new, nil
}

// makeTemplate make a new template
func makeTemplate(iso *v8go.Isolate) *v8go.ObjectTemplate {
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

func newIsolate() *Isolate {

	iso := v8go.NewIsolate()
	template := makeTemplate(iso)

	new := &Isolate{
		Isolate:  iso,
		template: template,
		status:   IsoReady,
		contexts: map[*Script]chan *v8go.Context{},
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
		ch := make(chan *v8go.Context, runtimeOption.ContetxQueueSize)
		iso.contexts[script] = ch
		if runtimeOption.Mode == "performance" {
			for i := 0; i < runtimeOption.ContetxQueueSize; i++ {
				newContext, err := iso.MakeContext(script)
				if err != nil {
					log.Error("[V8] %s make context error %s", script.ID, err.Error())
					continue
				}
				ch <- newContext
			}
		}
	}

	for _, script := range RootScripts {
		timeout := script.Timeout
		if timeout == 0 {
			timeout = time.Millisecond * 100
		}

		ch := make(chan *v8go.Context, runtimeOption.ContetxQueueSize)
		iso.contexts[script] = ch
		if runtimeOption.Mode == "performance" {
			for i := 0; i < runtimeOption.ContetxQueueSize; i++ {
				newContext, err := iso.MakeContext(script)
				if err != nil {
					log.Error("[V8] %s make context error %s", script.ID, err.Error())
					continue
				}
				ch <- newContext
			}
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
	list.Data.Store(iso, true)
	list.Len = list.Len + 1
	chIsoReady <- iso
}

// Remove a isolate
func (list *Isolates) Remove(iso *Isolate) {

	// Remove the contexts
	for script, ch := range iso.contexts {

		// close the contexts
		for i := 0; i < len(ch); i++ {
			ctx := <-ch
			ctx.Close()
		}

		// close channel
		close(ch)

		// remove the context
		delete(iso.contexts, script)
	}

	iso.Isolate.Dispose()
	iso.Isolate = nil
	iso.contexts = nil
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

// SelectContext select a context
func (iso *Isolate) SelectContext(script *Script, timeout time.Duration) (*v8go.Context, error) {

	// for performance mode
	if runtimeOption.Mode == "performance" {
		return iso.NewContext(script, timeout)
	}

	// for normal mode
	if iso.Isolate == nil {
		return nil, fmt.Errorf("[V8] %s isolate was removed", script.ID)
	}

	return iso.MakeContext(script)
}

// NewContext create a new context
func (iso *Isolate) NewContext(script *Script, timeout time.Duration) (*v8go.Context, error) {

	if iso.Isolate == nil {
		return nil, fmt.Errorf("[V8] %s isolate was removed", script.ID)
	}

	var ch chan *v8go.Context
	ch, has := iso.contexts[script]
	if !has {
		ch = make(chan *v8go.Context, runtimeOption.ContetxQueueSize)
		iso.contexts[script] = ch

		// Create ContetxQueueSize contexts
		// for performance, we can create the context when the isolate is created
		if runtimeOption.Mode == "performance" {
			for i := 0; i < runtimeOption.ContetxQueueSize; i++ {
				newContext, err := iso.MakeContext(script)
				if err != nil {
					log.Error("[V8] %s make context error %s", script.ID, err.Error())
					continue
				}
				ch <- newContext
			}
		}
	}

	go func() {
		newContext, err := iso.MakeContext(script)
		if err != nil {
			log.Error("[V8] %s make context error %s", script.ID, err.Error())
			return
		}
		ch <- newContext
	}()

	// make a timer
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, fmt.Errorf("Select context timeout %v", timeout)

	case ctx := <-ch:
		return ctx, nil
	}
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
