package v8

import (
	"rogchap.com/v8go"
)

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
	ctx.Iso.Unlock()
	ctx.Context = nil
	ctx.Data = nil
	ctx.SID = ""
	return nil
}
