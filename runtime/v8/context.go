package v8

import (
	"context"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/objects/console"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

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
		log.Error("%s.%s %s", context.ID, method, err.Error())
		return nil, err
	}

	goRes, err := bridge.GoValue(jsRes, context.Context)
	if err != nil {
		return nil, err
	}

	return goRes, nil
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
		log.Error("%s.%s %s", context.ID, method, err.Error())
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
