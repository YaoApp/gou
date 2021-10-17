package gou

import (
	"strings"

	"github.com/yaoapp/gou/query/share"
)

// Engines 已加载数据分析引擎
var Engines = map[string]share.DSL{}

// RegisterEngine 注册数据分析引擎
func RegisterEngine(name string, engine share.DSL) {
	name = strings.ToLower(name)
	Engines[name] = engine
}
