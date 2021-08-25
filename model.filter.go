package gou

import (
	"github.com/yaoapp/kun/maps"
)

// FliterIn 输入前过滤解码
func (mod *Model) FliterIn(row maps.MapStrAny) {
	for name, value := range row {
		column, has := mod.Columns[name]

		// 删除无效字段
		if !has {
			row.Del(name)
			continue
		}

		// 过滤输入信息
		column.FliterIn(value, row)
	}
}

// FliterOut 输出前过滤解码
func (mod *Model) FliterOut(row maps.MapStrAny) {
}
