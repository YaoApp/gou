package gou

import (
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/dbal"
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

// FliterSelect 选择字段
func (mod *Model) FliterSelect(columns []string) []interface{} {
	res := []interface{}{}
	for _, name := range columns {
		column, has := mod.Columns[name]
		if !has {
			continue
		}

		// 加密字段
		if column.Crypt == "AES" {
			icrypt := SelectCrypt(column.Crypt)
			raw, err := icrypt.Decode(name)
			if err != nil {
				exception.Err(err, 500).Throw()
			}
			res = append(res, dbal.Raw(raw))
		} else {
			res = append(res, name)
		}
	}
	return res
}

// FliterOut 输出前过滤解码
func (mod *Model) FliterOut(row maps.MapStrAny) {
	for name, value := range row {
		column, has := mod.Columns[name]

		// 删除无效字段
		if !has {
			continue
		}

		// 过滤输入信息
		column.FliterOut(value, row)
	}
}
