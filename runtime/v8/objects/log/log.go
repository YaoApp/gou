package log

import (
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// Object Javascript API
type Object struct{}

// New create a new Log Object
func New() *Object {
	return &Object{}
}

// ExportObject Export as a Log Object
// log.Trace("%s %v", "name", {"foo":"bar"} )
// log.Error("%s %v", "name", {"foo":"bar"} )
func (obj *Object) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Trace", obj.run(iso, log.TraceLevel))
	tmpl.Set("Debug", obj.run(iso, log.DebugLevel))
	tmpl.Set("Info", obj.run(iso, log.InfoLevel))
	tmpl.Set("Warn", obj.run(iso, log.WarnLevel))
	tmpl.Set("Error", obj.run(iso, log.ErrorLevel))
	tmpl.Set("Fatal", obj.run(iso, log.FatalLevel))
	tmpl.Set("Panic", obj.run(iso, log.PanicLevel))
	return tmpl
}

func (obj *Object) run(iso *v8go.Isolate, level log.Level) *v8go.FunctionTemplate {

	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Log: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		message := args[0].String()
		values := []interface{}{}
		var err error
		if len(args) > 1 {
			values, err = bridge.GoValues(args[1:], info.Context())
			if err != nil {
				msg := fmt.Sprintf("Log: %s", err.Error())
				log.Error(msg)
				return bridge.JsException(info.Context(), msg)
			}
		}

		switch level {
		case log.TraceLevel:
			log.Trace(message, values...)
			break
		case log.DebugLevel:
			log.Debug(message, values...)
			break
		case log.InfoLevel:
			log.Info(message, values...)
			break
		case log.WarnLevel:
			log.Warn(message, values...)
			break
		case log.ErrorLevel:
			log.Error(message, values...)
			break
		case log.FatalLevel:
			log.Fatal(message, values...)
			break
		case log.PanicLevel:
			log.Panic(message, values...)
			break
		default:
			log.Error(message, values...)
		}

		return v8go.Null(iso)
	})
}
