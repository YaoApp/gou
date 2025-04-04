package console

import (
	"fmt"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// Object Javascript API
type Object struct {
	mode string // production, development
}

// New create a new Console Object
func New(mode string) *Object {

	// validate mode
	if mode == "" || mode != "development" {
		mode = "production"
	}

	return &Object{
		mode: mode, // production, development
	}
}

// ExportObject Export as a Console Object
// console.log("name", {"foo":"bar"} )
func (obj *Object) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("log", obj.log(iso))
	tmpl.Set("info", obj.info(iso))
	tmpl.Set("warn", obj.warn(iso))
	tmpl.Set("error", obj.error(iso))
	return tmpl
}

// Set new obj instance
func (obj *Object) Set(name string, ctx *v8go.Context) error {
	tmpl := v8go.NewObjectTemplate(ctx.Isolate())
	tmpl.Set("log", obj.log(ctx.Isolate()))
	tmpl.Set("info", obj.info(ctx.Isolate()))
	tmpl.Set("warn", obj.warn(ctx.Isolate()))
	tmpl.Set("error", obj.error(ctx.Isolate()))

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

/**
 * Log
 *
 * @param args
 * @return *v8go.Value
 */
func (obj *Object) log(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if obj.mode != "development" {
			return v8go.Null(iso)
		}
		return obj.dump(info, helper.Dump)
	})
}

/**
 * Warn
 *
 * @param args
 * @return *v8go.Value
 */
func (obj *Object) warn(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return obj.dump(info, helper.DumpWarn)
	})
}

/**
 * Error
 *
 * @param args
 * @return *v8go.Value
 */
func (obj *Object) error(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return obj.dump(info, helper.DumpError)
	})
}

/**
 * Info
 *
 * @param args
 * @return *v8go.Value
 */
func (obj *Object) info(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return obj.dump(info, helper.DumpInfo)
	})
}

/**
 * Dump
 *
 * @param args
 * @return *v8go.Value
 */
func (obj *Object) dump(info *v8go.FunctionCallbackInfo, method func(...interface{})) *v8go.Value {
	args := info.Args()
	if len(args) < 1 {
		msg := fmt.Sprintf("console: Missing parameters")
		log.Error(msg)
		return bridge.JsException(info.Context(), msg)
	}

	goArgs := []interface{}{}
	var err error
	goArgs, err = bridge.GoValues(args, info.Context())
	if err != nil {
		msg := fmt.Sprintf("console: %s", err.Error())
		log.Error(msg)
		return bridge.JsException(info.Context(), msg)
	}
	method(goArgs...)
	return v8go.Null(info.Context().Isolate())
}
