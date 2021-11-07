package gou

import "github.com/robertkrimen/otto"

// ScriptVM 脚本接口
type ScriptVM interface {
	Compile(script *Script) error
	WithProcess(except ...string) ScriptVM
	Run(name string, method string, args ...interface{}) (interface{}, error)
	RunScript(script *Script, method string, args ...interface{}) (interface{}, error)
	Load(filename string, name string) error
	MustLoad(filename string, name string) ScriptVM
	Get(name string) (*Script, error)
	MustGet(name string) *Script
}

// Script 脚本
type Script struct {
	File      string
	Source    string
	Functions map[string]Function
}

// Function 脚本函数
type Function struct {
	Name      string
	NumOfArgs int
	Line      int
	Compiled  interface{}
}

// JavaScript 脚本程序运行器
type JavaScript struct {
	Scripts map[string]*Script
	*otto.Otto
}
