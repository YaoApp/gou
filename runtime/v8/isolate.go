package v8

import (
	"fmt"
	"slices"
	"time"

	atobT "github.com/yaoapp/gou/runtime/v8/functions/atob"
	btoaT "github.com/yaoapp/gou/runtime/v8/functions/btoa"
	evalT "github.com/yaoapp/gou/runtime/v8/functions/eval"
	langT "github.com/yaoapp/gou/runtime/v8/functions/lang"
	processT "github.com/yaoapp/gou/runtime/v8/functions/process"
	exceptionT "github.com/yaoapp/gou/runtime/v8/objects/exception"
	fsT "github.com/yaoapp/gou/runtime/v8/objects/fs"
	httpT "github.com/yaoapp/gou/runtime/v8/objects/http"
	jobT "github.com/yaoapp/gou/runtime/v8/objects/job"
	logT "github.com/yaoapp/gou/runtime/v8/objects/log"
	planT "github.com/yaoapp/gou/runtime/v8/objects/plan"
	queryT "github.com/yaoapp/gou/runtime/v8/objects/query"
	storeT "github.com/yaoapp/gou/runtime/v8/objects/store"
	timeT "github.com/yaoapp/gou/runtime/v8/objects/time"
	websocketT "github.com/yaoapp/gou/runtime/v8/objects/websocket"
	"github.com/yaoapp/gou/runtime/v8/store"

	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

var isoReady chan *store.Isolate

// thirdPartyObjects third party objects
var keepWords = []string{"log", "time", "http", "Exception", "FS", "Job", "Store", "Plan", "Query", "WebSocket", "$L", "Process", "Eval"}
var thirdPartyObjects map[string]*ThirdPartyObject = make(map[string]*ThirdPartyObject)
var thirdPartyFunctions map[string]*ThirdPartyFunction = make(map[string]*ThirdPartyFunction)

// initialize create a new Isolate
// in performance mode, the minSize isolates will be created
func initialize() {

	log.Info("[V8] initialize mode: %s", runtimeOption.Mode)
	v8go.YaoInit(uint(runtimeOption.HeapSizeLimit / 1024 / 1024))

	// Performance mode
	if runtimeOption.Mode == "performance" {
		dispatcher = NewDispatcher(runtimeOption.MinSize, runtimeOption.MaxSize)
		dispatcher.Start()
		return
	}

	// Standard mode
	makeGlobalIsolate()
	isoReady = make(chan *store.Isolate, runtimeOption.MinSize)

}

func release() {
	v8go.YaoDispose()
	if runtimeOption.Mode == "performance" {
		dispatcher.Stop()
	}
}

// RegisterObject register a third party object
func RegisterObject(name string, object EmbedObject, attributes ...v8go.PropertyAttribute) error {

	// Validate the name
	if slices.Contains(keepWords, name) {
		log.Error("[V8] Register object %s failed: %s", name, "The name is reserved")
		return fmt.Errorf("the name is reserved")
	}

	syncLock.Lock()
	defer syncLock.Unlock()
	thirdPartyObjects[name] = &ThirdPartyObject{
		Name:       name,
		Object:     object,
		Attributes: attributes,
	}
	return nil
}

// RegisterFunction register a third party function
func RegisterFunction(name string, function EmbedFunction, attributes ...v8go.PropertyAttribute) error {

	// Validate the name
	if slices.Contains(keepWords, name) {
		log.Error("[V8] Register function %s failed: %s", name, "The name is reserved")
		return fmt.Errorf("the name is reserved")
	}

	syncLock.Lock()
	defer syncLock.Unlock()
	thirdPartyFunctions[name] = &ThirdPartyFunction{
		Name:       name,
		Function:   function,
		Attributes: attributes,
	}
	return nil
}

// precompile compile the loaded scirpts
// it cost too much time and memory to compile all scripts
// ignore the error
// func precompile(iso *store.Isolate) {
// }

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
	template.Set("Plan", planT.New().ExportFunction(iso))
	template.Set("Query", queryT.New().ExportFunction(iso))
	template.Set("WebSocket", websocketT.New().ExportFunction(iso))
	template.Set("$L", langT.ExportFunction(iso))
	template.Set("Process", processT.ExportFunction(iso))
	template.Set("Eval", evalT.ExportFunction(iso))

	// Deprecated Studio and Require
	// template.Set("Studio", studioT.ExportFunction(iso))
	// template.Set("Require", Require(iso))

	// Window object (std functions)
	template.Set("atob", atobT.ExportFunction(iso))
	template.Set("btoa", btoaT.ExportFunction(iso))

	// Register third party objects
	for name, object := range thirdPartyObjects {
		err := template.Set(name, object.Object(iso), object.Attributes...)
		if err != nil {
			log.Error("[V8] Register object %s failed: %s", name, err.Error())
		}
	}

	// Register third party functions
	for name, function := range thirdPartyFunctions {
		err := template.Set(name, function.Function(iso), function.Attributes...)
		if err != nil {
			log.Error("[V8] Register function %s failed: %s", name, err.Error())
		}
	}

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
