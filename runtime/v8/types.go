package v8

import (
	"sync"
	"time"

	"rogchap.com/v8go"
)

// var isoMaxSize = 10
// var isoInitSize = 2
// var isoHeapSizeLimit uint64 = 1518338048 // 1.5G
// var isoHeapSizeRelease uint64 = 52428800 // 50M

// Option runtime option
type Option struct {
	MinSize           int    `json:"minSize,omitempty"`           // the number of V8 VM when runtime start. max value is 100, the default value is 2
	MaxSize           int    `json:"maxSize,omitempty"`           // the maximum of V8 VM should be smaller than minSize, the default value is 10
	HeapSizeLimit     uint64 `json:"heapSizeLimit,omitempty"`     // the isolate heap size limit should be smaller than 1.5G, and the default value is 1518338048 (1.5G)
	HeapSizeRelease   uint64 `json:"heapSizeRelease,omitempty"`   // the isolate will be re-created when reaching this value, and the default value is 52428800 (50M)
	HeapAvailableSize uint64 `json:"heapAvailableSize,omitempty"` // the isolate will be re-created when the available size is smaller than this value, and the default value is 524288000 (500M)
	Precompile        bool   `json:"precompile,omitempty"`        // if true compile scripts when the VM is created. this will increase the load time, but the script will run faster. the default value is false
	DataRoot          string `json:"dataRoot,omitempty"`          // the data root path
}

// Script v8 scripts
type Script struct {
	ID      string
	File    string
	Source  string
	Root    bool
	Timeout time.Duration
}

// Isolate v8 Isolate
type Isolate struct {
	*v8go.Isolate
	status   uint8
	contexts map[*Script]*v8go.Context
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
	SID     string                 // set the session id
	Data    map[string]interface{} // set the global data
	Timeout time.Duration          // terminate the execution after this time
	Iso     *Isolate
	Root    bool
	*v8go.Context
}

const (

	// IsoReady isolate is ready
	IsoReady uint8 = 0

	// IsoBusy isolate is in used
	IsoBusy uint8 = 1
)
