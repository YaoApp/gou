package use

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// ExportFunction exports the Use function for resource management
// Use(Constructor, ...args, callback) automatically calls __release() after callback execution
// Errors thrown inside the callback will properly propagate to the caller
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, exec)
}

// exec implements the Use function using V8 Go API
func exec(info *v8go.FunctionCallbackInfo) *v8go.Value {
	ctx := info.Context()
	args := info.Args()

	// Validate arguments: at least Constructor and callback
	if len(args) < 2 {
		return bridge.JsException(ctx, "Use() requires at least 2 arguments: Constructor and callback")
	}

	// First argument must be a function (Constructor)
	constructor := args[0]
	if !constructor.IsFunction() {
		return bridge.JsException(ctx, "Use() requires a constructor function as the first argument")
	}

	// Last argument must be a function (callback)
	callback := args[len(args)-1]
	if !callback.IsFunction() {
		return bridge.JsException(ctx, "Use() requires a callback function as the last argument")
	}

	// Middle arguments are constructor arguments
	constructorArgs := []v8go.Valuer{}
	if len(args) > 2 {
		for _, arg := range args[1 : len(args)-1] {
			constructorArgs = append(constructorArgs, arg)
		}
	}

	// Call constructor with 'new' to create instance
	constructorFunc, err := constructor.AsFunction()
	if err != nil {
		return bridge.JsException(ctx, "failed to get constructor function: "+err.Error())
	}

	instance, err := constructorFunc.NewInstance(constructorArgs...)
	if err != nil {
		return bridge.JsException(ctx, "failed to create instance: "+err.Error())
	}

	// Get the instance value and object
	instanceValue := instance.Value
	instanceObj, err := instanceValue.AsObject()
	if err != nil {
		return bridge.JsException(ctx, "failed to get instance object: "+err.Error())
	}

	// Call the callback with the instance
	callbackFunc, err := callback.AsFunction()
	if err != nil {
		return bridge.JsException(ctx, "failed to get callback function: "+err.Error())
	}

	result, err := callbackFunc.Call(v8go.Undefined(ctx.Isolate()), instanceValue)

	// Always call __release(), even if there was an error
	// This is safe because Function.Call() consumes the exception and returns it as a Go error
	// At this point, there is NO pending exception in V8, so calling __release() is allowed
	if releaseMethod, releaseErr := instanceObj.Get("__release"); releaseErr == nil && releaseMethod.IsFunction() {
		if releaseFunc, releaseErr := releaseMethod.AsFunction(); releaseErr == nil {
			_, _ = releaseFunc.Call(instanceObj) // Ignore errors in release
		}
	}

	// Now handle the error if there was one
	if err != nil {
		// Re-throw the exception to propagate it properly
		errVal, _ := v8go.NewValue(ctx.Isolate(), err.Error())
		return ctx.Isolate().ThrowException(errVal)
	}

	return result
}
