package yao

import "rogchap.com/v8go"

// Yao runtime engine ( base on v8)
type Yao struct {
	scripts         map[string]script
	iso             *v8go.Isolate
	template        *v8go.ObjectTemplate
	objectTemplates map[string]*v8go.ObjectTemplate
}

type script struct {
	name     string
	filename string
	source   string
	compiled *v8go.UnboundScript
}
