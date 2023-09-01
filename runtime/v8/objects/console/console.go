package console

import (
	"fmt"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// Object Javascript API
type Object struct{}

// New create a new Console Object
func New() *Object {
	return &Object{}
}

// ExportObject Export as a Console Object
// console.log("name", {"foo":"bar"} )
func (obj *Object) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("log", obj.run(iso))
	return tmpl
}

// Set new obj instance
func (obj *Object) Set(name string, ctx *v8go.Context) error {
	tmpl := v8go.NewObjectTemplate(ctx.Isolate())
	tmpl.Set("log", obj.run(ctx.Isolate()))

	instance, err := tmpl.NewInstance(ctx)
	if err != nil {
		return err
	}

	err = ctx.Global().Set(name, instance)
	if err != nil {
		return err
	}
	return nil
}

func (obj *Object) run(iso *v8go.Isolate) *v8go.FunctionTemplate {

	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("console.log: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		goArgs := []interface{}{}
		var err error
		if len(args) > 0 {
			goArgs, err = bridge.GoValues(args, info.Context())
			if err != nil {
				msg := fmt.Sprintf("console.log: %s", err.Error())
				log.Error(msg)
				return bridge.JsException(info.Context(), msg)
			}

			helper.Dump(goArgs...)
		}

		return v8go.Null(iso)
	})
}
