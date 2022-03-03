package values

import (
	"rogchap.com/v8go"
)

// Error Return javascript error object
func Error(ctx *v8go.Context, message string) *v8go.Value {
	global := ctx.Global()
	errorObj, _ := global.Get("Error")
	if errorObj.IsFunction() {
		fn, _ := errorObj.AsFunction()
		m, _ := v8go.NewValue(ctx.Isolate(), message)
		v, _ := fn.Call(v8go.Undefined(ctx.Isolate()), m)
		return v
	}

	tmpl := v8go.NewObjectTemplate(ctx.Isolate())
	inst, _ := tmpl.NewInstance(ctx)
	inst.Set("message", message)
	return inst.Value
}
