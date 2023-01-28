package studio

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// ExportFunction function template
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, exec)
}

// exec
func exec(info *v8go.FunctionCallbackInfo) *v8go.Value {

	v, err := info.Context().Global().Get("__YAO_SU_ROOT")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !v.Boolean() {
		return bridge.JsException(info.Context(), "function is not allowed")
	}

	jsGlobal, err := info.Context().Global().Get("__yao_global")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	jsSID, _ := info.Context().Global().Get("__yao_sid")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	goGlobal, err := bridge.GoValue(jsGlobal)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	global, ok := goGlobal.(map[string]interface{})
	if !ok {
		global = map[string]interface{}{}
	}

	jsArgs := info.Args()
	if len(jsArgs) < 1 {
		return bridge.JsException(info.Context(), "missing parameters")
	}

	if !jsArgs[0].IsString() {
		return bridge.JsException(info.Context(), "the first parameter should be a string")
	}

	goArgs := []interface{}{}
	if len(jsArgs) > 1 {
		goArgs, err = bridge.GoValues(jsArgs[1:])
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}
	}

	goRes, err := process.New(jsArgs[0].String(), goArgs...).
		WithGlobal(global).
		WithSID(jsSID.String()).
		Exec()

	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	jsRes, err := bridge.JsValue(info.Context(), goRes)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	return jsRes
}
