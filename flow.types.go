package gou

import (
	"context"
	"regexp"

	"github.com/robertkrimen/otto"
)

var reVar = regexp.MustCompile("{{[ ]*([^\\s]+)[ ]*}}")                     // {{in.2}}
var reFun = regexp.MustCompile("{{[ ]*([0-9a-zA-Z_]+)[ ]*\\((.*)\\)[ ]*}}") // {{pluck($res.users, 'id')}}
var reFunArg = regexp.MustCompile("([^\\s,]+)")                             // $res.users, 'id'

// Flow  工作流
type Flow struct {
	Name         string            `json:"-"`
	Source       string            `json:"-"`
	ScriptSource map[string]string `json:"-"`
	Scripts      map[string]string `json:"-"`
	Label        string            `json:"label"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	Nodes        []FlowNode        `json:"nodes,omitempty"`
	Output       interface{}       `json:"output,omitempty"`
}

// FlowNode 工作流节点
type FlowNode struct {
	Name    string        `json:"name,omitempty"`
	Process string        `json:"process,omitempty"`
	Script  string        `json:"script,omitempty"`
	Args    []interface{} `json:"args,omitempty"`
	Outs    []interface{} `json:"outs,omitempty"`
}

// FlowContext 工作流上下文
type FlowContext struct {
	In      []interface{}
	Res     map[string]interface{}
	Context *context.Context
	Cancel  context.CancelFunc
}

// FlowVM 数据处理脚本程序运行器
type FlowVM struct {
	*otto.Otto
}

// Helper 转换器
type Helper func(...interface{}) interface{}
