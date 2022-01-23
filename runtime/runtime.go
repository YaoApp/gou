package runtime

import (
	"context"

	"github.com/yaoapp/gou/runtime/yao"
	"github.com/yaoapp/kun/exception"
)

// Yao create a pure javascript ES6 javascript runtime
func Yao() *Runtime {
	return &Runtime{Name: "yao", Scripts: map[string]Script{}, Engine: yao.New()}
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

// AddFunction add a global JavaScript function
func (runtime *Runtime) AddFunction(name string, fn func(global map[string]interface{}, sid string, args ...interface{}) interface{}) *Runtime {
	err := runtime.Engine.AddFunction(name, fn)
	if err != nil {
		exception.New("runtime AddFunction %s: %s", 500, name, err.Error()).Throw()
	}
	return runtime
}

// AddObject add a global JavaScript object
func (runtime *Runtime) AddObject(name string, methods map[string]func(global map[string]interface{}, sid string, args ...interface{}) interface{}) *Runtime {
	err := runtime.Engine.AddObject(name, methods)
	if err != nil {
		exception.New("runtime AddVariable %s: %s", 500, name, err.Error()).Throw()
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
