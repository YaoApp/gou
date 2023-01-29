package flow

import (
	"context"

	"github.com/yaoapp/gou/query/share"
)

// Flow  工作流
type Flow struct {
	ID          string                 `json:"-"`
	File        string                 `json:"-"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Nodes       []Node                 `json:"nodes,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Global      map[string]interface{} // 全局变量
	Sid         string                 // 会话ID
}

// Node 工作流节点
type Node struct {
	Name    string        `json:"name,omitempty"`
	Process string        `json:"process,omitempty"`
	Engine  string        `json:"engine,omitempty"` // 数据分析引擎名称
	Query   interface{}   `json:"query,omitempty"`  // 数据分析语言 Query Source
	DSL     share.DSL     `json:"-"`                // 数据分析语言 Query DSL
	Args    []interface{} `json:"args,omitempty"`
	Outs    []interface{} `json:"outs,omitempty"`
}

// Context 工作流上下文
type Context struct {
	In      []interface{}
	Res     map[string]interface{}
	Context *context.Context
	Cancel  context.CancelFunc
}
