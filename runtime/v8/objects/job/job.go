package job

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

var jobs = sync.Map{}

// Object Javascript API
type Object struct{}

// Job Job struct
type Job struct {
	id      string
	status  uint8
	process string
	args    []interface{}
	data    interface{}
	cancel  context.CancelFunc
	err     error
	created int64
}

const (
	// StatusCreate Job status: create
	StatusCreate uint8 = iota
	// StatusRunning Job status: running
	StatusRunning
	// StatusDone Job status: done
	StatusDone
)

// New create a new FS Object
func New() *Object {
	return &Object{}
}

// ExportObject Export as a FS Object
func (obj *Object) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Pending", obj.pending(iso))
	tmpl.Set("Data", obj.data(iso))
	tmpl.Set("Cancel", obj.cancel(iso))
	return tmpl
}

// ExportFunction Export as a javascript FS function
// var job = new Job(processName, args...);
func (obj *Object) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := obj.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		jsArgs := info.Args()
		if len(jsArgs) < 1 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		if !jsArgs[0].IsString() {
			return bridge.JsException(info.Context(), "the first parameter should be a string")
		}

		var err error
		goArgs := []interface{}{}
		if len(jsArgs) > 2 {
			goArgs, err = bridge.GoValues(jsArgs[1:])
			if err != nil {
				return bridge.JsException(info.Context(), err)
			}
		}

		this, err := object.NewInstance(info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), err.Error())
		}

		id := uuid.New().String()
		exec := jsArgs[0].String()
		ctx, cancel := context.WithCancel(context.Background())

		jobs.Store(id, &Job{
			id:      id,
			status:  StatusRunning,
			process: exec,
			args:    goArgs,
			created: time.Now().UnixNano(),
			cancel:  cancel,
			data:    nil,
			err:     nil,
		})

		_, global, sid, v := bridge.ShareData(info.Context())
		if v != nil {
			return v
		}

		go func() {
			select {
			case <-ctx.Done():
				return
			default:
				goRes, err := process.New(exec, goArgs...).
					WithGlobal(global).
					WithSID(sid).
					Exec()
				jobs.Store(id, &Job{
					id:      id,
					status:  StatusDone,
					process: exec,
					args:    goArgs,
					created: time.Now().UnixNano(),
					data:    goRes,
					err:     err,
				})
			}
		}()

		this.Set("id", id)
		return this.Value
	})
	return tmpl
}

func (obj *Object) pending(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "the first parameter should be a string")
		}

		if !args[0].IsFunction() {
			return bridge.JsException(info.Context(), "the first parameter should be a function")
		}

		id, err := info.This().Get("id")
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}

		cbFun, err := args[0].AsFunction()
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}

		var v interface{}
		var job *Job
		var ok bool = false

		v, ok = jobs.Load(id.String())
		if !ok {
			return v8go.Undefined(info.Context().Isolate())
		}

		job, ok = v.(*Job)
		if !ok {
			return v8go.Undefined(info.Context().Isolate())
		}

		for job.status == StatusRunning {

			v, ok = jobs.Load(id.String())
			if !ok {
				return v8go.Undefined(info.Context().Isolate())
			}

			job, ok = v.(*Job)
			if !ok {
				return v8go.Undefined(info.Context().Isolate())
			}

			res, err := cbFun.Call(v8go.Undefined(info.Context().Isolate()))
			if err != nil {
				return bridge.JsException(info.Context(), err)
			}

			if res.IsBoolean() && !res.Boolean() {
				break
			}
		}

		return v8go.Undefined(info.Context().Isolate())
	})
}

func (obj *Object) data(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		id, err := info.This().Get("id")
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}

		v, ok := jobs.Load(id.String())
		if !ok {
			return v8go.Undefined(info.Context().Isolate())
		}

		job, ok := v.(*Job)
		if !ok {
			return v8go.Undefined(info.Context().Isolate())
		}

		jsData, err := bridge.JsValue(info.Context(), job.data)
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}

		defer jobs.Delete(id.String())
		return jsData
	})
}

func (obj *Object) cancel(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		id, err := info.This().Get("id")
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}

		v, ok := jobs.Load(id.String())
		if !ok {
			return v8go.Undefined(info.Context().Isolate())
		}

		job, ok := v.(*Job)
		if !ok {
			return v8go.Undefined(info.Context().Isolate())
		}

		job.cancel()
		defer jobs.Delete(id.String())
		return v8go.Undefined(info.Context().Isolate())
	})
}
