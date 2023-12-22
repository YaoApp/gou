package process

import (
	"context"
)

// Process the process sturct
type Process struct {
	Name    string
	Group   string
	Method  string
	Handler string
	ID      string
	Args    []interface{}
	Global  map[string]interface{} // Global vars
	Sid     string                 // Session ID
	Context context.Context        // Context
	Runtime Runtime                // Runtime
}

// Runtime interface
type Runtime interface {
	Dispose()
}

// Handler the process handler
type Handler func(process *Process) interface{}
