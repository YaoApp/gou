package xun

import (
	"fmt"

	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/xun/dbal/schema"
)

// setIndex
func setIndex(table schema.Blueprint, index types.Index) error {
	if index.Name == "" {
		return fmt.Errorf("missing name %v", index)
	}

	if len(index.Columns) == 0 {
		return fmt.Errorf("index %s missing columns", index.Name)
	}

	switch index.Type {
	case "unique":
		table.AddUnique(index.Name, index.Columns...)
		return nil
	case "index":
		table.AddIndex(index.Name, index.Columns...)
		return nil
	case "primary":
		table.AddPrimary(index.Columns...)
		return nil
	case "fulltext":
		table.AddFulltext(index.Name, index.Columns...)
		return nil
	}
	return fmt.Errorf("Index %s, Type %s does not support", index.Name, index.Type)
}
