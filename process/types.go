package process

import (
	"context"
)

// Process the process sturct
type Process struct {
	Name      string
	Group     string
	Method    string
	Handler   string
	ID        string
	Args      []interface{}
	Global    map[string]interface{} // Global vars
	Sid       string                 // Session ID
	Context   context.Context        // Context
	V8Context interface{}            `json:"-"` // V8 Context (for thread affinity in JavaScript calls)
	Runtime   Runtime                `json:"-"` // Runtime
	Callback  CallbackFunc           `json:"-"` // Callback
	_val      *interface{}           // Value // The result of the process
}

// CallbackFunc the callback function
type CallbackFunc func(process *Process, data map[string]interface{}) error

// Runtime interface
type Runtime interface {
	Dispose()
}

// Handler the process handler
type Handler func(process *Process) interface{}
