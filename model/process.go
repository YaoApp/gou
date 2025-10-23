package model

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/str"
)

// ModelHandlers 模型运行器
var ModelHandlers = map[string]process.Handler{

	// Atomic operations
	"find":                processFind,
	"get":                 processGet,
	"paginate":            processPaginate,
	"count":               processCount,
	"create":              processCreate,
	"update":              processUpdate,
	"save":                processSave,
	"delete":              processDelete,
	"destroy":             processDestroy,
	"insert":              processInsert,
	"updatewhere":         processUpdateWhere,
	"deletewhere":         processDeleteWhere,
	"destroywhere":        processDestroyWhere,
	"eachsave":            processEachSave,
	"eachsaveafterdelete": processEachSaveAfterDelete,
	"upsert":              processUpsert,

	// Select operations - will be deprecated
	"selectoption": processSelectOption,

	// Snapshot operations
	"takesnapshot":            processTakeSnapshot,
	"restoresnapshot":         processRestoreSnapshot,
	"restoresnapshotbyrename": processRestoreSnapshotByRename,
	"dropsnapshot":            processDropSnapshot,
	"snapshotexists":          processSnapshotExists,

	// DSL operations
	"migrate":  processMigrate,
	"load":     processLoad,
	"reload":   processReload,
	"metadata": processGetMetaData,
	"read":     processRead,
	"exists":   processExists,
}

func init() {

	// Model instance
	process.RegisterGroup("models", ModelHandlers)

	// Model DSL operations
	process.RegisterGroup("model", map[string]process.Handler{
		"list":    processList,
		"read":    processRead,
		"dsl":     processDSL,
		"exists":  processModelExists,
		"reload":  processModelReload,
		"migrate": processModelMigrate,
		"load":    processModelLoad,
		"unload":  processModelUnload,
	})
}

// processModelReload reload the model
func processModelReload(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	mod := Select(id)
	_, err := mod.Reload()
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return nil
}

// processModelMigrate migrate the model
func processModelMigrate(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	mod := Select(id)
	return mod.Migrate(process.ArgsBool(1))
}

// processModelLoad load the model
func processModelLoad(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	id := process.ArgsString(0)
	source := process.ArgsString(1)
	_, err := LoadSourceSync([]byte(source), id, "")
	return err
}

// processModelUnload unload the model
func processModelUnload(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	rwlock.Lock()
	defer rwlock.Unlock()
	delete(Models, id)
	return nil
}

// processModelExists Check if the model is loaded
func processModelExists(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	return Exists(id)
}

// processDSL get the model dsl
func processDSL(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)
	options := process.ArgsMap(1)
	mod := Select(id)
	data := map[string]interface{}{
		"id":          id,
		"name":        mod.MetaData.Name,
		"description": mod.MetaData.Table.Comment,
		"file":        mod.File,
		"table":       mod.MetaData.Table,
		"primary":     mod.PrimaryKey,
	}
	if v, ok := options["metadata"].(bool); ok && v {
		data["metadata"] = mod.MetaData
	}
	if v, ok := options["columns"].(bool); ok && v {
		data["columns"] = mod.Columns
	}
	return data
}

// Return the loaded models
func processList(process *process.Process) interface{} {
	models := []map[string]interface{}{}
	options := process.ArgsMap(0)
	withMetadata := false
	withColumns := false
	if v, ok := options["metadata"].(bool); ok && v {
		withMetadata = v
	}

	if v, ok := options["columns"].(bool); ok && v {
		withColumns = v
	}

	for _, model := range Models {
		file := model.File
		if !strings.HasPrefix(file, "/") {
			file = fmt.Sprintf("/models/%s", file)
		}

		description := ""
		if model.MetaData.Table.Comment != "" {
			description = model.MetaData.Table.Comment
		}

		data := map[string]interface{}{
			"id":          model.ID,
			"name":        model.MetaData.Name,
			"description": description,
			"file":        file,
			"table":       model.MetaData.Table,
			"primary":     model.PrimaryKey,
		}

		if withMetadata {
			data["metadata"] = model.MetaData
		}

		if withColumns {
			data["columns"] = model.Columns
		}
		models = append(models, data)
	}
	return models
}

// processFind 运行模型 MustFind
func processFind(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	mod := Select(process.ID)
	params, ok := AnyToQueryParam(process.Args[1])
	if !ok {
		params = QueryParam{}
	}
	return mod.MustFind(process.Args[0], params)
}

// processGet 运行模型 MustGet
func processGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
	}
	return mod.MustGet(params)
}

// processPaginate 运行模型 MustPaginate
func processPaginate(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	mod := Select(process.ID)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
	}

	page := any.Of(process.Args[1]).CInt()
	pagesize := any.Of(process.Args[2]).CInt()
	return mod.MustPaginate(params, page, pagesize)
}

// processCount 运行模型 MustCount
func processCount(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
	}
	return mod.MustCount(params)
}

// processCreate 运行模型 MustCreate
func processCreate(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	row := any.Of(process.Args[0]).Map().MapStrAny
	return mod.MustCreate(row)
}

// processUpdate 运行模型 MustUpdate
func processUpdate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	mod := Select(process.ID)
	id := process.Args[0]
	row := any.Of(process.Args[1]).Map().MapStrAny
	mod.MustUpdate(id, row)
	return nil
}

// processSave 运行模型 MustSave
func processSave(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	row := any.Of(process.Args[0]).Map().MapStrAny
	return mod.MustSave(row)
}

// processDelete 运行模型 MustDelete
func processDelete(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	mod.MustDelete(process.Args[0])
	return nil
}

// processDestroy 运行模型 MustDestroy
func processDestroy(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	mod.MustDestroy(process.Args[0])
	return nil
}

// processInsert 运行模型 MustInsert
func processInsert(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	mod := Select(process.ID)
	var colums = []string{}
	colums, ok := process.Args[0].([]string)
	if !ok {
		anyColums, ok := process.Args[0].([]interface{})
		if !ok {
			exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
		}
		for _, col := range anyColums {
			colums = append(colums, string(str.Of(col)))
		}
	}

	var rows = [][]interface{}{}
	rows, ok = process.Args[1].([][]interface{})
	if !ok {
		anyRows, ok := process.Args[1].([]interface{})
		if !ok {
			exception.New("第2个查询参数错误 %v", 400, process.Args[1]).Throw()
		}
		for _, anyRow := range anyRows {

			row, ok := anyRow.([]interface{})
			if !ok {
				exception.New("第2个查询参数错误 %v", 400, process.Args[1]).Throw()
			}
			rows = append(rows, row)
		}
	}

	mod.MustInsert(colums, rows)
	return nil
}

// processUpdateWhere 运行模型 MustUpdateWhere
func processUpdateWhere(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	mod := Select(process.ID)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
	}
	row := any.Of(process.Args[1]).Map().MapStrAny
	return mod.MustUpdateWhere(params, row)
}

// processDeleteWhere 运行模型 MustDeleteWhere
func processDeleteWhere(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		params = QueryParam{}
	}
	return mod.MustDeleteWhere(params)
}

// processDestroyWhere 运行模型 MustDestroyWhere
func processDestroyWhere(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		params = QueryParam{}
	}
	return mod.MustDestroyWhere(params)
}

// processEachSave 运行模型 MustEachSave
func processEachSave(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	rows := process.ArgsRecords(0)
	eachrow := map[string]interface{}{}
	if process.NumOfArgsIs(2) {
		eachrow = process.ArgsMap(1)
	}
	return mod.MustEachSave(rows, eachrow)
}

// processEachSaveAfterDelete 运行模型 MustDeleteWhere 后 MustEachSave
func processEachSaveAfterDelete(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	mod := Select(process.ID)
	eachrow := map[string]interface{}{}
	ids := []int{}
	if v, ok := process.Args[0].([]int); ok {
		ids = v
	} else if v, ok := process.Args[0].([]interface{}); ok {
		for _, i := range v {
			ids = append(ids, any.Of(i).CInt())
		}
	}
	rows := process.ArgsRecords(1)
	if process.NumOfArgsIs(3) {
		eachrow = process.ArgsMap(2)
	}
	if len(ids) > 0 {
		mod.MustDeleteWhere(QueryParam{Wheres: []QueryWhere{{Column: "id", OP: "in", Value: ids}}})
	}
	return mod.MustEachSave(rows, eachrow)
}

// processSelectOption 运行模型 MustGet
func processSelectOption(process *process.Process) interface{} {

	// TODO: will be deprecated
	color.Yellow("SelectOption will be deprecated, use Get instead")

	mod := Select(process.ID)
	keyword := "%%"
	if process.NumOfArgs() > 0 {
		keyword = fmt.Sprintf("%%%s%%", process.ArgsString(0))
	}
	name := "name"
	if process.NumOfArgs() > 1 && process.ArgsString(1, "name") != "" {
		name = process.ArgsString(1, "name")
	}

	value := "id"
	if process.NumOfArgs() > 2 && process.ArgsString(2, "id") != "" {
		value = process.ArgsString(2, "id")
	}

	limit := 20
	if process.NumOfArgs() > 3 && process.ArgsInt(3, 20) > 0 {
		limit = process.ArgsInt(3)
	}

	query := QueryParam{
		Select: []interface{}{name, value},
		Wheres: []QueryWhere{
			{Column: name, OP: "like", Value: keyword},
		},
		Limit: limit,
	}

	data := mod.MustGet(query)
	res := []maps.StrAny{}
	for _, row := range data {
		new := maps.StrAny{
			"name": row.Get(name),
			"id":   row.Get(value),
		}
		res = append(res, new)
	}

	return res
}

// processMigrate migrate model
func processMigrate(process *process.Process) interface{} {
	mod := Select(process.ID)
	if process.NumOfArgs() > 0 {
		return mod.Migrate(process.ArgsBool(0))
	}
	return mod.Migrate(false)
}

// processLoad load model
func processLoad(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ID
	file := process.ArgsString(0)
	if process.NumOfArgs() > 1 {
		source := process.ArgsString(1)
		_, err := LoadSourceSync([]byte(source), id, file)
		return err
	}
	_, err := LoadSync(file, id)
	return err
}

func processReload(process *process.Process) interface{} {
	mod := Select(process.ID)
	_, err := mod.Reload()
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return nil
}

// processGetMetaData get the model meta data
func processGetMetaData(process *process.Process) interface{} {
	return GetMetaData(process.ID)
}

// processRead read the model source
func processRead(process *process.Process) interface{} {
	mod := Select(process.ID)
	return string(mod.source)
}

// processExists Check if the model is loaded
func processExists(process *process.Process) interface{} {
	return Exists(process.ID)
}

// processTakeSnapshot Create a snapshot of the model
func processTakeSnapshot(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	inMemory := process.ArgsBool(0)
	name, err := mod.TakeSnapshot(inMemory)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return name
}

// processRestoreSnapshot Restore the model from the snapshot
func processRestoreSnapshot(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	name := process.ArgsString(0)
	err := mod.RestoreSnapshot(name)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return nil
}

// processRestoreSnapshotByRename Restore the model from the snapshot by renaming
func processRestoreSnapshotByRename(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	name := process.ArgsString(0)
	err := mod.RestoreSnapshotByRename(name)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return nil
}

// processDropSnapshot Drop the snapshot table
func processDropSnapshot(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	name := process.ArgsString(0)
	err := mod.DropSnapshotTable(name)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return nil
}

// processSnapshotExists Check if the snapshot table exists
func processSnapshotExists(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	mod := Select(process.ID)
	name := process.ArgsString(0)
	exists, err := mod.SnapshotExists(name)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return exists
}

// processUpsert runs the model Upsert method
func processUpsert(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	mod := Select(process.ID)
	row := any.Of(process.Args[0]).Map().MapStrAny

	// Validate uniqueBy parameter
	var uniqueBy []interface{}
	switch v := process.Args[1].(type) {
	case string:
		uniqueBy = []interface{}{v}
	case []string:
		for _, s := range v {
			uniqueBy = append(uniqueBy, s)
		}
	case []interface{}:
		uniqueBy = v
	default:
		exception.New("uniqueBy parameter must be a string or string array", 400).Throw()
	}

	if len(uniqueBy) == 0 {
		exception.New("uniqueBy parameter cannot be empty", 400).Throw()
	}

	// Process updateColumns parameter if provided
	var updateColumns []interface{}
	if process.NumOfArgsIs(3) {
		switch v := process.Args[2].(type) {
		case []string:
			for _, s := range v {
				updateColumns = append(updateColumns, s)
			}
		case []interface{}:
			updateColumns = v
		default:
			exception.New("updateColumns parameter must be a string array", 400).Throw()
		}
	}

	id, err := mod.Upsert(row, uniqueBy, updateColumns)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return id
}
