package model

import (
	"github.com/yaoapp/xun/dbal/schema"
)

// SetIndex 设置索引
func (index Index) SetIndex(table schema.Blueprint) {
	switch index.Type {
	case "unique":
		table.AddUnique(index.Name, index.Columns...)
		break
	case "index":
		table.AddIndex(index.Name, index.Columns...)
		break
	case "primary":
		table.AddPrimary(index.Columns...)
		break
	case "fulltext":
		table.AddFulltext(index.Name, index.Columns...)
		break
	}
}
