package gou

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
