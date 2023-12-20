package v8

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// Call call the script function
func (ctx *Context) Call(method string, args ...interface{}) (interface{}, error) {

	global := ctx.Context.Global()
	jsArgs, err := bridge.JsValues(ctx.Context, args)
	if err != nil {
		return nil, fmt.Errorf("%s.%s %s", ctx.ID, method, err.Error())
	}

	defer bridge.FreeJsValues(jsArgs)

	jsData, err := ctx.setData(global)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !jsData.IsNull() && !jsData.IsUndefined() {
			jsData.Release()
		}
	}()

	jsRes, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)
	if err != nil {
		return nil, fmt.Errorf("%s.%s %+v", ctx.ID, method, err)
	}

	goRes, err := bridge.GoValue(jsRes, ctx.Context)
	if err != nil {
		return nil, fmt.Errorf("%s.%s %s", ctx.ID, method, err.Error())
	}

	return goRes, nil
}

// CallWith call the script function
func (ctx *Context) CallWith(context context.Context, method string, args ...interface{}) (interface{}, error) {

	global := ctx.Context.Global()
	jsArgs, err := bridge.JsValues(ctx.Context, args)
	if err != nil {
		return nil, fmt.Errorf("%s.%s %s", ctx.ID, method, err.Error())
	}

	defer bridge.FreeJsValues(jsArgs)

	jsData, err := ctx.setData(global)
	if err != nil {
		return nil, fmt.Errorf("%s.%s %s", ctx.ID, method, err.Error())
	}
	defer func() {
		if !jsData.IsNull() && !jsData.IsUndefined() {
			jsData.Release()
		}
	}()

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

			goRes, err := bridge.GoValue(jsRes, ctx.Context)
			if err != nil {
				errChan <- err
				return
			}

			resChan <- goRes
		}
	}()

	select {
	case <-context.Done():
		doneChan <- true
		return nil, context.Err()

	case err := <-errChan:
		return nil, fmt.Errorf("%s.%s %s", ctx.ID, method, err.Error())

	case goRes := <-resChan:
		return goRes, nil
	}
}

func (ctx *Context) setData(global *v8go.Object) (*v8go.Value, error) {
	goData := map[string]interface{}{
		"SID":  ctx.SID,
		"ROOT": ctx.Root,
		"DATA": ctx.Data,
	}

	jsData, err := bridge.JsValue(ctx.Context, goData)
	if err != nil {
		return nil, err
	}

	err = global.Set("__yao_data", jsData)
	if err != nil {
		return nil, err
	}

	return jsData, nil
}

// Close Context
func (ctx *Context) Close() error {
	ctx.Context.Close()
	ctx.Context = nil
	ctx.Data = nil
	ctx.SID = ""
	return ctx.Iso.Unlock()
}
