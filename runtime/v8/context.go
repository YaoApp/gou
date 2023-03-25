package v8

import (
	"time"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// NewContext create a new context
func (script *Script) NewContext(sid string, global map[string]interface{}) (*Context, error) {

	timeout := script.Timeout
	if timeout == 0 {
		timeout = 100 * time.Millisecond
	}

	iso, err := SelectIso(timeout)
	if err != nil {
		return nil, err
	}

	var context *v8go.Context
	var has bool

	// load from cache
	context, has = iso.contexts[script]

	// re-compile and save to cache
	if !has {
		context, err = script.Compile(iso, timeout)
		if err != nil {
			iso.Unlock() // unlock iso
			return nil, err
		}
	}

	return &Context{
		ID:      script.ID,
		Context: context,
		SID:     sid,
		Data:    global,
		Root:    script.Root,
		Iso:     iso,
	}, nil
}

// Call call the script function
func (ctx *Context) Call(method string, args ...interface{}) (interface{}, error) {

	global := ctx.Context.Global()
	jsArgs, err := bridge.JsValues(ctx.Context, args)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	goRes, err := bridge.GoValue(jsRes)
	if err != nil {
		return nil, err
	}

	return goRes, nil
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
	defer ctx.Iso.Unlock()
	ctx.Data = nil
	ctx.SID = ""
	return nil
}
