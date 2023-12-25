package v8

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// Call call the script function
func (context *Context) Call(method string, args ...interface{}) (interface{}, error) {

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

	// Run the method
	jsArgs, err := bridge.JsValues(context.Context, args)
	if err != nil {
		return nil, err
	}
	defer bridge.FreeJsValues(jsArgs)

	jsRes, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)
	if err != nil {
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
		return nil, fmt.Errorf("%s.%s %s", context.ID, method, err.Error())

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

	context.Context.Close()
	context.Context = nil
	context.UnboundScript = nil
	context.Data = nil

	if runtimeOption.Mode == "standard" {
		context.Isolate.Dispose()
		context.Isolate = nil
		return nil
	}
	// Performance Mode
	context.Isolate.Unlock()
	context.Isolate = nil
	return nil
}
