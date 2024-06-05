package v8

import (
	"sync"
	"time"

	"github.com/yaoapp/gou/runtime/v8/store"
	"rogchap.com/v8go"
)

// var isoMaxSize = 10
// var isoInitSize = 2
// var isoHeapSizeLimit uint64 = 1518338048 // 1.5G
// var isoHeapSizeRelease uint64 = 52428800 // 50M

// Option runtime option
type Option struct {
	Mode              string `json:"mode,omitempty"`              // the mode of the runtime, the default value is "standard" and the other value is "performance". "performance" mode need more memory but will run faster
	MinSize           uint   `json:"minSize,omitempty"`           // the number of V8 VM when runtime start. max value is 100, the default value is 2
	MaxSize           uint   `json:"maxSize,omitempty"`           // the maximum of V8 VM should be smaller than minSize, the default value is 10
	HeapSizeLimit     uint64 `json:"heapSizeLimit,omitempty"`     // the isolate heap size limit should be smaller than 1.5G, and the default value is 1518338048 (1.5G)
	HeapSizeRelease   uint64 `json:"heapSizeRelease,omitempty"`   // the isolate will be re-created when reaching this value, and the default value is 52428800 (50M)
	HeapAvailableSize uint64 `json:"heapAvailableSize,omitempty"` // the isolate will be re-created when the available size is smaller than this value, and the default value is 524288000 (500M)
	Precompile        bool   `json:"precompile,omitempty"`        // if true compile scripts when the VM is created. this will increase the load time, but the script will run faster. the default value is false
	DefaultTimeout    int    `json:"defaultTimeout,omitempty"`    // the default timeout for the script, the default value is 200ms
	ContextTimeout    int    `json:"contextTimeout,omitempty"`    // the default timeout for the context, the default value is 200ms
	ContetxQueueSize  int    `json:"contextQueueSize,omitempty"`  // the default queue size for the context, the default value is 10, performance only
	DataRoot          string `json:"dataRoot,omitempty"`          // the data root path

	// The following options are experimental features and not stable.
	// They may be removed once the features become stable. Please do not use them in a production environment.
	Import bool `json:"import,omitempty"` // If true, TypeScript import will be enabled. Default value is false.

}

// Script v8 scripts
type Script struct {
	ID      string
	File    string
	Source  string
	Root    bool
	Timeout time.Duration
}

// Module the module
type Module struct {
	GlobalName string
	File       string
	Source     string
}

// Import module
type Import struct {
	Name    string
	Path    string
	AbsPath string
	Clause  string
}

// Isolate v8 Isolate
type Isolate struct {
	*v8go.Isolate
	status   uint8
	template *v8go.ObjectTemplate
}

// Isolates loaded isolate
type Isolates struct {
	Len  int
	Data *sync.Map
}

// Context v8 Context
type Context struct {
	ID      string                 // the script id
	Sid     string                 // set the session id
	Data    map[string]interface{} // set the global data
	Root    bool
	Timeout time.Duration // terminate the execution after this time
	*Runner
	*store.Isolate
	*v8go.UnboundScript
	*v8go.Context
}

const (

	// IsoReady isolate is ready
	IsoReady uint8 = 0

	// IsoBusy isolate is in used
	IsoBusy uint8 = 1
)
