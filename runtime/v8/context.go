package v8

import (
	"fmt"
	"time"

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
			return nil, err
		}
	}

	return &Context{
		ID:      script.ID,
		Context: context,
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

		fmt.Println("---", err)
		return nil, err
	}

	return res.String(), nil
}

// Close Context
func (ctx *Context) Close() error {
	defer ctx.Iso.Unlock()
	// ctx.Context.Close()
	ctx.Context = nil
	ctx.Data = nil
	ctx.SID = ""

	return nil
}
