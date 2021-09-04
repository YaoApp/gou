package gou

import (
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/schema"
)

// SchemaTableUpgrade 旧表数据结构差别对比后升级
func (mod *Model) SchemaTableUpgrade() {
}

// SchemaTableDiff 旧表数据结构差别对比
func (mod *Model) SchemaTableDiff() {
}

// SchemaTableCreate 创建新的数据表
func (mod *Model) SchemaTableCreate() {

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

		// 创建时间, 更新时间
		if mod.MetaData.Option.Timestamps {
			table.Timestamps()
		}

		// 软删除
		if mod.MetaData.Option.SoftDeletes {
			table.SoftDeletes()
			table.JSON("__restore_data").Null()
		}

		// 追溯ID
		if mod.MetaData.Option.Trackings || mod.MetaData.Option.Logging {
			table.BigInteger("__tracking_id").Index().Null()
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
