package gou

// Fliters 处理函数
var Fliters map[string]Helper = map[string]Helper{
	"pluck": func(args ...interface{}) interface{} { return args },
}
