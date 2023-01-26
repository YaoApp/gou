package v8

import (
	"time"

	"rogchap.com/v8go"
)

// Script v8 scripts
type Script struct {
	ID      string
	File    string
	Source  string
	Timeout time.Duration
}

// Isolate v8 Isolate
type Isolate struct {
	*v8go.Isolate
	status uint8
}

// Context v8 Context
type Context struct {
	ID      string                 // the script id
	SID     string                 // set the session id
	Data    map[string]interface{} // set the global data
	Timeout time.Duration          // terminate the execution after this time
	Iso     *Isolate
	*v8go.Context
}

const (

	// IsoReady isolate is ready
	IsoReady uint8 = 0

	// IsoBusy isolate is in used
	IsoBusy uint8 = 1
)
