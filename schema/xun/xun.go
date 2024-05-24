package xun

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/schema"
)

// Xun the XUN datebase driver
type Xun struct{ Option }

// Option the xun database option
type Option struct {
	Name    string `json:"name"`
	Manager *capsule.Manager
}

// SetOption set the option
func (x *Xun) SetOption(option interface{}) error {
	opt, ok := option.(Option)
	if !ok {
		return fmt.Errorf("the given option is: %v,  not a XunOption", option)
	}
	x.Option = opt
	return nil
}

// Close connetion
func (x *Xun) Close() error {
	x.Manager.Connections.Range(func(key, value any) bool {
		if conn, ok := value.(*capsule.Connection); ok {
			conn.Close()
		}
		return true
	})
	return nil
}

// Create create a schema (temporary, move to @xun in the next version)
func (x *Xun) Create(name string) error {
	m, err := x.Manager.Primary()
	if err != nil {
		return err
	}

	db := m.DB
	driver := m.Config.Driver
	collation := "utf8mb4"
	if x.Manager.Option.Collation != "" {
		collation = x.Manager.Option.Collation
	}

	charset := "utf8mb4_general_ci"
	if x.Manager.Option.Charset != "" {
		charset = x.Manager.Option.Charset
	}

	switch driver {
	case "mysql":
		_, err := db.Exec(
			fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s",
				strings.ReplaceAll(name, "`", "\\`"),
				collation,
				charset,
			),
		)
		if err != nil {
			return err
		}
	case "sqlite3":
	}

	return nil
}

// Drop drop a schema (temporary, move to @xun in the next version)
func (x *Xun) Drop(name string) error {
	m, err := x.Manager.Primary()
	if err != nil {
		return err
	}

	db := m.DB
	driver := m.Config.Driver
	switch driver {
	case "mysql":
		_, err := db.Exec(
			fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", strings.ReplaceAll(name, "`", "\\`")),
		)
		if err != nil {
			return err
		}
	case "sqlite3":
	}
	return nil
}

// Tables get the list of table
func (x *Xun) Tables(prefix ...string) ([]string, error) {
	filters := []string{}
	sch := x.Manager.Schema()
	tables, err := sch.GetTables()
	if err != nil {
		return nil, err
	}
	if len(prefix) == 0 {
		return tables, nil
	}
	for _, table := range tables {
		if strings.HasPrefix(table, prefix[0]) {
			filters = append(filters, table)
		}
	}
	return filters, nil
}

// TableGet get a table blueprint
func (x *Xun) TableGet(name string) (types.Blueprint, error) {
	sch := x.Manager.Schema()
	table, err := sch.GetTable(name)
	if err != nil {
		return types.Blueprint{}, err
	}
	return TableToBlueprint(table), nil
}

// TableExists check if a table exists
func (x *Xun) TableExists(name string) (bool, error) {
	sch := x.Manager.Schema()
	has, err := sch.HasTable(name)
	if err != nil {
		return false, err
	}
	return has, nil
}

// TableCreate a table
func (x *Xun) TableCreate(name string, blueprint types.Blueprint) error {
	sch := x.Manager.Schema()
	err := sch.CreateTable(name, func(table schema.Blueprint) {

		// Create columns
		for _, column := range blueprint.Columns {
			_, err := setColumn(table, column)
			if err != nil {
				log.Error("[TableCreate] table:%s column: %s %s", name, column.Name, err)
			}
		}

		// Create indexes
		for _, index := range blueprint.Indexes {
			err := setIndex(table, index)
			if err != nil {
				log.Error("[TableCreate] table:%s index %s %s", name, index.Name, err)
			}
		}

		// +created_at, updated_at
		if blueprint.Option.Timestamps {
			table.Timestamps()
		}

		// +deleted_at
		if blueprint.Option.SoftDeletes {
			table.SoftDeletes()
			table.JSON("__restore_data").Null()
		}
	})

	return err
}

// TableDrop a table if exist
func (x *Xun) TableDrop(name string) error {
	sch := x.Manager.Schema()
	return sch.DropTableIfExists(name)
}

// TableRename a table
func (x *Xun) TableRename(name string, new string) error {
	sch := x.Manager.Schema()
	return sch.RenameTable(name, new)
}

// TableDiff compare the two tables, return the difference
func (x *Xun) TableDiff(blueprint types.Blueprint, another types.Blueprint) (types.Diff, error) {
	return types.Compare(blueprint, another)
}

// TableSave a table, if the table exists update, otherwise create
func (x *Xun) TableSave(name string, blueprint types.Blueprint) error {
	sch := x.Manager.Schema()
	table, err := sch.GetTable(name)
	if err != nil && !strings.Contains(err.Error(), "does not exists") {
		return err
	}

	// the table does not exists, create
	if err != nil {
		return x.TableCreate(name, blueprint)
	}

	// Update
	diff, err := types.Compare(TableToBlueprint(table), blueprint)
	if err != nil {
		return err
	}

	return diff.Apply(x, name)
}

// ColumnAdd add a column to the given table
func (x *Xun) ColumnAdd(name string, column types.Column) error {
	sch := x.Manager.Schema()
	return sch.AlterTable(name, func(table schema.Blueprint) {
		_, err := setColumn(table, column)
		if err != nil {
			log.Error("[ColumnAdd] table: %s column: %s %s", name, column.Name, err)
		}
	})
}

// ColumnAlt alter a column to the given table, if the column does not exists add it to the table
func (x *Xun) ColumnAlt(name string, column types.Column) error {
	sch := x.Manager.Schema()

	// drop index
	if column.RemoveIndex || column.Index {
		x.IndexDel(name, fmt.Sprintf("%s_index", column.Name))
	}

	// drop unique
	if column.RemoveUnique || column.Unique {
		x.IndexDel(name, fmt.Sprintf("%s_unique", column.Name))
	}

	// drop primary
	if column.RemovePrimary {
		x.IndexDel(name, "PRIMARY")
	}

	return sch.AlterTable(name, func(table schema.Blueprint) {
		_, err := setColumn(table, column)
		if err != nil {
			log.Error("[ColumnAlt] table: %s column: %s %s", name, column.Name, err)
		}
	})
}

// ColumnDel delete a column from the given table
func (x *Xun) ColumnDel(name string, columns ...string) error {
	if len(columns) == 0 {
		return fmt.Errorf("missing columns")
	}
	sch := x.Manager.Schema()
	return sch.AlterTable(name, func(table schema.Blueprint) {
		for _, col := range columns {
			table.RenameColumn(col, fmt.Sprintf("__DEL__%s", col))
		}
	})
}

// IndexAdd add a index to the given table
func (x *Xun) IndexAdd(name string, index types.Index) error {
	sch := x.Manager.Schema()
	return sch.AlterTable(name, func(table schema.Blueprint) {
		err := setIndex(table, index)
		if err != nil {
			log.Error("[IndexAdd] table: %s index: %s %s", name, index.Name, err)
		}
	})
}

// IndexDel delete a index from the given table
func (x *Xun) IndexDel(name string, indexes ...string) error {
	if len(indexes) == 0 {
		return fmt.Errorf("missing indexes")
	}
	sch := x.Manager.Schema()
	return sch.AlterTable(name, func(table schema.Blueprint) {
		table.DropIndex(indexes...)
	})
}

// IndexAlt alter a index to the given table, if the column does not exists add it to the table
func (x *Xun) IndexAlt(name string, index types.Index) error {
	log.Warn(`[IndexAlt] does not support yet`)
	return nil
}
