package gou

import (
	"context"

	"github.com/yaoapp/gou/query/share"
)

// Flow  工作流
type Flow struct {
	Name        string                 `json:"-"`
	Source      string                 `json:"-"`
	Scripts     map[string]string      `json:"-"`
	Label       string                 `json:"label"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Nodes       []FlowNode             `json:"nodes,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Global      map[string]interface{} // 全局变量
	Sid         string                 // 会话ID
}

// FlowNode 工作流节点
type FlowNode struct {
	Name    string        `json:"name,omitempty"`
	Process string        `json:"process,omitempty"`
	Engine  string        `json:"engine,omitempty"` // 数据分析引擎名称
	Query   interface{}   `json:"query,omitempty"`  // 数据分析语言 Query Source
	DSL     share.DSL     `json:"-"`                // 数据分析语言 Query DSL
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
