package gou

import (
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/schema"
)

// SchemaDiffTable 旧表数据结构差别对比
func (mod *Model) SchemaDiffTable() {
}

// SchemaCreateTable 创建新的数据表
func (mod *Model) SchemaCreateTable() {

	sch := capsule.Schema()
	err := sch.CreateTable(mod.MetaData.Table.Name, func(table schema.Blueprint) {

		// 创建字段
		for _, column := range mod.MetaData.Columns {
			col := column.SetType(table)
			column.SetOption(col)
		}

		// 创建索引
		for _, index := range mod.MetaData.Indexes {
			index.SetIndex(table)
		}
	})

	if err != nil {
		exception.Err(err, 500).Throw()
	}

	// 添加默认值
	for _, row := range mod.MetaData.Values {
		mod.MustCreate(row)
	}
}
