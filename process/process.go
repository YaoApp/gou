package process

import (
	"fmt"
	"strings"

	"github.com/yaoapp/kun/exception"
)

// Handlers ProcessHanlders
var Handlers = map[string]Handler{}

// New make a new process
func New(name string, args ...interface{}) *Process {
	process, err := Of(name, args...)
	if err != nil {
		exception.New("%s", 500, err.Error()).Throw()
	}
	return process
}

// Of make a new process and return error
func Of(name string, args ...interface{}) (*Process, error) {
	process := &Process{Name: name, Args: args, Global: map[string]interface{}{}}
	err := process.make()
	if err != nil {
		return nil, err
	}
	return process, nil
}

// Run the process
func (process *Process) Run() interface{} {
	hd, err := process.handler()
	if err != nil {
		exception.New("%s", 500, err.Error()).Throw()
		return nil
	}
	return hd(process)
}

// Exec execute the process and return error
func (process *Process) Exec() (value interface{}, err error) {

	var hd Handler
	hd, err = process.handler()
	if err != nil {
		return
	}

	defer func() { err = exception.Catch(recover()) }()
	value = hd(process)
	return
}

// Register register a process handler
func Register(name string, handler Handler) {
	name = strings.ToLower(name)
	Handlers[name] = handler
}

// RegisterGroup register a process handler group
func RegisterGroup(name string, group map[string]Handler) {
	for method, handler := range group {
		id := fmt.Sprintf("%s.%s", strings.ToLower(name), strings.ToLower(method))
		Handlers[id] = handler
	}
}

// Alias set an alias a process
func Alias(name string, alias string) {
	name = strings.ToLower(name)
	alias = strings.ToLower(alias)
	Handlers[alias] = Handlers[name]
}

// WithSID set the session id
func (process *Process) WithSID(sid string) *Process {
	process.Sid = sid
	return process
}

// WithGlobal set the global vars
func (process *Process) WithGlobal(global map[string]interface{}) *Process {
	process.Global = global
	return process
}

// handler get the process handler
func (process *Process) handler() (Handler, error) {
	if hander, has := Handlers[process.Handler]; has {
		return hander, nil
	}
	return nil, fmt.Errorf("Exception|404:%s (%s) not found", process.Name, process.Handler)
}

// make parse the process
func (process *Process) make() error {
	fields := strings.Split(process.Name, ".")
	if len(fields) < 2 {
		return fmt.Errorf("Exception|404:%s not found", process.Name)
	}

	process.Group = fields[0]
	switch process.Group {

	case "models", "schemas", "stores", "fs", "tasks", "schedules":
		// models.user.pet.Find
		process.Method = fields[len(fields)-1]
		process.ID = strings.ToLower(strings.Join(fields[1:len(fields)-1], "."))
		process.Handler = strings.ToLower(fmt.Sprintf("%s.%s", process.Group, process.Method))
		break

	case "flows":
		process.Handler = process.Group
		process.ID = strings.ToLower(strings.Join(fields[1:], "."))
		break

	case "aigcs":
		if len(fields) < 2 {
			return fmt.Errorf("Exception|404:%s not found", process.Name)
		}
		// aigcs.translate
		process.Handler = strings.ToLower(process.Group)
		process.ID = strings.ToLower(strings.ToLower(strings.Join(fields[1:], ".")))
		break

	case "scripts", "studio", "plugins":
		if len(fields) < 3 {
			return fmt.Errorf("Exception|404:%s not found", process.Name)
		}
		// scripts.runtime.basic.Hello
		process.Handler = strings.ToLower(process.Group)
		process.ID = strings.ToLower(strings.ToLower(strings.Join(fields[1:len(fields)-1], ".")))
		process.Method = fields[len(fields)-1]
		break

	case "session", "http":
		process.Method = fields[len(fields)-1]
		process.Handler = strings.ToLower(fmt.Sprintf("%s.%s", process.Group, process.Method))
		break

	case "widgets":
		// widgets.reload, widgets.dyform.save
		process.Method = fields[len(fields)-1]
		process.ID = strings.ToLower(strings.Join(fields[1:len(fields)-1], "."))
		process.Handler = strings.ToLower(fmt.Sprintf("widgets.%s.%s", process.ID, process.Method))
		fmt.Println("widgets:: ", process.Handler)
		break
	default:
		process.Handler = strings.ToLower(process.Name)
		break
	}

	return nil
}

// extraProcess 解析执行方法  name = "models.user.Find", name = "plugins.user.Login"
// return type=models, name=login, class=user
// @下一版优化这个函数
// func (process *Process) make() error {
// 	namer := strings.Split(process.Name, ".")
// 	last := len(namer) - 1

// 	if _, has := whitelist[namer[0]]; last < 2 && !has {
// 		exception.New("Process:%s format error", 400, process.Name).Throw()
// 	}

// 	process.Type = strings.ToLower(namer[0])
// 	if last > 1 {
// 		process.Class = strings.ToLower(strings.Join(namer[1:last], "."))
// 		process.Method = strings.ToLower(namer[last])
// 	} else {
// 		process.Class = strings.ToLower(namer[1])
// 		process.Method = ""
// 	}

// // Handler groups
// if handlers, has := HandlerGroups[process.Type]; has {
// 	method := process.Method
// 	if method == "" {
// 		method = process.Class
// 	}

// 	process.Name = strings.ToLower(process.Name)
// 	handler, has := handlers[method]
// 	if !has {
// 		exception.New("%s: %s %s does not exist", 404, process.Type, process.Name, process.Method).Throw()
// 	}
// 	process.Handler = handler
// 	return
// }

// switch process.Type {

// case "plugins":
// 	process.Name = strings.ToLower(process.Name)
// 	process.Handler = processPlugin
// 	return

// case "flows":
// 	process.Name = strings.ToLower(process.Name)
// 	process.Handler = processFlow
// 	return

// case "scripts":
// 	process.Class = strings.ToLower(strings.Join(namer[1:last], "."))
// 	process.Method = namer[last]
// 	process.Handler = processScript
// 	return

// case "session":
// 	process.Method = strings.ToLower(namer[last])
// 	process.Handler = processSession
// 	return

// case "stores":
// 	process.Name = strings.ToLower(process.Name)
// 	handler, has := StoreHandlers[process.Method]
// 	if !has {
// 		exception.New("Store: %s %s does not exist", 404, process.Name, process.Method).Throw()
// 	}
// 	process.Handler = handler
// 	return

// case "widgets":

// 	if widgetHanlders, has := WidgetCustomHandlers[strings.ToLower(process.Class)]; has {
// 		if handler, has := widgetHanlders[strings.ToLower(process.Method)]; has {
// 			process.Name = strings.ToLower(process.Name)
// 			process.Handler = handler
// 			return
// 		}
// 	}
// 	process.Name = strings.ToLower(process.Name)
// 	handler, has := WidgetHandlers[strings.ToLower(process.Method)]
// 	if !has {
// 		exception.New("Widget: %s %s does not exist", 404, process.Name, process.Method).Throw()
// 	}
// 	process.Handler = handler
// 	return

// case "schemas":
// 	process.Name = strings.ToLower(process.Name)
// 	handler, has := SchemaHandlers[process.Method]
// 	if !has {
// 		exception.New("Schema: %s %s does not exist", 404, process.Name, process.Method).Throw()
// 	}
// 	process.Handler = handler
// 	return

// case "tasks":
// 	process.Name = strings.ToLower(process.Name)
// 	handler, has := TaskHandlers[process.Method]
// 	if !has {
// 		exception.New("Task: %s %s does not exist", 404, process.Name, process.Method).Throw()
// 	}
// 	process.Handler = handler
// 	return

// case "schedules":
// 	process.Name = strings.ToLower(process.Name)
// 	handler, has := ScheduleHandlers[process.Method]
// 	if !has {
// 		exception.New("Schedule: %s %s does not exist", 404, process.Name, process.Method).Throw()
// 	}
// 	process.Handler = handler
// 	return

// case "models":
// 	process.Name = strings.ToLower(process.Name)
// 	handler, has := ModelHandlers[process.Method]
// 	if !has {
// 		exception.New("Model: %s %s does not exist", 404, process.Name, process.Method).Throw()
// 	}
// 	process.Handler = handler
// 	return

// default:
// 	if handler, has := Handlers[strings.ToLower(process.Name)]; has {
// 		process.Name = strings.ToLower(process.Name)
// 		process.Handler = handler
// 		return
// 	} else if handler, has := Handlers[process.Type]; has {
// 		process.Name = strings.ToLower(process.Name)
// 		process.Handler = handler
// 		return
// 	}
// }

// exception.New("%s does not found", 404, process.Name).Throw()
// }

// processPlugin 运行插件中的方法
// func processPlugin(process *Process) interface{} {
// 	plugin := SelectPluginModel(process.Class)
// 	res, err := plugin.Exec(process.Method, process.Args...)
// 	if err != nil {
// 		exception.Err(err, 500).Throw()
// 	}
// 	return res.MustValue()
// }

// // processFlow 运行工作流
// func processFlow(process *Process) interface{} {
// 	name := strings.TrimPrefix(process.Name, "flows.")
// 	flow := SelectFlow(name).WithGlobal(process.Global).WithSID(process.Sid)
// 	return flow.Exec(process.Args...)
// }

// // processScript 运行脚本中定义的处理器
// func processScript(process *Process) interface{} {
// 	res, err := Yao.New(process.Class, process.Method).
// 		WithGlobal(process.Global).
// 		WithSid(process.Sid).
// 		Call(process.Args...)

// 	if err != nil {
// 		message := err.Error()

// 		// JS Exception
// 		if strings.HasPrefix(message, "Exception|") {
// 			message = strings.Replace(message, "Exception|", "", -1)
// 			values := strings.Split(message, ":")
// 			if len(values) == 2 {
// 				code := 500
// 				if v, err := strconv.Atoi(values[0]); err == nil {
// 					code = v
// 				}
// 				message = strings.TrimSpace(values[1])
// 				exception.New(message, code).Throw()
// 			}
// 		}

// 		// Other
// 		code := 500
// 		values := strings.Split(message, "|")
// 		if len(values) == 2 {
// 			if v, err := strconv.Atoi(values[0]); err == nil {
// 				code = v
// 			}
// 			message = values[0]
// 		}

// 		exception.New(message, code).Throw()
// 	}
// 	return res
// }

// processSession
// **WARN** refactor in the next version
// func processSession(process *Process) interface{} {

// 	if process.Method == "start" {
// 		process.Sid = session.ID()
// 		return process.Sid
// 	}

// 	ss := session.Global()

// 	if process.Sid != "" {
// 		ss = ss.ID(process.Sid)
// 	}

// 	switch process.Method {

// 	case "id":
// 		return process.Sid

// 	case "get":
// 		process.ValidateArgNums(1)
// 		if process.NumOfArgs() == 2 {
// 			ss = session.Global().ID(process.ArgsString(1))
// 			return ss.MustGet(process.ArgsString(0))
// 		}
// 		return ss.MustGet(process.ArgsString(0))

// 	case "set":
// 		process.ValidateArgNums(2)
// 		if process.NumOfArgs() == 3 {
// 			ss.MustSetWithEx(process.ArgsString(0), process.Args[1], time.Duration(process.ArgsInt(2))*time.Second)
// 			return nil

// 		} else if process.NumOfArgs() == 4 {
// 			ss = session.Global().ID(process.ArgsString(3))
// 			ss.MustSetWithEx(process.ArgsString(0), process.Args[1], time.Duration(process.ArgsInt(2))*time.Second)
// 		}

// 		ss.MustSet(process.ArgsString(0), process.Args[1])
// 		return nil

// 	case "setmany":
// 		process.ValidateArgNums(1)
// 		if process.NumOfArgs() == 2 {
// 			ss.MustSetManyWithEx(process.ArgsMap(0), time.Duration(process.ArgsInt(1))*time.Second)
// 			return nil

// 		} else if process.NumOfArgs() == 3 {
// 			ss = session.Global().ID(process.ArgsString(2))
// 			ss.MustSetManyWithEx(process.ArgsMap(0), time.Duration(process.ArgsInt(1))*time.Second)
// 			return nil
// 		}
// 		ss.MustSetMany(process.ArgsMap(0))
// 		return nil
// 	case "dump":
// 		if process.NumOfArgs() == 1 {
// 			ss = session.Global().ID(process.ArgsString(0))
// 			return ss.MustDump()
// 		}
// 		return ss.MustDump()
// 	}
// 	return nil
// }
