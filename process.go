package gou

import (
	"fmt"
	"strings"

	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/str"
)

// NewProcess 创建运行器
func NewProcess(name string, args ...interface{}) *Process {
	process := &Process{Name: name, Args: args}
	process.extraProcess()
	return process
}

// RegisterProcessHandler 注册 ProcessHandler
func RegisterProcessHandler(name string, handler ProcessHandler) {
	ThirdHandlers[name] = handler
}

// Run 运行方法
func (process *Process) Run() interface{} {
	return process.Handler(process)
}

// extraProcess 解析执行方法  name = "models.user.Find", name = "plugins.user.Login"
// return type=models, name=login, class=user
func (process *Process) extraProcess() {
	namer := strings.Split(process.Name, ".")
	last := len(namer) - 1
	if last < 2 {
		exception.New(
			fmt.Sprintf("Process:%s 格式错误", process.Name),
			400,
		).Throw()
	}
	process.Type = strings.ToLower(namer[0])
	process.Class = strings.ToLower(strings.Join(namer[1:last], "."))
	process.Method = strings.ToLower(namer[last])
	if process.Type == "plugins" { // Plugin
		process.Handler = processExec
		return
	} else if process.Type == "models" { // Model
		handler, has := ModelHandlers[process.Method]
		if !has {
			exception.New("%s 方法不存在", 404, process.Method).Throw()
		}
		process.Handler = handler
		return
	} else if handler, has := ThirdHandlers[strings.ToLower(process.Name)]; has {
		process.Handler = handler
		return
	} else if handler, has := ThirdHandlers[process.Type]; has {
		process.Handler = handler
		return
	}

	exception.New("%s 未找到处理器", 404, process.Name).Throw()
}

// validateArgs( args )
func (process *Process) validateArgNums(length int) {
	if len(process.Args) < length {
		exception.New(
			fmt.Sprintf("Model:%s%s(args...); 缺少查询参数", process.Class, process.Name),
			400,
		).Throw()
	}
}

// processExec 运行插件中的方法
func processExec(process *Process) interface{} {
	mod := SelectPluginModel(process.Class)
	res, err := mod.Exec(process.Method, process.Args...)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

// processFind 运行模型 MustFind
func processFind(process *Process) interface{} {
	process.validateArgNums(2)
	mod := Select(process.Class)
	params, ok := process.Args[1].(QueryParam)
	if !ok {
		params = QueryParam{}
	}
	return mod.MustFind(process.Args[0], params)
}

// processGet 运行模型 MustGet
func processGet(process *Process) interface{} {
	process.validateArgNums(1)
	mod := Select(process.Class)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
	}
	return mod.MustGet(params)
}

// processPaginate 运行模型 MustPaginate
func processPaginate(process *Process) interface{} {
	process.validateArgNums(3)
	mod := Select(process.Class)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
	}

	page := any.Of(process.Args[1]).CInt()
	pagesize := any.Of(process.Args[2]).CInt()
	return mod.MustPaginate(params, page, pagesize)
}

// processCreate 运行模型 MustCreate
func processCreate(process *Process) interface{} {
	process.validateArgNums(1)
	mod := Select(process.Class)
	row := any.Of(process.Args[0]).Map().MapStrAny
	return mod.MustCreate(row)
}

// processUpdate 运行模型 MustUpdate
func processUpdate(process *Process) interface{} {
	process.validateArgNums(2)
	mod := Select(process.Class)
	id := process.Args[0]
	row := any.Of(process.Args[1]).Map().MapStrAny
	mod.MustUpdate(id, row)
	return nil
}

// processSave 运行模型 MustSave
func processSave(process *Process) interface{} {
	process.validateArgNums(1)
	mod := Select(process.Class)
	row := any.Of(process.Args[0]).Map().MapStrAny
	return mod.MustSave(row)
}

// processDelete 运行模型 MustDelete
func processDelete(process *Process) interface{} {
	process.validateArgNums(1)
	mod := Select(process.Class)
	mod.MustDelete(process.Args[0])
	return nil
}

// processDestroy 运行模型 MustDestroy
func processDestroy(process *Process) interface{} {
	process.validateArgNums(1)
	mod := Select(process.Class)
	mod.MustDestroy(process.Args[0])
	return nil
}

// processInsert 运行模型 MustInsert
func processInsert(process *Process) interface{} {
	process.validateArgNums(2)
	mod := Select(process.Class)
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
func processUpdateWhere(process *Process) interface{} {
	process.validateArgNums(2)
	mod := Select(process.Class)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		exception.New("第1个查询参数错误 %v", 400, process.Args[0]).Throw()
	}
	row := any.Of(process.Args[1]).Map().MapStrAny
	return mod.MustUpdateWhere(params, row)
}

// processDeleteWhere 运行模型 MustDeleteWhere
func processDeleteWhere(process *Process) interface{} {
	process.validateArgNums(1)
	mod := Select(process.Class)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		params = QueryParam{}
	}
	return mod.MustDeleteWhere(params)
}

// processDestroyWhere 运行模型 MustDestroyWhere
func processDestroyWhere(process *Process) interface{} {
	process.validateArgNums(1)
	mod := Select(process.Class)
	params, ok := AnyToQueryParam(process.Args[0])
	if !ok {
		params = QueryParam{}
	}
	return mod.MustDestroyWhere(params)
}
