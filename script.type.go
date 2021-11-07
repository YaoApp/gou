package gou

import "github.com/robertkrimen/otto"

// ScriptVM 脚本接口
type ScriptVM interface {
	Compile(script *Script) error
	Run(script *Script, method string, args ...interface{}) (interface{}, error)
	WithProcess(except ...string) ScriptVM
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
	Process string // 当前运行的处理器 | 防止递归调用
	*otto.Otto
}
