package gou

// Process 运行器
type Process struct {
	Name    string
	Type    string
	Class   string
	Method  string
	Args    []interface{}
	Global  map[string]interface{} // 全局变量
	Sid     string                 // 会话ID
	Handler ProcessHandler
}

// ProcessHandler 处理程序
type ProcessHandler func(process *Process) interface{}

// ThirdHandlers 第三方处理器
var ThirdHandlers = map[string]ProcessHandler{}

// HandlerGroups registered process handler groups
var HandlerGroups = map[string]map[string]ProcessHandler{}
