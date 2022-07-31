package xun

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/utils"
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
	db := x.Manager.GetPrimary().DB
	driver := x.Manager.GetPrimary().Config.Driver
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
	db := x.Manager.GetPrimary().DB
	driver := x.Manager.GetPrimary().Config.Driver
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
	fliters := []string{}
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
			fliters = append(fliters, table)
		}
	}
	return fliters, nil
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

// TableCreate a table
func (x *Xun) TableCreate(name string, blueprint types.Blueprint) error {
	sch := x.Manager.Schema()
	err := sch.CreateTable(name, func(table schema.Blueprint) {

		// Create columns
		for _, column := range blueprint.Columns {
			_, err := setColumn(table, column)
			if err != nil {
				log.Error("[TableCreate] column %s %s", column.Name, err)
			}
		}

		// Create indexes
		for _, index := range blueprint.Indexes {
			err := SetIndex(table, index)
			if err != nil {
				log.Error("[TableCreate] index %s %s", index.Name, err)
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
	utils.Dump(blueprint, another)
	diff := types.NewDiff()
	return diff, nil
}

// TableSave a table, if the table exists update, otherwise create
func (x *Xun) TableSave(name string, blueprint types.Blueprint) error {
	return nil
}

// ColumnAdd add a column to the given table
func (x *Xun) ColumnAdd(name string, column types.Column) error {
	return nil
}

// ColumnAlt alter a column to the given table, if the column does not exists add it to the table
func (x *Xun) ColumnAlt(name string, column types.Column) error {
	return nil
}

// ColumnDel delete a column from the given table
func (x *Xun) ColumnDel(name string) error {
	return nil
}

// IndexAdd add a index to the given table
func (x *Xun) IndexAdd(name string, index types.Index) error {
	return nil
}

// IndexAlt alter a index to the given table, if the column does not exists add it to the table
func (x *Xun) IndexAlt(name string, index types.Index) error {
	return nil
}

// IndexDel delete a index from the given table
func (x *Xun) IndexDel(name string) error {
	return nil
}
