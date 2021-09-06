package gou

// Filters 处理函数
var Filters map[string]Helper = map[string]Helper{
	"pluck": func(args ...interface{}) interface{} { return args },
}

// RegisterHelper 注册 helper
func RegisterHelper(name string, helper Helper) {
	Filters[name] = helper
}
