package v8

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/objects/console"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

var reFuncHead = regexp.MustCompile(`\s*function\s+(\w+)\s*\(([^)]*)\)\s*\{`)

// Call call the script function
func (context *Context) Call(method string, args ...interface{}) (interface{}, error) {

	// Performance Mode
	if context.Runner != nil {
		return context.Runner.Exec(context.script), nil
	}

	// Set the global data
	global := context.Global()
	err := bridge.SetShareData(context.Context, global, &bridge.Share{
		Sid:    context.Sid,
		Root:   context.Root,
		Global: context.Data,
	})
	if err != nil {
		return nil, err
	}

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New().Set("console", context.Context)
	if err != nil {
		return nil, err
	}

	// Run the method
	jsArgs, err := bridge.JsValues(context.Context, args)
	if err != nil {
		return nil, err
	}
	defer bridge.FreeJsValues(jsArgs)

	jsRes, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)
	if err != nil {
		if e, ok := err.(*v8go.JSError); ok {
			PrintException(method, args, e, context.SourceRoots)
		}
		log.Error("%s.%s %s", context.ID, method, err.Error())
		return nil, err
	}

	goRes, err := bridge.GoValue(jsRes, context.Context)
	if err != nil {
		return nil, err
	}

	return goRes, nil
}

// CallAnonymous call the script function with anonymous function
func (context *Context) CallAnonymous(source string, args ...interface{}) (interface{}, error) {

	// Remove the function name from the source, if it exists regex
	source = reFuncHead.ReplaceAllString(source, "")
	name := fmt.Sprintf("__anonymous_%s", uuid.New().String())

	script, err := context.Isolate.CompileUnboundScript(source, name, v8go.CompileOptions{})
	if err != nil {
		return nil, err
	}

	fn, err := script.Run(context.Context)
	if err != nil {
		return nil, err
	}
	defer fn.Release()

	global := context.Global()
	global.Set(name, fn)
	defer global.Delete(name)

	res, err := context.Call(name, args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// CallAnonymousWith call the script function with anonymous function
func (context *Context) CallAnonymousWith(ctx context.Context, source string, args ...interface{}) (interface{}, error) {

	source = reFuncHead.ReplaceAllString(source, "($2) => {")
	name := fmt.Sprintf("__anonymous_%s", uuid.New().String())

	script, err := context.Isolate.CompileUnboundScript(source, name, v8go.CompileOptions{})
	if err != nil {
		return nil, err
	}

	fn, err := script.Run(context.Context)
	if err != nil {
		return nil, err
	}
	defer fn.Release()

	global := context.Global()
	global.Set(name, fn)
	defer global.Delete(name)

	return context.CallWith(ctx, name, args...)
}

// CallWith call the script function
func (context *Context) CallWith(ctx context.Context, method string, args ...interface{}) (interface{}, error) {

	// Performance Mode
	if context.Runner != nil {
		return context.Runner.Exec(context.script), nil
	}

	// Set the global data
	global := context.Global()
	err := bridge.SetShareData(context.Context, global, &bridge.Share{
		Sid:    context.Sid,
		Root:   context.Root,
		Global: context.Data,
	})
	if err != nil {
		return nil, err
	}

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New().Set("console", context.Context)
	if err != nil {
		return nil, err
	}

	// Run the method
	jsArgs, err := bridge.JsValues(context.Context, args)
	if err != nil {
		return nil, err
	}
	defer bridge.FreeJsValues(jsArgs)

	doneChan := make(chan bool, 1)
	resChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {

		defer func() {
			close(resChan)
			close(errChan)
		}()

		select {
		case <-doneChan:
			return

		default:

			jsRes, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)
			if err != nil {
				if e, ok := err.(*v8go.JSError); ok {
					PrintException(method, args, e, context.SourceRoots)
				}
				errChan <- err
				return
			}

			goRes, err := bridge.GoValue(jsRes, context.Context)
			if err != nil {
				errChan <- err
				return
			}

			resChan <- goRes
		}
	}()

	select {
	case <-ctx.Done():
		doneChan <- true
		return nil, ctx.Err()

	case err := <-errChan:
		log.Error("%s.%s %v", context.ID, method, err)
		return nil, err

	case goRes := <-resChan:
		return goRes, nil
	}
}

// WithFunction add a function to the context
func (context *Context) WithFunction(name string, cb v8go.FunctionCallback) {
	tmpl := v8go.NewFunctionTemplate(context.Isolate.Isolate, cb)
	context.Global().Set(name, tmpl.GetFunction(context.Context))
}

// WithGlobal add a global variable to the context
func (context *Context) WithGlobal(name string, value interface{}) error {
	switch value.(type) {
	case v8go.Valuer:
		context.Global().Set(name, value)
	default:
		jsValue, err := bridge.JsValue(context.Context, value)
		if err != nil {
			return err
		}
		context.Global().Set(name, jsValue)
	}
	return nil
}

// Close Context
func (context *Context) Close() error {

	// Standard Mode Release Isolate
	if runtimeOption.Mode == "standard" {
		context.Context.Close()
		context.Context = nil
		context.UnboundScript = nil
		context.Data = nil

		context.Isolate.Dispose()
		context.Isolate = nil
		return nil
	}

	// Performance Mode Release Runner
	if context.Runner != nil {
		context.Runner.Reset()
		context.Context = nil
		context.Data = nil
		context.Runner = nil
		return nil
	}

	return nil
}
