package plan

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// Object Plan Object
type Object struct {
	id     string
	plan   *plan.Plan
	shared plan.SharedSpace
	ctx    *context.Context
	cancel context.CancelFunc
}

// New create a new Plan Object
func New() *Object {
	return &Object{}
}

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
	tmpl.Set("Add", obj.add(iso))
	tmpl.Set("Status", obj.status(iso))
	tmpl.Set("TaskStatus", obj.taskStatus(iso))
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

		// Create a new plan
		obj.id = id // Store the id
		ctx, cancel := context.WithCancel(context.Background())
		obj.ctx = &ctx
		obj.cancel = cancel

		shared := plan.NewMemorySharedSpace()
		obj.shared = shared
		obj.plan = plan.NewPlan(ctx, id, shared)

		// Set the id
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

		// Validate the task function
		if !args[2].IsFunction() {
			return bridge.JsException(info.Context(), "the task function should be a function")
		}

		// Get the task id
		taskID := args[0].String()

		// Get the order
		order := int(args[1].Int32())

		// Get the task function
		taskFn, err := args[2].AsFunction()
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("failed to get the task function %s", err.Error()))
		}

		// Add the task to the plan
		obj.plan.AddTask(taskID, order, func(ctx context.Context, shared plan.SharedSpace, signals <-chan plan.Signal) error {
			_, err := taskFn.Call(info.This())
			if err != nil {
				return fmt.Errorf("failed to call the task function %s", err.Error())
			}
			return nil
		})

		return info.This().Value
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
		status, err := obj.plan.GetTaskStatus(taskID)
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
		plan, tasks := obj.plan.GetStatus()

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
