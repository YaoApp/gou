package gou

import (
	"fmt"
	"strings"

	"github.com/yaoapp/kun/exception"
)

// Caller 运行器
type Caller struct {
	Name    string
	Type    string
	Class   string
	Method  string
	Args    []interface{}
	Handler func(caller *Caller) interface{}
}

// ModelHandlers 模型运行器
var ModelHandlers = map[string]func(caller *Caller) interface{}{
	"find": callerFind,
}

// NewCaller 创建运行器
func NewCaller(name string, args ...interface{}) *Caller {
	caller := &Caller{Name: name, Args: args}
	caller.extraProcess()
	return caller
}

// Run 运行方法
func (caller *Caller) Run() interface{} {
	return caller.Handler(caller)
}

// extraProcess 解析执行方法  name = "models.user.Find", name = "plugins.user.Login"
// return type=models, name=login, class=user
func (caller *Caller) extraProcess() {
	namer := strings.Split(caller.Name, ".")
	last := len(namer) - 1
	if last < 2 {
		exception.New(
			fmt.Sprintf("Process:%s 格式错误", caller.Name),
			400,
		).Throw()
	}
	caller.Type = strings.ToLower(namer[0])
	caller.Class = strings.ToLower(strings.Join(namer[1:last], "."))
	caller.Method = strings.ToLower(namer[last])
	if caller.Type == "plugins" { // Plugin
		caller.Handler = callerExec
	} else if caller.Type == "models" { // Model
		handler, has := ModelHandlers[caller.Method]
		if !has {
			exception.New("%s 方法不存在", 404, caller.Method).Throw()
		}
		caller.Handler = handler
	}
}

// validateArgs( args )
func (caller *Caller) validateArgNums(length int) {
	if len(caller.Args) < length {
		exception.New(
			fmt.Sprintf("Model:%s%s(args...); 参数错误", caller.Class, caller.Name),
			400,
		).Throw()
	}
}

// callerExec 运行插件中的方法
func callerExec(caller *Caller) interface{} {
	mod := SelectPluginModel(caller.Class)
	res, err := mod.Exec(caller.Method, caller.Args...)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

// callerFind 运行模型 MustFind
func callerFind(caller *Caller) interface{} {
	caller.validateArgNums(2)
	mod := Select(caller.Class)
	params, ok := caller.Args[1].(QueryParam)
	if !ok {
		params = QueryParam{}
	}
	return mod.MustFind(caller.Args[0], params)
}
