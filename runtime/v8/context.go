package v8

import (
	"fmt"
	"time"

	"rogchap.com/v8go"
)

var contexts = map[*Isolate]map[string]*v8go.Context{}

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

	ctx, ok := contexts[iso][script.ID]
	if !ok {
		return nil, fmt.Errorf("context not compiled")
	}

	return &Context{
		ID:      script.ID,
		Context: ctx,
		SID:     sid,
		Data:    global,
		Iso:     iso,
	}, nil

}

// Call call the script function
func (ctx *Context) Call(method string, args ...interface{}) (interface{}, error) {
	global := ctx.Global()
	arg, err := v8go.NewValue(ctx.Isolate(), "world")
	if err != nil {
		return nil, err
	}

	// fmt.Println("method:", method, arg)
	res, err := global.MethodCall("Hello", arg)
	if err != nil {
		return nil, err
	}

	return res.String(), nil
}

// Close Context
func (ctx *Context) Close() error {
	// ctx.Context.Close()
	ctx.Context = nil
	ctx.Data = nil
	ctx.SID = ""
	ctx.Iso.Unlock()
	return nil
}
