package exception

import (
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// Exception Javascript Exception
type Exception struct{}

// New create a new exception object
func New() *Exception {
	return &Exception{}
}

// ExportObject Export as a javascript Object
func (e *Exception) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Code", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		code, err := info.This().Get("code")
		if err != nil {
			log.Error("Exception: %s", err.Error())
		}
		return code
	}))

	tmpl.Set("Message", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		message, err := info.This().Get("message")
		if err != nil {
			log.Error("Exception: %s", err.Error())
		}
		return message
	}))
	return tmpl
}

// ExportFunction Export as a javascript function
func (e *Exception) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {

	inst := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		ctx := info.Context()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "Missing parameters")
		}

		var code, message *v8go.Value
		if len(args) == 1 {
			code, _ = v8go.NewValue(ctx.Isolate(), int32(500))
			message = args[0]
		} else if len(args) >= 2 {
			code = args[1]
			message = args[0]
		}

		global := info.Context().Global()
		errorObj, _ := global.Get("Error")
		if errorObj.IsFunction() {
			fn, err := errorObj.AsFunction()
			if err != nil {
				log.Error("Exception: %s", err.Error())
			}

			v, err := fn.Call(v8go.Undefined(ctx.Isolate()), message)
			if err != nil {
				log.Error("Exception: %s", err.Error())
			}

			obj, err := v.AsObject()
			if err != nil {
				log.Error("Exception: %s", err.Error())
			}

			// extend error object
			obj.Set("code", code)
			obj.Set("name", fmt.Sprintf("Exception|%v", code))
			return obj.Value
		}

		object := e.ExportObject(iso)
		this, err := object.NewInstance(info.Context())
		if err != nil {
			log.Error("Exception: %s", err.Error())
			return nil
		}
		err = this.Set("message", message)
		if err != nil {
			log.Error("Exception: %s", err.Error())
		}

		err = this.Set("code", code)
		if err != nil {
			log.Error("Exception: %s", err.Error())
		}

		err = this.Set("name", "Exception")
		if err != nil {
			log.Error("Exception: %s", err.Error())
		}
		return this.Value
	})

	return inst
}
