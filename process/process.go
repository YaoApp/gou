package process

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
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

// NewWithContext make a new process with context
func NewWithContext(ctx context.Context, name string, args ...interface{}) *Process {
	process, err := Of(name, args...)
	if err != nil {
		exception.New("%s", 500, err.Error()).Throw()
	}
	process.Context = ctx
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

// Execute execute the process and return error only
func (process *Process) Execute() (err error) {
	if process.Context == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		process.Context = ctx
	}

	var hd Handler
	hd, err = process.handler()
	if err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			recovered := recover()
			err = exception.Catch(recovered)
			if err != nil {
				exception.DebugPrint(err, "%s", process)
			}
		}()
		value := hd(process)
		process._val = &value
	}()

	select {
	case <-process.Context.Done():
		return process.Context.Err()
	case <-done:
		return err
	}
}

// ExecuteSync execute the process synchronously in current thread (for V8 context sharing)
// This method is designed for calls from JavaScript with shared V8 context
// It executes in the current thread without creating goroutines to maintain thread affinity
func (process *Process) ExecuteSync() (err error) {
	var hd Handler
	hd, err = process.handler()
	if err != nil {
		return err
	}

	defer func() {
		recovered := recover()
		err = exception.Catch(recovered)
		if err != nil {
			exception.DebugPrint(err, "%s", process)
		}
	}()

	value := hd(process)
	process._val = &value
	return nil
}

// Release the value of the process
func (process *Process) Release() {
	process._val = nil
}

// Dispose the process after run success
func (process *Process) Dispose() {
	if process == nil {
		return
	}
	if process.Runtime != nil {
		process.Runtime.Dispose()
	}

	process.Args = nil
	process.Global = nil
	process.Context = nil
	process.Runtime = nil
	process._val = nil
}

// Value get the result of the process
func (process *Process) Value() interface{} {
	if process._val != nil {
		return *process._val
	}
	return nil
}

// Run the process
// ****
//
// This function causes a memory leak, will be disposed in the future,
// Use Execute() instead
//
// ****
func (process *Process) Run() interface{} {
	hd, err := process.handler()
	if err != nil {
		exception.New("%s", 500, err.Error()).Throw()
		return nil
	}

	defer func() { process.Release() }()
	return hd(process)
}

// Exec execute the process and return error
//
// ****
//
// This function causes a memory leak, will be disposed in the future,
// Use Execute() instead
// Example:
//
//	process := Of("models.user.pet.Find", 1, {})
//	err := process.Execute();
//	if err != nil {
//	 	// handle error
//	}
//	defer process.Release()  // or  process.Dispose() if you want to relese the runtime isolate after run success
//	result := process.Value() // Get the result
//
// ****
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

// Exists check if the process exists
func Exists(name string) bool {

	// Exclude the scripts, assistants, agents, ai, services
	if strings.HasPrefix(name, "scripts.") || strings.HasPrefix(name, "assistants.") || strings.HasPrefix(name, "agents.") || strings.HasPrefix(name, "ai.") || strings.HasPrefix(name, "services.") {
		return true
	}

	name = strings.ToLower(name)
	return Handlers[name] != nil
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

// WithCallback set the callback function
func (process *Process) WithCallback(callback CallbackFunc) *Process {
	process.Callback = callback
	return process
}

// WithV8Context set the V8 context for thread affinity
func (process *Process) WithV8Context(v8ctx interface{}) *Process {
	process.V8Context = v8ctx
	return process
}

// String the process as string
func (process Process) String() string {
	args, _ := jsoniter.MarshalToString(process.Args)
	global, _ := jsoniter.MarshalToString(process.Global)
	return fmt.Sprintf("%s%s\n%s%s\n%s%s\n%s%s\n",
		color.YellowString("Process: "),
		color.WhiteString(process.Name),
		color.YellowString("Sid: "),
		color.WhiteString(process.Sid),
		color.YellowString("Args: \n"),
		color.WhiteString(args),
		color.YellowString("Global: \n"),
		color.WhiteString(global),
	)
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

	// The services scripts under the services directory
	case "services":
		if len(fields) < 3 {
			return fmt.Errorf("Exception|404:%s not found", process.Name)
		}

		// add scripts to the beginning of the fields
		fields = append([]string{"scripts"}, fields...)
		fields[1] = "__yao_service"
		process.Group = "scripts"

		// services.foo.Bar
		process.Handler = strings.ToLower(process.Group)
		process.ID = strings.ToLower(strings.ToLower(strings.Join(fields[1:len(fields)-1], ".")))
		process.Method = fields[len(fields)-1]
		break

	// The assistants scripts under the assistants directory
	case "agents", "assistants", "ai":
		if len(fields) < 3 {
			return fmt.Errorf("Exception|404:%s not found", process.Name)
		}

		// add scripts to the beginning of the fields
		fields = append([]string{"scripts"}, fields...)
		process.Group = "scripts"
		fields[1] = "assistants"

		// agents.foo.Bar
		process.Handler = strings.ToLower(process.Group)
		process.ID = strings.ToLower(strings.ToLower(strings.Join(fields[1:len(fields)-1], ".")))
		process.Method = fields[len(fields)-1]

	// the scripts under the scripts directory, or plugins under the plugins directory
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
