package gou

// Filters 处理函数
var Filters map[string]Helper = map[string]Helper{
	"pluck": func(args ...interface{}) interface{} { return args },
}
