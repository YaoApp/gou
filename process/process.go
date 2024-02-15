package process

import (
	"context"
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
	if _, has := Handlers[name]; has {
		Handlers[alias] = Handlers[name]
		return
	}
	exception.New("Process: %s does not exist", 404, name).Throw()
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

// WithContext set the context
func (process *Process) WithContext(ctx context.Context) *Process {
	process.Context = ctx
	return process
}

// WithRuntime set the runtime interface
func (process *Process) WithRuntime(runtime Runtime) *Process {
	process.Runtime = runtime
	return process
}

// Dispose the process after run success
func (process *Process) Dispose() {
	if process.Runtime != nil {
		process.Runtime.Dispose()
	}

	process.Args = nil
	process.Global = nil
	process.Context = nil
	process.Runtime = nil
	process = nil
}

// handler get the process handler
func (process *Process) handler() (Handler, error) {
	if hander, has := Handlers[process.Handler]; has && hander != nil {
		return hander, nil
	}
	return nil, fmt.Errorf("Exception|404:%s Handler -> %s not found", process.Name, process.Handler)
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

	case "flows", "pipes":
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
		process.Method = fields[len(fields)-1]
		process.ID = strings.ToLower(strings.Join(fields[1:len(fields)-1], "."))
		process.Handler = strings.ToLower(fmt.Sprintf("widgets.%s.%s", process.ID, process.Method))
		break

	default:
		process.Handler = strings.ToLower(process.Name)
		break
	}

	return nil
}
