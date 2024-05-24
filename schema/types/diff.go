package types

import (
	"reflect"
	"strings"
)

// Compare the two Blueprint, return the difference
func Compare(blueprint, another Blueprint) (Diff, error) {
	diff := NewDiff()
	diff.ColumnsDiff(blueprint, another)
	diff.IndexesDiff(blueprint, another)
	diff.OptionDiff(blueprint, another)
	return diff, nil
}

// Apply apply the changes
func (diff Diff) Apply(sch Schema, name string) error {

	// columns Add
	for _, column := range diff.Columns.Add {
		err := sch.ColumnAdd(name, column)
		if err != nil {
			return err
		}
	}

	// columns Alter
	for _, column := range diff.Columns.Alt {
		err := sch.ColumnAlt(name, column)
		if err != nil {
			return err
		}
	}

	// columns Del
	deletes := []string{}
	for _, column := range diff.Columns.Del {
		deletes = append(deletes, column.Name)
	}
	if len(deletes) > 0 {
		err := sch.ColumnDel(name, deletes...)
		if err != nil {
			return err
		}
	}

	// Indexes Add
	for _, index := range diff.Indexes.Add {
		err := sch.IndexAdd(name, index)
		if err != nil {
			return err
		}
	}

	// Indexes Del
	deletes = []string{}
	for _, index := range diff.Indexes.Del {
		deletes = append(deletes, index.Name)
	}

	if len(deletes) > 0 {
		err := sch.IndexDel(name, deletes...)
		if err != nil {
			return err
		}
	}

	return nil
}

// OptionDiff find the option difference
func (diff *Diff) OptionDiff(blueprint, another Blueprint) {
	if blueprint.Option.SoftDeletes != another.Option.SoftDeletes {
		diff.Option["soft_deletes"] = another.Option.SoftDeletes
	}

	if blueprint.Option.Timestamps != another.Option.Timestamps {
		diff.Option["timestamps"] = another.Option.Timestamps
	}
}

// IndexesDiff find the index difference
func (diff *Diff) IndexesDiff(blueprint, another Blueprint) {
	mapping := blueprint.IndexesMapping()
	mappingAnother := another.IndexesMapping()
	diff.indexesDel(blueprint.Indexes, mappingAnother)
	diff.indexesAddAlt(another.Indexes, mapping)

}

func (diff *Diff) indexesDel(indexes []Index, another map[string]Index) {
	for _, index := range indexes {
		_, nameHas := another[index.Name]
		_, hashHas := another[index.Hash()]
		if !nameHas && !hashHas {
			diff.Indexes.Del = append(diff.Indexes.Del, index)
		}
	}
}

func (diff *Diff) indexesAddAlt(indexes []Index, blueprint map[string]Index) {
	for _, index := range indexes {
		_, has := blueprint[index.Name]
		// if !has {
		// 	origin, has = blueprint[index.Hash()]
		// }

		if !has {
			diff.Indexes.Add = append(diff.Indexes.Add, index)
			continue
		}
		// index.Origin = index.Name
		// if origin.Hash() != index.Hash() {
		// 	diff.Indexes.Alt = append(diff.Indexes.Alt, index)
		// 	continue
		// }
	}
}

// ColumnsDiff find the column difference
func (diff *Diff) ColumnsDiff(blueprint, another Blueprint) {
	mapping := blueprint.ColumnsMapping()
	mappingAnother := another.ColumnsMapping()
	diff.columnsDel(blueprint.Columns, mappingAnother)
	diff.columnsAddAlt(another.Columns, mapping)
}

func (diff *Diff) columnsDel(columns []Column, another map[string]Column) {
	for _, column := range columns {
		_, nameHas := another[column.Name]
		// _, hashHas := another[column.Hash()]
		if !nameHas {
			diff.Columns.Del = append(diff.Columns.Del, column)
		}
	}
}

func (diff *Diff) columnsAddAlt(columns []Column, blueprint map[string]Column) {
	for _, column := range columns {

		// Ignore created_at, updated_at, deleted_at
		if column.Type == "timestamp" &&
			(column.Name == "created_at" || column.Name == "updated_at" || column.Name == "deleted_at") {
			continue
		}

		origin, has := blueprint[column.Name]
		// if !has {
		// 	origin, has = blueprint[column.Hash()]
		// }

		if !has {
			diff.Columns.Add = append(diff.Columns.Add, column)
			continue
		}

		// column.Origin = origin.Name
		// if origin.Name != column.Name {
		// 	diff.Columns.Alt = append(diff.Columns.Alt, column)
		// 	continue
		// }

		if strings.ToLower(origin.Type) == "id" {
			continue
		}

		if strings.ToLower(origin.Type) == "json" {
			continue
		}

		if strings.ToLower(origin.Type) != strings.ToLower(column.Type) {
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.Type == "enum" && !reflect.DeepEqual(origin.Option, column.Option) {
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.Type == "string" && origin.Length != column.Length && column.Length > 0 {
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.Default != column.Default {
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.DefaultRaw != column.DefaultRaw {
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.Nullable != column.Nullable {
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.Index != column.Index {
			column.RemoveIndex = true
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.Unique != column.Unique {
			column.RemoveUnique = true
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}

		if origin.Primary != column.Primary {
			column.RemovePrimary = true
			diff.Columns.Alt = append(diff.Columns.Alt, column)
			continue
		}
	}
}
