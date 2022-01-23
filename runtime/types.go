package runtime

import "context"

// Engine 脚本接口
type Engine interface {
	Load(filename string, name string) error
	AddFunction(name string, fn func(global map[string]interface{}, sid string, args ...interface{}) interface{}) error
	AddObject(name string, methods map[string]func(global map[string]interface{}, sid string, args ...interface{}) interface{}) error

	Call(data map[string]interface{}, name string, method string, args ...interface{}) (interface{}, error)
}

// Runtime 运行时
type Runtime struct {
	Name    string
	Engine  Engine
	Scripts map[string]Script
}

// Script 脚本
type Script struct {
	File   string
	Source string
}

// Request 请求
type Request struct {
	name    string
	method  string
	runtime *Runtime
	sid     string
	global  map[string]interface{}
	context context.Context
}
