package xun

import (
	"strings"

	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/xun/dbal/schema"
)

// TableToBlueprint cast schema.Table to types.blueprint
func TableToBlueprint(table schema.Blueprint) types.Blueprint {
	blueprint := types.New()
	prikeys := []string{}
	columns := table.GetColumns()
	indexes := table.GetIndexes()

	if table.Get().Primary != nil && len(table.Get().Primary.Columns) > 0 {
		for _, col := range table.Get().Primary.Columns {
			prikeys = append(prikeys, col.Name)
		}
	}

	// option
	createdAt, hasCreatedAt := columns["created_at"]
	updatedAt, hasUpdatedAt := columns["updated_at"]
	deletedAt, hasDeletedAt := columns["deleted_at"]
	restoreData, hasRestoreData := columns["__restore_data"]

	if hasCreatedAt && hasUpdatedAt &&
		createdAt.Type == "timestamp" &&
		updatedAt.Type == "timestamp" {
		blueprint.Option.Timestamps = true
	}

	if hasDeletedAt && hasRestoreData &&
		deletedAt.Type == "timestamp" &&
		(restoreData.Type == "text" || restoreData.Type == "json") {
		blueprint.Option.SoftDeletes = true

	}

	// Indexes
	for _, name := range table.GetIndexNames() {
		idx, has := indexes[name]
		if !has {
			continue
		}

		index := IndexToBlueprint(idx, prikeys)
		if index.Name != "" {
			blueprint.Indexes = append(blueprint.Indexes, index)
		}
	}

	// Columns
	for _, name := range table.GetColumnNames() {
		col, has := columns[name]
		if !has ||
			strings.HasPrefix(name, "__DEL") ||
			(name == "created_at" && blueprint.Option.Timestamps) ||
			(name == "updated_at" && blueprint.Option.Timestamps) ||
			(name == "deleted_at" && blueprint.Option.SoftDeletes) ||
			(name == "__restore_data" && blueprint.Option.SoftDeletes) {
			continue
		}
		blueprint.Columns = append(blueprint.Columns, ColumnToBlueprint(col, prikeys))
	}

	return blueprint
}

// IndexToBlueprint cast schema.Index to types.Index
func IndexToBlueprint(idx *schema.Index, prikeys []string) types.Index {
	index := types.Index{Columns: []string{}}
	if len(idx.Columns) <= 1 &&
		(idx.Type == "index" || idx.Type == "unique" || idx.Type == "primary") {
		return index
	}

	index.Name = idx.Name
	index.Type = idx.Type
	if idx.Comment != nil {
		index.Comment = *idx.Comment
	}

	for _, col := range idx.Columns {
		index.Columns = append(index.Columns, col.Name)
	}
	return index
}

// ColumnToBlueprint cast schema.Column to types.Column
func ColumnToBlueprint(col *schema.Column, prikeys []string) types.Column {
	column := types.Column{Name: col.Name}

	primary := ""
	if len(prikeys) == 1 {
		primary = prikeys[0]
	}

	column.Label = strings.ToUpper(col.Name)

	if col.Name == primary {
		column.Primary = true
	}

	if col.Nullable {
		column.Nullable = true
	}

	if col.Comment != nil {
		column.Comment = *col.Comment
		column.Label = column.Comment
	}

	if col.Default != nil {
		column.Default = col.Default
	}

	if col.Length != nil {
		column.Length = *col.Length
	}

	if col.Precision != nil {
		column.Precision = *col.Precision
	}

	if col.Scale != nil {
		column.Scale = *col.Scale
	}

	for _, idx := range col.Indexes {
		if len(idx.Columns) != 1 {
			continue
		}

		if idx.Type == "index" {
			column.Index = true
		}

		if idx.Type == "unique" {
			column.Unique = true
		}
	}

	parseColumnType(col, &column)
	return column
}

func parseColumnType(col *schema.Column, column *types.Column) {
	column.Type = col.Type

	switch col.Type {

	case "enum":
		column.Option = col.Option
		switch column.Default.(type) {
		case []byte:
			column.Default = strings.ReplaceAll(string(column.Default.([]byte)), "'", "")
		case string:
			column.Default = strings.ReplaceAll(column.Default.(string), "'", "")
		}
		break

	case "integer", "tinyInteger", "smallInteger", "bigInteger":
		column.Length = 0
		if column.Primary && col.Extra != nil {
			column.Type = "ID"
			column.Nullable = false
		}

		switch column.Default.(type) {
		case []byte:
			column.Default = any.Of(string(column.Default.([]byte))).CInt()
		case string:
			column.Default = any.Of(column.Default.(string)).CInt()
		}
		break

	case "float", "decimal", "double":
		column.Length = 0
		switch column.Default.(type) {
		case []byte:
			column.Default = any.Of(string(column.Default.([]byte))).CFloat()
		case string:
			column.Default = any.Of(column.Default.(string)).CFloat()
		}
		break

	case "boolean":
		if v, ok := column.Default.(string); ok {
			if v == "false" {
				column.Default = false
			} else if v == "true" {
				column.Default = true
			}
		}
		break

	case "timestamp", "datetime":
		if col.OctetLength != nil {
			// fmt.Println("OctetLength:", column.Name, *col.OctetLength)
			column.Length = *col.OctetLength
		}
		switch column.Default.(type) {
		case []byte:
			column.DefaultRaw = string(column.Default.([]byte))
			column.Default = nil
		case string:
			column.DefaultRaw = string(column.Default.(string))
			column.Default = nil
		}
		break
	case "json":
		column.Comment = strings.TrimLeft(column.Comment, "T:json|")
		column.Label = strings.TrimLeft(column.Label, "T:json|")
		break

	default:
		column.Type = col.Type
	}
}
