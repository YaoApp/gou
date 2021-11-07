package gou

import "github.com/robertkrimen/otto"

// Function 脚本函数
type Function struct {
	Name      string
	NumOfArgs int
	Line      int
	Compiled  interface{}
}

// Script 脚本
type Script struct {
	File      string
	Source    string
	Functions map[string]Function
}

// JavaScript 脚本程序运行器
type JavaScript struct {
	*otto.Otto
}

// ScriptVM 脚本接口
type ScriptVM interface {
	Compile(script *Script) error
	Run(script *Script, method string, args ...interface{}) (interface{}, error)
}
