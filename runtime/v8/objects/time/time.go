package time

import (
	"fmt"
	"time"

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
	tmpl.Set("Sleep", obj.sleep(iso))
	return tmpl
}

// Set new obj instance
func (obj *Object) Set(name string, ctx *v8go.Context) error {
	tmpl := v8go.NewObjectTemplate(ctx.Isolate())
	tmpl.Set("Sleep", obj.sleep(ctx.Isolate()))

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

func (obj *Object) sleep(iso *v8go.Isolate) *v8go.FunctionTemplate {

	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("time.Sleep: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		ms := args[0].Integer()
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return v8go.Null(iso)
	})
}
