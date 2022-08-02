package widget

import "github.com/yaoapp/gou/runtime"

// Widget the widget structs
type Widget struct {
	Name            string
	Path            string
	Label           string   `json:"label,omitempty"`
	Description     string   `json:"description,omitempty"`
	Version         string   `json:"version,omitempty"`
	Root            string   `json:"root,omitempty"`
	Extension       string   `json:"extension,omitempty"`
	Modules         []string `json:"modules,omitempty"`
	Handlers        map[string]Handler
	Instances       map[string]*Instance
	Runtime         *runtime.Runtime
	ModuleRegister  ModuleRegister
	ProcessRegister ProcessRegister
}

// Handler the javascript process
type Handler func()

// Instance the widget instance
type Instance struct {
	Name string
	DSL  map[string]interface{}
}

// ModuleRegister  register  model, flow, etc.
type ModuleRegister map[string]func(name string, source []byte) error

// ProcessRegister register process
type ProcessRegister func(widget, name string, process func(args ...interface{}) interface{}) error
