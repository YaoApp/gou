package time

import (
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
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
	tmpl.Set("After", obj.after(iso))
	return tmpl
}

// Set new obj instance
func (obj *Object) Set(name string, ctx *v8go.Context) error {
	tmpl := v8go.NewObjectTemplate(ctx.Isolate())
	tmpl.Set("Sleep", obj.sleep(ctx.Isolate()))
	tmpl.Set("After", obj.after(ctx.Isolate()))

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

func (obj *Object) after(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("time.After: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		ms := args[0].Integer()
		name := args[1].String()
		goArgs := []interface{}{}

		if (len(args)) > 2 {
			jsArgs := args[2:]
			args, err := bridge.GoValues(jsArgs, info.Context())
			if err != nil {
				msg := fmt.Sprintf("time.After: %s", err.Error())
				log.Error(msg)
				return bridge.JsException(info.Context(), msg)
			}
			goArgs = args
		}

		go func() {
			select {
			case <-time.After(time.Duration(ms) * time.Millisecond):
				p, err := process.Of(name, goArgs...)
				if err != nil {
					log.Error("time.after %d: %s process error: %s, args: %v", ms, name, err.Error(), goArgs)
					return
				}
				if p == nil {
					log.Error("time.after %d: %s process error: %s, args: %v", ms, name, err.Error(), goArgs)
					return
				}

				res, err := p.Exec()
				if err != nil {
					log.Error("time.after %d: %s process error: %s, args: %v", ms, name, err.Error(), goArgs)
					return
				}
				log.Info("time.after %d: %s process result: %v, args: %v", ms, name, res, goArgs)
			}
		}()

		return v8go.Null(iso)
	})
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
