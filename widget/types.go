package widget

import "github.com/yaoapp/gou/runtime"

// Widget the widget structs
type Widget struct {
	Name        string
	Path        string
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Root        string `json:"root,omitempty"`
	Extension   string `json:"extension,omitempty"`
	Runtime     *runtime.Runtime
	Handlers    map[string]Handler
	Instances   map[string]*Instance
}

// Handler the javascript process
type Handler func()

// Instance the widget instance
type Instance struct {
	Name string
	DSL  map[string]interface{}
}
