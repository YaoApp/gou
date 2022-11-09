package runtime

import (
	"context"
	"io"

	"github.com/yaoapp/gou/runtime/yao"
	"github.com/yaoapp/kun/exception"
)

// Yao create a pure javascript ES6 javascript runtime
func Yao(option Option) *Runtime {
	engine := yao.New(option.WorkerNums, option.FileRoot)
	return &Runtime{Name: "yao", Scripts: map[string]Script{}, Engine: engine}
}

// Node create a NodeJS runtime ( not support yet )
func Node() *Runtime {
	exception.New("the node runtime does not support yet, use Yao() instead.", 500).Throw()
	return &Runtime{}
}

// Load load and compile script
func (runtime *Runtime) Load(filename string, name string) error {
	return runtime.Engine.Load(filename, name)
}

// LoadReader load and compile script
func (runtime *Runtime) LoadReader(reader io.Reader, name string, filename ...string) error {
	return runtime.Engine.LoadReader(reader, name, filename...)
}

// RootLoad load and compile script root mode
func (runtime *Runtime) RootLoad(filename string, name string) error {
	return runtime.Engine.RootLoad(filename, name)
}

// RootReader load and compile script root mode
func (runtime *Runtime) RootReader(reader io.Reader, name string, filename ...string) error {
	return runtime.Engine.RootLoadReader(reader, name, filename...)
}

// Has check the given name of the script is load
func (runtime *Runtime) Has(name string) bool {
	return runtime.Engine.Has(name)
}

// AddFunction add a global JavaScript function
func (runtime *Runtime) AddFunction(name string, fn func(global map[string]interface{}, sid string, args ...interface{}) interface{}) *Runtime {
	err := runtime.Engine.AddFunction(name, fn)
	if err != nil {
		exception.New("runtime AddFunction %s: %s", 500, name, err.Error()).Throw()
	}
	return runtime
}

// AddRootFunction add a global JavaScript function (root only)
func (runtime *Runtime) AddRootFunction(name string, fn func(global map[string]interface{}, sid string, args ...interface{}) interface{}) *Runtime {
	err := runtime.Engine.AddRootFunction(name, fn)
	if err != nil {
		exception.New("runtime AddFunction %s: %s", 500, name, err.Error()).Throw()
	}
	return runtime
}

// AddObject add a global JavaScript object
func (runtime *Runtime) AddObject(name string, methods map[string]func(global map[string]interface{}, sid string, args ...interface{}) interface{}) *Runtime {
	err := runtime.Engine.AddObject(name, methods)
	if err != nil {
		exception.New("runtime AddObject %s: %s", 500, name, err.Error()).Throw()
	}
	return runtime
}

// New create a new ctx
func (runtime *Runtime) New(name string, method string) *Request {
	return &Request{
		name:    name,
		method:  method,
		runtime: runtime,
		global:  map[string]interface{}{},
		context: context.Background(),
	}
}

// Init initialize Engine
func (runtime *Runtime) Init() error {
	return runtime.Engine.Init()
}

// WithGlobal with global data
func (request *Request) WithGlobal(data map[string]interface{}) *Request {
	request.global = data
	return request
}

// WithSid with global session id
func (request *Request) WithSid(sid string) *Request {
	request.sid = sid
	return request
}

// WithContext with context
func (request *Request) WithContext(context context.Context) *Request {
	request.context = context
	return request
}

// Call execute JavaScript function and return
func (request *Request) Call(args ...interface{}) (interface{}, error) {
	data := map[string]interface{}{
		"__yao_global": request.global,
		"__yao_sid":    request.sid,
	}
	return request.runtime.Engine.Call(data, request.name, request.method, args...)
}

// RootCall execute JavaScript function and return
func (request *Request) RootCall(args ...interface{}) (interface{}, error) {
	data := map[string]interface{}{
		"__yao_global": request.global,
		"__yao_sid":    request.sid,
	}
	return request.runtime.Engine.RootCall(data, request.name, request.method, args...)
}
