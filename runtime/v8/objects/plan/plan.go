package plan

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

var plans = map[string]*Instance{}

// KeepKeys are the keys that should not be set in the shared space
var keepKeys = map[string]bool{
	"TaskCompleted": true,
	"TaskError":     true,
	"TaskStarted":   true,
	"Released":      true,
}

// Object Plan Object
type Object struct {
}

// Instance is the instance of the plan
type Instance struct {
	id     string
	plan   *plan.Plan
	shared plan.SharedSpace
	ctx    *context.Context
	cancel context.CancelFunc
}

// New create a new Plan Object
func New() *Object {
	obj := &Object{}
	return obj
}

var globalSharedSpace = plan.NewMemorySharedSpace()

// StringStatus convert the plan status to a string
func StringStatus(status plan.Status) string {
	switch status {
	case plan.StatusRunning:
		return "running"
	case plan.StatusPaused:
		return "paused"
	case plan.StatusCompleted:
		return "completed"
	case plan.StatusFailed:
		return "failed"
	case plan.StatusDestroyed:
		return "destroyed"
	case plan.StatusCreated:
		return "created"
	case plan.StatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// ExportObject Export as a FS Object
func (obj *Object) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Run", obj.run(iso))         // Run the plan
	tmpl.Set("Add", obj.add(iso))         // Add a task to the plan
	tmpl.Set("Status", obj.status(iso))   // Get the status of the plan and each task
	tmpl.Set("Release", obj.release(iso)) // Release the plan

	// Tasks methods
	tmpl.Set("TaskStatus", obj.taskStatus(iso)) // Get the status of a task
	tmpl.Set("TaskData", obj.taskData(iso))     // Get or set the data of a task

	// Shared methods
	tmpl.Set("Subscribe", obj.subscribe(iso)) // Subscribe to the plan
	tmpl.Set("Set", obj.set(iso))             // Set a value in the shared space
	tmpl.Set("Get", obj.get(iso))             // Get a value from the shared space
	tmpl.Set("Del", obj.del(iso))             // Delete a value from the shared space
	tmpl.Set("Clear", obj.clear(iso))         // Clear the shared space
	return tmpl
}

// ExportFunction Export as a javascript Plan function
// const plan = new Plan("plan-id");
//
//	plan.Add("task-id", 1, function(task, shared) {
//		shared.Set("foo", "bar");
//	});
func (obj *Object) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := obj.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "the first parameter should be a string")
		}

		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the first parameter should be a string")
		}

		id := args[0].String()
		this, err := object.NewInstance(info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to create plan object %s", err.Error()))
		}

		// Get the existing plan
		if _, ok := plans[id]; ok {
			this.Set("id", id)
			return this.Value
		}

		// Create a new plan
		instance := &Instance{id: id}
		ctx, cancel := context.WithCancel(context.Background())
		instance.ctx = &ctx
		instance.cancel = cancel

		shared := plan.NewMemorySharedSpace()
		instance.shared = shared
		instance.plan = plan.NewPlan(ctx, id, shared)

		// Create a new object for the task functions
		// Store the plan object in the plans map
		plans[id] = instance // Store the plan object in the plans map
		this.Set("id", id)
		return this.Value
	})
	return tmpl
}

func (obj *Object) add(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 3 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		// Validate the task id
		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the task id should be a string")
		}

		// Validate the order
		if !args[1].IsNumber() {
			return bridge.JsException(info.Context(), "the order should be a number")
		}

		// Validate the task process name
		if !args[2].IsString() {
			return bridge.JsException(info.Context(), "the task function should be a string")
		}

		// Get the task id
		taskID := args[0].String()

		// Get the order
		order := int(args[1].Int32())

		// Get the task function
		method := args[2].String()

		// Validate the process exists
		if !strings.HasPrefix(method, "scripts.") && !process.Exists(method) {
			return bridge.JsException(info.Context(), fmt.Sprintf("the process %s does not exist", method))
		}

		// The rest of the arguments are the arguments to the process
		rest := make([]interface{}, 0)
		for i := 3; i < len(args); i++ {
			v, err := bridge.GoValue(args[i], info.Context())
			if err != nil {
				return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the value %s", err.Error()))
			}
			rest = append(rest, v)
		}

		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}
		instance := plans[id]

		// Add the task to the plan
		instance.plan.AddTask(taskID, order, func(ctx context.Context, shared plan.SharedSpace, signals <-chan plan.Signal) error {

			// Trigger the TaskStarted signal
			instance.plan.Trigger("TaskStarted", map[string]any{"task": taskID})
			args := []interface{}{instance.plan.ID, taskID}
			args = append(args, rest...)

			p, err := process.Of(method, args...)
			if err != nil {
				instance.plan.Trigger("TaskError", map[string]any{"message": err.Error(), "task": taskID})
				return err
			}
			defer p.Release()

			err = p.Execute()
			if err != nil {
				instance.plan.Trigger("TaskError", map[string]any{"message": err.Error(), "task": taskID})
				return err
			}

			result := p.Value()
			instance.plan.Trigger("TaskCompleted", map[string]any{"task": taskID, "result": result})
			return nil
		})

		return info.This().Value
	})
}

func (obj *Object) run(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		err = instance.plan.Start()
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to start the plan %s", err.Error()))
		}
		return info.This().Value
	})
}

func (obj *Object) release(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		instance.plan.Release()
		delete(plans, id)
		return nil
	})
}

// ID get the id of the plan
func (obj *Object) ID(info *v8go.FunctionCallbackInfo) (string, error) {
	id, err := info.This().Get("id")
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (obj *Object) subscribe(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		// Validate the key
		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the key should be a string")
		}

		// Validate the callback
		if !args[1].IsString() {
			return bridge.JsException(info.Context(), "the callback should be a string")
		}

		// Get the key
		key := args[0].String()
		method := args[1].String()

		// Validate the process exists
		if !strings.HasPrefix(method, "scripts.") && !process.Exists(method) {
			return bridge.JsException(info.Context(), fmt.Sprintf("the process %s does not exist", method))
		}

		// The rest of the arguments are the arguments to the process
		rest := make([]interface{}, 0)
		for i := 2; i < len(args); i++ {
			v, err := bridge.GoValue(args[i], info.Context())
			if err != nil {
				return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the value %s", err.Error()))
			}
			rest = append(rest, v)
		}

		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		// Subscribe to the key
		instance := plans[id]
		err = instance.shared.Subscribe(key, func(key string, value interface{}) {
			args := []interface{}{instance.plan.ID, key, value}
			args = append(args, rest...)
			p, err := process.Of(method, args...)
			if err != nil {
				return
			}
			defer p.Release()
			p.Execute()
		})

		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to subscribe to the key %s", err.Error()))
		}

		return nil
	})
}

func (obj *Object) set(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		// Validate the key
		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the key should be a string")
		}

		// Get the key
		key := args[0].String()

		// Keep key as a string
		if _, ok := keepKeys[key]; ok {
			return bridge.JsException(info.Context(), "the key should not be a reserved key")
		}
		// Get the value
		value := args[1]
		goValue, err := bridge.GoValue(value, info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the value %s", err.Error()))
		}

		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		err = instance.shared.Set(key, goValue)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to set the value %s", err.Error()))
		}
		return nil
	})
}

func (obj *Object) get(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		// Validate the key
		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the key should be a string")
		}

		// Get the key
		key := args[0].String()

		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		value, err := instance.shared.Get(key)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the value %s", err.Error()))
		}

		output, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the value %s", err.Error()))
		}
		return output
	})
}

func (obj *Object) del(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		// Validate the key
		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the key should be a string")
		}

		// Get the key
		key := args[0].String()

		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		err = instance.shared.Delete(key)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to delete the value %s", err.Error()))
		}
		return nil
	})
}

func (obj *Object) clear(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		err = instance.shared.Clear()
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to clear the shared space %s", err.Error()))
		}
		return nil
	})
}

func (obj *Object) taskData(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		// Validate the task id
		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the task id should be a string")
		}

		taskID := args[0].String()

		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		// Get the task
		instance := plans[id]
		task, exists := instance.plan.Tasks[args[0].String()]
		if !exists {
			return bridge.JsException(info.Context(), fmt.Sprintf("the task %s does not exist", taskID))
		}

		// Get the data
		if len(args) < 2 {
			res, err := bridge.JsValue(info.Context(), task.Data)
			if err != nil {
				return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the data %s", err.Error()))
			}
			return res
		}

		// Get the data from the argument
		data := args[1]
		goData, err := bridge.JsValue(info.Context(), data)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the data %s", err.Error()))
		}

		// Set the data
		task.Data = goData
		return nil
	})
}

// Get the status of a task
func (obj *Object) taskStatus(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		// Validate the task id
		if !args[0].IsString() {
			return bridge.JsException(info.Context(), "the task id should be a string")
		}

		taskID := args[0].String()

		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		status, err := instance.plan.GetTaskStatus(taskID)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the status %s", err.Error()))
		}

		output, err := bridge.JsValue(info.Context(), StringStatus(status))
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the status %s", err.Error()))
		}
		return output
	})
}

// Get the status of the plan and each task
func (obj *Object) status(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		id, err := obj.ID(info)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the id %s", err.Error()))
		}

		instance := plans[id]
		plan, tasks := instance.plan.GetStatus()

		// Convert the plan status to a string
		tasksStatus := make(map[string]string)
		if tasks != nil {
			for taskID, status := range tasks {
				tasksStatus[taskID] = StringStatus(status)
			}
		}
		output, err := bridge.JsValue(info.Context(), map[string]any{"plan": StringStatus(plan), "tasks": tasksStatus})
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the status %s", err.Error()))
		}
		return output
	})
}
