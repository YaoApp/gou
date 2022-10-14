package yao

import "rogchap.com/v8go"

// Yao runtime engine ( base on v8)
type Yao struct {
	scripts         map[string]script
	rootScripts     map[string]script
	iso             *v8go.Isolate
	ctx             *v8go.Context
	template        *v8go.ObjectTemplate
	objectTemplates map[string]*v8go.ObjectTemplate
	// contexts        *Pool
	numOfContexts int
}

type script struct {
	name     string
	filename string
	source   string
	Ctx      *v8go.Context
	IsRoot   bool
	// compiled *v8go.UnboundScript
}

// Pool JS contect pool
type Pool struct {
	contexts chan *v8go.Context
	size     int
	lock     bool
}
