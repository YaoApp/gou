package schema

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// SchemaHandlers Processes
var SchemaHandlers = map[string]process.Handler{
	"create": processSchemaCreate,
	"drop":   processSchemaDrop,

	"tables":      processSchemaTables,
	"tableget":    processSchemaTableGet,
	"tablecreate": processSchemaTableCreate,
	"tabledrop":   processSchemaTableDrop,
	"tablerename": processSchemaTableRename,
	"tablediff":   processSchemaTableDiff,
	"tablesave":   processSchemaTableSave,

	"columnadd": processSchemaColumnAdd,
	"columnalt": processSchemaColumnAlt,
	"columndel": processSchemaColumnDel,

	"indexadd": processSchemaIndexAdd,
	"indexdel": processSchemaIndexDel,
}

func init() {
	process.RegisterGroup("schemas", SchemaHandlers)
}

// schemas.<connector>.Create
// args: [name:String]
// Create a schema
func processSchemaCreate(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	err := sch.Create(name)
	if err != nil {
		log.Error("schemas.%s.Create: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.Drop
// args: [name:String]
// Drop a schema
func processSchemaDrop(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	err := sch.Drop(name)
	if err != nil {
		log.Error("schemas.%s.Drop: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.Tables
// args: [preifx:String<optional>]
// Tables Get the list of table
func processSchemaTables(process *process.Process) interface{} {
	sch := Use(process.ID)
	preifx := []string{}
	if process.NumOfArgsIs(1) {
		preifx = append(preifx, process.ArgsString(0))
	}
	tables, err := sch.Tables(preifx...)
	if err != nil {
		log.Error("schemas.%s.Tables: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return tables
}

// schemas.<connector>.TableGet
// args: [tableName:String]
// TableGet Get a table blueprint
func processSchemaTableGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	table, err := sch.TableGet(name)
	if err != nil {
		log.Error("schemas.%s.TableGet: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return table
}

// schemas.<connector>.TableCreate
// args: [tableName:String, blueprint:Blueprint]
// TableCreate Create a table
func processSchemaTableCreate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	blueprint, err := types.NewAny(process.Args[1])
	if err != nil {
		log.Error("schemas.%s.TableCreate: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	err = sch.TableCreate(name, blueprint)
	if err != nil {
		log.Error("schemas.%s.TableCreate: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.TableDrop
// args: [tableName:String]
// TableDrop Drop a table
func processSchemaTableDrop(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	err := sch.TableDrop(name)
	if err != nil {
		log.Error("schemas.%s.TableDrop: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.TableRename
// args: [tableName:String, newTableName:String]
// TableRename Rename a table
func processSchemaTableRename(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	new := process.ArgsString(1)
	err := sch.TableRename(name, new)
	if err != nil {
		log.Error("schemas.%s.TableRename: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.TableDiff
// args: [blueprint:Blueprint, anotherBlueprint:Blueprint]
// TableDiff Compare the two tables, return the difference
func processSchemaTableDiff(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	blueprint, err := types.NewAny(process.Args[0])
	if err != nil {
		log.Error("schemas.%s.TableDiff: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	anotherBlueprint, err := types.NewAny(process.Args[1])
	if err != nil {
		log.Error("schemas.%s.TableDiff: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	diff, err := sch.TableDiff(blueprint, anotherBlueprint)
	if err != nil {
		log.Error("schemas.%s.TableDiff: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	return diff
}

// schemas.<connector>.TableSave
// args: [tableName:String, blueprint:Blueprint]
// TableSave Save a table, if the table exists update, otherwise create
func processSchemaTableSave(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	blueprint, err := types.NewAny(process.Args[1])
	if err != nil {
		log.Error("schemas.%s.TableSave: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	err = sch.TableSave(name, blueprint)
	if err != nil {
		log.Error("schemas.%s.TableSave: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.ColumnAdd
// args: [tableName:String, column:Column]
// ColumnAdd Add a column to the given table
func processSchemaColumnAdd(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	column, err := types.NewColumnAny(process.Args[1])
	if err != nil {
		log.Error("schemas.%s.ColumnAdd: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	err = sch.ColumnAdd(name, column)
	if err != nil {
		log.Error("schemas.%s.ColumnAdd: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.ColumnAlt
// args: [tableName:String, column:Column]
// ColumnAlt alter a column to the given table, if the column does not exists add it to the table
func processSchemaColumnAlt(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	column, err := types.NewColumnAny(process.Args[1])
	if err != nil {
		log.Error("schemas.%s.ColumnAlt: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	err = sch.ColumnAlt(name, column)
	if err != nil {
		log.Error("schemas.%s.ColumnAlt: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.ColumnDel
// args: [tableName:String, columnName:String]
// ColumnDel delete a column from the given table
func processSchemaColumnDel(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	column := process.ArgsString(1)
	err := sch.ColumnDel(name, column)
	if err != nil {
		log.Error("schemas.%s.ColumnDel: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.IndexAdd
// args: [tableName:String, index:Index]
// IndexAdd add a index to the given table
func processSchemaIndexAdd(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	index, err := types.NewIndexAny(process.Args[1])
	if err != nil {
		log.Error("schemas.%s.IndexAdd: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	err = sch.IndexAdd(name, index)
	if err != nil {
		log.Error("schemas.%s.IndexAdd: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}

// schemas.<connector>.IndexDel
// args: [tableName:String, indexName:String]
// IndexDel delete a index from the given table
func processSchemaIndexDel(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	sch := Use(process.ID)
	name := process.ArgsString(0)
	index := process.ArgsString(1)
	err := sch.IndexDel(name, index)
	if err != nil {
		log.Error("schemas.%s.IndexDel: %s", process.ID, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}
	return nil
}
